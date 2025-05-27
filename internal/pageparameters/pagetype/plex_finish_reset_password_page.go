package pagetype

import "userclouds.com/internal/pageparameters/parameter"

// PlexFinishResetPasswordPage is the page type for the plex finish reset password page
const PlexFinishResetPasswordPage Type = "plex_finish_reset_password_page"

func init() {
	b := parameter.NewBuilder().
		AddParameter(parameter.ActionSubmitButtonText, "Sign in").
		AddParameter(parameter.HeadingText, "Reset Password").
		AddParameter(parameter.MissingOTPCodeText, "Missing required 'otp_code' parameter").
		AddParameter(parameter.NewPasswordLabel, "New Password").
		AddParameter(parameter.ResetPasswordFailStatusText, "Password change failed").
		AddParameter(parameter.ResetPasswordStartStatusText, "Changing password...").
		AddParameter(parameter.ResetPasswordSuccessStatusText, "Redirecting...")

	if err := registerPageParameters(PlexFinishResetPasswordPage, b.Build()); err != nil {
		panic(err)
	}
}
