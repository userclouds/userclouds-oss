package messageelements

import (
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/messaging/email/emailaddress"
	phone "userclouds.com/infra/uctypes/messaging/telephony/phonenumber"
)

// functions that implement this interface can be registered with an element type
// and used for validation
type elementValidator func(element string) error

var emailValidator = func(element string) error {
	address := emailaddress.Address(element)
	if err := address.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

var emailNameValidator = func(element string) error {
	if _, err := emailaddress.CombineAddress(element, getDefaultEmailSender()); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

var phoneNumberValidator = func(element string) error {
	phoneNumber := phone.PhoneNumber(element)
	if err := phoneNumber.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

var sizeValidator = func(element string) error {
	// TODO: can use this to make sure the element does not exceed a specific size
	return nil
}

var sanitizationValidator = func(element string) error {
	// TODO: can use this to make sure the element has been appropriately sanitized
	return nil
}

// this function can be used to chain two validators together, allowing us to have a
// series of validations for a given element type
func chainElementValidator(v1 elementValidator, v2 elementValidator) elementValidator {
	return func(element string) error {
		if err := v1(element); err != nil {
			return ucerr.Wrap(err)
		}
		if err := v2(element); err != nil {
			return ucerr.Wrap(err)
		}
		return nil
	}
}
