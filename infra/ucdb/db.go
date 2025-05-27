package ucdb

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq" // also registers Postgres driver

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucdb/errorcode"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/ucmetrics"
	"userclouds.com/infra/uctrace"
)

const (

	// Environment variables names for overriding DB Host & port
	envKeyDBHostOverride = "UC_DB_HOST_OVERRIDE"
	envKeyDBPortOverride = "UC_DB_PORT_OVERRIDE"
)

// Validator allows the caller to run a validation system against the new DB
// specifically this allows us to ensure that a prod DB is up-to-date on schemas
// before a service starts. Bypassed with a no-op system for test/migrate.
// We specifically don't use the Option pattern here because we don't want it
// to be optional in production services to validate...some day we'll have a
// ServiceFramework system to handle all this but not yet
type Validator interface {
	Validate(context.Context, *DB) error
}

// ValidationError wraps any validation error so we can detect them
// TODO can we generalize this typed wrapping pattern?
type ValidationError struct {
	wrapped error
}

// Error implements error
func (v ValidationError) Error() string {
	return fmt.Sprintf("validation error: %v", v.wrapped)
}

// Unwrap implements errors.Unwrap
func (v ValidationError) Unwrap() error {
	return v.wrapped
}

// TODO - We need to make sure that we don't open too many connections to any singular DB when it backs up. We will
// eventually need to build a connection pool which is aware of machine capabilities, current load, and reads backload from DB
// For now put a reasonable limit on this given our current infra
const maxConnectionsPerDB int = 10
const maxIdleConnectionsPerDB int = 10

// New creates a new database connection from Config with default connection pool limits
func New(ctx context.Context, cfg *Config, validator Validator) (*DB, error) {
	return NewWithLimits(ctx, cfg, validator, maxConnectionsPerDB, maxIdleConnectionsPerDB)
}

type contextKey string

const (
	ctxDBRetryConfig contextKey = "ucdb.retryConfig"
)

type retryConfig struct {
	waitMax []int
}

func newDefaultRetryConfig() *retryConfig {
	return &retryConfig{waitMax: []int{10, 1500, 5000}}
}

func (rc *retryConfig) retryDelay(try int) {
	// Handle nil receiver
	if rc == nil {
		// Use default behavior with a small delay
		time.Sleep(time.Millisecond * 10)
		return
	}

	// the second condition should never happen but is just for safety
	if try >= len(rc.waitMax) {
		try = len(rc.waitMax) - 1
	}
	wait := rand.Intn(rc.waitMax[try])
	time.Sleep(time.Millisecond * time.Duration(wait))
}

func (rc *retryConfig) retriesExceeded(try int) bool {
	if rc == nil {
		return true
	}
	return try >= len(rc.waitMax)
}

func getRetryConfig(ctx context.Context) *retryConfig {
	val := ctx.Value(ctxDBRetryConfig)
	rc, ok := val.(*retryConfig)
	if !ok {
		return newDefaultRetryConfig()
	}
	return rc
}

// SetRetryConfig sets the retry configuration for database connections
func SetRetryConfig(ctx context.Context, newWaitMax ...int) context.Context {
	var rc *retryConfig
	if len(newWaitMax) == 0 {
		rc = newDefaultRetryConfig()
	} else {
		rc = &retryConfig{waitMax: newWaitMax}
	}
	return context.WithValue(ctx, ctxDBRetryConfig, rc)
}

// NewWithLimits creates a new database connection from Config with given connection pool limits
func NewWithLimits(ctx context.Context, cfg *Config, validator Validator, maxConn, maxIdle int) (*DB, error) {
	return uctrace.Wrap1(ctx, tracer, "ucdb.NewWithLimits", true, func(ctx context.Context) (*DB, error) {
		db, err := newDB(ctx, cfg, maxConn, maxIdle, getRetryConfig(ctx))
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		// use this to validate schemas are correct, etc.
		// TODO: retry if non-validation error
		if err := validator.Validate(ctx, db); err != nil {
			uclog.Errorf(ctx, "DB %v failed validation: %v", cfg.DBName, err)
			if errc := db.Close(ctx); errc != nil {
				uclog.Errorf(ctx, "Failed to close a connection to a DB that failed validation: %v", errc)
			}
			return nil, ucerr.Wrap(ValidationError{err})
		}

		return db, ucerr.Wrap(err)
	})
}
func getDBHosts(universe universe.Universe, machineRegion region.MachineRegion, cfg *Config) (regionalHost string, masterHost string) {
	if dbHost := os.Getenv(envKeyDBHostOverride); !universe.IsCloud() && dbHost != "" {
		// We allow overriding DB host via environment variables in non cloud (debug, staging & prod) environments/universes.
		// This is useful for running our services in a container & CI environment where we don't want
		// to dynamically (or at all) update a bunch of config files to point to the right DB.
		return dbHost, dbHost
	}

	// Check if we have an explicitly specified host for the region
	if host := cfg.GetRegionalHost(machineRegion); host != "" {
		return host, cfg.Host
	}

	if !universe.IsProdOrStaging() {
		return cfg.Host, cfg.Host
	}

	// TODO: this is a terrible awful hack but I want it to work tonight
	// design doc coming shortly with a couple ideas for a longer-term solution
	// scoping to CDB only because that's where it matters right now
	// "production" is part of the dedicated MR cluster hostname but "free-tier" for serverless
	if region.IsValid(machineRegion, universe) && !strings.Contains(cfg.Host, string(machineRegion)) {
		cockroachHost := strings.Replace(cfg.Host, "aws-us-west-2", string(machineRegion), 1)
		return cockroachHost, cockroachHost
	}

	return cfg.Host, cfg.Host
}

// GetPostgresURLs returns the regional and master DB URLs for the given config
func GetPostgresURLs(ctx context.Context, hashed bool, cfg *Config, universe universe.Universe, machineRegion region.MachineRegion) (regional string, master string, err error) {
	user := url.User(cfg.User)
	password, err := cfg.Password.Resolve(ctx)

	if err != nil {
		return "", "", ucerr.Wrap(err)
	}
	if hashed {
		password = crypto.GetMD5Hash(password)
	}
	if password != "" {
		user = url.UserPassword(cfg.User, password)
	}

	port := cfg.Port
	if !universe.IsCloud() {
		// We allow overriding DB host & port via environment variables in non cloud (debug, staging & prod) environments/universes.
		// This is useful for running our services in a container & CI environment where we don't want
		// to dynamically (or at all) update a bunch of config files to point to the right DB.
		if dbPort := os.Getenv(envKeyDBPortOverride); dbPort != "" {
			port = dbPort
		}
	}
	regionalHost, masterHost := getDBHosts(universe, machineRegion, cfg)
	rawQuery := ""
	if cfg.IsLocal() {
		rawQuery = "sslmode=disable"
	}
	regionalURL := url.URL{
		Scheme:   "postgresql",
		User:     user,
		Host:     fmt.Sprintf("%s:%s", regionalHost, port),
		Path:     cfg.DBName,
		RawQuery: rawQuery,
	}
	masterURL := url.URL{
		Scheme:   "postgresql",
		User:     user,
		Host:     fmt.Sprintf("%s:%s", masterHost, port),
		Path:     cfg.DBName,
		RawQuery: rawQuery,
	}

	// TODO: url.URL.String() escapes passwords per HTTP spec, but DBs don't expect that
	return regionalURL.String(), masterURL.String(), nil
}

// this just exists to make retries easy
func newDB(ctx context.Context, cfg *Config, maxConn, maxIdle int, rc *retryConfig) (*DB, error) {
	if rc == nil {
		rc = newDefaultRetryConfig()
	}

	if cfg.DBDriver == "" {
		// Assume postgres if the driver is not specified
		cfg.DBDriver = PostgresDriver
	}

	var regionalURL, masterURL, regionalHashedURL, masterHashedURL string
	var err error
	switch cfg.DBDriver {
	case PostgresDriver:
		regionalURL, masterURL, err = GetPostgresURLs(ctx, false, cfg, universe.Current(), region.Current())
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		regionalHashedURL, masterHashedURL, err = GetPostgresURLs(ctx, true, cfg, universe.Current(), region.Current())
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
	default:
		return nil, ucerr.Errorf("Unknown dbdriver in the db configuration - %s", cfg.DBDriver)
	}

	logPrefix := fmt.Sprintf("%s regional", cfg.DBProduct)
	uclog.Debugf(ctx, "[%s] trying to connect to regional DB %s via DB URL (PW is hashed): %s", logPrefix, cfg.DBName, regionalHashedURL)
	db, err := connectWithRetries(ctx, cfg.DBDriver, regionalURL, regionalHashedURL, maxConn, maxIdle, cfg.DBName, logPrefix, rc)
	if err != nil {
		return nil, ucerr.Friendlyf(err, "[%s] Failed to connect to DB URL (PW is hashed): %s", logPrefix, regionalHashedURL)
	}
	uclog.Debugf(ctx, "[%s] Connected to DB %s via DB URL (PW is hashed): %s", logPrefix, cfg.DBName, regionalHashedURL)

	var masterDB *sqlx.DB
	if regionalURL != masterURL {
		logPrefix := fmt.Sprintf("%s master", cfg.DBProduct)
		uclog.Debugf(ctx, "[%s] trying to connect to master DB %s via DB URL (PW is hashed): %s", logPrefix, cfg.DBName, masterHashedURL)
		masterDB, err = connectWithRetries(ctx, cfg.DBDriver, masterURL, masterHashedURL, maxConn, maxIdle, cfg.DBName, logPrefix, rc)
		if err != nil {
			if err := db.Close(); err != nil {
				uclog.Warningf(ctx, "failed to close DB connection after failed master DB connect: %v", err)
			}
			return nil, ucerr.Friendlyf(err, "Failed to connect to master DB URL (PW is hashed): %s", masterHashedURL)
		}
		uclog.Debugf(ctx, "[%s] Connected to master DB %s via DB URL (PW is hashed): %s", logPrefix, cfg.DBName, masterHashedURL)
	}

	if cfg.SessionVars != nil {
		if _, err := db.ExecContext(ctx, *cfg.SessionVars); err != nil {
			if err := db.Close(); err != nil {
				uclog.Warningf(ctx, "failed to close DB connection after failed session vars exec: %v", err)
			}
			return nil, ucerr.Wrap(err)
		}
	}

	metricsCollector, err := ucmetrics.RegisterDBStatsCollector(db.DB, cfg.DBName)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	wrappedDB := &DB{
		DB:               db,
		MasterDB:         masterDB, // nil if regionalDB == masterDB
		dbName:           cfg.DBName,
		DBProduct:        cfg.DBProduct,
		timeout:          defaultTimeout,
		metricsCollector: &metricsCollector,
		rc:               rc,
	}

	return wrappedDB, nil
}

func connectWithRetries(ctx context.Context, dbDriver string, dbURI, hashedURI string, maxConn int, maxIdle int, dbName string, logPrefix string, rc *retryConfig) (*sqlx.DB, error) {
	start := time.Now().UTC()

	var retries int
	var db *sqlx.DB
	var err error
	for {
		db, err = connect(ctx, dbDriver, dbURI, maxConn, maxIdle, dbName, logPrefix)
		if err == nil {
			break
		}

		retries++
		if rc.retriesExceeded(retries) {
			d := time.Now().UTC().Sub(start)
			return nil, ucerr.Errorf("[%s] failed to connect to DB %s via %s in %d: tries over %v: %w", logPrefix, dbName, hashedURI, retries-1, d, err)
		}

		var ne net.Error
		if !errors.As(err, &ne) {
			return nil, ucerr.Wrap(err)
		}

		uclog.Errorf(ctx, "[%s] failed to connect to DB %s via %s (try %d): %v", logPrefix, dbName, hashedURI, retries-1, err)
		rc.retryDelay(retries)
	}

	return db, nil
}

func connect(ctx context.Context, dbDriver string, dbURI string, maxConn, maxIdle int, dbName string, logPrefix string) (*sqlx.DB, error) {
	start := time.Now().UTC()
	db, err := sqlx.Connect(dbDriver, dbURI)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	end := time.Now().UTC()
	uclog.Verbosef(ctx, "[%s] sqlx.Connect took %v for %s", logPrefix, end.Sub(start), dbName)

	// See https://github.com/userclouds/userclouds/issues/2554 for more details. Limiting lifetime of idle connections to minimize probability of
	// dbs with broken connections being returned from the pool.
	db.SetConnMaxIdleTime(time.Minute * 10)

	db.SetMaxOpenConns(maxConn)
	db.SetMaxIdleConns(maxIdle)

	return db, nil
}

// IsUniqueViolation returns true if the error is a postgres unique
// violation error, and false otherwise (including if err is nil).
func IsUniqueViolation(err error) bool {
	var perr *pq.Error
	if ok := errors.As(err, &perr); ok && perr.Code == errorcode.UniqueViolation() {
		return true
	}
	return false
}

// IsSQLParseError returns true if the error is a postgres parse error, likely due to bad user input
func IsSQLParseError(err error) (bool, string) {
	var perr *pq.Error
	if ok := errors.As(err, &perr); ok {
		if perr.Code == errorcode.InvalidTextRepresentation() || perr.Code == errorcode.InvalidDateTimeFormat() {
			return true, perr.Message
		}
	}
	return false, ""
}

// IsTransactionConflict returns true if the error is a postgres transaction commit
// error, and false otherwise (including if err is nil).
func IsTransactionConflict(err error) bool {
	var perr *pq.Error
	if ok := errors.As(err, &perr); ok && perr.Code == errorcode.TransactionCommit() {
		return true
	}
	return false
}
