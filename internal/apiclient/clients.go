package apiclient

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/security"
	"userclouds.com/internal/tenantcache"
)

type optionsAuthClient struct {
	useTenantState       bool
	tenantID             uuid.UUID
	tenantURL            string
	companyConfigStorage *companyconfig.Storage
	clientCacheConfig    *cache.Config
	authzOptions         []authz.Option
}

// OptionsAuthClient makes clients.optionsAuthClient extensible
type OptionsAuthClient interface {
	apply(*optionsAuthClient)
}

type optFunc func(*optionsAuthClient)

func (o optFunc) apply(opts *optionsAuthClient) {
	o(opts)
}

// UseTenantState true means that the client config should be retrieved from the context
func UseTenantState() OptionsAuthClient {
	return optFunc(func(opts *optionsAuthClient) {
		opts.useTenantState = true
	})
}

// TenantConfig sets ID, URL of the tenant in cases where tenant state is not available and uses DB to read tenant cache config
func TenantConfig(tenantID uuid.UUID, tenantURL string, s *companyconfig.Storage) OptionsAuthClient {
	return optFunc(func(opts *optionsAuthClient) {
		opts.tenantID = tenantID
		opts.tenantURL = tenantURL
		opts.companyConfigStorage = s
	})
}

// ClientCacheConfig sets the redis config for the client cache
func ClientCacheConfig(cfg *cache.Config) OptionsAuthClient {
	return optFunc(func(opts *optionsAuthClient) {
		opts.clientCacheConfig = cfg
	})
}

// AuthZ is a wrapper around authz.Option
func AuthZ(opt ...authz.Option) OptionsAuthClient {
	return optFunc(func(opts *optionsAuthClient) {
		opts.authzOptions = append(opts.authzOptions, opt...)
	})
}

// NewAuthzClientFromTenantStateWithPassthroughAuth constructs a new authz client from tenant state
func NewAuthzClientFromTenantStateWithPassthroughAuth(ctx context.Context, opts ...OptionsAuthClient) (*authz.Client, error) {
	opts = append(opts,
		UseTenantState(),
		AuthZ(authz.PassthroughAuthorization()),
	)
	return NewAuthzClient(ctx, opts...)
}

// NewAuthzClientWithTokenSource constructs a new authz client from companyconfig DB state
func NewAuthzClientWithTokenSource(ctx context.Context, s *companyconfig.Storage, tenantID uuid.UUID, tenantURL string, tokenSource jsonclient.Option,
	opts ...OptionsAuthClient) (*authz.Client, error) {
	opts = append(opts,
		TenantConfig(tenantID, tenantURL, s),
		AuthZ(authz.JSONClient(tokenSource)),
	)
	return NewAuthzClient(ctx, opts...)
}

// NewAuthzClientFromTenantStateWithClientSecret constructs a new authz client from tenant state given client credentials
func NewAuthzClientFromTenantStateWithClientSecret(ctx context.Context, clientID string, clientSecret secret.String, opts ...OptionsAuthClient) (*authz.Client, error) {
	cs, err := clientSecret.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	ts := multitenant.GetTenantState(ctx)
	if ts == nil {
		return nil, ucerr.Errorf("tenant state not found in context")
	}
	tokenSource, err := jsonclient.ClientCredentialsForURL(ts.GetTenantURL(), clientID, cs, nil)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	opts = append(opts,
		UseTenantState(),
		AuthZ(authz.JSONClient(tokenSource)),
	)
	return NewAuthzClient(ctx, opts...)
}

// NewAuthzClient constructs a new authz client
func NewAuthzClient(ctx context.Context, opts ...OptionsAuthClient) (*authz.Client, error) {
	var options optionsAuthClient
	for _, opt := range opts {
		opt.apply(&options)
	}
	var cp cache.Provider
	var tenantURL string
	var tenantID uuid.UUID
	var err error

	// Get tenant state from context (used in services that use multitenant middleware)
	if options.useTenantState {
		ts := multitenant.MustGetTenantState(ctx)
		uclog.Verbosef(ctx, "Creating cache client for %v using tenant state config %v", ts.ID, ts)
		tenantURL = ts.GetTenantURL()
		tenantID = ts.ID
		cp, err = ts.GetCacheProvider(ctx)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
	} else if options.tenantURL != "" {
		tenantURL = options.tenantURL
		tenantID = options.tenantID
		uclog.Verbosef(ctx, "Creating cache client for %v using cache config %v", tenantID, options.clientCacheConfig)
		cp, err = tenantcache.Connect(ctx, options.clientCacheConfig, tenantID)
		if err != nil {
			// log the error but don't fail
			uclog.Errorf(ctx, "failed to connect to client cache: %v", err)
			cp = nil
			err = nil
		}
	} else {
		return nil, ucerr.New("no tenant state or company config storage provided")
	}

	if options.authzOptions == nil {
		options.authzOptions = make([]authz.Option, 0)
	}
	options.authzOptions = append(options.authzOptions, authz.CacheProvider(cp))
	options.authzOptions = append(options.authzOptions, authz.TenantID(tenantID))
	options.authzOptions = append(options.authzOptions, authz.JSONClient(security.PassXForwardedFor()))
	azc, err := authz.NewCustomClient(authz.DefaultObjTypeTTL, authz.DefaultEdgeTypeTTL, authz.DefaultObjTTL, authz.DefaultEdgeTTL,
		tenantURL, options.authzOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// TODO if PassthroughAuthorization is specified we don't cache the client but if it has a token source we can cache it
	return azc, nil
}
