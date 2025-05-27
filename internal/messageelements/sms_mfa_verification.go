package messageelements

// SMSMFAVerify is the message type for a multi-factor authentication SMS verification
const SMSMFAVerify MessageType = "sms_mfa_verify"

const mfaSMSVerifyBodyTemplate = "MFA Verification Request for {{.AppName}}. Use code {{.Code}} to confirm your phone number for use with MFA."

func init() {
	defaultParameterGetter :=
		func() any {
			return MFATemplateData{
				AppName: "app",
				Code:    "code",
			}
		}
	registerSMSType(SMSMFAVerify, defaultSMSSender, mfaSMSVerifyBodyTemplate, defaultParameterGetter)
}
