package oidc

import "fmt"

type mfaAuth0EmailChannel struct{}

func (mfaAuth0EmailChannel) canConfigure() bool {
	return false
}

func (mfaAuth0EmailChannel) canReissueChallenge() bool {
	return true
}

func (mfaAuth0EmailChannel) getAuditLogType() string {
	return "Auth0 Email MFA"
}

func (c mfaAuth0EmailChannel) getChallengeDescription(mfac MFAChannel, shouldMask bool, firstChallenge bool) string {
	return fmt.Sprintf("Enter Code Sent Via Email to %s", c.getChannelDescription(mfac, shouldMask))
}

func (mfaAuth0EmailChannel) getChannelDescription(mfac MFAChannel, shouldMask bool) string {
	return mfac.ChannelName
}

func (mfaAuth0EmailChannel) getRegistrationInfo(MFAChannel) (string, string, bool) {
	return "", "", false
}

func (mfaAuth0EmailChannel) getUniqueID(mfac MFAChannel) string {
	return fmt.Sprintf("%v:%s", MFAAuth0EmailChannel, mfac.ChannelTypeID)
}

func (mfaAuth0EmailChannel) getUserDetailDescription(mfac MFAChannel) string {
	return mfac.ChannelTypeID
}

func (mfaAuth0EmailChannel) validateChannel(*MFAChannel) error {
	return nil
}

func init() {
	mfaChannelTypes[MFAAuth0EmailChannel] = mfaAuth0EmailChannel{}
}
