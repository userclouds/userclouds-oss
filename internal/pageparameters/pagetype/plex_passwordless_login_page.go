package pagetype

import "userclouds.com/internal/pageparameters/parameter"

// PlexPasswordlessLoginPage is the page type for the plex passwordless login page
const PlexPasswordlessLoginPage Type = "plex_passwordless_login_page"

func init() {
	b := parameter.NewBuilder().
		AddParameter(parameter.ActionSubmitButtonText, "Sign in").
		AddParameter(parameter.EmailLabel, "Email").
		AddParameter(parameter.HeadingText, "Sign in to {{.AppName}}").
		AddParameter(parameter.LoginFailStatusText, "Login failed").
		AddParameter(parameter.LoginStartStatusText, "Signing in...").
		AddParameter(parameter.LoginSuccessStatusText, "Redirecting...").
		AddParameter(parameter.OtpCodeLabel, "Code").
		AddParameter(parameter.PasswordlessSendEmailStartStatusText, "Validating email...").
		AddParameter(parameter.PasswordlessSendEmailSuccessStatusText, "Email sent! Check for instructions at email address")

	if err := registerPageParameters(PlexPasswordlessLoginPage, b.Build()); err != nil {
		panic(err)
	}
}
