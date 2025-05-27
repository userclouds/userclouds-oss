package api

import (
	"reflect"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
	message "userclouds.com/internal/messageelements"
)

// MessageElement contains a default element value and a custom
// element value for an element type. The custom value is set
// to the empty string if not specified.
type MessageElement struct {
	Type         message.ElementType `json:"type"`
	CustomValue  string              `json:"custom_value"`
	DefaultValue string              `json:"default_value" validate:"notempty"`
}

//go:generate genvalidate MessageElement

// MessageParameter contains a parameter name and an appropriate
// default value for that parameter.
type MessageParameter struct {
	Name         string `json:"name" validate:"notempty"`
	DefaultValue string `json:"default_value" validate:"notempty"`
}

//go:generate genvalidate MessageParameter

// MessageTypeMessageElements contains a map of element types to message
// elements and an array of parameters for the given message type.
type MessageTypeMessageElements struct {
	Type              message.MessageType                    `json:"type"`
	MessageElements   map[message.ElementType]MessageElement `json:"message_elements"`
	MessageParameters []MessageParameter                     `json:"message_parameters" validate:"unique,Name" keytype:"string"`
}

//go:generate genvalidate MessageTypeMessageElements

func (mtme *MessageTypeMessageElements) extraValidate() error {
	if len(mtme.MessageElements) != len(message.ElementTypes(mtme.Type)) {
		return ucerr.Friendlyf(nil, "Expected '%d' MessageElements but have '%d'", len(message.ElementTypes(mtme.Type)), len(mtme.MessageElements))
	}
	for key, val := range mtme.MessageElements {
		if err := val.Validate(); err != nil {
			return ucerr.Friendlyf(err, "MessageElements value for key '%s' is invalid", key)
		}
		if key != val.Type {
			return ucerr.Friendlyf(nil, "MessageElements key '%s' does not match MessageElement.Type '%s'", key, val.Type)
		}
	}

	return nil
}

// AppMessageElements contains a map of message type to message type message elements for the given app id.
type AppMessageElements struct {
	AppID                      uuid.UUID                                          `json:"app_id" validate:"notnil"`
	MessageTypeMessageElements map[message.MessageType]MessageTypeMessageElements `json:"message_type_message_elements"`
}

//go:generate genvalidate AppMessageElements

func (ame *AppMessageElements) extraValidate() error {
	for key, val := range ame.MessageTypeMessageElements {
		if err := val.Validate(); err != nil {
			return ucerr.Friendlyf(err, "MessageTypeMessageElements value for key '%s' is invalid", key)
		}
		if key != val.Type {
			return ucerr.Friendlyf(nil, "MessageTypeMessageElements key '%s' does not match MessageTypeMessageElements.Type '%s'", key, val.Type)
		}
	}

	return nil
}

// TenantAppMessageElements contains an array of app message elements
// for the given tenant id.
type TenantAppMessageElements struct {
	TenantID           uuid.UUID            `json:"tenant_id" validate:"notnil"`
	AppMessageElements []AppMessageElements `json:"app_message_elements" validate:"unique,AppID" keytype:"github.com/gofrs/uuid.UUID"`
}

//go:generate genvalidate TenantAppMessageElements

// ModifiedMessageTypeMessageElements contains a map of message element types to modified
// message elements for the specified tenant, app, and message type
type ModifiedMessageTypeMessageElements struct {
	TenantID        uuid.UUID                      `json:"tenant_id" validate:"notnil"`
	AppID           uuid.UUID                      `json:"app_id" validate:"notnil"`
	MessageType     message.MessageType            `json:"message_type"`
	MessageElements map[message.ElementType]string `json:"message_elements"`
}

//go:generate genvalidate ModifiedMessageTypeMessageElements

func (mmtme *ModifiedMessageTypeMessageElements) extraValidate() error {
	if len(mmtme.MessageElements) != len(message.ElementTypes(mmtme.MessageType)) {
		return ucerr.Friendlyf(nil, "Expected '%d' MessageElements but have '%d'", len(message.ElementTypes(mmtme.MessageType)), len(mmtme.MessageElements))
	}
	for elt := range mmtme.MessageElements {
		if err := elt.Validate(); err != nil {
			return ucerr.Friendlyf(err, "MessageElement element type '%s' is invalid", elt)
		}
	}

	return nil
}

var messageParametersForType = map[message.MessageType][]MessageParameter{}

func init() {
	for _, mt := range message.MessageTypes() {
		mp := []MessageParameter{}
		if sampleData := message.DefaultParameters(mt); sampleData != nil {
			v := reflect.ValueOf(sampleData)
			for i := range v.NumField() {
				mp = append(mp, MessageParameter{Name: v.Type().Field(i).Name, DefaultValue: v.Field(i).String()})
			}
		}
		messageParametersForType[mt] = mp
	}
}
