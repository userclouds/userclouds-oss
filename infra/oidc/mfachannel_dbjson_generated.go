// NOTE: automatically generated file -- DO NOT EDIT

package oidc

import (
	"database/sql/driver"
	"encoding/json"

	"userclouds.com/infra/ucerr"
)

// Value implements sql.Valuer
func (o MFAChannel) Value() (driver.Value, error) {
	return json.Marshal(o)
}

// Scan implements sql.Scanner
func (o *MFAChannel) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return ucerr.New("type assertion failed for MFAChannel.Scan()")
	}
	return ucerr.Wrap(json.Unmarshal(b, &o))
}
