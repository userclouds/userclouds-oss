package messageelements

// EmailMFAChallenge is the message type for a multi-factor authentication email challenge
const EmailMFAChallenge MessageType = "mfa_email_challenge"

const mfaEmailChallengeSenderName = defaultEmailSenderName
const mfaEmailChallengeSubjectTemplate = "MFA Email Challenge Required"
const mfaEmailChallengeTextTemplate = "Challenge Code: {{.Code}}"

const mfaEmailChallengeHTMLTemplate = `
<h1>Login Request to {{.AppName}}</h1>
<p>Use code {{.Code}} to complete the sign in process</p>
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
		EmailMFAChallenge,
		getDefaultEmailSender(),
		mfaEmailChallengeSenderName,
		mfaEmailChallengeSubjectTemplate,
		mfaEmailChallengeHTMLTemplate,
		mfaEmailChallengeTextTemplate,
		defaultParameterGetter)
}
