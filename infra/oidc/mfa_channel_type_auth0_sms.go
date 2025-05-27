package oidc

import "fmt"

type mfaAuth0SMSChannel struct{}

func (mfaAuth0SMSChannel) canConfigure() bool {
	return false
}

func (mfaAuth0SMSChannel) canReissueChallenge() bool {
	return true
}

func (mfaAuth0SMSChannel) getAuditLogType() string {
	return "Auth0 SMS MFA"
}

func (c mfaAuth0SMSChannel) getChallengeDescription(mfac MFAChannel, shouldMask bool, firstChallenge bool) string {
	return fmt.Sprintf("Enter Code Sent Via SMS to %s", c.getChannelDescription(mfac, shouldMask))
}

func (mfaAuth0SMSChannel) getChannelDescription(mfac MFAChannel, shouldMask bool) string {
	return mfac.ChannelName
}

func (mfaAuth0SMSChannel) getRegistrationInfo(MFAChannel) (string, string, bool) {
	return "", "", false
}

func (mfaAuth0SMSChannel) getUniqueID(mfac MFAChannel) string {
	return fmt.Sprintf("%v:%s", MFAAuth0SMSChannel, mfac.ChannelTypeID)
}

func (mfaAuth0SMSChannel) getUserDetailDescription(mfac MFAChannel) string {
	return mfac.ChannelTypeID
}

func (mfaAuth0SMSChannel) validateChannel(*MFAChannel) error {
	return nil
}

func init() {
	mfaChannelTypes[MFAAuth0SMSChannel] = mfaAuth0SMSChannel{}
}
