package paths

// PlexUIRoot is the root path on Plex for the React-based UI.
// NOTE: there are 2 other things that need to be kept in sync:
// 1. `plexui/package.json` has a "homepage" property that must match this exactly
// 2. `plexui/src/index.tsx` sets `BrowserRouter.basename` to this value.
const PlexUIRoot = "/plexui"

// LogoutPath is the full path of the Plex URL that clients redirect to in order to log out.
const LogoutPath = "/logout"

// ResetPasswordRootPath is the root of the password reset handler
const ResetPasswordRootPath = "/resetpassword"

// SocialRootPath is the root of the social login handler
const SocialRootPath = "/social"

// SocialCallbackSubPath is the subpath to a method under the social login handler which handles
// callbacks from external OIDC providers
const SocialCallbackSubPath = "/callback"

// These constants are sub-paths under the Plex UI path for the React-based UI - NOT handler paths.
// NOTE: this must be kept in sync with the corresponding <Route> elements set up in `plexui/src/index.tsx`.
const (
	LoginUISubPath               = "/login"
	CreateUserUISubPath          = "/createuser"
	MFAChannelUISubPath          = "/mfachannel"
	MFAShowRecoveryCodeUISubPath = "/mfashowrecoverycode"
	MFACodeUISubPath             = "/mfasubmit"
	StartResetPasswordUISubPath  = "/startresetpassword"
	FinishResetPasswordUISubPath = "/finishresetpassword"
	PasswordlessLoginUISubPath   = "/passwordlesslogin"
	EmailExistsUIPath            = "/userwithemailexists"
)

// Auth0RedirectCallbackPath is the callback for Auth0 providers with Redirect set, where we actually redirect
// the user off to Auth0 and then back to us (instead of MITMing the login HTML), before rewriting the token
// and redirecting them back again to the customer
const Auth0RedirectCallbackPath = "/callback"

// AccountChooser lets you choose between delegated accounts
// TODO: get rid of trailing Path in names in paths package
const AccountChooser = "/chooser"
