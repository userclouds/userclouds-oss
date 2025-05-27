package oidc

import "userclouds.com/infra/ucerr"

// GetDefaultIssuerURL returns the default issuer URL for a provider type if one exists
func (pt ProviderType) GetDefaultIssuerURL() string {
	switch pt {
	case ProviderTypeGoogle:
		return googleIssuerURL
	case ProviderTypeFacebook:
		return fbIssuerURL
	case ProviderTypeLinkedIn:
		return linkedInIssuerURL
	case ProviderTypeMicrosoft:
		return MicrosoftIssuerURL
	default:
		return ""
	}
}

// IsNative returns true if the OIDC provider type is a native provider type
func (pt ProviderType) IsNative() bool {
	switch pt {
	case ProviderTypeGoogle:
	case ProviderTypeFacebook:
	case ProviderTypeLinkedIn:
	case ProviderTypeMicrosoft:
	default:
		return false
	}

	return true
}

// IsSupported returns true if the OIDC provider type is recognized and not unsupported or none
func (pt ProviderType) IsSupported() bool {
	if err := pt.Validate(); err != nil {
		return false
	}

	return pt != ProviderTypeUnsupported && pt != ProviderTypeNone
}

// ValidateIssuerURL will make sure the issuer URL is valid for the OIDC provider type
func (pt ProviderType) ValidateIssuerURL(issuerURL string) error {
	if pt.IsNative() {
		if issuerURL != pt.GetDefaultIssuerURL() {
			return ucerr.Errorf("issuer URL '%s' is invalid for native type '%v'", issuerURL, pt)
		}
	} else if pt == ProviderTypeCustom && issuerURL == "" {
		return ucerr.Errorf("isser URL cannot be empty for type '%v'", pt)
	}
	return nil
}

// NativeOIDCProviderTypes returns the supported native OIDC provider types
func NativeOIDCProviderTypes() []ProviderType {
	return []ProviderType{ProviderTypeFacebook, ProviderTypeGoogle, ProviderTypeLinkedIn, ProviderTypeMicrosoft}
}
