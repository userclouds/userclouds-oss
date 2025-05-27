package storage

import (
	"context"

	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// Storage defines the interface for storing the metrics and event metadata
type Storage struct {
	db *ucdb.DB
}

// NewStorage returns a new DB-backed for storing the metrics and metrics metadata
func NewStorage(db *ucdb.DB) *Storage {
	return &Storage{db}
}

// WriteCounters returns a list of all object types
func (s *Storage) WriteCounters(ctx context.Context, query string) error {

	_, err := s.db.ExecContext(ctx, "WriteCounters", query)

	return ucerr.Wrap(err)
}

// ReadCounters returns a list of all object types
func (s *Storage) ReadCounters(ctx context.Context, query string) (*[]MetricsRow, error) {
	var metrics []MetricsRow
	if err := s.db.SelectContext(ctx, "ReadCounters", &metrics, query); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &metrics, nil
}

// ReadCount returns a list of agregated count
func (s *Storage) ReadCount(ctx context.Context, query string) (*int, error) {
	var count int
	if err := s.db.GetContext(ctx, "ReadCount", &count, query); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &count, nil
}
