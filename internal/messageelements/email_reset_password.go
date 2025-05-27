package messageelements

// EmailResetPassword is the message type for the reset password email
const EmailResetPassword MessageType = "reset_password"

const rpSenderName = defaultEmailSenderName
const rpSubjectTemplate = "Password Reset"
const rpTextTemplate = "Password Reset for {{.AppName}}. Click here to reset your password: {{.Link}}"

const rpHTMLTemplate = `
<h1>Password Reset for {{.AppName}}</h1>
<p>
  If you didn't request to reset your password, please ignore this email.
  Otherwise click {{.WorkaroundLink}}here</a> to reset your {{.AppName}}
  password
</p>
`

func init() {
	defaultParameterGetter :=
		func() any {
			return OTPEmailTemplateData{
				AppName:        "app",
				Code:           "code",
				Link:           "link",
				WorkaroundLink: "workaroundlink",
			}
		}
	registerEmailType(EmailResetPassword, getDefaultEmailSender(), rpSenderName, rpSubjectTemplate, rpHTMLTemplate, rpTextTemplate, defaultParameterGetter)
}
