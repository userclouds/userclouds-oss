package messageelements

import (
	"fmt"
	"strings"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
)

type defaultParameterGetter func() any

const defaultEmailSenderName = "UserClouds"

const defaultSMSSender = "+1111111111"

var defaultElements = map[MessageType]map[ElementType]string{}

var defaultParameterGetters = map[MessageType]defaultParameterGetter{}

var emailMessageTypes = []MessageType{}

var smsMessageTypes = []MessageType{}

func getDefaultEmailSender() string {
	var emailDomain string
	if universe.Current().IsProd() {
		emailDomain = "userclouds.com"
	} else {
		emailDomain = fmt.Sprintf("%v.userclouds.com", universe.Current())
	}
	return fmt.Sprintf("info@%s", emailDomain)
}

func sanitizeDefault(s string) string {
	return strings.TrimSpace(s)
}

func registerEmailType(mt MessageType, sender string, senderName string, subjectTemplate string, htmlTemplate string, textTemplate string, dpg defaultParameterGetter) {
	if _, present := defaultElements[mt]; present {
		panic(fmt.Sprintf("duplicate registration for message type %s", mt))
	}

	defaultElements[mt] = map[ElementType]string{}
	defaultElements[mt][EmailSender] = sanitizeDefault(sender)
	defaultElements[mt][EmailSenderName] = sanitizeDefault(senderName)
	defaultElements[mt][EmailSubjectTemplate] = sanitizeDefault(subjectTemplate)
	defaultElements[mt][EmailHTMLTemplate] = sanitizeDefault(htmlTemplate)
	defaultElements[mt][EmailTextTemplate] = sanitizeDefault(textTemplate)

	defaultParameterGetters[mt] = dpg

	emailMessageTypes = append(emailMessageTypes, mt)
}

func registerSMSType(mt MessageType, sender string, bodyTemplate string, dpg defaultParameterGetter) {
	if _, present := defaultElements[mt]; present {
		panic(fmt.Sprintf("duplicate registration for message type %s", mt))
	}

	defaultElements[mt] = map[ElementType]string{}
	defaultElements[mt][SMSBodyTemplate] = sanitizeDefault(bodyTemplate)
	defaultElements[mt][SMSSender] = sanitizeDefault(sender)

	defaultParameterGetters[mt] = dpg

	smsMessageTypes = append(smsMessageTypes, mt)
}

// DefaultParameters returns a struct containing appropriate default parameters
// for the associated MessageType, which can be used for validating a message element
func DefaultParameters(mt MessageType) any {
	if defaultParameterGetter, present := defaultParameterGetters[mt]; present {
		return defaultParameterGetter()
	}

	return nil
}

// EmailMessageTypes returns a copy of the slice of all registered email message types
func EmailMessageTypes() (types []MessageType) {
	return append(types, emailMessageTypes...)
}

// SMSMessageTypes returns a copy of the slice of all registered sms message types
func SMSMessageTypes() (types []MessageType) {
	return append(types, smsMessageTypes...)
}

// MessageTypes returns all registered message types
func MessageTypes() (types []MessageType) {
	return append(append(types, EmailMessageTypes()...), SMSMessageTypes()...)
}

// ElementTypes returns the element types that apply for a message type
func ElementTypes(mt MessageType) (elementTypes []ElementType) {
	elementTypes = []ElementType{}
	messageTypeElements := defaultElements[mt]
	for elt := range messageTypeElements {
		elementTypes = append(elementTypes, elt)
	}
	return elementTypes
}

// MakeElementGetter returns a function with the signature of ElementGetter
// that will return the appropriate default message element for a MessageType
// and the passed in ElementType
func MakeElementGetter(mt MessageType) ElementGetter {
	return func(elt ElementType) string {
		return defaultElements[mt][elt]
	}
}

// Validate implements the Validatable interface and verifies the MessageType is valid
func (mt MessageType) Validate() error {
	if _, present := defaultElements[mt]; present {
		return nil
	}

	return ucerr.Errorf("invalid message type: %s", mt)
}
