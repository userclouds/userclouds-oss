package timestamparray

// these wrappers are in a separate file so that we don't pull them into the sdk

import (
	"database/sql/driver"
	"time"

	"github.com/lib/pq"

	"userclouds.com/infra/ucerr"
)

// NOTE: The pq library does not support scanning of []time.Time,
//       so we are representing an array of time.Time as an array
//       of strings in the database, using a JSON representation
//       to make them human readable and more easily queryable.

// Value implements the driver.Valuer interface.
func (tsa TimestampArray) Value() (driver.Value, error) {
	strs := make(pq.StringArray, 0, len(tsa))
	for i := range tsa {
		b, err := tsa[i].MarshalText()
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		strs = append(strs, string(b))
	}
	return strs.Value()
}

// Scan implements the sql.Scanner interface.
func (tsa *TimestampArray) Scan(src any) error {
	var strs pq.StringArray
	if err := strs.Scan(src); err != nil {
		return ucerr.Wrap(err)
	}

	for i := range strs {
		var t time.Time
		if err := t.UnmarshalText([]byte(strs[i])); err != nil {
			return ucerr.Wrap(err)
		}
		*tsa = append(*tsa, t)
	}

	return nil
}
