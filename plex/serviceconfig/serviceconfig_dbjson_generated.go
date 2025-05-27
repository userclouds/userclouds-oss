// NOTE: automatically generated file -- DO NOT EDIT

package serviceconfig

import (
	"database/sql/driver"
	"encoding/json"

	"userclouds.com/infra/ucerr"
)

// Value implements sql.Valuer
func (o ServiceConfig) Value() (driver.Value, error) {
	return json.Marshal(o)
}

// Scan implements sql.Scanner
func (o *ServiceConfig) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return ucerr.New("type assertion failed for ServiceConfig.Scan()")
	}
	return ucerr.Wrap(json.Unmarshal(b, &o))
}
