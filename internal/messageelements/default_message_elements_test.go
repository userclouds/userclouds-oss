package messageelements_test

import (
	"os"
	"testing"

	"userclouds.com/infra/assert"
	message "userclouds.com/internal/messageelements"
)

func TestDefaultMessageTypes(t *testing.T) {
	// verify that all default message elements pass validation
	for _, mt := range message.EmailMessageTypes() {
		defaultGetter := message.MakeElementGetter(mt)
		elementTypes := message.ElementTypes(mt)
		assert.True(t, len(elementTypes) > 0)
		for _, elt := range elementTypes {
			assert.NoErr(t, message.ValidateMessageElement(mt, elt, defaultGetter(elt)))
		}
	}
	for _, mt := range message.SMSMessageTypes() {
		defaultGetter := message.MakeElementGetter(mt)
		elementTypes := message.ElementTypes(mt)
		assert.True(t, len(elementTypes) > 0)
		for _, elt := range elementTypes {
			assert.NoErr(t, message.ValidateMessageElement(mt, elt, defaultGetter(elt)))
		}
	}
}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../../..")
	os.Exit(m.Run())
}
