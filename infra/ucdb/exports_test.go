package ucdb

import (
	"github.com/jmoiron/sqlx"
)

// export these functions for test only ... we need these since we also use testdb.* in ucdb_test package

func (d *DB) TESTONLYChooseDB(dirty bool) (*sqlx.DB, string) {
	return d.chooseDBConnection(dirty)
}

func TESTONLYNewValidationError(err error) error {
	return ValidationError{err}
}

func (d *DB) TESTONLYQueryUpdate(q string) string {
	return d.queryUpdate(q)
}
