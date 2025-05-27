package oidc

import (
	"userclouds.com/infra/ucerr"
)

var mfaChannelTypes = map[MFAChannelType]mfaChannelType{}

type mfaChannelType interface {
	canConfigure() bool
	canReissueChallenge() bool
	getAuditLogType() string
	getChallengeDescription(mfac MFAChannel, shouldMask bool, firstChallenge bool) string
	getChannelDescription(mfac MFAChannel, shouldMask bool) string
	getRegistrationInfo(MFAChannel) (link string, qrCode string, ok bool)
	getUniqueID(MFAChannel) string
	getUserDetailDescription(MFAChannel) string
	validateChannel(*MFAChannel) error
}

type invalidChannel struct{}

func (invalidChannel) canConfigure() bool {
	return false
}

func (invalidChannel) canReissueChallenge() bool {
	return false
}

func (invalidChannel) getAuditLogType() string {
	return ""
}

func (invalidChannel) getChallengeDescription(MFAChannel, bool, bool) string {
	return ""
}

func (invalidChannel) getChannelDescription(MFAChannel, bool) string {
	return ""
}

func (invalidChannel) getRegistrationInfo(MFAChannel) (string, string, bool) {
	return "", "", false
}

func (invalidChannel) getUniqueID(MFAChannel) string {
	return ""
}

func (invalidChannel) getUserDetailDescription(MFAChannel) string {
	return ""
}

func (invalidChannel) validateChannel(*MFAChannel) error {
	return ucerr.New("invalid channel type")
}

func (ct MFAChannelType) getChannelType() mfaChannelType {
	channelType, found := mfaChannelTypes[ct]
	if found {
		return channelType
	}

	return invalidChannel{}
}

// CanConfigure returns true if channels of this type can be configured
func (ct MFAChannelType) CanConfigure() bool {
	return ct.getChannelType().canConfigure()
}

// CanReissueChallenge returns true if channels of this type can reissue a new challenge
func (ct MFAChannelType) CanReissueChallenge() bool {
	return ct.getChannelType().canReissueChallenge()
}

// GetAuditLogType returns a string description appropriate for the audit log
func (ct MFAChannelType) GetAuditLogType() string {
	return ct.getChannelType().getAuditLogType()
}

// IsRecoveryCode returns true if this is the MFA recovery code channel type
func (ct MFAChannelType) IsRecoveryCode() bool {
	return ct == MFARecoveryCodeChannel
}

// String will convert an MFAChannelType to a string
func (ct MFAChannelType) String() string {
	return string(ct)
}

// Validate implements the validatable interface
func (ct MFAChannelType) Validate() error {
	if _, found := mfaChannelTypes[ct]; !found {
		return ucerr.Errorf("invalid MFAChannelType: %v", ct)
	}

	return nil
}

// MFAChannelTypeSet is a set of MFA channel types
type MFAChannelTypeSet map[MFAChannelType]bool

// MFAChannelTypes is a db-appropriate set of MFA channel types
type MFAChannelTypes struct {
	ChannelTypes MFAChannelTypeSet `json:"mfa_channel_types" yaml:"mfa_channel_types"`
}

// IncludesType returns true if the channelType is found
func (mfact MFAChannelTypes) IncludesType(channelType MFAChannelType) bool {
	return mfact.ChannelTypes[channelType]
}

// Validate implements the Validatable interface
func (mfact *MFAChannelTypes) Validate() error {
	for ct := range mfact.ChannelTypes {
		if err := ct.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

//go:generate gendbjson MFAChannelTypes
