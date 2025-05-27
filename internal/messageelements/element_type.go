package messageelements

import (
	"fmt"

	"userclouds.com/infra/ucerr"
)

// ElementType constants for customizable elements of
// user-facing message - these are used for storing message
// element overrides in plexmap and must be kept in
// synch
const (
	EmailSender          ElementType = "sender"
	EmailSenderName      ElementType = "sender_name"
	EmailSubjectTemplate ElementType = "subject_template"
	EmailHTMLTemplate    ElementType = "html_template"
	EmailTextTemplate    ElementType = "text_template"
	SMSBodyTemplate      ElementType = "sms_body_template"
	SMSSender            ElementType = "sms_sender"
)

var elementTypeTransformers = map[ElementType]elementTransformer{}
var elementTypeValidators = map[ElementType]elementValidator{}

func registerElementType(elt ElementType, et elementTransformer, ev elementValidator) {
	if _, present := elementTypeTransformers[elt]; present {
		panic(fmt.Sprintf("duplicate registration for element type %s", elt))
	}

	elementTypeTransformers[elt] = et
	elementTypeValidators[elt] = ev
}

// Validate implements the Validatable interface and verifies the ElementType is valid
func (elt ElementType) Validate() error {
	if _, present := elementTypeTransformers[elt]; present {
		return nil
	}

	return ucerr.Errorf("invalid element type: %s", elt)
}

func init() {
	registerElementType(EmailSender, identityTransformer, emailValidator)
	registerElementType(EmailSenderName, identityTransformer, emailNameValidator)
	registerElementType(EmailSubjectTemplate, templateTransformer, sizeValidator)
	registerElementType(EmailHTMLTemplate, templateTransformer, sanitizationValidator)
	registerElementType(EmailTextTemplate, templateTransformer, chainElementValidator(sizeValidator, sanitizationValidator))
	registerElementType(SMSBodyTemplate, templateTransformer, chainElementValidator(sizeValidator, sanitizationValidator))
	registerElementType(SMSSender, identityTransformer, phoneNumberValidator)
}
