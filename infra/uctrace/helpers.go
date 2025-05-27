package uctrace

import (
	"github.com/gofrs/uuid"
)

// SetTenantAttributes sets the tenant attributes on the span
func SetTenantAttributes(span Span, tenantID uuid.UUID, tenantName string, tenantURL string, companyID uuid.UUID, companyName string) {
	span.SetStringAttribute(AttributeTenantID, tenantID.String())
	span.SetStringAttribute(AttributeTenantName, tenantName)
	span.SetStringAttribute(AttributeCompanyID, companyID.String())
	if companyName != "" {
		// Can be empty in the worker on the create tenant task
		span.SetStringAttribute(AttributeCompanyName, companyName)
	}
	if tenantURL != "" {
		span.SetStringAttribute(AttributeTenantURL, tenantURL)
	}

}
