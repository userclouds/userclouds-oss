package uctrace

// Attribute is an enum of UserClouds-specific properties that we want to
// attach to traces.
type Attribute string

const (
	// AttributeCompanyID captures the UUID of the company owning a tenant
	AttributeCompanyID Attribute = "uc.company_id"
	// AttributeCompanyName captures the name of the company owning a tenant
	AttributeCompanyName Attribute = "uc.company_name"
	// AttributeTenantID captures the UUID of the tenant
	AttributeTenantID Attribute = "uc.tenant_id"
	// AttributeTenantName captures the name of the tenant
	AttributeTenantName Attribute = "uc.tenant_name"
	// AttributeTenantURL captures the URL of the tenant
	AttributeTenantURL Attribute = "uc.tenant_url"
	// AttributeHandlerName captures the name of the handler that is processing
	// the request
	AttributeHandlerName Attribute = "uc.handler_name"
	// AttributeSdkVersion captures the version of the UC SDK that sent the request
	AttributeSdkVersion Attribute = "uc.sdk_version"
	// AttributeUserFriendlyError captures the user-friendly error message in the event of a Friendlyf error
	AttributeUserFriendlyError Attribute = "uc.user_friendly_error"
)
