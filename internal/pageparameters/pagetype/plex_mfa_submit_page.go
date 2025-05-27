package pagetype

import "userclouds.com/internal/pageparameters/parameter"

// PlexMfaSubmitPage is the page type for the plex mfa submit page
const PlexMfaSubmitPage Type = "plex_mfa_submit_page"

func init() {
	b := parameter.NewBuilder().
		AddParameter(parameter.ActionSubmitButtonText, "Sign in").
		AddParameter(parameter.HeadingText, "Multifactor authentication required").
		AddParameter(parameter.LoginFailStatusText, "Login failed").
		AddParameter(parameter.LoginStartStatusText, "Signing in...").
		AddParameter(parameter.LoginSuccessStatusText, "Redirecting...").
		AddParameter(parameter.MFACodeLabel, "MFA Code")

	if err := registerPageParameters(PlexMfaSubmitPage, b.Build()); err != nil {
		panic(err)
	}
}
