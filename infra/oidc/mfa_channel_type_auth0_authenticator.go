package oidc

import "fmt"

type mfaAuth0AuthenticatorChannel struct{}

func (mfaAuth0AuthenticatorChannel) canConfigure() bool {
	return false
}

func (mfaAuth0AuthenticatorChannel) canReissueChallenge() bool {
	return false
}

func (mfaAuth0AuthenticatorChannel) getAuditLogType() string {
	return "Auth0 Authenticator MFA"
}

func (mfaAuth0AuthenticatorChannel) getChallengeDescription(MFAChannel, bool, bool) string {
	return "Enter Code from Code Generator"
}

func (mfaAuth0AuthenticatorChannel) getChannelDescription(MFAChannel, bool) string {
	return "Auth0 Code Generator"
}

func (mfaAuth0AuthenticatorChannel) getRegistrationInfo(MFAChannel) (string, string, bool) {
	return "", "", false
}

func (mfaAuth0AuthenticatorChannel) getUniqueID(mfac MFAChannel) string {
	return fmt.Sprintf("%v:%s", MFAAuth0AuthenticatorChannel, mfac.ChannelTypeID)
}

func (mfaAuth0AuthenticatorChannel) getUserDetailDescription(mfac MFAChannel) string {
	return mfac.ChannelTypeID
}

func (mfaAuth0AuthenticatorChannel) validateChannel(*MFAChannel) error {
	return nil
}

func init() {
	mfaChannelTypes[MFAAuth0AuthenticatorChannel] = mfaAuth0AuthenticatorChannel{}
}
