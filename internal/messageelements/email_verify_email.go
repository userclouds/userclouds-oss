package messageelements

// EmailVerifyEmail is the message type for the verify email address email
const EmailVerifyEmail MessageType = "verify_email"

const veSenderName = defaultEmailSenderName
const veSubjectTemplate = "Verify your Email"
const veTextTemplate = "Verify your email address for {{.AppName}}.\n\nClick\n\n {{.Link}}\n\n to sign in"

const veHTMLTemplate = `
<h1>Welcome to {{.AppName}}</h1>
<p>Verify your email address by clicking {{.WorkaroundLink}}here</a>.</p>
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
	registerEmailType(EmailVerifyEmail, getDefaultEmailSender(), veSenderName, veSubjectTemplate, veHTMLTemplate, veTextTemplate, defaultParameterGetter)
}
