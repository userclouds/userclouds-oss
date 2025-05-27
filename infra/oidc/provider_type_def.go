package oidc

// ProviderType is an OIDC provider type
type ProviderType int

// Supported OIDC provider types
const (
	// when synching data from other IDPs, we may encounter OIDC providers that
	// are not supported, in which case we will store ProviderTypeUnsupported
	// in the DB
	ProviderTypeUnsupported ProviderType = -1

	// not having an OIDC provider is the default
	ProviderTypeNone ProviderType = 0

	// valid OIDC providers are numbered starting with 1
	ProviderTypeGoogle    ProviderType = 1
	ProviderTypeFacebook  ProviderType = 2
	ProviderTypeLinkedIn  ProviderType = 3
	ProviderTypeCustom    ProviderType = 4
	ProviderTypeMicrosoft ProviderType = 5
)

//go:generate genconstant ProviderType
