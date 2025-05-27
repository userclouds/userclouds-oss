package create

import (
	"net/http"
	"slices"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/messaging/email"
	"userclouds.com/internal/auditlog"
	message "userclouds.com/internal/messageelements"
	"userclouds.com/plex"
	"userclouds.com/plex/internal/addauthn"
	"userclouds.com/plex/internal/invite"
	"userclouds.com/plex/internal/loginapp"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
)

type handler struct {
	emailClient *email.Client
	factory     provider.Factory
}

// NewHandler returns a new create-user handler for plex
func NewHandler(email *email.Client, factory provider.Factory) http.Handler {
	h := &handler{email, factory}

	hb := builder.NewHandlerBuilder()
	handlerBuilder(hb, h)

	hb.HandleFunc("/submit", h.createUserHandler)
	hb.HandleFunc("/grantordenyauthnaddpermission", h.grantOrDenyAuthnAddPermission)

	return hb.Build()
}

//go:generate genhandler /create

func (h *handler) createUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req plex.CreateUserRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "MarshalError")
		return
	}
	if err := req.Validate(); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "InvalidEmail", jsonapi.Code(http.StatusBadRequest))
		return
	}
	s := tenantconfig.MustGetStorage(ctx)

	var validInvite bool
	var inviteEmail string

	var session *storage.OIDCLoginSession
	var err error
	if req.SessionID != uuid.Nil {
		session, err = s.GetOIDCLoginSession(ctx, req.SessionID)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "GetSession")
			return
		}

		validInvite, inviteEmail, err = invite.CheckForValidInvite(ctx, session)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "InvalidInvite")
			return
		}
	}

	tc := tenantconfig.MustGet(ctx)
	if tc.DisableSignUps && !validInvite && !slices.Contains(tc.BootstrapAccountEmails, req.Email) {
		jsonapi.MarshalErrorL(ctx, w, ucerr.New("tenant does not allow new user sign ups (except with a valid invite)"), "SignUpsDisabled", jsonapi.Code(http.StatusBadRequest))
		return
	}
	if tc.VerifyEmails && h.emailClient == nil {
		jsonapi.MarshalErrorL(ctx, w, ucerr.Friendlyf(nil, "Email not available, cannot create user when verify email is required."), "VerifyEmailsDisabled", jsonapi.Code(http.StatusNotImplemented))
		return
	}
	emailClient := tc.PlexMap.GetEmailClient(*h.emailClient)

	prov, err := provider.NewActiveManagementClient(ctx, h.factory, req.ClientID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetProviderError")
		return
	}

	uclog.Debugf(ctx, "got provider: %T", prov)

	// Check if the email is already in use by another account
	authns, err := addauthn.CheckForExistingAccounts(ctx, session, req.Email, req.Password, s, prov)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "CheckForExistingAccountsError")
		return
	}
	if authns != nil {
		// We found existing accounts, so return them and allow user to opt to add this authn to one of them
		jsonapi.Marshal(w, authns, jsonapi.Code(http.StatusAccepted))
		return
	}

	// If we've made it here, we should be able to create the new user successfully
	profile := iface.UserProfile{
		UserBaseProfile: idp.UserBaseProfile{
			Email:         req.Email,
			EmailVerified: req.Email == inviteEmail,
			Name:          req.Name,
			Nickname:      req.Nickname,
			Picture:       req.Picture,
		},
	}

	userID, err := prov.CreateUserWithPassword(ctx, req.Username, req.Password, profile)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "CreateAccountError")
		return
	}

	app, _, err := tc.PlexMap.FindAppForClientID(req.ClientID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FindApp")
		return
	}

	if err := loginapp.AddLoginAccessForUserIfNecessary(ctx, tc, app, userID); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "AddLoginAccessForUserIfNecessary")
		return
	}

	// At this point the account is created in the primary IDP and the rest of the flow deals with email verification, marking invite as used
	// and creating account in the secondary IDPs
	auditlog.Post(ctx, auditlog.NewEntry(userID, auditlog.AccountCreated,
		auditlog.Payload{"ID": app.ID, "Name": app.Name, "InviteEmail": inviteEmail, "Actor": req.Username, "Provider": req.ClientID}))

	if validInvite {
		if err := otp.BindInviteToUser(ctx, s, session, userID, req.Username, app); err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "BindInvite")
			return
		}
	}

	followers, err := provider.NewFollowerManagementClients(ctx, h.factory, req.ClientID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GetFollowerError")
		return
	}

	// TODO: if any of these fail, we should do something smarter than bail with an error
	for _, b := range followers {
		if _, err := b.CreateUserWithPassword(ctx, req.Username, req.Password, profile); err != nil {
			uclog.Warningf(ctx, "create user '%s' failed on follower provider (desc: '%s'): %v", req.Username, b, err)
			jsonapi.MarshalErrorL(ctx, w, err, "FollowerCreateFailure")
			return
		}
		// Add the audit entry for account creation in every secondary IDP
		auditlog.Post(ctx, auditlog.NewEntry(userID, auditlog.AccountCreated,
			auditlog.Payload{"ID": app.ID, "Name": app.Name, "InviteEmail": inviteEmail, "Actor": req.Username, "Provider": "Follower"}))
	}

	if tc.VerifyEmails && !profile.EmailVerified {
		// TODO: refactor email verification into a shared method since it can be used from other places.
		verifySessionID, otpCode, err := otp.CreateAccountVerificationSession(ctx, s, req.ClientID, userID, req.Email)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "StartFlowFail")
			return
		}

		uclog.InfofPII(ctx, "client '%s' (app '%s') has VerifyEmails=true, sending verification email to '%s'",
			req.ClientID, app.Name, req.Email)

		if err := otp.SendOTPEmailUI(ctx, emailClient, verifySessionID, app.Name, req.Email, app.MakeElementGetter(message.EmailVerifyEmail), otpCode); err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "SendEmailError")
			return
		}

		// TODO: Maybe return a payload indicating that an account verification email was sent?
	}

	jsonapi.Marshal(w, plex.CreateUserResponse{UserID: userID}, jsonapi.Code(http.StatusCreated))
}

// GrantOrDenyAuthnAddPermissionRequest is the request body for the grantOrDenyAuthnAddPermission handler.
type GrantOrDenyAuthnAddPermissionRequest struct {
	SessionID  uuid.UUID `json:"session_id"`
	Permission bool      `json:"permission"`
}

func (h *handler) grantOrDenyAuthnAddPermission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req GrantOrDenyAuthnAddPermissionRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GrantOrDenyPermsBadRequest")
		return
	}

	s := tenantconfig.MustGetStorage(ctx)

	session, err := s.GetOIDCLoginSession(ctx, req.SessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GrantOrDenyPermsGetSession")
		return
	}

	if session.AddAuthnProviderData.NewProviderAuthnType == "" {
		jsonapi.MarshalErrorL(ctx, w, ucerr.New("session does not have a valid AddAuthnProviderData"), "GrantOrDenyPermsInvalidSession")
		return
	}

	if req.Permission {
		session.AddAuthnProviderData.PermissionToAdd = true
		session.AddAuthnProviderData.DoNotAdd = false
	} else {
		session.AddAuthnProviderData.PermissionToAdd = false
		session.AddAuthnProviderData.DoNotAdd = true
	}

	if err := s.SaveOIDCLoginSession(ctx, session); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "GrantOrDenyPermsSaveSessionError")
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(http.StatusAccepted))
}
