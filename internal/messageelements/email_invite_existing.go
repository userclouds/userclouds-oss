package messageelements

// EmailInviteExistingUser is the message type for the invite existing user email
const EmailInviteExistingUser MessageType = "invite_existing"

const ieuSenderName = defaultEmailSenderName
const ieuSubjectTemplate = "Invitation to {{.AppName}}"
const ieuTextTemplate = "{{.InviterName}} has invited you to {{.AppName}}.\n\n {{.InviteText}}\n\n There is an account associated with this email address. Please click {{.Link}} to sign in."

const ieuHTMLTemplate = `
<h1>{{.InviterName}} has invited you to {{.AppName}}.</h1>
<p>{{.InviteText}}</p>
<p>
  There is an account associated with this email address. Please click
  {{.WorkaroundLink}}here</a> to sign in.
</p>
`

func init() {
	defaultParameterGetter :=
		func() any {
			return InviteUserTemplateData{
				AppName:        "app",
				InviterName:    "inviter",
				InviteText:     "Invitation",
				Link:           "link",
				WorkaroundLink: "workaroundlink",
			}
		}
	registerEmailType(EmailInviteExistingUser, getDefaultEmailSender(), ieuSenderName, ieuSubjectTemplate, ieuHTMLTemplate, ieuTextTemplate, defaultParameterGetter)
}
