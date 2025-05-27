package oidc_test

import (
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/oidc"
)

func TestChannelTypeValidity(t *testing.T) {
	assert.NoErr(t, oidc.MFAEmailChannel.Validate())
	assert.NoErr(t, oidc.MFASMSChannel.Validate())
	assert.NoErr(t, oidc.MFAAuthenticatorChannel.Validate())
	assert.NoErr(t, oidc.MFAAuth0AuthenticatorChannel.Validate())
	assert.NoErr(t, oidc.MFAAuth0EmailChannel.Validate())
	assert.NoErr(t, oidc.MFAAuth0SMSChannel.Validate())
	assert.NoErr(t, oidc.MFARecoveryCodeChannel.Validate())
	assert.NotNil(t, oidc.MFAInvalidChannel.Validate())
	badChannelType := oidc.MFAChannelType("foo")
	assert.NotNil(t, badChannelType.Validate())
}
