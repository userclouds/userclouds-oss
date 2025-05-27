package messageelements

import (
	"userclouds.com/infra/ucerr"
)

// ValidateMessageElement validates a message element for a given message type and element type
func ValidateMessageElement(mt MessageType, elt ElementType, element string) error {
	if err := mt.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if err := elt.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	elementTransformer, ok := elementTypeTransformers[elt]
	if !ok {
		return ucerr.Errorf("no element transformer for element type %s", elt)
	}

	transformedElement, err := elementTransformer(elt, DefaultParameters(mt), element)
	if err != nil {
		return ucerr.Wrap(err)
	}

	elementValidator, ok := elementTypeValidators[elt]
	if !ok {
		return ucerr.Errorf("no element validator for element type %s", elt)
	}

	if err := elementValidator(transformedElement); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
