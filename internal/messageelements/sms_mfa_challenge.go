package messageelements

// SMSMFAChallenge is the message type for a multi-factor authentication SMS challenge
const SMSMFAChallenge MessageType = "sms_mfa_challenge"

const mfaSMSChallengeBodyTemplate = "Login Request to {{.AppName}}. Use code {{.Code}} to complete the sign in process."

func init() {
	defaultParameterGetter :=
		func() any {
			return MFATemplateData{
				AppName: "app",
				Code:    "code",
			}
		}
	registerSMSType(SMSMFAChallenge, defaultSMSSender, mfaSMSChallengeBodyTemplate, defaultParameterGetter)
}
