package main

import (
	"userclouds.com/idp/userstore"
)

type contact struct {
	name                string
	phoneNumber         string
	phoneNumberPurposes []string
}

func (c *contact) addPhoneNumberPurpose(purpose string) {
	c.phoneNumberPurposes = append(c.phoneNumberPurposes, purpose)
}

func (c contact) getPhoneNumberPurposeResourceIDs() []userstore.ResourceID {
	resourceIDs := []userstore.ResourceID{}
	for _, purpose := range c.phoneNumberPurposes {
		resourceIDs = append(resourceIDs, userstore.ResourceID{Name: purpose})
	}
	return resourceIDs
}

func newContact() contact {
	return contact{
		phoneNumberPurposes: []string{"operational"},
	}
}
