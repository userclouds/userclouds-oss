package ucdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucdb/metrics"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/ucmetrics"
	"userclouds.com/infra/uctrace"
)

var metricsSubsystem = ucmetrics.Subsystem("ucdb")
var (
	metricQueryCount = ucmetrics.CreateCounter(metricsSubsystem, "queries_total", "The total number of processed DB queries", "query_type", "query_name")
	// Note: we have two metrics for measuring query duration:
	// query_duration_seconds is a histogram that gives the distribution of
	// durations, which is very useful for flexible queries, but since it's
	// labeled by query type/name, it can end up being very high cardinality,
	// making prometheus very slow over large time ranges. By contrast, using
	// rate(query_duration_seconds_total) / rate(queries_total) will only
	// provide the average query duration, but it's very simple/compact and
	// will work over large time ranges.
	metricQueryDurationSeconds      = ucmetrics.CreateHistogram(metricsSubsystem, "query_duration_seconds", "Histogram of individual query duration in seconds", "query_type", "query_name")
	metricQueryDurationSecondsTotal = ucmetrics.CreateCounter(metricsSubsystem, "query_duration_seconds_total", "The total time spent processing DB queries", "query_type", "query_name")
)
var tracer = uctrace.NewTracer("ucdb")

// DB is a wrapper around sqlx.DB that lets us log queries
type DB struct {
	*sqlx.DB          // this is the default connection (may be a replica)
	MasterDB *sqlx.DB // the master DB connection (if different from the default)

	dbName string // Keep the DB name around so we can get better logging around connection close

	// DB operation timeout
	// we keep this here because our methods already take variadic args, so it's hard to use
	// an Option pattern cleanly, and making extra methods sucks, but since we only use this
	// for migration today, we're still being careful to set/reset it in the right places
	timeout time.Duration

	DBProduct DBProduct

	metricsCollector *ucmetrics.Collector
	rc               *retryConfig
}

func (d DB) unsafe() DB {
	d.DB = d.DB.Unsafe()
	return d
}

// Default DB operation timeout to prevent build up of requests when DB is slow
const defaultTimeout = 15 * time.Second

// Timeout returns the current timeout for this DB connection
func (d *DB) Timeout() time.Duration {
	if d.timeout != 0 {
		return d.timeout
	}
	return defaultTimeout
}

// SetTimeout sets the timeout for this DB connection
func (d *DB) SetTimeout(t time.Duration) {
	d.timeout = t
}

func recordQueryMetrics(ctx context.Context, queryType string, queryName string, took time.Duration, inFlight int) {
	metricQueryCount.WithLabelValues(queryType, queryName).Inc()
	if took.Seconds() < 0 {
		took = 0
	}

	metricQueryDurationSeconds.WithLabelValues(queryType, queryName).Observe(took.Seconds())
	metricQueryDurationSecondsTotal.WithLabelValues(queryType, queryName).Add(took.Seconds())
	switch queryType {
	case "GetContext":
		metrics.RecordGet(ctx, took, inFlight)
	case "SelectContext":
		metrics.RecordSelect(ctx, took, inFlight)
	case "ExecContext":
		metrics.RecordWrite(ctx, took, inFlight)
	default:
		// Calling ucerr.Errorf to get the stack trace
		uclog.Errorf(ctx, "%v", ucerr.Errorf("unknown query type %v passed to recordQueryMetrics", queryType))
	}
}

// track instruments the execution of a query
func (d *DB) track(ctx context.Context, queryType string, queryName string, q string, dbInfo string, retryNumber int, f func(context.Context) error) error {
	dbStatsPrecall := d.Stats()
	inFlight := metrics.IncrementInFlightCount()
	start := time.Now().UTC() // the utc doesn't matter for timing purposes but better than lint-ignore?
	err := uctrace.Wrap0(ctx, tracer, fmt.Sprintf("%s try %d", queryType, retryNumber), true, f)
	took := time.Now().UTC().Sub(start)
	metrics.DecrementInFlightCount()
	dbStatsPostcall := d.Stats()
	recordQueryMetrics(ctx, queryType, queryName, took, inFlight)
	// TODO: Sprintf in just the IDs for debugging purposes?
	uclog.Verbosef(ctx, "[%v] %v: %v %v err %v took %v: %v", d.DBProduct, queryType, queryName, dbInfo, err, took, q)
	uclog.Verbosef(ctx, "DB stats openconns [%v:%v] inuse[%v:%v] idle[%v:%v] wait[%v:%v]",
		dbStatsPrecall.OpenConnections, dbStatsPostcall.OpenConnections, dbStatsPrecall.InUse, dbStatsPostcall.InUse, dbStatsPrecall.Idle, dbStatsPostcall.Idle,
		dbStatsPrecall.WaitCount, dbStatsPostcall.WaitCount)

	return ucerr.Wrap(err)
}

// trackWithResult instruments the execution of a query that returns a result
func trackWithResult[item any](ctx context.Context, d *DB, queryType string, queryName string, q string, dbInfo string, retryNumber int, f func(context.Context) (item, error)) (item, error) {
	dbStatsPrecall := d.Stats()
	inFlight := metrics.IncrementInFlightCount()
	start := time.Now().UTC() // the utc doesn't matter for timing purposes but better than lint-ignore?
	res, err := uctrace.Wrap1(ctx, tracer, fmt.Sprintf("%s try %d", queryType, retryNumber), true, f)
	took := time.Now().UTC().Sub(start)
	metrics.DecrementInFlightCount()
	dbStatsPostcall := d.Stats()
	recordQueryMetrics(ctx, queryType, queryName, took, inFlight)
	// TODO: Sprintf in just the IDs for debugging purposes?
	uclog.Verbosef(ctx, "[%v] %v: %v %v err %v took %v: %v", d.DBProduct, queryType, queryName, dbInfo, err, took, q)
	uclog.Verbosef(ctx, "DB stats openconns [%v:%v] inuse[%v:%v] idle[%v:%v] wait[%v:%v]",
		dbStatsPrecall.OpenConnections, dbStatsPostcall.OpenConnections, dbStatsPrecall.InUse, dbStatsPostcall.InUse, dbStatsPrecall.Idle, dbStatsPostcall.Idle,
		dbStatsPrecall.WaitCount, dbStatsPostcall.WaitCount)
	return res, ucerr.Wrap(err)
}

// ExecContext implements sql.DB
func (d *DB) ExecContext(ctx context.Context, queryName, q string, args ...any) (sql.Result, error) {
	return d.ExecContextWithDirty(ctx, queryName, q, true, args...)
}

// ExecContextWithDirty is the same as ExecContext, but allows the caller to specify if the data query is retrieving is dirty in local cluster
func (d *DB) ExecContextWithDirty(ctx context.Context, queryName, q string, dirty bool, args ...any) (sql.Result, error) {
	return uctrace.Wrap1(ctx, tracer, "ucdb.ExecContext "+queryName, true, func(ctx context.Context) (sql.Result, error) {
		// create a child context with a timeout
		withTimeoutCtx, cancel := context.WithTimeout(ctx, d.Timeout())
		// release resources used in `withTimeoutCtx` if
		// the DB operation finishes faster than the timeout
		defer cancel()

		db, dbInfo := d.chooseDBConnection(dirty)
		q := d.queryUpdate(q)

		var res sql.Result
		var err error

		var try int
		for {
			res, err = trackWithResult(withTimeoutCtx, d, "ExecContext", queryName, q, dbInfo, try, func(ctx context.Context) (sql.Result, error) {
				return db.ExecContext(ctx, q, args...)
			})

			if err == nil {
				break
			}
			var ne net.Error
			if !errors.As(err, &ne) {
				if strings.Contains(err.Error(), "syntax error") {
					return nil, ucerr.Errorf("SQL %v: %w", q, err)
				}
				return nil, ucerr.Wrap(err)
			}

			try++
			if d.rc.retriesExceeded(try) {
				return nil, ucerr.Wrap(err)
			}

			uclog.Warningf(ctx, "ExecContext: %v failed (%d), retrying: %v", queryName, try, err)
			d.rc.retryDelay(try)

		}

		return res, nil
	})
}

// GetContext implements sql.DB
func (d *DB) GetContext(ctx context.Context, queryName string, dest any, q string, args ...any) error {
	return ucerr.Wrap(d.GetContextWithDirty(ctx, queryName, dest, q, true, args...))
}

// GetContextWithDirty is the same as GetContext, but allows the caller to specify if the data query is retrieving is dirty in local cluster
func (d *DB) GetContextWithDirty(ctx context.Context, queryName string, dest any, q string, dirty bool, args ...any) error {
	return uctrace.Wrap0(ctx, tracer, "ucdb.GetContext "+queryName, true, func(ctx context.Context) error {

		// create a child context with a timeout
		withTimeoutCtx, cancel := context.WithTimeout(ctx, d.Timeout())
		// release resources used in `withTimeoutCtx` if
		// the DB operation finishes faster than the timeout
		defer cancel()

		// choose the connection to use (primary or read-only if applicable & available)
		db, dbInfo := d.chooseDBConnection(dirty)
		q := d.queryUpdate(q)

		var try int
		for {
			err := d.track(withTimeoutCtx, "GetContext", queryName, q, dbInfo, try, func(ctx context.Context) error {
				return ucerr.Wrap(db.GetContext(ctx, dest, q, args...))
			})

			if err == nil {
				break
			}

			// we want to retry on all network errors except context.DeadlineExceeded,
			// but not on other errors (eg. sql.ErrNoRows would be a waste of time to retry)
			var ne net.Error
			if !errors.As(err, &ne) || errors.Is(err, context.DeadlineExceeded) {
				return ucerr.Wrap(err)
			}

			try++
			if d.rc.retriesExceeded(try) {
				return ucerr.Wrap(err)
			}

			uclog.Warningf(ctx, "GetContext: %v failed (%d), retrying: %v", queryName, try, err)
			d.rc.retryDelay(try)

			// if we failed once, try again with the primary connection
			db = d.DB
		}

		return nil
	})
}

// SelectContext implements sql.DB
func (d *DB) SelectContext(ctx context.Context, queryName string, dest any, q string, args ...any) error {
	return ucerr.Wrap(d.SelectContextWithDirty(ctx, queryName, dest, q, true, args...))
}

// SelectContextWithDirty is the same as SelectContext, but allows the caller to specify if the data query is retrieving is dirty in local cluster
func (d *DB) SelectContextWithDirty(ctx context.Context, queryName string, dest any, q string, dirty bool, args ...any) error {
	return ucerr.Wrap(d.SelectContextWithDirtyAndPrefix(ctx, "ucdb.SelectContext", queryName, dest, q, dirty, args...))

}

// UnsafeSelectContext calls SelectContext in unsafe mode, allowing callers to successfully scan when columns in the SQL result
// have no fields in the destination struct
func (d *DB) UnsafeSelectContext(ctx context.Context, queryName string, dest any, q string, accessPrimaryDBOnly bool, args ...any) error {
	unsafeDB := d.unsafe()
	return ucerr.Wrap(unsafeDB.SelectContextWithDirtyAndPrefix(ctx, "ucdb.UnsafeSelectContext", queryName, dest, q, accessPrimaryDBOnly, args...))
}

// SelectContextWithDirtyAndPrefix is the same as SelectContext, but allows the caller to specify if the data query is retrieving is dirty in local cluster
func (d *DB) SelectContextWithDirtyAndPrefix(ctx context.Context, tracePrefix string, queryName string, dest any, q string, dirty bool, args ...any) error {
	return uctrace.Wrap0(ctx, tracer, fmt.Sprintf("%s %s", tracePrefix, queryName), true, func(ctx context.Context) error {
		// create a child context with a timeout
		withTimeoutCtx, cancel := context.WithTimeout(ctx, d.Timeout())
		// release resources used in `withTimeoutCtx` if
		// the DB operation finishes faster than the timeout
		defer cancel()

		// SelectContext() is always used for read-only statements, but there's no harm in being extra careful
		// (and obviously can't use read-only connection if it's nil)
		db, dbInfo := d.chooseDBConnection(dirty)
		q := d.queryUpdate(q)

		var try int
		for {
			err := d.track(withTimeoutCtx, "SelectContext", queryName, q, dbInfo, try, func(ctx context.Context) error {
				return ucerr.Wrap(db.SelectContext(ctx, dest, q, args...))
			})
			if err == nil {
				break
			}

			var ne net.Error
			if !errors.As(err, &ne) {
				return ucerr.Wrap(err)
			}

			try++
			if d.rc.retriesExceeded(try) {
				return ucerr.Wrap(err)
			}

			uclog.Warningf(ctx, "SelectContext: %v failed (%d), retrying: %v", queryName, try, err)
			d.rc.retryDelay(try)

			// if we failed once, try again with the primary connection for safety
			db = d.DB
		}

		return nil
	})
}

// QueryContext implements sql.DB
func (d *DB) QueryContext(ctx context.Context, queryName string, q string) (*sql.Rows, error) {
	return d.QueryContextWithDirty(ctx, queryName, q, true)
}

// QueryContextWithDirty is the same as QueryContext, but allows the caller to specify if the data query is retrieving is dirty in local cluster
func (d *DB) QueryContextWithDirty(ctx context.Context, queryName string, q string, dirty bool) (*sql.Rows, error) {
	return uctrace.Wrap1(ctx, tracer, "ucdb.QueryContext "+queryName, true, func(ctx context.Context) (*sql.Rows, error) {
		// create a child context with a timeout
		withTimeoutCtx, cancel := context.WithTimeout(ctx, d.Timeout())
		// release resources used in `withTimeoutCtx` if
		// the DB operation finishes faster than the timeout
		defer cancel()

		db, dbInfo := d.chooseDBConnection(dirty)
		q := d.queryUpdate(q)

		var try int
		var rows *sql.Rows
		var err error

		for {

			rows, err = trackWithResult(withTimeoutCtx, d, "QueryContext", queryName, q, dbInfo, try, func(ctx context.Context) (*sql.Rows, error) {
				return db.QueryContext(ctx, q)
			})
			if err == nil {
				break
			}

			var ne net.Error
			if !errors.As(err, &ne) {
				return nil, ucerr.Wrap(err)
			}

			try++
			if d.rc.retriesExceeded(try) {
				return nil, ucerr.Wrap(err)
			}

			uclog.Warningf(ctx, "QueryContext: %v failed (%d), retrying: %v", queryName, try, err)
			d.rc.retryDelay(try)

			// if we failed once, try again with the primary connection for safety
			db = d.DB
		}

		return rows, nil
	})
}

// Close wraps sqlx.Close so we can have better logging
func (d *DB) Close(ctx context.Context) error {
	return uctrace.Wrap0(ctx, tracer, "ucdb.Close", true, func(ctx context.Context) error {
		if d == nil {
			return ucerr.New("tried to close a nil DB")
		}

		ucmetrics.UnregisterDBStatsCollector(d.metricsCollector)

		uclog.Verbosef(ctx, "Closing connection for %s", d.dbName)
		err := d.DB.Close()
		if err != nil {
			uclog.Warningf(ctx, "Failed to close db connection %v", *d)
		}

		if d.MasterDB != nil {
			uclog.Verbosef(ctx, "Closing master region connection for %s", d.dbName)
			if errMaster := d.MasterDB.Close(); err != nil {
				uclog.Warningf(ctx, "Failed to close master region connection %v", *d)
				return ucerr.Wrap(errMaster)
			}
		}

		if err != nil {
			return ucerr.Wrap(err)
		}

		return nil
	})
}

// returns a db connection and whether it is a master connection; if it is not a master connection, then it is a regional replica
func (d *DB) chooseDBConnection(dirty bool) (*sqlx.DB, string) {
	if dirty && d.MasterDB != nil {
		return d.MasterDB, "master"
	}
	return d.DB, string(region.Current())
}

func (d *DB) queryUpdate(q string) string {
	if strings.Contains(q, "AS OF SYSTEM TIME FOLLOWER_READ_TIMESTAMP()") {
		return strings.Replace(q, "AS OF SYSTEM TIME FOLLOWER_READ_TIMESTAMP()", "", 1)
	}

	return q
}
