package custompages

import (
	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucdb"
)

// CustomPage is a page with completely custom HTML, CSS, and JS for an app
type CustomPage struct {
	ucdb.BaseModel

	AppID      uuid.UUID `json:"app_id" db:"app_id"`
	PageName   string    `json:"page_name" validate:"notempty" db:"page_name"`
	PageSource string    `json:"page_source" validate:"notempty" db:"page_source"`
}

//go:generate genpageable CustomPage

//go:generate genvalidate CustomPage

//go:generate genorm CustomPage custom_pages tenantdb
