package emailaddress_test

import (
	"os"
	"testing"

	"userclouds.com/infra/assert"
	email "userclouds.com/infra/uctypes/messaging/email/emailaddress"
)

func confirmGoodAddress(t *testing.T, address email.Address, addressPart string) {
	t.Helper()

	assert.NoErr(t, address.Validate())
	parsedAddress, err := address.Parse()
	assert.NoErr(t, err)
	assert.Equal(t, parsedAddress.Name, "")
	assert.Equal(t, parsedAddress.Address, addressPart)
}

func confirmBadAddress(t *testing.T, address email.Address) {
	t.Helper()
	assert.NotNil(t, address.Validate(), assert.Errorf("expected error for %q", address))
}

func TestGoodEmailAddresses(t *testing.T) {
	confirmGoodAddress(t, "foo@bar.com", "foo@bar.com")
	confirmGoodAddress(t, "<foo@bar.com>", "foo@bar.com")
	confirmGoodAddress(t, " foo@bar.com ", "foo@bar.com")
	confirmGoodAddress(t, " <foo@bar.com> ", "foo@bar.com")
	confirmGoodAddress(t, "< foo@bar.com>", "foo@bar.com")
	confirmGoodAddress(t, `"n@me"@bar.com`, `n@me@bar.com`)

	// not sure we want to allow these?
	confirmGoodAddress(t, "foo@bar", "foo@bar")
	confirmGoodAddress(t, "foo@bar_baz.com", "foo@bar_baz.com")
	confirmGoodAddress(t, "foo@bar.baz_com", "foo@bar.baz_com")
	confirmGoodAddress(t, "foo@localdomainname", "foo@localdomainname")
	confirmGoodAddress(t, "verylonglocal1234567890123456789012345678901234567890123456789012345678901234567890@foo.com", "verylonglocal1234567890123456789012345678901234567890123456789012345678901234567890@foo.com")
	confirmGoodAddress(t, "postmaster@[123.123.123.123]", "postmaster@[123.123.123.123]")
}

func TestBadEmailAddresses(t *testing.T) {
	confirmBadAddress(t, "")
	confirmBadAddress(t, "foo.com")
	confirmBadAddress(t, "@foo.com")
	confirmBadAddress(t, "@foo.com")
	confirmBadAddress(t, "Foo Bar foo@bar.com")
	confirmBadAddress(t, "Foo Bar <foo@bar.com>")
	confirmBadAddress(t, "< foo@bar.com >")
	confirmBadAddress(t, "<foo@bar.com >")
	confirmBadAddress(t, "<foo@bar.com")
	confirmBadAddress(t, "foo@bar.com>")
	confirmBadAddress(t, "foo@bar@baz.com")
	confirmBadAddress(t, "not_an_email_address")
	confirmBadAddress(t, "postmaster@[IPv6:2001:0db8:85a3:0000:0000:8a2e:0370:7334]")
}

func confirmBadCombinedAddress(t *testing.T, name string, address string) {
	t.Helper()

	_, err := email.CombineAddress(name, address)
	assert.NotNil(t, err, assert.Errorf("expected error for %q %q", name, address))
}

func confirmGoodCombinedAddress(t *testing.T, name string, address string, combinedAddress string) {
	t.Helper()

	ca, err := email.CombineAddress(name, address)
	assert.NoErr(t, err)
	assert.Equal(t, ca, combinedAddress)
}

func TestCombinedAddresses(t *testing.T) {
	confirmBadCombinedAddress(t, "foo", "\"name\" <foo@bar.com>")
	confirmBadCombinedAddress(t, "foo", "foo")

	confirmGoodCombinedAddress(t, "Foo Bar", "<foo@bar.com>", "\"Foo Bar\" <foo@bar.com>")
	confirmGoodCombinedAddress(t, "Foo Bar", "foo@bar.com", "\"Foo Bar\" <foo@bar.com>")
	confirmGoodCombinedAddress(t, "Userclouds", "info@userclouds.com", "\"Userclouds\" <info@userclouds.com>")
}

func mask(t *testing.T, maskedAddress string, address email.Address) {
	t.Helper()

	assert.NoErr(t, address.Validate())
	assert.Equal(t, maskedAddress, address.Mask())
}

func TestMasking(t *testing.T) {
	mask(t, "*@foo.com", "a@foo.com")
	mask(t, "**@foo.com", "ab@foo.com")
	mask(t, "***@foo.com", "abc@foo.com")
	mask(t, "a***@foo.com", "abcd@foo.com")
	mask(t, "ab***@foo.com", "abcde@foo.com")
	mask(t, "abc***@foo.com", "abcdef@foo.com")
	mask(t, "abc****@foo.com", "abcdefg@foo.com")
	mask(t, "abc****@foo.com", `"abc@efg"@foo.com`)
}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../..")
	os.Exit(m.Run())
}
