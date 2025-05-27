package telephony

// ProviderType is a telephony provider type
type ProviderType int

// Supported telephony provider types
const (
	ProviderTypeUnsupported ProviderType = -1

	ProviderTypeNone ProviderType = 0

	ProviderTypeTwilio ProviderType = 1
)

//go:generate genconstant ProviderType
