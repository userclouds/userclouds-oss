package pagetype

import "userclouds.com/internal/pageparameters/parameter"

// PlexEmailExistsPage is the page type for the plex login page
const PlexEmailExistsPage Type = "plex_email_exists_page"

func init() {
	b := parameter.NewBuilder().
		AddParameter(parameter.ActionSubmitButtonText, "Sign in to existing account").
		AddParameter(parameter.EmailExistsCreateAccountText, "No, create a new, separate account").
		AddParameter(parameter.EmailExistsLoginText, "No, sign in to my account without adding this method").
		AddParameter(parameter.FooterHTML, "").
		AddParameter(parameter.HeadingText, "Email already in use").
		AddParameter(parameter.LoginFailStatusText, "Login failed").
		AddParameter(parameter.LoginStartStatusText, "Signing in...").
		AddParameter(parameter.LoginSuccessStatusText, "Redirecting...").
		AddParameter(parameter.PasswordLabel, "Password").
		AddParameter(parameter.SocialRedirectStatusText, "Redirecting you to sign-in page...").
		AddParameter(parameter.SubheadingText, "Your email address is already associated with a {{.AppName}} account, using the login method(s) below.\nWould you like to sign in to your existing account, and add this sign in method to that account?").
		AddParameter(parameter.UserNameLabel, "Username")

	if err := registerPageParameters(PlexEmailExistsPage, b.Build()); err != nil {
		panic(err)
	}
}
