package companyconfig

import "github.com/gofrs/uuid"

// TenantInfo contains information about the Console / CompanyConfig tenant itself,
// since those services use our AuthN/AuthZ system to let users log in and
// create/modify companies & tenants.
type TenantInfo struct {
	CompanyID uuid.UUID
	TenantID  uuid.UUID
	TenantURL string
}
