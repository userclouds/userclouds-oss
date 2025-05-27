package oidcproviders

import (
	"context"
	"slices"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	internaloidc "userclouds.com/internal/oidc"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/tenantplex/storage"
)

// Verifier is functionally an alias for oidc.IDTokenVerifier, but we need to wrap it in order to
// mock it for tests
type Verifier interface {
	Verify(context.Context, string) (*oidc.IDToken, error)
}

// Provider is functionally an alias for oidc.Provider, but we need to wrap it in order to
// mock it for tests
type Provider interface {
	Verifier(*oidc.Config) Verifier
}

// OIDCProvider wraps oidc.Provider and implements the Provider interface
type OIDCProvider struct {
	*oidc.Provider
}

// Verifier implements the Provider interface
func (o OIDCProvider) Verifier(config *oidc.Config) Verifier {
	return OIDCVerifier{o.Provider.Verifier(config)}
}

// OIDCVerifier wraps oidc.IDTokenVerifier and implements the Verifier interface
type OIDCVerifier struct {
	*oidc.IDTokenVerifier
}

// OIDCProviderMap provides a map from Tenant ID to a Provider objects, which caches information
// about an OIDC provider (e.g. OpenID configuration, JSON web key set, etc). This allows us to validate
// JWTs for API calls to a tenant without re-fetching this cacheable & mostly immutable state repeatedly.
type OIDCProviderMap struct {
	providers      map[string]Provider //maps issuer to provider
	providersMutex sync.RWMutex

	tenantIssuers      map[uuid.UUID]set.Set[string] // map from tenant ID to  a set of allowed issuers
	tenantIssuersMutex sync.RWMutex

	tenantInvalidationRegistered map[uuid.UUID]bool
	tenantInvalidationMutex      sync.Mutex

	// NB: fallbackProvider is a special provider which can be used to issue/validate tokens on behalf of any other tenant,
	// We use this so that Console can issue tokens for other tenants, but limit this power (slightly) by ensuring the audience matches.
	// This allows the developer console to read/write data to any tenant without needing to authenticate against each independent tenant's Plex.
	// We could have done that too (and may still do it) but it requires all tenants to have a guaranteed-to-exist, discoverable
	// OIDC client that Console can use the Client Credentials Flow with to get an M2M token.
	fallbackProvider *oidc.Provider

	// fallbackProviderURL allows us to lazy-load the fallback provider, which makes startup
	// less ordering-dependent (since otherwise IDP can't finish starting until Plex is fully up)
	fallbackProviderURL string
}

// NewOIDCProviderMap returns a new OIDCProviderMap
func NewOIDCProviderMap() *OIDCProviderMap {
	return &OIDCProviderMap{
		providers:                    map[string]Provider{},
		tenantIssuers:                map[uuid.UUID]set.Set[string]{},
		tenantInvalidationRegistered: map[uuid.UUID]bool{},
	}
}

var newProvider func(string) (Provider, error)

func init() {
	newProvider = func(providerURL string) (Provider, error) {
		// NB: the Go OIDC library has a pretty weird design flaw where it caches[1] the context passed in to NewProvider
		// and uses it much later to issue GET requests, e.g. to fetch JSON Web Key Sets (JWKS) manifests.
		// Since it is very likely that the context passed in to this method is associated with an inbound server request,
		// and since those get canceled[2] upon completion of the request, this means that if this provider is *ever* used
		// later to fetch JWKS then it will fail with a very-hard-to-trace "context canceled" error deep inside the key
		// management logic of the OIDC library.
		// Go ahead, ask me how I know.
		// [1] https://github.com/coreos/go-oidc/blob/v3/oidc/jwks.go#L70 & https://github.com/coreos/go-oidc/blob/v3/oidc/jwks.go#L219
		// [2] https://pkg.go.dev/net/http#Request.Context

		prov, err := oidc.NewProvider(context.Background(), providerURL)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		return OIDCProvider{prov}, nil
	}
}

func getIssuersForTenant(ctx context.Context, tenantState *tenantmap.TenantState) (set.Set[string], error) {
	mainTenantURL := tenantState.GetTenantURL()
	issuers := set.NewStringSet()
	// Add all the tenant URLs for this tenant
	tenantURLs, err := tenantState.CompanyConfigStorage.ListTenantURLsForTenant(ctx, tenantState.ID)
	if err != nil {
		return set.NewStringSet(), ucerr.Wrap(err)
	}
	s := storage.New(ctx, tenantState.TenantDB, tenantState.CacheConfig)
	tenantPlex, err := s.GetTenantPlex(ctx, tenantState.ID)
	if err != nil {
		return set.NewStringSet(), ucerr.Wrap(err)
	}
	for _, tenantURL := range tenantURLs {
		issuers.Insert(tenantURL.TenantURL)
	}
	// This happens once per per tenant, per process because the caller caches the result in memory.
	uclog.Infof(ctx, "issuers for tenant %v [%s/%s]: main URL: %s from URLs: %v External: %v", tenantState.ID, tenantState.CompanyName, tenantState.TenantName, mainTenantURL, issuers, tenantPlex.PlexConfig.ExternalOIDCIssuers)
	issuers.Insert(mainTenantURL)
	// Add all the external OIDC issuers for this tenant
	issuers.Insert(tenantPlex.PlexConfig.ExternalOIDCIssuers...)
	return issuers, nil
}

func (pm *OIDCProviderMap) getAllowedIssuers(ctx context.Context, tenantState *tenantmap.TenantState) (set.Set[string], error) {
	pm.tenantIssuersMutex.RLock()
	allowedIssuers, ok := pm.tenantIssuers[tenantState.ID]
	pm.tenantIssuersMutex.RUnlock()
	if ok {
		return allowedIssuers, nil
	}
	allowedIssuers, err := getIssuersForTenant(ctx, tenantState)
	if err != nil {
		return set.NewStringSet(), ucerr.Wrap(err)
	}
	pm.tenantIssuersMutex.Lock()
	pm.tenantIssuers[tenantState.ID] = allowedIssuers
	pm.tenantIssuersMutex.Unlock()
	pm.registerPlexChangeHandler(ctx, tenantState)
	return allowedIssuers, nil
}

func (pm *OIDCProviderMap) checkTenantIssuers(ctx context.Context, tenantState *tenantmap.TenantState, issuer string) (bool, error) {
	allowedIssuers, err := pm.getAllowedIssuers(ctx, tenantState)
	if err != nil {
		return false, ucerr.Wrap(err)
	}
	isAllowed := allowedIssuers.Contains(issuer)
	if !isAllowed {
		uclog.Warningf(ctx, "issuer %s is not allowed for tenant %v. allowed: %v", issuer, tenantState.ID, allowedIssuers)
	}
	return isAllowed, nil
}

func (pm *OIDCProviderMap) getProviderForTenantAndIssuer(ctx context.Context, tenantState *tenantmap.TenantState, issuer string) (Provider, error) {
	if issuerAllowed, err := pm.checkTenantIssuers(ctx, tenantState, issuer); err != nil {
		return nil, ucerr.Wrap(err)
	} else if !issuerAllowed {
		// Can't find issuer for the tenant, so the caller will try with the fallback provider
		uclog.Infof(ctx, "issuer %s is not allowed for tenant %v", issuer, tenantState.ID)
		return nil, nil
	}
	pm.providersMutex.RLock()
	provider, ok := pm.providers[issuer]
	pm.providersMutex.RUnlock()
	if ok {
		return provider, nil
	}
	uclog.Verbosef(ctx, "initiating OIDC provider for issuer: %s", issuer)
	provider, err := newProvider(issuer)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	pm.providersMutex.Lock()
	pm.providers[issuer] = provider
	pm.providersMutex.Unlock()
	return provider, nil
}

func (pm *OIDCProviderMap) registerPlexChangeHandler(ctx context.Context, tenantState *tenantmap.TenantState) {
	pm.tenantInvalidationMutex.Lock()
	defer pm.tenantInvalidationMutex.Unlock()
	if pm.tenantInvalidationRegistered[tenantState.ID] {
		return
	}

	handler := func(ctx context.Context, key cache.Key, flush bool) error {
		pm.resetIssuersForTenant(ctx, tenantState.ID, "tenant plex change")
		return nil
	}

	s := storage.New(ctx, tenantState.TenantDB, tenantState.CacheConfig)
	if err := s.RegisterTenantPlexChangeHandler(ctx, handler, tenantState.ID); err != nil {
		// log but don't fail since the providers were still successfully initialized
		uclog.Errorf(ctx, "error registering tenant %v plex change handler: %v", tenantState.ID, err)
	} else {
		pm.tenantInvalidationRegistered[tenantState.ID] = true
	}
}

func (pm *OIDCProviderMap) resetIssuersForTenant(ctx context.Context, tenantID uuid.UUID, reason string) {
	uclog.Debugf(ctx, "resetting allowed issuers for tenant %v: %s", tenantID, reason)
	pm.providersMutex.Lock()
	delete(pm.tenantIssuers, tenantID)
	pm.providersMutex.Unlock()
}

func (pm *OIDCProviderMap) initFallbackProvider() error {
	if pm.fallbackProviderURL == "" {
		return ucerr.New("can't call initFallbackProvider with empty fallbackProviderURL")
	}

	// short circuiting in this method keeps the locks more manageable
	pm.providersMutex.RLock()
	if pm.fallbackProvider != nil {
		pm.providersMutex.RUnlock()
		return nil
	}
	pm.providersMutex.RUnlock()

	// need to init it
	provider, err := oidc.NewProvider(context.Background(), pm.fallbackProviderURL)
	if err != nil {
		return ucerr.Wrap(err)
	}
	pm.providersMutex.Lock()
	pm.fallbackProvider = provider
	pm.providersMutex.Unlock()
	return nil
}

// useFallbackProvider simply makes locking a little easier
func (pm *OIDCProviderMap) useFallbackProvider(ctx context.Context, config *oidc.Config, rawJWT string) (*oidc.IDToken, error) {
	if err := pm.initFallbackProvider(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	// if we didn't get an error, we can try the fallback provider
	pm.providersMutex.RLock()
	defer pm.providersMutex.RUnlock()

	decodedJWT, err := pm.fallbackProvider.Verifier(config).Verify(ctx, rawJWT)
	if err != nil {
		uclog.Warningf(ctx, "error verifying JWT against fallback provider: %+v", err)
		return nil, ucerr.Wrap(err)
	}

	return decodedJWT, nil
}

// ClearFallbackProvider is only used by tests to validate the behavior of the fallback provider.
func (pm *OIDCProviderMap) ClearFallbackProvider() {
	pm.providersMutex.Lock()
	pm.fallbackProvider = nil
	pm.fallbackProviderURL = ""
	pm.providersMutex.Unlock()
}

// SetFallbackProviderToTenant initializes the special 'fallback' OIDC provider to the given tenant's
// Tenant URL which allows it to be used as a token source on behalf of any tenant.
func (pm *OIDCProviderMap) SetFallbackProviderToTenant(ctx context.Context, storage *companyconfig.Storage, tenantID uuid.UUID) error {
	// Init the JWT verification logic so that the given tenant is a "special" provider whose
	// tokens can be trusted for any other tenant.
	tenant, err := storage.GetTenant(ctx, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// just set the URL and we'll lazy-load it later to make startup more predictable
	pm.fallbackProviderURL = tenant.TenantURL

	return nil
}

// VerifyAndDecode implements ucjwt.Verifier
// NOTE: this requires use of the multitenant.Middleware to ensure that tenant state
// is properly set up in the context.
func (pm *OIDCProviderMap) VerifyAndDecode(ctx context.Context, rawJWT string) (*oidc.IDToken, error) {
	ts := multitenant.MustGetTenantState(ctx)

	claims, err := ucjwt.ParseJWTClaimsUnverified(rawJWT)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if claims["iss"] == nil {
		return nil, ucerr.Errorf("jwt is invalid because it is missing the 'iss' claim. clams: %+v", claims)
	}
	issuer, ok := claims["iss"].(string)
	if !ok {
		return nil, ucerr.Errorf("jwt is invalid because the 'iss' claim is not a string. type: %T", claims["iss"])
	}
	// Disable the Client ID check in favor of the audience check further down.
	config := &oidc.Config{
		SkipClientIDCheck: true,
	}

	provider, err := pm.getProviderForTenantAndIssuer(ctx, ts, issuer)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	var decodedJWT *oidc.IDToken
	if provider != nil {
		// TODO (sgarrity 3/24): we should refactor fallback provider to be in the providers array
		// and simplify all of this logic, as well as eliminate this logging.
		uclog.Verbosef(ctx, "found provider for issuer: %v", issuer)
		decodedJWT, err = provider.Verifier(config).Verify(ctx, rawJWT)
		if err != nil {
			// NB: if there was a provider for this issuer, verification is really a failure
			// (vs in the past, we didn't log this because it could be the console fallback)
			uclog.Warningf(ctx, "error verifying JWT against tenant, issuer: %s, : %+v", issuer, err)
			decodedJWT = nil
		} else if decodedJWT == nil {
			uclog.Warningf(ctx, "error verifying JWT against tenant, issuer: %s - no error returned", issuer)
		}
	}

	if decodedJWT == nil {
		if pm.fallbackProviderURL != "" {
			uclog.Infof(ctx, "Failed to verify JWT (issuer: %s) against tenant %v (will try fallback): %v", issuer, ts.ID, pm.fallbackProviderURL)
			decodedJWT, err = pm.useFallbackProvider(ctx, config, rawJWT)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
		} else {
			return nil, ucerr.Errorf("error verifying JWT against tenant %v and no fallback provider is configured.", ts.ID)
		}
	}

	if internaloidc.IsUsercloudsIssued(decodedJWT.Issuer) {
		// Tokens issued by Plex to be used with IDP & AuthZ APIs should have the appropriate IDP/AuthZ
		// tenant listed as an audience (currently by URL, but [TODO] we could go by UUID instead).
		// This ensures we're only using tokens intended for this purpose.
		if !slices.Contains(decodedJWT.Audience, ts.GetTenantURL()) {
			return nil, ucerr.Errorf("jwt is invalid because tenant URL '%s' is not in the list of audiences '%v'", ts.TenantURL, decodedJWT.Audience)
		}
	}
	// TODO jwang 2/24 - what should the appropriate audience should be for an external token be?
	return decodedJWT, nil
}
