package messageelements

// EmailPasswordlessLogin is the message type for the passwordless login email
const EmailPasswordlessLogin MessageType = "passwordless_login"

const plSenderName = defaultEmailSenderName
const plSubjectTemplate = "Login Request"
const plTextTemplate = "Confirm your email address to sign in to {{.AppName}}.\n\nUse the code\n\n {{.Code}}\n\n or click\n\n {{.Link}}\n\n to sign in"

const plHTMLTemplate = `
<h1>Login Request to {{.AppName}}</h1>
<p>
  Use code {{.Code}} to complete your sign in or click
  {{.WorkaroundLink}}here</a> to sign in
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
	registerEmailType(EmailPasswordlessLogin, getDefaultEmailSender(), plSenderName, plSubjectTemplate, plHTMLTemplate, plTextTemplate, defaultParameterGetter)
}
