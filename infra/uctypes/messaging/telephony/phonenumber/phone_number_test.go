package phonenumber_test

import (
	"testing"

	"userclouds.com/infra/assert"
	phone "userclouds.com/infra/uctypes/messaging/telephony/phonenumber"
)

func bad(t *testing.T, pn phone.PhoneNumber) {
	t.Helper()
	assert.NotNil(t, pn.Validate())
}

func good(t *testing.T, pn phone.PhoneNumber) {
	t.Helper()
	assert.IsNil(t, pn.Validate())
}

func mask(t *testing.T, maskedNumber string, pn phone.PhoneNumber) {
	t.Helper()
	assert.IsNil(t, pn.Validate())
	assert.Equal(t, maskedNumber, pn.Mask())
}

func TestBadPhoneNumbers(t *testing.T) {
	bad(t, "")
	bad(t, "123")
	bad(t, "abc")
	bad(t, "1234567")
	bad(t, "+1")
	bad(t, "+1234567890123456")
	bad(t, "+0123456")
	bad(t, "+12")
	bad(t, "+123")
	bad(t, "+1234")
	bad(t, "+12345")
	bad(t, "+123456")
}

func TestGoodPhoneNumbers(t *testing.T) {
	good(t, "+1234567")
	good(t, "+12345678")
	good(t, "+123456789")
	good(t, "+1234567890")
	good(t, "+12345678901")
	good(t, "+123456789012")
	good(t, "+1234567890123")
	good(t, "+12345678901234")
	good(t, "+123456789012345")
}

func TestNumberMasking(t *testing.T) {
	mask(t, "****4567", "+1234567")
	mask(t, "*****5678", "+12345678")
	mask(t, "******6789", "+123456789")
	mask(t, "*******7890", "+1234567890")
	mask(t, "********8901", "+12345678901")
	mask(t, "*********9012", "+123456789012")
	mask(t, "**********0123", "+1234567890123")
	mask(t, "***********1234", "+12345678901234")
	mask(t, "************2345", "+123456789012345")
}
