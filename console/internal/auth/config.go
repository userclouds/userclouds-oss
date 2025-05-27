package auth

import (
	"github.com/gofrs/uuid"

	"userclouds.com/infra/secret"
)

// Config represents config for Console Authentication & Authorization.
type Config struct {
	ClientID     string        `yaml:"client_id" json:"client_id" validate:"notempty"`
	ClientSecret secret.String `yaml:"client_secret" json:"client_secret"`
	TenantURL    string        `yaml:"tenant_url" json:"tenant_url" validate:"notempty"`
	TenantID     uuid.UUID     `yaml:"tenant_id" json:"tenant_id" validate:"notnil"`
	CompanyID    uuid.UUID     `yaml:"company_id" json:"company_id" validate:"notnil"`
}

//go:generate genvalidate Config
