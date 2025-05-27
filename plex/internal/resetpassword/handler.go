package resetpassword

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/messaging/email"
	"userclouds.com/internal/auditlog"
	message "userclouds.com/internal/messageelements"
	"userclouds.com/plex"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/reactdev"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
)

type handler struct {
	email   email.Client
	factory provider.Factory
}

// these constants control how often a single email address can be used to reset a password.
const resetNumberLimit = 5
const resetTimeLimit = time.Minute * 30

// NewHandler returns a new password-reset handler for plex
// TODO: when we have other credential types besides password we can generalize this.
func NewHandler(email email.Client, factory provider.Factory) http.Handler {
	h := &handler{email, factory}

	hb := builder.NewHandlerBuilder()
	handlerBuilder(hb, h)

	// Validates a request then redirects to UI to enter an email to which a reset password link is sent.
	// TODO: eventually support phone, reset codes, etc.
	hb.HandleFunc("/start", h.startScreenHandler)

	// POST-only API handler to start the reset password flow.
	hb.MethodHandler("/startsubmit").Post(h.startSubmitHandler)

	// POST-only API handler to process a reset password
	hb.MethodHandler("/resetsubmit").Post(h.resetSubmitHandler)

	return hb.Build()
}

//go:generate genhandler /resetpassword

func (h *handler) startScreenHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sessionIDStr := r.URL.Query().Get("session_id")
	sessionID, err := uuid.FromString(sessionIDStr)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	// Handle the password reset flow start on the server so we can throttle, log, etc
	// before handing control to client-side React UI.
	query := url.Values{"session_id": []string{sessionID.String()}}
	u := reactdev.UIBaseURL(ctx)
	u.Path = u.Path + paths.StartResetPasswordUISubPath
	u.RawQuery = query.Encode()

	uchttp.Redirect(w, r, u.String(), http.StatusSeeOther)
}

func (h *handler) startSubmitHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req plex.PasswordResetStartRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "MarshalError")
		return
	}

	uclog.Infof(ctx, "reset password request for %s", req.Email)

	// TODO: use security "checker" to throttle?

	s := tenantconfig.MustGetStorage(ctx)
	session, err := s.GetOIDCLoginSession(ctx, req.SessionID)
	if err != nil {
		uchttp.ErrorL(ctx, w, ucerr.Errorf("can't find login session for id '%s': %w", req.SessionID, err), http.StatusBadRequest, "BadSession")
		return
	}

	tc := tenantconfig.MustGet(ctx)
	app, _, err := tc.PlexMap.FindAppForClientID(session.ClientID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FindApp")
		return
	}

	activeClient, err := provider.NewActiveManagementClient(ctx, h.factory, session.ClientID)
	if err != nil {
		// TODO: differentiate error types (issue #103).
		jsonapi.MarshalErrorL(ctx, w, err, "ProviderInitErr")
		return
	}

	// very simple rate limiting on a per-email basis
	// TODO: this should be generalized code some day (soon?) but this solves our pen test issue
	tokens, err := s.ListRecentOTPStates(ctx, storage.OTPPurposeResetPassword, req.Email, resetNumberLimit)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetOTPStates")
		return
	}

	if len(tokens) >= resetNumberLimit && tokens[0].Created.Add(resetTimeLimit).After(time.Now().UTC()) {
		jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("too many recent reset password attempts for %s", req.Email), "TooManyAttempts", jsonapi.Code(http.StatusTooManyRequests))
		return
	}

	// TODO: try to find existing token for a given email?
	otpCode, err := otp.StartPasswordResetFlow(ctx, activeClient, s, session, req.Email)
	if err != nil {
		if errors.Is(err, otp.ErrNoUserForEmail) {
			finishStartSubmitHandler(w)
			return
		}
		jsonapi.MarshalErrorL(ctx, w, err, "StartOTPFlow")
		return
	}

	if err := otp.SendOTPEmail(ctx, tc.PlexMap.GetEmailClient(h.email), req.SessionID, app.Name, req.Email, app.MakeElementGetter(message.EmailResetPassword), otpCode); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "SendEmailError")
		return
	}

	finishStartSubmitHandler(w)
}

// in case we make this fancier in the future, always call this to prevent leaking data
func finishStartSubmitHandler(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) resetSubmitHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req plex.PasswordResetSubmitRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "MarshalError")
		return
	}

	s := tenantconfig.MustGetStorage(ctx)
	session, err := s.GetOIDCLoginSession(ctx, req.SessionID)
	if err != nil {
		uclog.Debugf(ctx, "invalid session ID specified: %s", req.SessionID)
		jsonapi.MarshalErrorL(ctx, w, ucerr.New("invalid session_id specified"), "InvalidID", jsonapi.Code(http.StatusBadRequest))
		return
	}

	if session.OTPStateID.IsNil() {
		jsonapi.MarshalErrorL(ctx, w, ucerr.New("invalid session"), "InvalidSession")
		return
	}

	otpState, err := s.GetOTPState(ctx, session.OTPStateID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, ucerr.New("invalid session"), "InvalidSession2")
		return
	}

	// Wrong code or used/expired code is all the same to us
	if otpState.Purpose != storage.OTPPurposeResetPassword || req.OTPCode != otpState.Code || otpState.Used || time.Now().UTC().After(otpState.Expires) {
		// Return 400 to be consistent with bad username/password login
		// See https://stackoverflow.com/questions/22586825/oauth-2-0-why-does-the-authorization-server-return-400-instead-of-401-when-the
		jsonapi.MarshalErrorL(ctx, w, ucerr.New("invalid code"), "BadCode", jsonapi.Code(http.StatusBadRequest))
		return
	}

	// Mark token used
	otpState.Used = true
	if err := s.SaveOTPState(ctx, otpState); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "SaveSession")
		return
	}

	activeClient, err := provider.NewActiveManagementClient(ctx, h.factory, session.ClientID)
	if err != nil {
		// TODO: differentiate error types (issue #103).
		jsonapi.MarshalErrorL(ctx, w, err, "FailedProviderGet")
		return
	}

	user, err := activeClient.GetUser(ctx, otpState.UserID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedGetUser")
		return
	}

	var username string
	for _, authn := range user.Authns {
		// TODO: what if an account is merged and there's more than one username+password associated
		// with the account? Logically we might expect one of the usernames to match the email
		// address used to reset the password, in which case we should just change that password?
		// Or should we change ALL passwords associated with this user?
		if authn.AuthnType == idp.AuthnTypePassword {
			username = authn.Username
			break
		}
	}

	if username == "" {
		jsonapi.MarshalErrorL(ctx, w, err, "UsernameNotFound")
		return
	}

	if err := activeClient.UpdateUsernamePassword(ctx, username, req.Password); err != nil {
		// TODO: translate to better human readable errors.
		jsonapi.MarshalErrorL(ctx, w, err, "UsernamePasswordError")
		return
	}

	pm := tenantconfig.MustGetPlexMap(ctx)
	app, _, err := pm.FindAppForClientID(session.ClientID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FindApp")
		return
	}

	// At this point the password is changed in the primary
	auditlog.Post(ctx, auditlog.NewEntry(user.ID, auditlog.PasswordReset,
		auditlog.Payload{"ID": app.ID, "Name": app.Name, "Code": otpState.Code, "Actor": username, "Provider": session.ClientID}))

	followerClients, err := provider.NewFollowerManagementClients(ctx, h.factory, session.ClientID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedProviderGet")
		return
	}

	for _, client := range followerClients {
		// TODO: is it safe to assume the username matches on both IDPs?
		err = client.UpdateUsernamePassword(ctx, username, req.Password)
		if err != nil {
			wrapErr := ucerr.Errorf("error updating password on follower client for email '%s' (account is in inconsistent state!): %w", otpState.Email, err)
			// TODO: flag account in bad state, require another password reset?
			jsonapi.MarshalErrorL(ctx, w, wrapErr, "UsernamePasswordError", jsonapi.Code(http.StatusInternalServerError))
			return
		}
		auditlog.Post(ctx, auditlog.NewEntry(user.ID, auditlog.PasswordReset,
			auditlog.Payload{"ID": app.ID, "Name": app.Name, "Code": otpState.Code, "Actor": username, "Provider": "Follower"}))
	}

	w.WriteHeader(http.StatusNoContent)
}
