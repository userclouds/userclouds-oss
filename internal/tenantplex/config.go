package tenantplex

import (
	"context"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	pageparams "userclouds.com/internal/pageparameters"
	"userclouds.com/internal/pageparameters/pagetype"
	param "userclouds.com/internal/pageparameters/parameter"
	"userclouds.com/internal/pageparameters/parametertype"
)

// TenantConfigs is just an array of TenantConfig objects that we can validate etc
type TenantConfigs []TenantConfig

// Validate implements Validateable
func (tc TenantConfigs) Validate() error {
	for _, t := range tc {
		if err := t.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}

// IssuersList is a list of external OIDC issuers for JWT tokens for a tenant
type IssuersList []string

// Validate implements Validateable
func (l IssuersList) Validate() error {
	for _, issuer := range l {
		if u, err := url.Parse(issuer); err != nil || u.Scheme == "" || u.Host == "" {
			return ucerr.Friendlyf(err, "Invalid issuer URL: %s", issuer)
		}
	}
	return nil
}

// TenantConfig holds the plex config that is per-tenant
// TODO: do we need to support YAML going forward?
// NOTE: keep this class in sync with TenantPlexConfig in TSX codebase.
type TenantConfig struct {
	PlexMap PlexMap `yaml:"plex_map" json:"plex_map"`

	OIDCProviders OIDCProviders `yaml:"oidc_providers" json:"oidc_providers"`

	ExternalOIDCIssuers IssuersList `yaml:"external_oidc_issuers" json:"external_oidc_issuers"`

	// If true, send email verification emails to users after signing up.
	VerifyEmails bool `yaml:"verify_emails" json:"verify_emails"`

	// If true, users may not sign up for new accounts from the main login page.
	// Instead, a tenant admin will need to create accounts OR users can sign up via invites.
	DisableSignUps bool `yaml:"disable_sign_ups" json:"disable_sign_ups"`

	// If signups are disabled, this is the email address that will be used to create the first account
	BootstrapAccountEmails []string `yaml:"bootstrap_account_emails" json:"bootstrap_account_emails,omitempty"`

	Keys Keys `yaml:"keys,omitempty" json:"keys"`

	PageParameters pageparams.ParameterByNameByPageType `yaml:"page_parameters" json:"page_parameters"`
}

//go:generate gendbjson TenantConfig

//go:generate genvalidate TenantConfig

func (tenant *TenantConfig) extraValidate() error {
	for i := range tenant.PlexMap.Apps {
		app := &tenant.PlexMap.Apps[i]
		if explanation, err := pageparams.ValidateCompositeParameters(
			MakeRenderParameterGetter(tenant, app),
			MakeParameterClientData(tenant, app)); err != nil {
			return ucerr.Friendlyf(err, "composite parameter validation failed for app '%s': '%s'", app.ID, explanation)
		}

	}

	return nil
}

// DeletePageParameter deletes a page parameter value override for the given tenant and page type
func (tenant *TenantConfig) DeletePageParameter(t pagetype.Type, n param.Name) {
	if tenant.PageParameters == nil {
		return
	}
	if _, found := tenant.PageParameters[t]; !found {
		return
	}
	delete(tenant.PageParameters[t], n)
}

// SetPageParameter sets a page parameter value override for the given tenant and page type
func (tenant *TenantConfig) SetPageParameter(t pagetype.Type, n param.Name, value string) {
	if tenant.PageParameters == nil {
		tenant.PageParameters = pageparams.ParameterByNameByPageType{}
	}
	if _, found := tenant.PageParameters[t]; !found {
		tenant.PageParameters[t] = pageparams.ParameterByName{}
	}
	tenant.PageParameters[t][n] = param.MakeParameter(n, value)
}

// GetOIDCRedirectURL returns an appropriate redirect URL for OIDC login for the specified provider
func (tenant *TenantConfig) GetOIDCRedirectURL(tenantURL *url.URL, providerType oidc.ProviderType, issuerURL string) (*url.URL, error) {
	p, err := tenant.OIDCProviders.GetProviderForIssuerURL(providerType, issuerURL)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if !p.IsConfigured() {
		return nil, ucerr.Errorf("oidc provider named '%s' is not configured", p.GetName())
	}

	redirectURL := *tenantURL
	if universe.Current().IsDev() && p.UseLocalHostRedirect() {
		if port := redirectURL.Port(); port != "" {
			redirectURL.Host = fmt.Sprintf("localhost:%s", port)
		} else {
			redirectURL.Host = "localhost"
		}
	}

	return &redirectURL, nil
}

// UpdateSaveSettings encodes any secrets returned from the UI and applies any read-only settings
// that had been filtered from the settings presented to the UI
func (tenant *TenantConfig) UpdateSaveSettings(ctx context.Context, tenantID uuid.UUID, source TenantConfig) error {
	tenant.applyReadOnlySettings(source)

	if err := tenant.encodeSecrets(ctx, tenantID, source); err != nil {
		uclog.Debugf(ctx, "Failed to encode secrets: %v", err)
		return ucerr.Wrap(err)
	}

	return nil
}

func (tenant *TenantConfig) applyReadOnlySettings(source TenantConfig) {
	// make sure the tenant has no pre-existing settings
	tenant.filterReadOnlySettings()

	// apply tenant page parameters
	for pt, parametersByName := range source.PageParameters {
		for _, p := range parametersByName {
			tenant.SetPageParameter(pt, p.Name, p.Value)
		}
	}

	// apply plex map read-only settings
	tenant.PlexMap.applyReadOnlySettings(source.PlexMap)
}

// UpdateUISettings updates the tenant config for the UI, filtering out read-only
// settings and decoding secrets as appropriate for the current universe
func (tenant *TenantConfig) UpdateUISettings(ctx context.Context) error {
	// remove the parts of the TenantConfig that are read-only
	tenant.filterReadOnlySettings()

	if err := tenant.decodeSecrets(ctx); err != nil {
		uclog.Debugf(ctx, "Failed to decode secrets: %v", err)
		return ucerr.Wrap(err)
	}

	return nil
}

func (tenant *TenantConfig) filterReadOnlySettings() {
	// clear tenant page parameters
	tenant.PageParameters = pageparams.ParameterByNameByPageType{}

	// filter plex map read-only settings
	tenant.PlexMap.filterReadOnlySettings()
}

func (tenant *TenantConfig) decodeSecrets(ctx context.Context) error {
	if err := tenant.PlexMap.EmailServer.DecodeSecrets(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	if err := tenant.PlexMap.TelephonyProvider.DecodeSecrets(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	// TODO (sgarrity 7/24): I really don't like this pattern but following
	// it for now to at least be consistent. Note that I'm at least using secret.X
	// constants rather than totally parallel fields that leave artifacts.
	// We have a lot of repetitive code that isn't easily unified, and a lot of
	// path strings that are getting strewn around. I think there's a better pattern
	// here that
	//
	// 1) replaces secret.String with a securely generated random ID on API write
	//    that includes a sentinel for write-back and is cached for ~N minutes
	//    and maps to the original secret location
	// 2) recognizes this in the UI (with the secret input component) and when you click
	//    "reveal", makes a server call to resolve it (if allowed), so we only
	//    ever resolve the secret from SM if the user asks
	// 3) relies on the ID generation at original API call time to ensure that the ACLs
	//    are correct for the resolve call (and/or we add more ACLs later?)
	// 4) encodes this token + any newly written secret in a structured JSON object
	//    so we can easily tell if it's a new secret or a write-back, and we can
	//    automatically store it on the server side during deserialization (this
	//    would require a little bit of thought to express the randomness in a
	//    secret location so you didn't overwrite if validation failed?)
	for i := range tenant.PlexMap.Apps {
		if err := tenant.PlexMap.Apps[i].DecodeSecrets(ctx); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if err := tenant.PlexMap.EmployeeApp.DecodeSecrets(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	for i := range tenant.PlexMap.Providers {
		if err := tenant.PlexMap.Providers[i].DecodeSecrets(ctx); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if err := tenant.PlexMap.EmployeeProvider.DecodeSecrets(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	for i := range tenant.OIDCProviders.Providers {
		if err := tenant.OIDCProviders.Providers[i].DecodeSecrets(ctx); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

// encodeSecrets encodes secret values on the basis of values from the UI
func (tenant *TenantConfig) encodeSecrets(ctx context.Context, tenantID uuid.UUID, source TenantConfig) error {
	if err := tenant.PlexMap.EmailServer.EncodeSecrets(ctx, tenantID, source.PlexMap.EmailServer); err != nil {
		return ucerr.Wrap(err)
	}

	if err := tenant.PlexMap.TelephonyProvider.EncodeSecrets(ctx, tenantID, source.PlexMap.TelephonyProvider); err != nil {
		return ucerr.Wrap(err)
	}

	// TODO (sgarrity 7/24): I don't particularly like this pattern but following
	// it for now to at least be consistent. Note that I'm at least using secret.X
	// constants rather than totally parallel fields that leave artifacts
	for i := range tenant.PlexMap.Apps {
		if err := tenant.PlexMap.Apps[i].EncodeSecrets(ctx, &source.PlexMap); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if err := tenant.PlexMap.EmployeeApp.EncodeSecrets(ctx, &source.PlexMap); err != nil {
		return ucerr.Wrap(err)
	}

	for i := range tenant.PlexMap.Providers {
		if err := tenant.PlexMap.Providers[i].EncodeSecrets(ctx, &source.PlexMap); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if err := tenant.PlexMap.EmployeeProvider.EncodeSecrets(ctx, &source.PlexMap); err != nil {
		return ucerr.Wrap(err)
	}

	for i := range tenant.OIDCProviders.Providers {
		if err := tenant.OIDCProviders.Providers[i].EncodeSecrets(ctx, source.OIDCProviders.Providers); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}

// GetMFASettings returns whether MFA is required and which MFA channel types are supported for the tenant
func (tenant *TenantConfig) GetMFASettings(clientID string) (mfaRequired bool, mfaChannelTypes oidc.MFAChannelTypeSet, err error) {
	// NOTE: as part of validation we ensure that MFA Required is only true if there are valid configured
	//       MFA channel types
	app, _, err := tenant.PlexMap.FindAppForClientID(clientID)
	if err != nil {
		return false, nil, ucerr.Wrap(err)
	}

	pg := MakeRenderParameterGetter(tenant, app)
	p, found := pg(pagetype.EveryPage, param.MFARequired)
	if found && p.Value == "true" {
		mfaRequired = true
	}

	mfaChannelTypes = oidc.MFAChannelTypeSet{}
	p, found = pg(pagetype.EveryPage, param.MFAMethods)
	if found {
		for _, ct := range parametertype.GetOptions(p.Value) {
			mfaChannelTypes[oidc.MFAChannelType(ct)] = true
		}
	}

	return mfaRequired, mfaChannelTypes, nil
}
