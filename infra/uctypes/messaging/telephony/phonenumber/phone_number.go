package phonenumber

import (
	"regexp"
	"strings"

	"userclouds.com/infra/ucerr"
)

// PhoneNumber represents a phone number, expressed in the E.164 format
type PhoneNumber string

// While there is no stated minimum length for an E.164 phone number (which is 3 to 16
// characters long, including the '+' prefix, it appears that it is a safe assumption
// that a valid E.164 phone number must be at least 8 characters long ('+' followed by
// at least 7 digits and at most 15):
//
// https://stackoverflow.com/questions/14894899/what-is-the-minimum-length-of-a-valid-international-phone-number
var e164Regexp = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

const totalUnmaskedDigits = 4

// Mask returns a masked version of a phone number
func (pn PhoneNumber) Mask() string {
	// TODO: gint - 5/26/23 - we should ultimately rely on transformers for performing
	// masking, or at least utilize a shared implementation so our masking is consistent
	// with the default transformer policy.
	number := string(pn)
	maskedNumber := strings.Split(number, "")
	for i := range len(maskedNumber) - totalUnmaskedDigits {
		maskedNumber[i] = "*"
	}
	return strings.Join(maskedNumber, "")
}

// Validate implements the Validatable interface for PhoneNumber
func (pn PhoneNumber) Validate() error {
	if !e164Regexp.MatchString(string(pn)) {
		return ucerr.Errorf("phone number '%v' is not in E.164 format", pn)
	}
	return nil
}
