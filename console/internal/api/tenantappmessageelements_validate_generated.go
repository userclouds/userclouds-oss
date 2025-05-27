// NOTE: automatically generated file -- DO NOT EDIT

package api

import (
	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o TenantAppMessageElements) Validate() error {
	if o.TenantID.IsNil() {
		return ucerr.Friendlyf(nil, "TenantAppMessageElements.TenantID can't be nil")
	}
	keysForAppMessageElements := map[uuid.UUID]bool{}
	for _, item := range o.AppMessageElements {
		if err := item.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
		if _, found := keysForAppMessageElements[item.AppID]; found {
			return ucerr.Friendlyf(nil, "duplicate AppID '%v' in AppMessageElements", item.AppID)
		}
		keysForAppMessageElements[item.AppID] = true
	}
	return nil
}
