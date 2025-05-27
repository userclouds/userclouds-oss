package internal_test

import (
	"testing"

	"userclouds.com/infra/assert"
	message "userclouds.com/internal/messageelements"
	"userclouds.com/internal/tenantplex"
)

func validateElement(t *testing.T, mt message.MessageType, elt message.ElementType, expected string, actual string) {
	t.Helper()

	// because we are testing all of the cases in a separate method, it's useful
	// to have a struct for comparison so we have context about which cases fail
	type elementParams struct {
		MessageType message.MessageType
		ElementType message.ElementType
		Element     string
	}

	assert.Equal(t,
		elementParams{MessageType: mt, ElementType: elt, Element: expected},
		elementParams{MessageType: mt, ElementType: elt, Element: actual})
}

func validateUnchanged(t *testing.T, app tenantplex.App, messageTypes []message.MessageType) {
	t.Helper()

	for _, mt := range messageTypes {
		validateMessageTypeUnchanged(t, app, mt, message.ElementTypes(mt))
	}
}

func validateMessageTypeUnchanged(t *testing.T, app tenantplex.App, messageType message.MessageType, elementTypes []message.ElementType) {
	t.Helper()

	defaultGetter := message.MakeElementGetter(messageType)
	appGetter := app.MakeElementGetter(messageType)
	for _, et := range elementTypes {
		validateElement(t, messageType, et, defaultGetter(et), appGetter(et))
	}
}

func TestUncustomizedElements(t *testing.T) {
	var app tenantplex.App
	validateUnchanged(t, app, message.MessageTypes())
}

func TestCustomizedElements(t *testing.T) {
	override := "override"
	messageTypes := message.MessageTypes()
	for i, mt := range messageTypes {
		elementTypes := message.ElementTypes(mt)
		for j, et := range elementTypes {
			var app tenantplex.App
			app.CustomizeMessageElement(mt, et, override)
			appGetter := app.MakeElementGetter(mt)
			validateElement(t, mt, et, override, appGetter(et))

			unchangedMessageTypes := make([]message.MessageType, 0)
			unchangedMessageTypes = append(unchangedMessageTypes, messageTypes[:i]...)
			unchangedMessageTypes = append(unchangedMessageTypes, messageTypes[i+1:]...)
			validateUnchanged(t, app, unchangedMessageTypes)

			unchangedElementTypes := make([]message.ElementType, 0)
			unchangedElementTypes = append(unchangedElementTypes, elementTypes[:j]...)
			unchangedElementTypes = append(unchangedElementTypes, elementTypes[j+1:]...)
			validateMessageTypeUnchanged(t, app, mt, unchangedElementTypes)
		}
	}
}

func TestRemoveCustomizedElement(t *testing.T) {
	testMessageType := message.EmailInviteNewUser
	testElementType := message.EmailSender
	testElement := "override"
	defaultGetter := message.MakeElementGetter(testMessageType)

	var app tenantplex.App
	app.CustomizeMessageElement(testMessageType, testElementType, testElement)
	appGetter := app.MakeElementGetter(testMessageType)

	assert.Equal(t, testElement, appGetter(testElementType))

	app.CustomizeMessageElement(testMessageType, testElementType, "")
	assert.Equal(t, defaultGetter(testElementType), appGetter(testElementType))
}
