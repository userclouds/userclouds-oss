package messageelements

// EmailMFAVerify is the message type for a multi-factor authentication email verification
const EmailMFAVerify MessageType = "mfa_email_verify"

const mfaEmailVerifySenderName = defaultEmailSenderName
const mfaEmailVerifySubjectTemplate = "MFA Email Verification Required"
const mfaEmailVerifyTextTemplate = "Verification Code: {{.Code}}"

const mfaEmailVerifyHTMLTemplate = `
<h1>MFA Email Verification Request for {{.AppName}}</h1>
<p>Use code {{.Code}} to confirm your email for use with MFA</p>
`

func init() {
	defaultParameterGetter :=
		func() any {
			return MFATemplateData{
				AppName: "app",
				Code:    "code",
			}
		}
	registerEmailType(
		EmailMFAVerify,
		getDefaultEmailSender(),
		mfaEmailVerifySenderName,
		mfaEmailVerifySubjectTemplate,
		mfaEmailVerifyHTMLTemplate,
		mfaEmailVerifyTextTemplate,
		defaultParameterGetter)
}
