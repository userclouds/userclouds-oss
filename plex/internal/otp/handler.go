package otp

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
	"userclouds.com/plex/internal/oidc"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/reactdev"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
	"userclouds.com/plex/internal/usermanager"
)

type handler struct {
	factory provider.Factory
}

// RootPath is the path to the handler on Plex for OTP endpoints.
const RootPath = "/otp"

// SubmitSubPath is the subpath to the endpoint on Plex used for OTP code & magic link submission
// (e.g. for passwordless login, email verification, invites, password reset, etc).
const SubmitSubPath = "/submit"

// NewHandler returns a new OTP Submit handler for plex
func NewHandler(factory provider.Factory) http.Handler {
	h := &handler{factory}

	hb := builder.NewHandlerBuilder()
	handlerBuilder(hb, h)

	hb.MethodHandler(SubmitSubPath).
		Get(h.submitHandler).
		Post(h.submitHandler)

	return hb.Build()
}

//go:generate genhandler /otp

// SubmitRequest defines the JSON request to the OTP submit handler.
type SubmitRequest struct {
	SessionID uuid.UUID `json:"session_id"`
	Email     string    `json:"email"`
	OTPCode   string    `json:"otp_code"`
}

func (h *handler) submitHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: for extra security should we require that this request come from the same IP / user agent
	// as the call to start the OTP flow? That helps protect against link hijacking

	ctx := r.Context()
	fromLink := r.Method == http.MethodGet
	fromForm := r.Method == http.MethodPost

	if !fromLink && !fromForm {
		// This really can't happen if the handler was installed right
		jsonapi.MarshalErrorL(ctx, w, ucerr.New("method not supported"), "MethodNotSupported", jsonapi.Code(http.StatusMethodNotAllowed))
		return
	}

	var req SubmitRequest
	if fromForm {
		if err := jsonapi.Unmarshal(r, &req); err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "InvalidRequest")
			return
		}
	} else {
		sessionID := r.URL.Query().Get("session_id")
		var err error
		if req.SessionID, err = uuid.FromString(sessionID); err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "InvalidRequest") //, jsonapi.Code(http.StatusBadRequest))
			return
		}
		req.Email = r.URL.Query().Get("email")
		req.OTPCode = r.URL.Query().Get("otp_code")
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

	uclog.Infof(ctx, "OTP submit (purpose: %s) for %s", otpState.Purpose.String(), req.Email)

	// Wrong code, email, or used/expired code is all the same to us
	if req.OTPCode != otpState.Code ||
		req.Email != otpState.Email ||
		(otpState.Used && otpState.Purpose != storage.OTPPurposeInvite) ||
		time.Now().UTC().After(otpState.Expires) {
		uclog.Debugf(ctx, "invalid OTP code: code wrong (%t) or email wrong (%t) or used (%t) or expired (%t)", req.OTPCode != otpState.Code, req.Email != otpState.Email, otpState.Used, time.Now().UTC().After(otpState.Expires))
		// Return 400 to be consistent with bad username/password login
		// See https://stackoverflow.com/questions/22586825/oauth-2-0-why-does-the-authorization-server-return-400-instead-of-401-when-the
		jsonapi.MarshalErrorL(ctx, w, ucerr.New("invalid code or email"), "BadCodeOrEmail", jsonapi.Code(http.StatusBadRequest))
		return
	}

	if otpState.Used && otpState.Purpose == storage.OTPPurposeInvite {
		// If the invite was already used (user clicking again on an old invite link), just redirect to the callback URL and let the app handle it
		uchttp.Redirect(w, r, session.RedirectURI, http.StatusSeeOther)
		return
	}

	// Mark token used for some use cases
	// TODO: this makes invite links somewhat fragile; they can be clicked only once even if the user
	// doesn't follow through with the invite. Invites in Console also have a "used" flag and an expiration,
	// so this is a bit redundant. We could have the app mark an invite as "used" when it's done, or maybe
	// after a user account is created OR a user logs in successfully?
	switch otpState.Purpose {
	case storage.OTPPurposeLogin:
		fallthrough
	case storage.OTPPurposeAccountVerify:
		otpState.Used = true
		if err := s.SaveOTPState(ctx, otpState); err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "SaveSession")
			return
		}
	case storage.OTPPurposeInvite:
		// Don't mark it used here; we'll mark it used when the user logs in or creates an account.
		// TODO: how do we ensure we don't miss a codepath which allows invites to get re-used?
	case storage.OTPPurposeResetPassword:
		// Don't mark it used here; we'll mark it used when the password gets reset
	default:
		jsonapi.MarshalErrorL(ctx, w, err, "InvalidOTPPurpose2")
		return
	}

	activeClient, err := provider.NewActiveManagementClient(ctx, h.factory, session.ClientID)
	if err != nil {
		// TODO: differentiate error types (issue #103).
		jsonapi.MarshalErrorL(ctx, w, err, "ActiveClientErr")
		return
	}

	var user *iface.UserProfile
	if otpState.UserID != "" {
		user, err = activeClient.GetUser(ctx, otpState.UserID)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "GetUserForEmail")
			return
		}
	}

	// Mark email verified if possible.
	// TODO: if a user signs up via an invite link (but doesn't yet have an account) should we also
	// mark the email verified upon sign-up?
	if user != nil {
		uclog.Debugf(ctx, "marking email verified for '%s' (ID: %s)", user.Email, user.ID)
		followers, err := provider.NewFollowerManagementClients(ctx, h.factory, session.ClientID)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "FollowerClientsErr")
			return
		}

		if err := usermanager.MarkEmailVerified(ctx, activeClient, followers, user); err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "VerifyEmailErr")
			return
		}
	}

	switch otpState.Purpose {
	case storage.OTPPurposeLogin:
		// User was logging in, generate token and either return it or redirect the user
		tc := tenantconfig.MustGet(ctx)
		tu := tenantconfig.MustGetTenantURLString(ctx)
		if err := storage.GenerateUserPlexToken(ctx, tu, &tc, s, user, session, nil /* no underlying token for OTP */); err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "LoginOTP")
			return
		}

		// TODO: handle the case where an invited user completes a passwordless login successfully, which
		// requires tracking multiple OTP states.

		if fromForm {
			redirectURL, err := oidc.NewLoginResponse(ctx, session)
			if err != nil {
				jsonapi.MarshalErrorL(ctx, w, err, "NewLoginResponse")
				return
			}

			plexApp, _, err := tc.PlexMap.FindAppForClientID(session.ClientID)
			if err != nil {
				jsonapi.MarshalErrorL(ctx, w, err, "MissingPlexApp")
				return
			}

			auditlog.Post(ctx, auditlog.NewEntry(user.ID, auditlog.LoginSuccess,
				auditlog.Payload{"ID": plexApp.ID, "Name": plexApp.Name, "Actor": user.Email, "Type": "UC Passwordless", "Code": otpState.Code}))

			jsonapi.Marshal(w, redirectURL)
		} else {
			redirectURL, err := oidc.GetLoginRedirectURL(ctx, session)
			if err != nil {
				jsonapi.MarshalErrorL(ctx, w, err, "GetLoginRedirectURL")
				return
			}
			// TODO - Steve G - what event should we inject into audit log here ?
			uchttp.Redirect(w, r, redirectURL, http.StatusSeeOther)
		}

	case storage.OTPPurposeAccountVerify:
		jsonapi.Marshal(w, fmt.Sprintf("successfully verified email '%s'", user.Email))

	case storage.OTPPurposeInvite:
		// Redirect to the Sign In page which lets a user create a new account
		// or sign in with an existing one.
		provClient, err := provider.NewActiveClient(ctx, h.factory, session.ClientID)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetActiveClient")
			return
		}

		tc := tenantconfig.MustGet(ctx)
		plexApp, _, err := tc.PlexMap.FindAppForClientID(session.ClientID)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetClientApp")
			return
		}

		var redirectURL *url.URL
		// TODO: There may be some UX issues to think through here; for example, the user may have have forgotten whether they used
		// social vs. u/p login and we're directing them to the login page without a hint/clue about which mechanism they used.
		// So the account dupe detection / merging code should try to ensure there's a good user experience here.
		redirectURL, err = provClient.LoginURL(ctx, session.ID, plexApp)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "FailedToGetRedirectURL")
			return
		}
		uchttp.Redirect(w, r, redirectURL.String(), http.StatusTemporaryRedirect)

	case storage.OTPPurposeResetPassword:
		// Handle the password reset flow on the server so we can throttle, log, etc
		// before handing control to client-side React UI.
		query := url.Values{
			"session_id": []string{session.ID.String()},
			"otp_code":   []string{otpState.Code}}
		u := reactdev.UIBaseURL(ctx)
		u.Path = u.Path + paths.FinishResetPasswordUISubPath
		u.RawQuery = query.Encode()

		uchttp.Redirect(w, r, u.String(), http.StatusSeeOther)
	default:
		// Unreachable since we validated OTP purpose already
		jsonapi.MarshalErrorL(ctx, w, ucerr.New("bug: hit unreachable block"), "Unreachable")
	}
}
