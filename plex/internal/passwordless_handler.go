package internal

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/messaging/email"
	"userclouds.com/infra/uctypes/messaging/email/emailaddress"
	message "userclouds.com/internal/messageelements"
	"userclouds.com/internal/security"
	"userclouds.com/plex/internal/loginapp"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/tenantconfig"
)

// TODO: this should probably be a subpackage but some internal state needs to be cleaned up / refactored first.

// PasswordlessLoginRequest defines the JSON request to the passwordless login handler.
// TODO: add support for SMS
type PasswordlessLoginRequest struct {
	SessionID uuid.UUID `json:"session_id" validate:"notnil"`
	Email     string    `json:"email"`
}

//go:generate genvalidate PasswordlessLoginRequest

func (plr PasswordlessLoginRequest) extraValidate() error {
	a := emailaddress.Address(plr.Email)
	if err := a.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

type passwordlessHandler struct {
	checker security.ReqValidator
	email   email.Client
	factory provider.Factory
}

func newPasswordlessHandler(checker security.ReqValidator, email email.Client, factory provider.Factory) http.Handler {
	h := &passwordlessHandler{
		checker: checker,
		email:   email,
		factory: factory,
	}

	hb := builder.NewHandlerBuilder()
	// API to trigger starting passwordless login.
	hb.MethodHandler("/start").Post(h.passwordlessStartHandler)
	return hb.Build()
}

// passwordlessStartHandler triggers a passwordless login flow.
// For now this means Plex sends the user a code + magic link via email if the email is valid.
// TODO: support SMS.
func (h *passwordlessHandler) passwordlessStartHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req PasswordlessLoginRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "InvalidRequest")
		return
	}

	uclog.Infof(ctx, "passwordless login request for %s", req.Email)

	if h.checker.IsCallBlocked(ctx, req.Email) {
		// TODO better API between security events and general errors
		uchttp.ErrorL(ctx, w, ucerr.New("call volume exceeded"), http.StatusForbidden, "CallBlocked")
		return
	}

	s := tenantconfig.MustGetStorage(ctx)
	session, err := s.GetOIDCLoginSession(ctx, req.SessionID)
	if err != nil {
		uclog.Debugf(ctx, "invalid session ID specified: %s", req.SessionID)
		jsonapi.MarshalErrorL(ctx, w, ucerr.New("invalid session_id specified"), "InvalidID", jsonapi.Code(http.StatusBadRequest))
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

	userID, err := otp.ResolveEmailToUser(ctx, activeClient, req.Email)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "ResolveEmailToUser")
		return
	}

	hasAccess, err := loginapp.CheckLoginAccessForUser(ctx, tc, app, userID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "RestrictedAccessError")
		return
	}
	if !hasAccess {
		jsonapi.MarshalErrorL(ctx, w, ucerr.Friendlyf(nil, "You are not permitted to login to this app"), "RestrictedAccessDenied", jsonapi.Code(http.StatusForbidden))
		return
	}

	// StartLoginFlow will ensure there is exactly 1 username-password authn associated with this email.
	otpCode, err := otp.StartLoginFlow(ctx, activeClient, s, session, userID, req.Email)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "StartOTPFlow")
		return
	}

	if err := otp.SendOTPEmail(ctx, tc.PlexMap.GetEmailClient(h.email), req.SessionID, app.Name, req.Email, app.MakeElementGetter(message.EmailPasswordlessLogin), otpCode); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "SendEmailError")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
