// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

import (
	"database/sql/driver"
	"encoding/json"

	"userclouds.com/infra/ucerr"
)

// Value implements sql.Valuer
func (o UserRegionDBConfigMap) Value() (driver.Value, error) {
	return json.Marshal(o)
}

// Scan implements sql.Scanner
func (o *UserRegionDBConfigMap) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return ucerr.New("type assertion failed for UserRegionDBConfigMap.Scan()")
	}
	return ucerr.Wrap(json.Unmarshal(b, &o))
}
