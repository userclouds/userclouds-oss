package storage

import (
	"userclouds.com/infra/ucdb"
)

// Storage defines the interface for storing per-tenant user events.
type Storage struct {
	db *ucdb.DB
}

// New returns a new DB-backed Storage object to access
// per-tenant user events in the Log DB.
func New(db *ucdb.DB) *Storage {
	return &Storage{
		db: db,
	}
}

//go:generate genorm userevent.UserEvent user_events logdb
