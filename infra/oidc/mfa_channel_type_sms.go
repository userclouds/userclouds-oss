package oidc

import (
	"fmt"

	"userclouds.com/infra/ucerr"
	phone "userclouds.com/infra/uctypes/messaging/telephony/phonenumber"
)

type mfaSMSChannel struct{}

func (mfaSMSChannel) canConfigure() bool {
	return true
}

func (mfaSMSChannel) canReissueChallenge() bool {
	return true
}

func (mfaSMSChannel) getAuditLogType() string {
	return "UC SMS MFA"
}

func (c mfaSMSChannel) getChallengeDescription(mfac MFAChannel, shouldMask bool, firstChallenge bool) string {
	return fmt.Sprintf("Enter Code Sent Via SMS to %s", c.getChannelDescription(mfac, shouldMask))
}

func (mfaSMSChannel) getChannelDescription(mfac MFAChannel, shouldMask bool) string {
	if shouldMask {
		phoneNumber := phone.PhoneNumber(mfac.ChannelName)
		return phoneNumber.Mask()
	}
	return mfac.ChannelName
}

func (mfaSMSChannel) getRegistrationInfo(MFAChannel) (string, string, bool) {
	return "", "", false
}

func (mfaSMSChannel) getUniqueID(mfac MFAChannel) string {
	return fmt.Sprintf("%v:%s", MFASMSChannel, mfac.ChannelTypeID)
}

func (mfaSMSChannel) getUserDetailDescription(mfac MFAChannel) string {
	return mfac.ChannelTypeID
}

func (mfaSMSChannel) validateChannel(mfac *MFAChannel) error {
	if mfac.ChannelTypeID != mfac.ChannelName {
		return ucerr.Errorf("channel type id and name must match for channel of type '%v'", MFASMSChannel)
	}

	phoneNumber := phone.PhoneNumber(mfac.ChannelName)
	if err := phoneNumber.Validate(); err != nil {
		return ucerr.Friendlyf(ucerr.Wrap(err), "phone number is invalid")
	}

	return nil
}

func init() {
	mfaChannelTypes[MFASMSChannel] = mfaSMSChannel{}
}
