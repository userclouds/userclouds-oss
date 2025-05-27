package otp

import (
	"context"
	"net/url"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/storage"
)

// CodeLength is the length of one-time use login codes
// TODO: possibly configure per-tenant?
const CodeLength = 6

// Default expirations for OTP types
const (
	DefaultPasswordlessExpiry  = time.Minute * 15
	DefaultVerificationExpiry  = time.Hour * 24
	DefaultInviteExpiry        = time.Hour * 24 * 7
	DefaultResetPasswordExpiry = time.Minute * 15
)

// UseDefaultExpiry is a sentinel value to indicate that the default expiration should be used.
var UseDefaultExpiry = time.Time{}

// ErrNoInviteAssociatedWithSession indicates there is no invite associated with a given login session.
var ErrNoInviteAssociatedWithSession = ucerr.New("no invite associated with session")

// ErrInviteBoundToAnotherUser indicates that the invite associated with a given session has already been used by another user.
var ErrInviteBoundToAnotherUser = ucerr.New("invite associated with session already used by another user")

// ErrNoUserForEmail indicates what it says, used to prevent unintentional information leakage on eg. reset password.
var ErrNoUserForEmail = ucerr.New("no user found for email")

func startFlow(ctx context.Context, s *storage.Storage, session *storage.OIDCLoginSession, userID, email string, purpose storage.OTPPurpose, expires time.Time) (string, error) {
	if err := purpose.Validate(); err != nil {
		return "", ucerr.Wrap(err)
	}
	// TODO: We need a way to deal with multiple OTP flows in a single login session.
	// Either we track multiple flows, or we support invalidating previous ones and overwriting in place.
	// Currently we just fail if we try to start a 2nd OTP flow.
	// This error codepath can get triggered if:
	// 1. A user goes 'back' in their browser after starting a passwordless flow and starts another.
	// 2. A user creates a new account and then tries to go back in their browser and start a passwordless flow.
	// (and probably some other weird edge cases involving navigating 'back').
	// 3. A user responds to an invite and - while in the process of accepting the invite -
	// tries to reset their password OR starts passwordless login. In this case, we'd attempt to start a new OTP
	// flow (purpose = Login) in a session that already has one (purpose = Invite).
	// For now we'll just accept this as a limitation.
	if session.OTPStateID != uuid.Nil {
		return "", ucerr.Errorf("session %s has already started an OTP flow")
	}

	if expires.IsZero() {
		var duration time.Duration
		// TODO: make expirations configurable in tenant
		switch purpose {
		case storage.OTPPurposeLogin:
			duration = DefaultPasswordlessExpiry
		case storage.OTPPurposeAccountVerify:
			duration = DefaultVerificationExpiry
		case storage.OTPPurposeInvite:
			duration = DefaultInviteExpiry
		case storage.OTPPurposeResetPassword:
			duration = DefaultResetPasswordExpiry
		default:
			return "", ucerr.Errorf("invalid purpose; %s", purpose.String())
		}
		expires = time.Now().UTC().Add(duration)
	} else {
		if expires.Before(time.Now().UTC()) {
			return "", ucerr.Errorf("invalid expiration time in OTP Flow, '%s' is in the past", expires.String())
		}
	}

	var otpCode string
	switch purpose {
	case storage.OTPPurposeLogin:
		fallthrough
	case storage.OTPPurposeAccountVerify:
		fallthrough
	case storage.OTPPurposeInvite:
		otpCode = crypto.MustRandomDigits(CodeLength)
	case storage.OTPPurposeResetPassword:
		otpCode = crypto.GenerateOpaqueAccessToken()
	default:
		return "", ucerr.Errorf("invalid purpose; %s", purpose.String())
	}

	otpState := &storage.OTPState{
		BaseModel: ucdb.NewBase(),
		SessionID: session.ID,
		UserID:    userID,
		Email:     email,
		Code:      otpCode,
		Expires:   expires,
		Used:      false,
		Purpose:   purpose,
	}

	if err := s.SaveOTPState(ctx, otpState); err != nil {
		return "", ucerr.Wrap(err)
	}

	session.OTPStateID = otpState.ID
	if err := s.SaveOIDCLoginSession(ctx, session); err != nil {
		return "", ucerr.Wrap(err)
	}

	return otpCode, nil
}

// UserWithEmailExists returns true if at least one user account is associated with this email.
func UserWithEmailExists(ctx context.Context, activeClient iface.ManagementClient, email string) bool {
	users, err := activeClient.ListUsersForEmail(ctx, email, idp.AuthnTypeAll)
	if err != nil {
		// TODO: differentiate between user not found vs. other error in the management client impls.
		uclog.Debugf(ctx, "did not retrieve account(s) for invited user with email '%s': %v", email, err)
		return false
	} else if len(users) > 0 {
		return true
	}
	return false
}

// ResolveEmailToUser returns the user ID associated with the given email
func ResolveEmailToUser(ctx context.Context, activeClient iface.ManagementClient, email string) (string, error) {
	// Ensure email is for a valid user
	users, err := activeClient.ListUsersForEmail(ctx, email, idp.AuthnTypePassword)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	if len(users) == 0 {
		uclog.Debugf(ctx, "no user for email, error: %v", err)
		return "", ErrNoUserForEmail
	} else if len(users) > 1 {
		err = ucerr.Errorf("expected 1 user for email '%s', got %d", email, len(users))
		uclog.Debugf(ctx, "%v", err)
		return "", ucerr.Wrap(err)
	}
	return users[0].ID, nil
}

// StartLoginFlow initializes state in an existing session for a passwordless login flow.
func StartLoginFlow(ctx context.Context, activeClient iface.ManagementClient, s *storage.Storage, session *storage.OIDCLoginSession, userID string, email string) (string, error) {
	// TOOD: should probably have a dedicated passwordless AuthN type and allow users (or tenants) to
	// explicitly opt-in to passwordless?
	otpCode, err := startFlow(ctx, s, session, userID, email, storage.OTPPurposeLogin, UseDefaultExpiry)
	return otpCode, ucerr.Wrap(err)
}

// StartPasswordResetFlow initializes state in an existing session to reset a user's password.
func StartPasswordResetFlow(ctx context.Context, activeClient iface.ManagementClient, s *storage.Storage, session *storage.OIDCLoginSession, email string) (string, error) {
	userID, err := ResolveEmailToUser(ctx, activeClient, email)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	otpCode, err := startFlow(ctx, s, session, userID, email, storage.OTPPurposeResetPassword, UseDefaultExpiry)
	return otpCode, ucerr.Wrap(err)
}

func createLoginSessionAndFlow(ctx context.Context, s *storage.Storage, clientID, userID, email, state string, responseTypes storage.ResponseTypes, redirectURL *url.URL, purpose storage.OTPPurpose, expires time.Time) (uuid.UUID, string, error) {
	sessionID, err := storage.CreateOIDCLoginSession(ctx, s, clientID, responseTypes, redirectURL, state, oidc.DefaultScopes)
	if err != nil {
		return uuid.Nil, "", ucerr.Wrap(err)
	}

	session, err := s.GetOIDCLoginSession(ctx, sessionID)
	if err != nil {
		return uuid.Nil, "", ucerr.Wrap(err)
	}

	otpCode, err := startFlow(ctx, s, session, userID, email, purpose, expires)
	if err != nil {
		return uuid.Nil, "", ucerr.Wrap(err)
	}

	return session.ID, otpCode, nil
}

// CreateAccountVerificationSession kicks off an account verification flow and returns an OTP code
// that can be emailed to the user to validate ownership of the account along with a Login Session
// (which is currently required because OTP code info lives there).
// TODO: don't [ab]use OIDC Login session for this, separate OTP flows more granularly from login session.
// TODO: generalize for SMS at some point? Same idea, different identifier
func CreateAccountVerificationSession(ctx context.Context, s *storage.Storage, clientID, userID, email string) (uuid.UUID, string, error) {
	// TODO: support a custom landing page after account verification?

	// TODO: putting in dummy values to make validation happy, but these aren't used for account verification.
	redirectURL := &url.URL{
		Scheme: "https://",
	}
	emptyState := "unused"
	responseTypes := storage.ResponseTypes{storage.UnusedResponseType}

	return createLoginSessionAndFlow(ctx, s, clientID, userID, email, emptyState, responseTypes, redirectURL, storage.OTPPurposeAccountVerify, UseDefaultExpiry)
}

// CreateInviteSession kicks off a user invite flow and returns an OTP code and Login session which
// can be used to construct an email to the invitee to accept the invite.
// TODO: generalize for SMS at some point? Same idea, different identifier
func CreateInviteSession(ctx context.Context, s *storage.Storage, clientID, email, state string, redirectURL *url.URL, expires time.Time) (uuid.UUID, string, error) {
	// TODO: allow other types of responses?
	responseTypes := storage.ResponseTypes{storage.AuthorizationCodeResponseType}
	// Don't pass UserID in for invites because it isn't necessarily unique for an email and it isn't used.
	return createLoginSessionAndFlow(ctx, s, clientID, "", email, state, responseTypes, redirectURL, storage.OTPPurposeInvite, expires)
}

// HasUnusedInvite checks if the given session has a valid unused/unbound invite associated with it,
// and returns the OTP state associated with the invite if so, or an error if not valid.
func HasUnusedInvite(ctx context.Context, s *storage.Storage, session *storage.OIDCLoginSession) (*storage.OTPState, error) {
	if session.OTPStateID.IsNil() {
		return nil, ucerr.Wrap(ErrNoInviteAssociatedWithSession)
	}

	otpState, err := s.GetOTPState(ctx, session.OTPStateID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if otpState.Purpose != storage.OTPPurposeInvite {
		return nil, ucerr.Wrap(ErrNoInviteAssociatedWithSession)
	}

	if otpState.Used || otpState.UserID != "" {
		// Already used and bound to a user.
		return nil, ucerr.Wrap(ErrInviteBoundToAnotherUser)
	}

	return otpState, nil
}

// BindInviteToUser succeeds if there's an invite associated with the session and it is already bound to this user or it is unbound
// (in which case it will be bound to this user and marked as used). If any of those conditions are not met, an error will be returned.
func BindInviteToUser(ctx context.Context, s *storage.Storage, session *storage.OIDCLoginSession, userID string, userName string, app *tenantplex.App) error {
	if session.OTPStateID.IsNil() {
		return ucerr.Wrap(ErrNoInviteAssociatedWithSession)
	}

	if userID == "" {
		return ucerr.New("must specify valid userID to BindInviteToUser")
	}

	otpState, err := s.GetOTPState(ctx, session.OTPStateID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if otpState.Purpose != storage.OTPPurposeInvite {
		return ucerr.Wrap(ErrNoInviteAssociatedWithSession)
	}

	if otpState.UserID != "" && otpState.UserID != userID {
		return ucerr.Wrap(ErrInviteBoundToAnotherUser)
	}

	// Only make an audit log entry when we are marking the invite as used for the first time
	makeAuditLogEntry := !otpState.Used

	otpState.Used = true
	otpState.UserID = userID
	if err := s.SaveOTPState(ctx, otpState); err != nil {
		return ucerr.Wrap(err)
	}

	if makeAuditLogEntry {
		auditlog.Post(ctx, auditlog.NewEntry(userID, auditlog.InviteRedemeed,
			auditlog.Payload{"ID": app.ID, "Name": app.Name, "Code": otpState.Code, "Actor": userName}))

	}

	return nil
}
