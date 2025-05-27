package custompages

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// Storage provides an object for database access
type Storage struct {
	db *ucdb.DB
}

// NewStorage returns a Storage object
func NewStorage(db *ucdb.DB) *Storage {
	return &Storage{db: db}
}

// GetCustomPageForAppPage loads a CustomPage by App ID and Page Name
func (s *Storage) GetCustomPageForAppPage(ctx context.Context, appID uuid.UUID, pageName string) (*CustomPage, error) {
	const q = "SELECT id, updated, deleted, app_id, page_name, page_source, created FROM custom_pages WHERE app_id=$1 AND page_name=$2 AND deleted='0001-01-01 00:00:00';"

	var obj CustomPage
	if err := s.db.GetContext(ctx, "GetCustomPageForAppPage", &obj, q, appID, pageName); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}
