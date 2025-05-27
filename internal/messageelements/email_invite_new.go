package messageelements

// EmailInviteNewUser is the message type for the invite new user email
const EmailInviteNewUser MessageType = "invite_new"

const inuSenderName = defaultEmailSenderName
const inuSubjectTemplate = "Invitation to {{.AppName}}"
const inuTextTemplate = "{{.InviterName}} has invited you to {{.AppName}}.\n\n {{.InviteText}}\n\n Click {{.Link}} to sign up."

// NB: see the comment in SendWithHTMLTemplate about why {{ .Link }} has weirdly specific structure here
const inuHTMLTemplate = `
<h1>{{.InviterName}} has invited you to {{.AppName}}.</h1>
<p>{{.InviteText}}</p>
<p>Click {{.WorkaroundLink}}here</a> to sign up.</p>
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
	registerEmailType(EmailInviteNewUser, getDefaultEmailSender(), inuSenderName, inuSubjectTemplate, inuHTMLTemplate, inuTextTemplate, defaultParameterGetter)
}
