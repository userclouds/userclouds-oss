package storage

import "userclouds.com/infra/uclog"

// MetricsRow describes one row in the metrics table
// We don't generate the ORM because created/deleted/updated mechanics don't apply
type MetricsRow struct {
	ID        uint64          `db:"id"`
	EventCode uclog.EventCode `db:"type"` // TODO: rename the db column to event_code?
	Timestamp int64           `db:"timestamp"`
	Count     int             `db:"count"`
}
