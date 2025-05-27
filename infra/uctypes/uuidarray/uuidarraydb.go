package uuidarray

// these wrappers are in a separate file so that we don't pull them into the sdk

import (
	"database/sql/driver"

	"github.com/lib/pq"
)

// Value implements the driver.Valuer interface.
func (a UUIDArray) Value() (driver.Value, error) {
	return pq.GenericArray{A: a}.Value()
}

// Scan implements the sql.Scanner interface.
func (a *UUIDArray) Scan(src any) error {
	return pq.GenericArray{A: a}.Scan(src)
}
