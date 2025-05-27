package emailaddress

import (
	"net/mail"
	"strings"

	"userclouds.com/infra/ucerr"
)

// Address represents an email address
type Address string

// Validate implements the Validatable interface for Address
func (a Address) Validate() error {
	if _, err := a.Parse(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// Parse will parse an address according to the RFC 5322 standard,
// applying additional restrictions we have chosen to enforce
func (a Address) Parse() (*mail.Address, error) {
	parsedAddress, err := mail.ParseAddress(string(a))
	if err != nil {
		return nil, ucerr.Friendlyf(err, "invalid email: %v", err) // lint-safe-wrap because this error is from net/mail
	}
	if parsedAddress.Name != "" {
		return nil, ucerr.New("invalid email: no name allowed")
	}

	return parsedAddress, nil
}

// CombineAddress will generate a valid address string from a name part
// and an address part, ensuring that the address part was valid and did
// not already have an associated name, and that the resulting combined
// address parses correctly
func CombineAddress(name string, address string) (string, error) {
	parsedAddress, err := mail.ParseAddress(address)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	if parsedAddress.Name != "" {
		return "", ucerr.New("address had an associated name")
	}
	parsedAddress.Name = name
	combinedAddress := parsedAddress.String()
	if _, err := mail.ParseAddress(combinedAddress); err != nil {
		return "", ucerr.Wrap(err)
	}

	return combinedAddress, nil
}

func getMaskBoundaries(address string) (maskedBoundary int, localBoundary int) {
	localBoundary = strings.LastIndex(address, "@")
	switch {
	case localBoundary > 5:
		return 3, localBoundary
	case localBoundary == 5:
		return 2, localBoundary
	case localBoundary == 4:
		return 1, localBoundary
	default:
		return 0, localBoundary
	}
}

// Mask returns a masked version of an email address
func (a Address) Mask() string {
	// TODO: gint - 5/26/23 - we should ultimately rely on transformers for performing
	// masking, or at least utilize a shared implementation so our masking is consistent
	// with the default transformer policy.
	address := string(a)
	parsedAddress, err := mail.ParseAddress(address)
	if err != nil {
		// this will not happen for a valid Address
		return address
	}

	maskedBoundary, localBoundary := getMaskBoundaries(parsedAddress.Address)

	maskedAddress := strings.Split(parsedAddress.Address, "")
	for i := maskedBoundary; i < localBoundary; i++ {
		maskedAddress[i] = "*"
	}

	return strings.Join(maskedAddress, "")
}
