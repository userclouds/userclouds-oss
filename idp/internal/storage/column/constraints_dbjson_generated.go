// NOTE: automatically generated file -- DO NOT EDIT

package column

import (
	"database/sql/driver"
	"encoding/json"

	"userclouds.com/infra/ucerr"
)

// Value implements sql.Valuer
func (o Constraints) Value() (driver.Value, error) {
	return json.Marshal(o)
}

// Scan implements sql.Scanner
func (o *Constraints) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return ucerr.New("type assertion failed for Constraints.Scan()")
	}
	return ucerr.Wrap(json.Unmarshal(b, &o))
}
