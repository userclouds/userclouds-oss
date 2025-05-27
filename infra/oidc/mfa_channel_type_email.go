package oidc

import (
	"fmt"

	"userclouds.com/infra/ucerr"
	email "userclouds.com/infra/uctypes/messaging/email/emailaddress"
)

type mfaEmailChannel struct{}

func (mfaEmailChannel) canConfigure() bool {
	return true
}

func (mfaEmailChannel) canReissueChallenge() bool {
	return true
}

func (mfaEmailChannel) getAuditLogType() string {
	return "UC Email MFA"
}

func (c mfaEmailChannel) getChallengeDescription(mfac MFAChannel, shouldMask bool, firstChallenge bool) string {
	return fmt.Sprintf("Enter Code Sent Via Email to %s", c.getChannelDescription(mfac, shouldMask))
}

func (mfaEmailChannel) getChannelDescription(mfac MFAChannel, shouldMask bool) string {
	if shouldMask {
		address := email.Address(mfac.ChannelName)
		return address.Mask()
	}
	return mfac.ChannelName
}

func (mfaEmailChannel) getRegistrationInfo(MFAChannel) (string, string, bool) {
	return "", "", false
}

func (mfaEmailChannel) getUniqueID(mfac MFAChannel) string {
	return fmt.Sprintf("%v:%s", MFAEmailChannel, mfac.ChannelTypeID)
}

func (mfaEmailChannel) getUserDetailDescription(mfac MFAChannel) string {
	return mfac.ChannelTypeID
}

func (mfaEmailChannel) validateChannel(mfac *MFAChannel) error {
	if mfac.ChannelTypeID != mfac.ChannelName {
		return ucerr.Errorf("channel type id and name must match for channel of type '%v'", MFAEmailChannel)
	}

	address := email.Address(mfac.ChannelName)
	if err := address.Validate(); err != nil {
		return ucerr.Friendlyf(ucerr.Wrap(err), "email address is invalid")
	}

	return nil
}

func init() {
	mfaChannelTypes[MFAEmailChannel] = mfaEmailChannel{}
}
