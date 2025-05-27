package pagetype

import "userclouds.com/internal/pageparameters/parameter"

// PlexStartResetPasswordPage is the page type for the plex start reset password page
const PlexStartResetPasswordPage Type = "plex_start_reset_password_page"

func init() {
	b := parameter.NewBuilder().
		AddParameter(parameter.ActionSubmitButtonText, "Reset Password").
		AddParameter(parameter.EmailLabel, "Email").
		AddParameter(parameter.HeadingText, "Reset Password").
		AddParameter(parameter.ResetPasswordFailStatusText, "Reset password failed").
		AddParameter(parameter.ResetPasswordStartStatusText, "Validating email...").
		AddParameter(parameter.ResetPasswordSuccessStatusText, "Please check further instructions at email address")

	if err := registerPageParameters(PlexStartResetPasswordPage, b.Build()); err != nil {
		panic(err)
	}
}
