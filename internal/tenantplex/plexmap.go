package tenantplex

import (
	"context"
	"net/url"
	"slices"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/messaging/email"
	"userclouds.com/infra/uctypes/messaging/telephony"
	"userclouds.com/infra/uctypes/set"
	message "userclouds.com/internal/messageelements"
	pageparams "userclouds.com/internal/pageparameters"
	"userclouds.com/internal/pageparameters/pagetype"
	param "userclouds.com/internal/pageparameters/parameter"
)

// ProviderType is a string enum of different backend auth providers.
type ProviderType string

// Const string enum values for ProviderType
const (
	ProviderTypeAuth0    ProviderType = "auth0"
	ProviderTypeEmployee ProviderType = "employee"
	ProviderTypeUC       ProviderType = "uc"
	ProviderTypeCognito  ProviderType = "cognito"
)

// PlexMap configures a Plex instance and maps Plex apps (client IDs)
// to underlying auth providers' apps/clients.
// TODO: Either nest a PlexMap inside some kind of PlexTenant struct,
// or add more non-mapping config to it and rename it?
type PlexMap struct {
	Providers []Provider `yaml:"providers,omitempty" json:"providers"`
	Apps      []App      `yaml:"apps,omitempty" json:"apps"`
	Policy    Policy     `yaml:"policy,omitempty" json:"policy"`

	EmployeeProvider *Provider `yaml:"employee_provider,omitempty" json:"employee_provider,omitempty"`
	EmployeeApp      *App      `yaml:"employee_app,omitempty" json:"employee_app,omitempty"`

	TelephonyProvider telephony.ProviderConfig `yaml:"telephony_provider,omitempty" json:"telephony_provider"`
	EmailServer       email.SMTPServer         `yaml:"email_server,omitempty" json:"email_server"`
}

// Provider is the config struct for a single underlying IDP provider tenant.
type Provider struct {
	ID   uuid.UUID    `yaml:"id" json:"id" validate:"notnil"`
	Name string       `yaml:"name" json:"name" validate:"notempty"`
	Type ProviderType `yaml:"type" json:"type"`

	Auth0   *Auth0Provider   `yaml:"auth0,omitempty" json:"auth0,omitempty" validate:"allownil"`
	UC      *UCProvider      `yaml:"uc,omitempty" json:"uc,omitempty" validate:"allownil"`
	Cognito *CognitoProvider `yaml:"cognito,omitempty" json:"cognito,omitempty" validate:"allownil"`
}

//go:generate genvalidate Provider

// CanSyncUsers returns true if the provider can sync users (assuming it is active)
func (p Provider) CanSyncUsers() bool {
	return p.Type == ProviderTypeAuth0 || p.Type == ProviderTypeCognito
}

// DecodeSecrets will replace any secrets in the provider config with placeholders
func (p *Provider) DecodeSecrets(ctx context.Context) error {
	if p == nil {
		return nil
	}

	if p.Auth0 != nil {
		for i := range p.Auth0.Apps {
			s, err := p.Auth0.Apps[i].ClientSecret.ResolveForUI(ctx)
			if err != nil {
				return ucerr.Wrap(err)
			}
			p.Auth0.Apps[i].ClientSecret = *s
		}

		s, err := p.Auth0.Management.ClientSecret.ResolveForUI(ctx)
		if err != nil {
			return ucerr.Wrap(err)
		}
		p.Auth0.Management.ClientSecret = *s
	}

	if p.Cognito != nil {
		for i := range p.Cognito.Apps {
			s, err := p.Cognito.Apps[i].ClientSecret.ResolveForUI(ctx)
			if err != nil {
				return ucerr.Wrap(err)
			}

			p.Cognito.Apps[i].ClientSecret = *s
		}
	}

	return nil
}

// EncodeSecrets will replace any UI secrets in the provider config with actual secrets
func (p *Provider) EncodeSecrets(ctx context.Context, source *PlexMap) error {
	if p == nil {
		return nil
	}

	if p.Auth0 != nil {
		var sourceAuth0Provider *Auth0Provider
		for i, sourceProvider := range source.Providers {
			if p.ID == sourceProvider.ID {
				sourceAuth0Provider = source.Providers[i].Auth0
			}
		}

		for i := range p.Auth0.Apps {
			if err := p.Auth0.Apps[i].EncodeSecrets(ctx, sourceAuth0Provider); err != nil {
				return ucerr.Wrap(err)
			}
		}

		// easy case first
		if p.Auth0.Management.ClientSecret == secret.EmptyString {
			return nil
		}

		// no changes, or sourceAuth0Provider is nil so the whole provider is new
		if p.Auth0.Management.ClientSecret == secret.UIPlaceholder || sourceAuth0Provider == nil {
			p.Auth0.Management.ClientSecret = sourceAuth0Provider.Management.ClientSecret
			return nil
		}

		// must be a new secret, or a new app (we didn't find the app ID in the source above)
		sec, err := p.Auth0.Management.ClientSecret.MarshalText()
		if err != nil {
			return ucerr.Wrap(err)
		}

		// TODO (sgarrity 7/24) this is my major objection to this pattern, this random thing
		// has to stay in sync with random places
		ns, err := crypto.CreateClientSecret(ctx, "auth0mgmt", string(sec))
		if err != nil {
			return ucerr.Wrap(err)
		}
		p.Auth0.Management.ClientSecret = *ns
	}

	if p.Cognito != nil {
		var sourceCognitoProvider *CognitoProvider
		for i, sourceProvider := range source.Providers {
			if p.ID == sourceProvider.ID {
				sourceCognitoProvider = source.Providers[i].Cognito
			}
		}

		for i := range p.Cognito.Apps {
			if err := p.Cognito.Apps[i].EncodeSecrets(ctx, sourceCognitoProvider); err != nil {
				return ucerr.Wrap(err)
			}
		}
	}

	return nil
}

// Auth0Provider defines Auth0 tenant config.
type Auth0Provider struct {
	Domain     string          `yaml:"domain" json:"domain" validate:"notempty"`
	Apps       []Auth0App      `yaml:"apps,omitempty" json:"apps"`
	Management Auth0Management `yaml:"management,omitempty" json:"management"`

	// TODO: is this the right long term design? this is for a demo 5/5/22
	// usually false, meaning our normal API-driven flow, but if set to true, we use
	// an OIDC redirect to log in via Auth0's UI and back to us, so we can rewrite
	// the token (eg for delegation purposes). This lets us use existing Auth0 management
	// infrastructure to fill out the impersonated token etc.
	Redirect bool `yaml:"redirect" json:"redirect"`
}

//go:generate genvalidate Auth0Provider

// FindProviderApp finds the auth0 app that matches the plex app
func (a *Auth0Provider) FindProviderApp(app *App) (*Auth0App, error) {
	for i, a0App := range a.Apps {
		if slices.Contains(app.ProviderAppIDs, a0App.ID) {
			return &a.Apps[i], nil
		}
	}
	return nil, ucerr.Errorf("no matching Auth0App found for %v", app.ID)
}

// Auth0App defines Auth0 application (client) config.
type Auth0App struct {
	ID           uuid.UUID     `yaml:"id" json:"id" validate:"notnil"`
	Name         string        `yaml:"name" json:"name" validate:"notempty"`
	ClientID     string        `yaml:"client_id" json:"client_id" validate:"notempty"`
	ClientSecret secret.String `yaml:"client_secret" json:"client_secret"`
}

//go:generate genvalidate Auth0App

// EncodeSecrets will replace any UI secrets in the provider config with actual secrets
func (app *Auth0App) EncodeSecrets(ctx context.Context, source *Auth0Provider) error {
	// easy case first
	if app.ClientSecret == secret.EmptyString {
		return nil
	}

	// no changes, or source is nil so the whole provider is new
	if app.ClientSecret == secret.UIPlaceholder || source == nil {
		for _, sourceApp := range source.Apps {
			if app.ID == sourceApp.ID {
				app.ClientSecret = sourceApp.ClientSecret
				return nil
			}
		}
	}

	// must be a new secret, or a new app (we didn't find the app ID in the source above)
	sec, err := app.ClientSecret.MarshalText()
	if err != nil {
		return ucerr.Wrap(err)
	}

	ns, err := crypto.CreateClientSecret(ctx, app.ID.String(), string(sec))
	if err != nil {
		return ucerr.Wrap(err)
	}
	app.ClientSecret = *ns
	return nil
}

// Auth0Management defines Auth0 client config for the management API
// This is very similar to Auth0App and could be shoehorned in, but
// using a separate type makes it easier to understand where we're using
// the management API vs OIDC-ish API, and probably ends up being 1-to-many
// with the OIDC clients eventually.
type Auth0Management struct {
	ClientID     string        `yaml:"client_id" json:"client_id" validate:"notempty"`
	ClientSecret secret.String `yaml:"client_secret" json:"client_secret"`
	Audience     string        `yaml:"audience" json:"audience" validate:"notempty"`
}

//go:generate genvalidate Auth0Management

// UCProvider defines UserClouds tenant config.
// TODO: maybe rename IDPURL to TenantURL?
type UCProvider struct {
	IDPURL string  `yaml:"idp_url" json:"idp_url"`
	Apps   []UCApp `yaml:"apps,omitempty" json:"apps"`
}

//go:generate genvalidate UCProvider

// UCApp defines UserClouds Client config.
type UCApp struct {
	ID   uuid.UUID `yaml:"id" json:"id" validate:"notnil"`
	Name string    `yaml:"name" json:"name" validate:"notempty"`
}

//go:generate genvalidate UCApp

// CognitoProvider defines Cognito tenant config.
type CognitoProvider struct {
	AWSConfig  ucaws.Config `yaml:"aws_config" json:"aws_config"`
	UserPoolID string       `yaml:"user_pool_id" json:"user_pool_id" validate:"notempty"`

	Apps []CognitoApp `yaml:"apps,omitempty" json:"apps"`
}

//go:generate genvalidate CognitoProvider

// CognitoApp defines a cognito login app to map to plex apps
type CognitoApp struct {
	ID   uuid.UUID `yaml:"id" json:"id" validate:"notnil"`
	Name string    `yaml:"name" json:"name" validate:"notempty"`

	ClientID     string        `yaml:"client_id" json:"client_id" validate:"notempty"`
	ClientSecret secret.String `yaml:"client_secret" json:"client_secret"`
}

// EncodeSecrets will replace any UI secrets in the provider config with actual secrets
func (app *CognitoApp) EncodeSecrets(ctx context.Context, source *CognitoProvider) error {
	// easy case first
	if app.ClientSecret == secret.EmptyString {
		return nil
	}

	// no changes, or source is nil so the whole provider is new
	if app.ClientSecret == secret.UIPlaceholder || source == nil {
		for _, sourceApp := range source.Apps {
			if app.ID == sourceApp.ID {
				app.ClientSecret = sourceApp.ClientSecret
				return nil
			}
		}
	}

	// must be a new secret, or a new app (we didn't find the app ID in the source above)
	sec, err := app.ClientSecret.MarshalText()
	if err != nil {
		return ucerr.Wrap(err)
	}

	ns, err := crypto.CreateClientSecret(ctx, app.ID.String(), string(sec))
	if err != nil {
		return ucerr.Wrap(err)
	}
	app.ClientSecret = *ns
	return nil
}

// GrantType represents the supported types of OAuth grants
type GrantType string

//go:generate genconstant GrantType

// GrantType constants
const (
	GrantTypeUnknown           GrantType = ""
	GrantTypeImplicit          GrantType = "implicit"
	GrantTypeAuthorizationCode GrantType = "authorization_code"
	GrantTypeRefreshToken      GrantType = "refresh_token"
	GrantTypeClientCredentials GrantType = "client_credentials"
	GrantTypePassword          GrantType = "password"
	GrantTypeDeviceCode        GrantType = "urn:ietf:params:oauth:grant-type:device_code"

	// non-standard grant types
	// Auth0 maps these to different grant types, but then doesn't let you
	// enable/disable them individually per this doc:
	// https://auth0.com/docs/get-started/applications/application-grant-types#auth0-extension-grants
	// so for now, we're just supporting one
	GrantTypeMFA GrantType = "mfa"
)

// GrantTypes is an array of GrantType that exists to hang things like Contains() on
type GrantTypes []GrantType

// SupportedGrantTypes is the list of grant types we support today
// (vs the ones that we can copy over from Auth0 :) )
// TODO: we should support more!
var SupportedGrantTypes = GrantTypes{
	GrantTypeAuthorizationCode,
	GrantTypeRefreshToken,
	GrantTypeClientCredentials,
	GrantTypePassword,
}

// Contains returns true if the given GrantType is in the array
// This is useful for validating eg. if a GrantType is allowed in an app
func (g GrantTypes) Contains(t GrantType) bool {
	return slices.Contains(g, t)
}

// App defines a Plex app which maps to 1 or more underlying Provider apps/clients.
type App struct {
	ID             uuid.UUID `yaml:"id" json:"id" validate:"notnil"`
	Name           string    `yaml:"name" json:"name" validate:"notempty"`
	Description    string    `yaml:"description" json:"description"`
	OrganizationID uuid.UUID `yaml:"organization_id" json:"organization_id"`

	ClientID     string        `yaml:"client_id" json:"client_id" validate:"notempty"`
	ClientSecret secret.String `yaml:"client_secret" json:"client_secret"`

	RestrictedAccess bool `yaml:"restricted_access" json:"restricted_access"`

	TokenValidity TokenValidity `yaml:"token_validity" json:"token_validity"`

	ProviderAppIDs []uuid.UUID `yaml:"provider_app_ids" json:"provider_app_ids" validate:"notnil"`

	AllowedRedirectURIs []string                                   `yaml:"allowed_redirect_uris" json:"allowed_redirect_uris"  validate:"skip"`
	AllowedLogoutURIs   []string                                   `yaml:"allowed_logout_uris" json:"allowed_logout_uris"  validate:"skip"`
	MessageElements     message.ElementsByElementTypeByMessageType `yaml:"message_elements" json:"message_elements" validate:"skip"`
	PageParameters      pageparams.ParameterByNameByPageType       `yaml:"page_parameters" json:"page_parameters"`

	// GrantTypes that this app supports
	GrantTypes GrantTypes `yaml:"grant_types" json:"grant_types"`

	// the ID (if any) of the provider this was synced from
	SyncedFromProvider uuid.UUID `yaml:"synced_from_provider" json:"synced_from_provider" validate:"skip"`

	ImpersonateUserConfig ImpersonateUserConfig `yaml:"impersonate_user_config" json:"impersonate_user_config" validate:"skip"`

	SAMLIDP *SAMLIDP `yaml:"saml_idp" json:"saml_idp,omitempty" validate:"allownil"`
}

// Equals checks if the two apps are the same
func (app App) Equals(o *App) bool {

	if len(app.ProviderAppIDs) != len(o.ProviderAppIDs) {
		return false
	}
	for i := range app.ProviderAppIDs {
		if app.ProviderAppIDs[i] != o.ProviderAppIDs[i] {
			return false
		}

	}

	if len(app.AllowedRedirectURIs) != len(o.AllowedRedirectURIs) {
		return false
	}
	for i := range app.AllowedRedirectURIs {
		if app.AllowedRedirectURIs[i] != o.AllowedRedirectURIs[i] {
			return false
		}

	}

	if len(app.AllowedLogoutURIs) != len(o.AllowedLogoutURIs) {
		return false
	}
	for i := range app.AllowedLogoutURIs {
		if app.AllowedLogoutURIs[i] != o.AllowedLogoutURIs[i] {
			return false
		}

	}

	if len(app.MessageElements) != len(o.MessageElements) {
		return false
	}
	for i := range app.MessageElements {
		if len(app.MessageElements[i]) != len(o.MessageElements[i]) {
			return false
		}
		for j := range app.MessageElements[i] {
			if app.MessageElements[i][j] != o.MessageElements[i][j] {
				return false
			}
		}
	}

	if len(app.PageParameters) != len(o.PageParameters) {
		return false
	}
	for i := range app.PageParameters {
		if len(app.PageParameters[i]) != len(o.PageParameters[i]) {
			return false
		}
		for j := range app.PageParameters[i] {
			if app.PageParameters[i][j] != o.PageParameters[i][j] {
				return false
			}

		}
	}

	origGrantTypes := set.NewStringSet()
	for _, gt := range app.GrantTypes {
		origGrantTypes.Insert(string(gt))
	}
	otherGrantTypes := set.NewStringSet()
	for _, gt := range o.GrantTypes {
		otherGrantTypes.Insert(string(gt))
	}
	if !origGrantTypes.Equal(otherGrantTypes) {
		return false
	}

	return app.ID == o.ID &&
		app.Name == o.Name &&
		app.Description == o.Description &&
		app.ClientID == o.ClientID &&
		app.ClientSecret == o.ClientSecret &&
		app.SyncedFromProvider == o.SyncedFromProvider &&
		app.ImpersonateUserConfig == o.ImpersonateUserConfig
}

// DecodeSecrets will replace any secrets in the provider config with
// the actual secret for the login app
func (app *App) DecodeSecrets(ctx context.Context) error {
	s, err := app.ClientSecret.ResolveInsecurelyForUI(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	app.ClientSecret = *s

	return nil
}

// EncodeSecrets will replace any UI secrets in the provider config with actual secrets
func (app *App) EncodeSecrets(ctx context.Context, source *PlexMap) error {
	// easy case first
	if app.ClientSecret == secret.EmptyString {
		return nil
	}

	// TODO (sgarrity 7/24): someday we should actually resolve from the UI as needed and fix this
	currentSecret, err := app.ClientSecret.MarshalText()
	if err != nil {
		return ucerr.Wrap(err)
	}

	var sourceSecret secret.String
	if source != nil {
		for _, sourceApp := range source.Apps {
			if app.ID == sourceApp.ID {
				sourceSecret = sourceApp.ClientSecret
				break
			}
		}
		if source.EmployeeApp != nil && app.ID == source.EmployeeApp.ID {
			sourceSecret = source.EmployeeApp.ClientSecret
		}
	}

	actualSourceSecret, err := sourceSecret.Resolve(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if string(currentSecret) == actualSourceSecret {
		app.ClientSecret = sourceSecret
		return nil
	}

	// must be a new secret, or a new app (we didn't find the app ID in the source above)
	ns, err := crypto.CreateClientSecret(ctx, app.ID.String(), string(currentSecret))
	if err != nil {
		return ucerr.Wrap(err)
	}
	app.ClientSecret = *ns
	return nil
}

//go:generate genvalidate App

// TokenValidity defines how long various token types are valid for
type TokenValidity struct {
	Access          int64 `json:"access" validate:"notzero"`
	Refresh         int64 `json:"refresh" validate:"notzero"`
	ImpersonateUser int64 `json:"impersonate_user" validate:"notzero"`
}

//go:generate genvalidate TokenValidity

// ImpersonateUserConfig defines configuration for rules around the impersonate user feature
// If CheckAttribute is set, then Plex will do an Authz check to see if the user has that attribute for the user they
// want to impersonate. Otherwise, if BypassCompanyAdminCheck is false, then Plex will do an Authz check to see if the user is a company admin.
type ImpersonateUserConfig struct {
	CheckAttribute          string `yaml:"check_attribute" json:"check_attribute"`
	BypassCompanyAdminCheck bool   `yaml:"bypass_company_admin_check" json:"bypass_company_admin_check"`
}

// Validate implements Validateable
func (app *App) extraValidate() error {
	for i, uri := range app.AllowedRedirectURIs {
		// NB: It turns out `url.Parse` actually succeeds at parsing basically anything you give it,
		// so maybe there's a better way to validate URLs in a more strict manner.
		if _, err := url.Parse(uri); err != nil {
			return ucerr.Friendlyf(err, "failed to parse redirect URI '%s' in app '%s' (ID: %v)", uri, app.ID, app.Name)
		}
		for j, otherURI := range app.AllowedRedirectURIs {
			if i != j && uri == otherURI {
				return ucerr.Friendlyf(nil, "duplicate redirect URI found at indices %d and %d: %s", i, j, uri)
			}
		}
	}

	for i, uri := range app.AllowedLogoutURIs {
		if _, err := url.Parse(uri); err != nil {
			return ucerr.Friendlyf(err, "failed to parse logout URI '%s' in app '%s' (ID: %v)", uri, app.ID, app.Name)
		}
		for j, otherURI := range app.AllowedLogoutURIs {
			if i != j && uri == otherURI {
				return ucerr.Friendlyf(nil, "duplicate logout URI found at indices %d and %d: %s", i, j, uri)
			}
		}
	}

	for mt, elements := range app.MessageElements {
		for elt, element := range elements {
			if err := message.ValidateMessageElement(mt, elt, element); err != nil {
				return ucerr.Friendlyf(err, "message element '%s' for message type '%s' and element type '%s' is invalid", element, mt, elt)
			}
		}
	}

	return nil
}

// ValidateRedirectURI ensures a given redirect URL matches one of the URLs found in the App's config.
func (app *App) ValidateRedirectURI(ctx context.Context, redirectURI string) (*url.URL, error) {
	redirectURIFound := slices.Contains(app.AllowedRedirectURIs, redirectURI)

	// allow us to redirect to ourselves for the SAML IDP flow
	// TODO: this is pretty janky, better way to handle this?
	if app.SAMLIDP != nil && redirectURI == strings.Replace(app.SAMLIDP.MetadataURL, "/metadata/", "/callback/", 1) {
		redirectURIFound = true
	}

	if !redirectURIFound {
		uclog.Warningf(ctx, "redirectURI %s not found in allowedRedirectURIs %v. plex app: %v (%s)", redirectURI, app.AllowedRedirectURIs, app.ID, app.Name)
		return nil, ucerr.Friendlyf(nil, "the specified redirect URI (%s) is not in the allowed list of redirect URIs for this client_id", redirectURI)
	}

	parsedURL, err := url.Parse(redirectURI)
	if err != nil {
		return nil, ucerr.Errorf("error parsing redirect URI: %w", err)
	}

	return parsedURL, nil
}

// ValidateLogoutURI ensures a given logout URL matches one of the URLs found in the App's config.
func (app *App) ValidateLogoutURI(ctx context.Context, logoutURI string) (*url.URL, error) {
	if !slices.Contains(app.AllowedLogoutURIs, logoutURI) {
		uclog.Warningf(ctx, "The logout URI '%s'is not in the allowed list: %v", logoutURI, app.AllowedLogoutURIs)
		return nil, ucerr.Friendlyf(nil, "the specified logout URI is not in the allowed list of redirect URIs for this client_id")
	}

	parsedURL, err := url.Parse(logoutURI)
	if err != nil {
		return nil, ucerr.Errorf("error parsing logout URI %s: %w", logoutURI, err)
	}

	return parsedURL, nil
}

// CustomizeMessageElement will set a message element override for an App for the specified MessageType and ElementType
func (app *App) CustomizeMessageElement(mt message.MessageType, elt message.ElementType, element string) {
	if app.MessageElements == nil {
		app.MessageElements = message.ElementsByElementTypeByMessageType{}
	}
	if _, ok := app.MessageElements[mt]; !ok {
		app.MessageElements[mt] = message.ElementsByElementType{}
	}
	if len(element) > 0 {
		app.MessageElements[mt][elt] = element
	} else {
		delete(app.MessageElements[mt], elt)
	}
}

// DeletePageParameter delete a page parameter value override for the given app and page type
func (app *App) DeletePageParameter(t pagetype.Type, n param.Name) {
	if app.PageParameters == nil {
		return
	}
	if _, found := app.PageParameters[t]; !found {
		return
	}
	delete(app.PageParameters[t], n)
}

// SetPageParameter sets a page parameter value override for the given app and page type
func (app *App) SetPageParameter(pt pagetype.Type, pn param.Name, value string) {
	if app.PageParameters == nil {
		app.PageParameters = pageparams.ParameterByNameByPageType{}
	}
	if _, found := app.PageParameters[pt]; !found {
		app.PageParameters[pt] = pageparams.ParameterByName{}
	}
	app.PageParameters[pt][pn] = param.MakeParameter(pn, value)
}

// MakeElementGetter returns a function with the signature of ElementGetter
// that will return the appropriate message element override if one exists
// for the MessageType, App, and ElementType, or fall back to the default
// element for the Message and ElementType
func (app App) MakeElementGetter(mt message.MessageType) message.ElementGetter {
	return func(elt message.ElementType) string {
		element, ok := app.MessageElements[mt][elt]
		if ok {
			return element
		}

		return message.MakeElementGetter(mt)(elt)
	}
}

// Policy determines which apps or providers should be primary/active,
// and which should be followers. It will eventually contain
// more complex rules for how to route users based on settings and liveness.
// For now, it's very simple: just mark 1 provider as active and the rest
// are implied to be followers.
type Policy struct {
	ActiveProviderID uuid.UUID `yaml:"active_provider_id" json:"active_provider_id" validate:"notnil"`
}

//go:generate genvalidate Policy

func (pm *PlexMap) hasEmployeeSettings() bool {
	return pm.EmployeeProvider != nil && pm.EmployeeApp != nil
}

func (pm *PlexMap) validateTenantSettings() error {
	activeProviderFound := false
	for _, p := range pm.Providers {
		if err := p.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
		if p.ID == pm.Policy.ActiveProviderID {
			activeProviderFound = true
		}
	}

	if !activeProviderFound {
		return ucerr.Errorf("active provider (ID: %v) not found in provider list", pm.Policy.ActiveProviderID)
	}

	for _, app := range pm.Apps {
		if err := app.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
		for _, provAppID := range app.ProviderAppIDs {
			// TODO: Can validate that there aren't duplicates,
			// or that there is at most 1 app ID from a given provider, etc.
			if err := pm.ValidateProviderAppID(provAppID); err != nil {
				return ucerr.Wrap(err)
			}
		}
	}

	return nil
}

func (pm *PlexMap) validateEmployeeSettings() error {
	if pm.EmployeeProvider == nil && pm.EmployeeApp == nil {
		return nil
	}

	if !pm.hasEmployeeSettings() {
		return ucerr.New("EmployeeApp and EmployeeProvider must both be configured")
	}

	if pm.EmployeeProvider.Type != ProviderTypeEmployee {
		return ucerr.Errorf("EmployeeProvider must be of type %s", ProviderTypeEmployee)
	}

	if err := pm.EmployeeProvider.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if len(pm.EmployeeProvider.UC.Apps) != 1 {
		return ucerr.New("EmployeeProvider must have exactly one UCApp")
	}

	if err := pm.EmployeeApp.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if len(pm.EmployeeApp.ProviderAppIDs) != 1 {
		return ucerr.New("EmployeeApp must have exactly one ProviderAppID")
	}

	if err := pm.EmployeeProvider.ValidateAppID(pm.EmployeeApp.ProviderAppIDs[0]); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// Validate implements Validateable
func (pm *PlexMap) Validate() error {
	if pm.EmailServer.IsConfigured() {
		if err := pm.EmailServer.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}

	if err := pm.TelephonyProvider.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if err := pm.validateTenantSettings(); err != nil {
		return ucerr.Wrap(err)
	}

	if err := pm.validateEmployeeSettings(); err != nil {
		return ucerr.Wrap(err)
	}

	// validate that there aren't duplicate ClientIDs
	plexCIDs := set.NewStringSet()
	for _, a := range pm.Apps {
		if plexCIDs.Contains(a.ClientID) {
			return ucerr.Friendlyf(nil, "found duplicate clientID %v", a.ClientID)
		}

		plexCIDs.Insert(a.ClientID)
	}
	if pm.hasEmployeeSettings() && plexCIDs.Contains(pm.EmployeeApp.ClientID) {
		return ucerr.Friendlyf(nil, "found duplicate clientID %v", pm.EmployeeApp.ClientID)
	}

	return nil
}

// ValidateProviderAppID checks if the given provider app ID is a valid
// app/client in any underlying provider.
func (pm *PlexMap) ValidateProviderAppID(appID uuid.UUID) error {
	for i := range pm.Providers {
		if pm.Providers[i].ValidateAppID(appID) == nil {
			return nil
		}
	}
	return ucerr.Friendlyf(nil, "app ID %v not found in providers %v", appID, pm.Providers)
}

// FindAppForClientID returns the Plex App config for a given Plex ClientID.
func (pm PlexMap) FindAppForClientID(plexClientID string) (*App, *Policy, error) {
	if pm.hasEmployeeSettings() && pm.EmployeeApp.ClientID == plexClientID {
		policy := Policy{ActiveProviderID: pm.EmployeeProvider.ID}
		return pm.EmployeeApp, &policy, nil
	}

	for _, app := range pm.Apps {
		if app.ClientID == plexClientID {
			return &app, &pm.Policy, nil
		}
	}

	// TODO: support searching provider apps too?
	return nil, nil, ucerr.Friendlyf(nil, "no plex app with Plex client ID '%s' found", plexClientID)
}

// FindProviderForAppID returns the Provider which contains a provider-specific
// app config with a given App ID.
func (pm PlexMap) FindProviderForAppID(providerAppID uuid.UUID) (*Provider, error) {
	if pm.hasEmployeeSettings() && pm.EmployeeProvider.UC.Apps[0].ID == providerAppID {
		return pm.EmployeeProvider, nil
	}

	for i := range pm.Providers {
		if pm.Providers[i].ValidateAppID(providerAppID) == nil {
			return &pm.Providers[i], nil
		}
	}

	return nil, ucerr.Friendlyf(nil, "no provider with app ID '%s' found", providerAppID)
}

// GetActiveProvider returns the Provider matching the active policy, or an error
func (pm PlexMap) GetActiveProvider() (*Provider, error) {
	for i, p := range pm.Providers {
		if pm.Policy.ActiveProviderID == p.ID {
			return &pm.Providers[i], nil
		}
	}
	return nil, ucerr.Friendlyf(nil, "no provider with policy active ID: %s", pm.Policy.ActiveProviderID)
}

// ListFollowerProviders returns a list of all providers that are not the active provider
// doesn't return an error because the default is just an empty list
func (pm PlexMap) ListFollowerProviders() []Provider {
	var fs []Provider
	for i, p := range pm.Providers {
		if pm.Policy.ActiveProviderID != p.ID {
			fs = append(fs, pm.Providers[i])
		}
	}

	return fs
}

// applyReadOnlySettings merges read-only settings from a source PlexMap into the target
func (pm *PlexMap) applyReadOnlySettings(source PlexMap) {
	// restore employee provider
	pm.EmployeeProvider = source.EmployeeProvider

	// preserve the source employee app settings if either the source or target
	// are not configured - they should either both be configured or neither be
	// configured, so this is a defensive check
	if source.EmployeeApp == nil || pm.EmployeeApp == nil {
		pm.EmployeeApp = source.EmployeeApp
	} else {
		// preserve the following fields but take all other settings from the target
		pm.EmployeeApp.ID = source.EmployeeApp.ID
		pm.EmployeeApp.Name = source.EmployeeApp.Name
		pm.EmployeeApp.ProviderAppIDs = append(pm.EmployeeApp.ProviderAppIDs, source.EmployeeApp.ProviderAppIDs...)
		pm.EmployeeApp.GrantTypes = append(pm.EmployeeApp.GrantTypes, source.EmployeeApp.GrantTypes...)
		pm.EmployeeApp.SyncedFromProvider = source.EmployeeApp.SyncedFromProvider
	}

	// apply app page parameters and message elements
	for _, sourceApp := range source.Apps {
		for i, targetApp := range pm.Apps {
			if targetApp.ID != sourceApp.ID {
				continue
			}

			for pt, parametersByName := range sourceApp.PageParameters {
				for _, p := range parametersByName {
					pm.Apps[i].SetPageParameter(pt, p.Name, p.Value)
				}
			}

			for mt, elementsByElementType := range sourceApp.MessageElements {
				for elt, element := range elementsByElementType {
					pm.Apps[i].CustomizeMessageElement(mt, elt, element)
				}
			}
		}
	}

	// for any existing SAML trusted SPs, preserve existing data since the
	// client (console) really only deals with MetadataURL (EntityID) right now
	for _, sourceApp := range source.Apps {
		for i, targetApp := range pm.Apps {
			if targetApp.ID != sourceApp.ID {
				continue
			}

			if sourceApp.SAMLIDP == nil || targetApp.SAMLIDP == nil {
				continue
			}

			for _, sourceSP := range sourceApp.SAMLIDP.TrustedServiceProviders {
				for j, targetSP := range targetApp.SAMLIDP.TrustedServiceProviders {
					if targetSP.EntityID == sourceSP.EntityID {
						pm.Apps[i].SAMLIDP.TrustedServiceProviders[j] = sourceSP
					}
				}
			}
		}
	}
}

// filterReadOnlySettings removes parts of the PlexMap that are read-only
func (pm *PlexMap) filterReadOnlySettings() {
	// clear employee provider
	pm.EmployeeProvider = nil

	// clear read-only fields of employee app
	if pm.EmployeeApp != nil {
		pm.EmployeeApp.MessageElements = message.ElementsByElementTypeByMessageType{}
		pm.EmployeeApp.PageParameters = pageparams.ParameterByNameByPageType{}
		pm.EmployeeApp.ProviderAppIDs = []uuid.UUID{}
		pm.EmployeeApp.GrantTypes = GrantTypes{}
		pm.EmployeeApp.SyncedFromProvider = uuid.Nil
	}

	// clear app page parameters and message elements
	for i := range pm.Apps {
		pm.Apps[i].PageParameters = pageparams.ParameterByNameByPageType{}
		pm.Apps[i].MessageElements = message.ElementsByElementTypeByMessageType{}
	}
}

// GetEmailClient either returns the default email client or per app email client
// TODO possibly add caching otherwise we make extra calls to resolve the secret
func (pm PlexMap) GetEmailClient(defaultClient email.Client) email.Client {
	if pm.EmailServer.IsConfigured() {
		return pm.EmailServer.NewClient()
	}
	return defaultClient
}

func (p *Provider) extraValidate() error {
	switch p.Type {
	case ProviderTypeAuth0:
		if p.Auth0 == nil {
			return ucerr.New("Auth0 config is required for Auth0 provider")
		}
	case ProviderTypeEmployee:
		// no config required for this
	case ProviderTypeUC:
		if p.UC == nil {
			return ucerr.New("UC config is required for UC provider")
		}
	case ProviderTypeCognito:
		if p.Cognito == nil {
			return ucerr.New("Cognito config is required for Cognito provider")
		}
	default:
		return ucerr.Friendlyf(nil, "unrecognized provider.Type %s", p.Type)
	}
	return nil
}

// ValidateAppID checks if the given app ID is defined in this provider.
func (p *Provider) ValidateAppID(providerAppID uuid.UUID) error {
	switch p.Type {
	case ProviderTypeAuth0:
		for _, a := range p.Auth0.Apps {
			if providerAppID == a.ID {
				return nil
			}
		}
	case ProviderTypeEmployee, ProviderTypeUC:
		for _, a := range p.UC.Apps {
			if providerAppID == a.ID {
				return nil
			}
		}
	case ProviderTypeCognito:
		for _, a := range p.Cognito.Apps {
			if providerAppID == a.ID {
				return nil
			}
		}
	default:
		return ucerr.Friendlyf(nil, "unrecognized provider.Type %s", p.Type)
	}
	return ucerr.Friendlyf(nil, "provider app ID %s not found in provider '%s' (%s)", providerAppID, p.Name, p.ID)
}
