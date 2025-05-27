package column

import (
	"regexp"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/messaging/email/emailaddress"
	"userclouds.com/infra/uctypes/messaging/telephony/phonenumber"
	"userclouds.com/infra/uctypes/set"
)

// boolValidatorMaker is a function that produces a bool validator
type boolValidatorMaker func(DataType, Constraints) func(bool) error

// compositeValidatorMaker is a function that produces a composite validator
type compositeValidatorMaker func(DataType, Constraints) func(userstore.CompositeValue) error

// intValidatorMaker is a function that produces an int validator
type intValidatorMaker func(DataType, Constraints) func(int) error

// stringValidatorMaker is a function that produces a string validator
type stringValidatorMaker func(DataType, Constraints) func(string) error

// timestampValidatorMaker is a function that produces a timestamp validator
type timestampValidatorMaker func(DataType, Constraints) func(time.Time) error

// uuidValidatorMaker is a function that produces a UUID validator
type uuidValidatorMaker func(DataType, Constraints) func(uuid.UUID) error

var invalidBool = func(DataType, Constraints) func(bool) error {
	return func(bool) error { return ucerr.New("bool unsupported") }
}

var invalidComposite = func(DataType, Constraints) func(userstore.CompositeValue) error {
	return func(userstore.CompositeValue) error { return ucerr.New("userstore.Composite unsupported") }
}

var invalidInt = func(DataType, Constraints) func(int) error {
	return func(int) error { return ucerr.New("int unsupported") }
}

var invalidString = func(DataType, Constraints) func(string) error {
	return func(string) error { return ucerr.New("string unsupported") }
}

var invalidTimestamp = func(DataType, Constraints) func(time.Time) error {
	return func(time.Time) error { return ucerr.New("time.Time unsupported") }
}

var invalidUUID = func(DataType, Constraints) func(uuid.UUID) error {
	return func(uuid.UUID) error { return ucerr.New("uuid.UUID unsupported") }
}

var makeUniqueValidator = func(dt DataType, c Constraints) func(any) error {
	uniqueValues := map[any]bool{}

	return func(v any) error {
		if c.UniqueRequired {
			// exclude any aspects of value that should be ignored
			// for uniqueness comparison
			uniqueValue, err := dt.getComparableValue(v, true)
			if err != nil {
				return ucerr.Wrap(err)
			}

			if uniqueValues[uniqueValue] {
				return ucerr.Friendlyf(nil, "value '%v' is not unique", v)
			}
			uniqueValues[uniqueValue] = true
		}

		return nil
	}
}

var makeUniqueIDValidator = func(dt DataType, c Constraints) func(any) error {
	uniqueIDs := set.NewStringSet()

	return func(v any) error {
		id, err := dt.GetUniqueID(v)
		if err != nil {
			return ucerr.Wrap(err)
		}

		if c.UniqueIDRequired {
			if id == "" {
				return ucerr.Friendlyf(nil, "ID is required but not specified for value '%v'", v)
			}

			if uniqueIDs.Contains(id) {
				return ucerr.Friendlyf(nil, "ID is not unique for value '%v'", v)
			}

			uniqueIDs.Insert(id)
		}

		return nil
	}
}

var makeChainedValidator = func(
	dt DataType,
	c Constraints,
	validatorMakers ...func(DataType, Constraints) func(any) error,
) func(any) error {
	var validators []func(any) error
	for _, validatorMaker := range validatorMakers {
		validators = append(validators, validatorMaker(dt, c))
	}

	return func(v any) error {
		for _, validator := range validators {
			if err := validator(v); err != nil {
				return ucerr.Wrap(err)
			}
		}
		return nil
	}
}

var validateBool = func(dt DataType, c Constraints) func(bool) error {
	validator := makeUniqueValidator(dt, c)
	return func(b bool) error { return ucerr.Wrap(validator(b)) }
}

var validateInt = func(dt DataType, c Constraints) func(int) error {
	validator := makeUniqueValidator(dt, c)
	return func(i int) error { return ucerr.Wrap(validator(i)) }
}

var validateString = func(dt DataType, c Constraints) func(string) error {
	validator := makeUniqueValidator(dt, c)
	return func(s string) error { return ucerr.Wrap(validator(s)) }
}

var validateTimestamp = func(dt DataType, c Constraints) func(time.Time) error {
	validator := makeUniqueValidator(dt, c)
	return func(t time.Time) error { return ucerr.Wrap(validator(t)) }
}

var validateUUID = func(dt DataType, c Constraints) func(uuid.UUID) error {
	validator := makeUniqueValidator(dt, c)
	return func(id uuid.UUID) error { return ucerr.Wrap(validator(id)) }
}

var validateComposite = func(dt DataType, c Constraints) func(userstore.CompositeValue) error {
	validator := makeChainedValidator(dt, c, makeUniqueValidator, makeUniqueIDValidator)
	return func(cv userstore.CompositeValue) error {
		foundFields := set.NewStringSet()

		for _, field := range dt.CompositeAttributes.Fields {
			if value, found := cv[field.StructName]; found {
				foundFields.Insert(field.StructName)
				var v Value
				if err := v.setFieldValue(field.DataTypeID, value); err != nil {
					return ucerr.Friendlyf(err, "composite value field '%s' has invalid value '%v'", field.Name, value)
				}
			} else if field.Required {
				return ucerr.Friendlyf(nil, "composite value is missing required field '%s'", field.StructName)
			}
		}

		if len(cv) > foundFields.Size() {
			var unsupportedFields []string
			for name := range cv {
				if !foundFields.Contains(name) {
					unsupportedFields = append(unsupportedFields, name)
				}
			}
			return ucerr.Friendlyf(nil, "composite value has unsupported fields: '%v'", unsupportedFields)
		}

		return ucerr.Wrap(validator(cv))
	}
}

var validateDate = func(dt DataType, c Constraints) func(time.Time) error {
	validator := makeUniqueValidator(dt, c)
	return func(t time.Time) error {
		if t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0 || t.Nanosecond() != 0 || t.Location() != time.UTC {
			return ucerr.Friendlyf(nil, "invalid date: %v", t)
		}

		return ucerr.Wrap(validator(t))
	}
}

var validateEmail = func(dt DataType, c Constraints) func(string) error {
	validator := makeUniqueValidator(dt, c)
	return func(s string) error {
		if s != "" {
			ea := emailaddress.Address(s)
			if err := ea.Validate(); err != nil {
				return ucerr.Wrap(err)
			}
		}

		return ucerr.Wrap(validator(s))
	}
}

var validateE164Phone = func(dt DataType, c Constraints) func(string) error {
	validator := makeUniqueValidator(dt, c)
	return func(s string) error {
		if s != "" {
			pn := phonenumber.PhoneNumber(s)
			if err := pn.Validate(); err != nil {
				return ucerr.Wrap(err)
			}
		}

		return ucerr.Wrap(validator(s))
	}
}

var phoneRegex = regexp.MustCompile(`^([+]?[\s0-9]+)?(\d{3}|[(]?[0-9]+[)])?([-]?[\s]?[0-9])+$`)
var validatePhone = func(dt DataType, c Constraints) func(string) error {
	validator := makeUniqueValidator(dt, c)
	return func(s string) error {
		if s != "" && !phoneRegex.MatchString(s) {
			return ucerr.Friendlyf(nil, "invalid phone number: %s", s)
		}

		return ucerr.Wrap(validator(s))
	}
}

var ssnRegex = regexp.MustCompile(`^\d{3}-?\d{2}-?\d{4}$`)
var validateSSN = func(dt DataType, c Constraints) func(string) error {
	validator := makeUniqueValidator(dt, c)
	return func(s string) error {
		if s != "" && !ssnRegex.MatchString(s) {
			return ucerr.Friendlyf(nil, "invalid SSN: %s", s)
		}

		return ucerr.Wrap(validator(s))
	}
}
