package app

import (
	"context"
	"fmt"
	"os"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // register Postgres driver

	"userclouds.com/infra/ucerr"
)

// Storage defines the interface for storing events & users.
type Storage struct {
	db *sqlx.DB
}

// NewStorage creates a new Postgres-backed Storage object.
func NewStorage() (*Storage, error) {
	ctx := context.Background()

	username := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DBNAME")
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbURI := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable password=%s", dbHost, dbPort, username, dbName, password)

	db, err := sqlx.Connect("postgres", dbURI)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	s := &Storage{
		db: db,
	}

	if err := s.initTables(ctx); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return s, nil
}

func (s *Storage) initTables(ctx context.Context) error {
	const userT = `CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY,
		name VARCHAR NOT NULL
		);
		ALTER TABLE users DROP COLUMN IF EXISTS username;`
	// Events table uses UUID as both event ID and AuthZ ID
	const eventT = `CREATE TABLE IF NOT EXISTS events (
		id UUID PRIMARY KEY,
		title VARCHAR NOT NULL
		);`

	if _, err := s.db.ExecContext(ctx, userT); err != nil {
		return fmt.Errorf("error creating users table: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, eventT); err != nil {
		return fmt.Errorf("error creating events table: %w", err)
	}
	return nil
}

// User is the data model for a user of the events app.
type User struct {
	ID   uuid.UUID `db:"id" json:"id" yaml:"id"`
	Name string    `db:"name" json:"name" yaml:"name"`
}

// SaveUser upserts a user.
func (s *Storage) SaveUser(ctx context.Context, user User) error {
	const query = `INSERT INTO users (id, name) VALUES ($1, $2)
		ON CONFLICT (id) DO NOTHING;`
	_, err := s.db.ExecContext(ctx, query, user.ID, user.Name)
	return ucerr.Wrap(err)
}

// ListUsers returns all users.
func (s *Storage) ListUsers(ctx context.Context) ([]User, error) {
	const query = `SELECT * FROM users;`
	var users []User
	if err := s.db.SelectContext(ctx, &users, query); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return users, nil
}

// Event is the data model for an event in the events app.
type Event struct {
	ID    uuid.UUID `db:"id" json:"id" yaml:"id"`
	Title string    `db:"title" json:"title" yaml:"title"`
}

// CreateEvent creates an event.
func (s *Storage) CreateEvent(ctx context.Context, id uuid.UUID, title string) error {
	const query = `INSERT INTO events (id, title) VALUES ($1, $2);`
	_, err := s.db.ExecContext(ctx, query, id, title)
	return ucerr.Wrap(err)
}

// GetEvent returns info on a single event by ID.
func (s *Storage) GetEvent(ctx context.Context, id uuid.UUID) (*Event, error) {
	const query = `SELECT * FROM events WHERE id=$1;`
	var event Event
	if err := s.db.GetContext(ctx, &event, query, id); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &event, nil
}

// UpdateEvent updates a single existing event.
func (s *Storage) UpdateEvent(ctx context.Context, event Event) error {
	const query = `UPDATE events SET title=$2 WHERE id=$1;`
	_, err := s.db.ExecContext(ctx, query, event.ID, event.Title)
	return ucerr.Wrap(err)
}

// DeleteEvent delets an event by ID.
func (s *Storage) DeleteEvent(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM events WHERE id=$1;`
	_, err := s.db.ExecContext(ctx, query, id)
	return ucerr.Wrap(err)
}
