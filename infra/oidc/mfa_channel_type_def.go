package oidc

// MFAChannelType defines the types of channels supported for MFA
type MFAChannelType string

// MFAChannelType constants
const (
	MFAInvalidChannel            MFAChannelType = "invalid"
	MFAEmailChannel              MFAChannelType = "email"
	MFASMSChannel                MFAChannelType = "sms"
	MFAAuthenticatorChannel      MFAChannelType = "authenticator"
	MFAAuth0AuthenticatorChannel MFAChannelType = "auth0_authenticator"
	MFAAuth0EmailChannel         MFAChannelType = "auth0_email"
	MFAAuth0SMSChannel           MFAChannelType = "auth0_sms"
	MFARecoveryCodeChannel       MFAChannelType = "recovery_code"
)
