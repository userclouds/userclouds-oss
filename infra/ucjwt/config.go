package ucjwt

import (
	"github.com/gofrs/uuid"
)

// Config represents config for Console Authentication & Authorization.
type Config struct {
	ClientID     string    `json:"client_id" yaml:"client_id" validate:"notempty"`
	ClientSecret string    `json:"client_secret" yaml:"client_secret" validate:"notempty"` // TODO: convert to secret.String
	TenantURL    string    `json:"tenant_url" yaml:"tenant_url" validate:"notempty"`
	TenantID     uuid.UUID `json:"tenant_id" yaml:"tenant_id" validate:"notnil"`
	CompanyID    uuid.UUID `json:"company_id" yaml:"company_id" validate:"notnil"`
}

//go:generate genvalidate Config
