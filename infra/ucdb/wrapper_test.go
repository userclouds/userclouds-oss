package ucdb_test

import (
	"strings"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucdb"
)

func TestQueryRewriting(t *testing.T) {

	wrappedDB := &ucdb.DB{
		DBProduct: ucdb.AWSAuroraPostgres,
	}

	q_in := "SELECT id, updated, deleted, type_name, created FROM object_types WHERE type_name='' AND deleted='0001-01-01 00:00:00';"
	q_out := wrappedDB.TESTONLYQueryUpdate(q_in)
	assert.Equal(t, q_out, q_in)

	q_cdb := "SELECT id, updated, deleted, type_name, created FROM object_types AS OF SYSTEM TIME FOLLOWER_READ_TIMESTAMP() WHERE type_name='' AND deleted='0001-01-01 00:00:00';"
	q_out = wrappedDB.TESTONLYQueryUpdate(q_cdb)
	assert.Equal(t, strings.Join(strings.Fields(q_out), " "), q_in)

}

func TestConnectionRouting(t *testing.T) {

	tdbP := testdb.New(t)
	tdbR := testdb.New(t)

	wrappedDB := &ucdb.DB{
		DB:        tdbP.DB,
		DBProduct: ucdb.AWSAuroraPostgres,
		MasterDB:  tdbR.DB,
	}

	db, dbInfo := wrappedDB.TESTONLYChooseDB(false)
	assert.Equal(t, db == tdbP.DB, true)
	assert.Equal(t, dbInfo, string(region.Current()))

	db, dbInfo = wrappedDB.TESTONLYChooseDB(true)
	assert.Equal(t, db == tdbR.DB, true)
	assert.Equal(t, dbInfo, "master")

	wrappedDB = &ucdb.DB{
		DB:        tdbP.DB,
		DBProduct: ucdb.AWSAuroraPostgres,
		MasterDB:  nil,
	}

	db, dbInfo = wrappedDB.TESTONLYChooseDB(false)
	assert.Equal(t, db == tdbP.DB, true)
	assert.Equal(t, dbInfo, string(region.Current()))

	db, dbInfo = wrappedDB.TESTONLYChooseDB(true)
	assert.Equal(t, db == tdbP.DB, true)
	assert.Equal(t, dbInfo, string(region.Current()))

}
