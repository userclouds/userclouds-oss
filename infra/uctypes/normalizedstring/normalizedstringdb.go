package normalizedstring

// these wrappers are in a separate file so that we don't pull them into the sdk

import (
	"database/sql/driver"
	"strings"

	"userclouds.com/infra/ucerr"
)

// Value implements the driver.Valuer interface.
func (ns String) Value() (driver.Value, error) {
	return strings.ToLower(string(ns)), nil
}

// Scan implements the sql.Scanner interface.
func (ns *String) Scan(src any) error {
	if s, ok := src.(string); ok {
		*ns = String(strings.ToLower(s))
		return nil
	}

	return ucerr.Errorf("could not convert '%v' to a normalized string", src)
}
