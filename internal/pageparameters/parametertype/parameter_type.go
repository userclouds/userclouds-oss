package parametertype

import (
	"userclouds.com/infra/ucerr"
)

type parameterValidator func(parameterValue string) bool

var parameterTypes = []Type{}

var parameterValidators = map[Type]parameterValidator{}

func registerParameterType(t Type, v parameterValidator) error {
	if _, present := parameterValidators[t]; present {
		return ucerr.Errorf("duplicate registration for parameter type '%s'", t)
	}

	parameterValidators[t] = v

	parameterTypes = append(parameterTypes, t)

	return nil
}

// Types returns a copy of the slice of all registered parameter types
func Types() (types []Type) {
	return append(types, parameterTypes...)
}

// IsValid will apply parameter type specific validation logic to the passed in parameter value
func (t Type) IsValid(parameterValue string) bool {
	validator, present := parameterValidators[t]
	if !present {
		return false
	}

	return validator(parameterValue)
}

// Validate implements the Validatable interface and verifies the ParameterType is valid
func (t Type) Validate() error {
	if _, present := parameterValidators[t]; !present {
		return ucerr.Errorf("invalid parameter type: %s", t)
	}

	return nil
}
