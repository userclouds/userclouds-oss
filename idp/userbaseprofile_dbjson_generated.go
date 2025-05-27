// NOTE: automatically generated file -- DO NOT EDIT

package idp

import (
	"database/sql/driver"
	"encoding/json"

	"userclouds.com/infra/ucerr"
)

// Value implements sql.Valuer
func (o UserBaseProfile) Value() (driver.Value, error) {
	return json.Marshal(o)
}

// Scan implements sql.Scanner
func (o *UserBaseProfile) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return ucerr.New("type assertion failed for UserBaseProfile.Scan()")
	}
	return ucerr.Wrap(json.Unmarshal(b, &o))
}
