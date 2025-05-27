package oidc

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"

	"github.com/pquerna/otp"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/messaging/email/emailaddress"
)

// AuthenticatorType is an authenticator app type
type AuthenticatorType int

// Supported authenticator app types
const (
	AuthenticatorTypeAuth0Guardian          AuthenticatorType = 1
	AuthenticatorTypeDuoMobile              AuthenticatorType = 2
	AuthenticatorTypeGoogleAuthenticator    AuthenticatorType = 3
	AuthenticatorTypeMicrosoftAuthenticator AuthenticatorType = 4
	AuthenticatorTypeTwilioAuthy            AuthenticatorType = 5
)

//go:generate genconstant AuthenticatorType

var authenticatorTypeDescriptions = map[string]string{}

func initAuthenticatorType(at AuthenticatorType, description string) {
	if _, found := authenticatorTypeDescriptions[at.String()]; found {
		panic(fmt.Sprintf("duplicate AuthenticatorType %v", at))
	}
	authenticatorTypeDescriptions[at.String()] = description
}

// AuthenticatorTypeDescription includes a description of the associated AuthenticatorType
type AuthenticatorTypeDescription struct {
	AuthenticatorType AuthenticatorType `json:"authenticator_type" yaml:"authenticator_type"`
	Description       string            `json:"description" yaml:"description"`
}

// GetAuthenticatorTypes returns a list of all AuthenticatorTypeDescriptions
func GetAuthenticatorTypes() (ats []AuthenticatorTypeDescription) {
	for _, at := range AllAuthenticatorTypes {
		if description, found := authenticatorTypeDescriptions[at.String()]; found {
			ats = append(ats, AuthenticatorTypeDescription{at, description})
		}
	}

	return ats
}

type mfaAuthenticatorChannel struct{}

func (mfaAuthenticatorChannel) canConfigure() bool {
	return true
}

func (mfaAuthenticatorChannel) canReissueChallenge() bool {
	return false
}

func (mfaAuthenticatorChannel) getAuditLogType() string {
	return "UC Authenticator MFA"
}

func (c mfaAuthenticatorChannel) getChallengeDescription(mfac MFAChannel, shouldMask bool, firstChallenge bool) string {
	if !mfac.Verified && firstChallenge {
		return fmt.Sprintf("Scan QR Code with %s to Register and Enter Code",
			c.getChannelDescription(mfac, shouldMask))
	}

	return fmt.Sprintf("Enter Code from %s", c.getChannelDescription(mfac, shouldMask))
}

func (mfaAuthenticatorChannel) getChannelDescription(mfac MFAChannel, shouldMask bool) string {
	description, found := authenticatorTypeDescriptions[mfac.ChannelName]
	if found {
		return description
	}

	return ""
}

func (mfaAuthenticatorChannel) getRegistrationInfo(mfac MFAChannel) (link string, qrCode string, ok bool) {
	key, err := otp.NewKeyFromURL(mfac.ChannelTypeID)
	if err != nil {
		return "", "", false
	}

	qrCodeImg, err := key.Image(200, 200)
	if err != nil {
		return "", "", false
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, qrCodeImg); err != nil {
		return "", "", false
	}

	return mfac.ChannelTypeID,
		fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(buf.Bytes())),
		true
}

func (mfaAuthenticatorChannel) getUniqueID(mfac MFAChannel) string {
	return fmt.Sprintf("%v:%s", MFAAuthenticatorChannel, mfac.ChannelName)
}

func (mfaAuthenticatorChannel) getUserDetailDescription(mfac MFAChannel) string {
	return fmt.Sprintf("%s: %s", mfac.ChannelName, mfac.ChannelTypeID)
}

func (c mfaAuthenticatorChannel) validateChannel(mfac *MFAChannel) error {
	if _, found := authenticatorTypeDescriptions[mfac.ChannelName]; !found {
		return ucerr.Errorf("unsupported channel name: %s", mfac.ChannelName)
	}

	key, err := otp.NewKeyFromURL(mfac.ChannelTypeID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if key.Type() != "totp" {
		return ucerr.Errorf("authenticator app otpauth URI has unexpected type: '%s'", mfac.ChannelTypeID)
	}

	if key.Issuer() == "" {
		return ucerr.Errorf("authenticator app otpauth URI is missing issuer: '%s'", mfac.ChannelTypeID)
	}

	address := emailaddress.Address(key.AccountName())
	if err := address.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if key.Secret() == "" {
		return ucerr.Errorf("authenticator app otpauth URI is missing secret: '%s'", mfac.ChannelTypeID)
	}

	return nil
}

func init() {
	mfaChannelTypes[MFAAuthenticatorChannel] = mfaAuthenticatorChannel{}

	initAuthenticatorType(AuthenticatorTypeAuth0Guardian, "Auth0 Guardian")
	initAuthenticatorType(AuthenticatorTypeDuoMobile, "Duo Mobile")
	initAuthenticatorType(AuthenticatorTypeGoogleAuthenticator, "Google Authenticator")
	initAuthenticatorType(AuthenticatorTypeMicrosoftAuthenticator, "Microsoft Authenticator")
	initAuthenticatorType(AuthenticatorTypeTwilioAuthy, "Twilio Authy")

	for _, at := range AllAuthenticatorTypes {
		if _, found := authenticatorTypeDescriptions[at.String()]; !found {
			panic(fmt.Sprintf("AuthenticatorType %v does not have a description", at))
		}
	}
}
