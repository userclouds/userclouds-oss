package oidc

import "userclouds.com/infra/ucerr"

type mfaRecoveryCodeChannel struct{}

func (mfaRecoveryCodeChannel) canConfigure() bool {
	return false
}

func (mfaRecoveryCodeChannel) canReissueChallenge() bool {
	return false
}

func (mfaRecoveryCodeChannel) getAuditLogType() string {
	return "Recovery Code"
}

func (mfaRecoveryCodeChannel) getChallengeDescription(MFAChannel, bool, bool) string {
	return "Enter Recovery Code"
}

func (mfaRecoveryCodeChannel) getChannelDescription(MFAChannel, bool) string {
	return "Recovery Code"
}

func (mfaRecoveryCodeChannel) getRegistrationInfo(MFAChannel) (string, string, bool) {
	return "", "", false
}

func (mfaRecoveryCodeChannel) getUniqueID(MFAChannel) string {
	return MFARecoveryCodeChannel.String()
}

func (mfaRecoveryCodeChannel) getUserDetailDescription(mfac MFAChannel) string {
	return mfac.ChannelTypeID
}

func (mfaRecoveryCodeChannel) validateChannel(mfac *MFAChannel) error {
	if mfac.Primary {
		return ucerr.Errorf("channel of type '%v' cannot be primary channel", MFARecoveryCodeChannel)
	}

	return nil
}

func init() {
	mfaChannelTypes[MFARecoveryCodeChannel] = mfaRecoveryCodeChannel{}
}
