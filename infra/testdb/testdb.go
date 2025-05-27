package testdb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uclog"
)

// this var is set during New from an env var, but we keep it around in a global so we can generate test configs
var connStr string

// because this is assigned at runtime and not link-time now, we need to lock it to prevent race detector from complaining
var connStrLock sync.RWMutex

// Options defines an interface for messing with the database after creation,
// eg to apply migrations to a test db
type Options interface {
	Apply(*ucdb.DB, ucdb.DBProduct) error
}

// New returns a new handle to a test database that's unique per test
func New(t testing.TB, opts ...Options) *ucdb.DB {
	// no need to parse the env var again if we've already done it
	connStrLock.RLock()
	if connStr != "" {
		connStrLock.RUnlock()
		db, _ := NewWithConnStr(t, connStr, opts...)
		return db
	}
	connStrLock.RUnlock()

	testDBPointerPath := os.Getenv("UC_TESTDB_POINTER_PATH")
	assert.NotEqual(t, testDBPointerPath, "")

	// read the file that points to the testdb directory
	// (we need this level of indirection since VSCode can't create & pass temp directories
	// around as part of the debugging tool cycle)
	bs, err := os.ReadFile(testDBPointerPath)
	assert.NoErr(t, err)

	// in that directory, read ./connfile, which contains the connection string
	bs, err = os.ReadFile(strings.TrimSpace(string(bs)) + "/connfile")
	assert.NoErr(t, err)

	// NB: assign to global so TestConfig can use it
	connStrLock.Lock()
	if connStr == "" {
		connStr = strings.TrimSpace(string(bs))
	}
	connStrLock.Unlock()

	db, _ := NewWithConnStr(t, connStr, opts...)
	return db
}

// NewWithConnStr provides a way to override the connstring from New
// We need this specifically for genschemas
// TODO: there should be a more elegant way to do this, but testdb.Options
// is keyed to the *ucdb.DB (which is too late) and no good refactoring was
// obvious to me, so I'll come back to it.
func NewWithConnStr(t testing.TB, connStr string, opts ...Options) (*ucdb.DB, string) {
	ctx := context.Background()
	// connect to the "root" DB
	rdb, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		uclog.Fatalf(ctx, "error creating test db: %v", err)
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			uclog.Fatalf(ctx, "failed to close root db: %v", err)
		}
	}()

	// create a new DB that's unique
	dbID := uuid.Must(uuid.NewV4()).String()
	// cdb doesn't like hyphens in db names, or first-char numbers :)
	dbID = fmt.Sprintf("db%s", strings.ReplaceAll(dbID, "-", ""))
	// Postgres does not allow bindvar substitution for identifer names, only values,
	// so we have to use Sprintf here (see https://www.postgresql.org/docs/9.1/xfunc-sql.html).
	// "The arguments can only be used as data values, not as identifiers"
	s := fmt.Sprintf(`CREATE DATABASE %s;`, dbID)
	if _, err := rdb.Exec(s); err != nil {
		uclog.Fatalf(ctx, "failed to create unique test DB %v: %v", dbID, err)
	}

	conn, err := url.Parse(connStr)
	if err != nil {
		uclog.Fatalf(ctx, "failed to parse root DB conn string: %v", err)
	}
	conn.Path = dbID

	db, err := sqlx.Connect("postgres", conn.String())
	if err != nil {
		uclog.Fatalf(ctx, "failed to connect to new unique db %v: %v", conn.String(), err)
	}

	wrappedDB := &ucdb.DB{DB: db}

	// figure out what product we're running in order to apply the right creates
	wrappedDB.DBProduct = mustGetDBProduct(ctx, wrappedDB, dbID)

	// run Options, if any
	for _, o := range opts {
		if err := o.Apply(wrappedDB, wrappedDB.DBProduct); err != nil {
			uclog.Fatalf(ctx, "failed to apply Options to testdb: %v", err)
		}
	}

	// register in testing to clean up after ourselves
	// t.Cleanup(func() {
	// 	s := fmt.Sprintf("DROP DATABASE %s;", dbID)
	// 	if _, err := db.Exec(s); err != nil {
	// 		uclog.Fatalf(ctx, "failed to drop unique db %v: %v", dbID, err)
	// 	}

	// 	if err := db.Close(); err != nil {
	// 		uclog.Fatalf(ctx, "failed to close unique db %v: %v", dbID, err)
	// 	}
	// })

	return wrappedDB, dbID
}

func mustGetDBProduct(ctx context.Context, db *ucdb.DB, dbID string) ucdb.DBProduct {
	var version string
	if err := db.Get(&version, "SELECT version();"); err != nil {
		uclog.Fatalf(ctx, "failed to get version for unique db %v: %v", dbID, err)
	}
	if strings.Contains(version, "PostgreSQL") {
		return ucdb.Postgres
	}

	uclog.Fatalf(ctx, "unknown DB product for unique db %v: %v", dbID, version)
	return "unknown"
}

// TestConfig generates a DB config object from the auto-created testdb
// This is useful when you need to eg embed a ucdb.Config object in an IDP tenant struct
// TODO: this should probably take a db name or tenant ID or something,
// rather than pointing to the default DB
func TestConfig(t *testing.T, tdb *ucdb.DB) ucdb.Config {
	connStrLock.RLock()
	connURL, err := url.Parse(connStr)
	connStrLock.RUnlock()
	assert.NoErr(t, err)

	var dbname string
	if err := tdb.Get(&dbname, "SELECT CURRENT_DATABASE();"); err != nil {
		uclog.Fatalf(context.Background(), "failed to get current database name for TestConfig: %v", err)
	}

	cfg := ucdb.Config{
		User:      connURL.User.Username(),
		Host:      connURL.Hostname(),
		Port:      connURL.Port(),
		DBName:    dbname,
		DBDriver:  ucdb.PostgresDriver,
		DBProduct: mustGetDBProduct(context.Background(), tdb, dbname), // use background here since we're only in test code
	}

	if pass, ok := connURL.User.Password(); ok {
		cfg.Password = secret.NewTestString(pass)
	}

	return cfg
}
