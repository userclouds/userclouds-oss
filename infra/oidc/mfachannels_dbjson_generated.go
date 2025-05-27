// NOTE: automatically generated file -- DO NOT EDIT

package oidc

import (
	"database/sql/driver"
	"encoding/json"

	"userclouds.com/infra/ucerr"
)

// Value implements sql.Valuer
func (o MFAChannels) Value() (driver.Value, error) {
	return json.Marshal(o)
}

// Scan implements sql.Scanner
func (o *MFAChannels) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return ucerr.New("type assertion failed for MFAChannels.Scan()")
	}
	return ucerr.Wrap(json.Unmarshal(b, &o))
}
