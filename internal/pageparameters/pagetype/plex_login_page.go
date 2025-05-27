package pagetype

import "userclouds.com/internal/pageparameters/parameter"

// PlexLoginPage is the page type for the plex login page
const PlexLoginPage Type = "plex_login_page"

func init() {
	b := parameter.NewBuilder().
		AddParameter(parameter.ActionSubmitButtonText, "Sign in").
		AddParameter(parameter.AllowCreation, "{{.AllowCreation}}").
		AddParameter(parameter.CreateAccountText, "Create an account").
		AddParameter(parameter.CustomLoginPagePostFieldsetHTMLCSS, "").
		AddParameter(parameter.CustomLoginPagePreFieldsetHTMLCSS, "").
		AddParameter(parameter.CustomLoginPagePostFormHTMLCSS, "").
		AddParameter(parameter.CustomLoginPagePreFormHTMLCSS, "").
		AddParameter(parameter.CustomLoginPagePreMainHTMLCSS, "").
		AddParameter(parameter.FooterHTML, "").
		AddParameter(parameter.ForgotPasswordText, "Forgot username or password?").
		AddParameter(parameter.HeadingText, "Sign in to {{.AppName}}").
		AddParameter(parameter.LoginFailStatusText, "Login failed").
		AddParameter(parameter.LoginStartStatusText, "Signing in...").
		AddParameter(parameter.LoginSuccessStatusText, "Redirecting...").
		AddParameter(parameter.PasswordLabel, "Password").
		AddParameter(parameter.PasswordlessLoginText, "Hate Passwords?...").
		AddParameter(parameter.PasswordResetEnabled, "{{.PasswordResetEnabled}}").
		AddParameter(parameter.SocialRedirectStatusText, "Redirecting you to sign-in page...").
		AddParameter(parameter.SubheadingText, "").
		AddParameter(parameter.UserNameLabel, "Username")

	if err := registerPageParameters(PlexLoginPage, b.Build()); err != nil {
		panic(err)
	}
}
