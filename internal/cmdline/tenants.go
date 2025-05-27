package cmdline

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/manager"
)

// TenantClientInfo is a struct that has info needed to call the APIs for a tenant
type TenantClientInfo struct {
	ID          uuid.UUID
	Name        string
	TenantURL   string
	TokenSource jsonclient.Option
}

// GetTenantByIDOrName takes a string and tries to parse it to UUID if it succeeds it tries to load tenant by that ID,
// otherwise, it treats the string as a case insensitive tenant name and tries to load the tenant by that name
func GetTenantByIDOrName(ctx context.Context, s *companyconfig.Storage, idOrName string) (*companyconfig.Tenant, error) {
	tenantID, err := uuid.FromString(idOrName)
	if err == nil {
		// tenant ID
		tenant, err := s.GetTenant(ctx, tenantID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		return tenant, nil
	}

	tenants, err := s.ListTenantsByName(ctx, idOrName)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if len(tenants) == 0 {
		return nil, ucerr.Errorf("tenant name: %v doesn't exist", idOrName)
	}
	if len(tenants) > 1 {
		return nil, ucerr.Errorf("tenant name: %v is not unique, multiple tenants returned (%v)", idOrName, len(tenants))
	}
	return &tenants[0], nil
}

// GetTenantInternalByID retrieves the passed in TenantInternal structure for the tenant id
func GetTenantInternalByID(ctx context.Context, s *companyconfig.Storage, id uuid.UUID) (*companyconfig.TenantInternal, error) {
	tenantInternal, err := s.GetTenantInternal(ctx, id)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return tenantInternal, nil
}

// GetAllTenantIDs retrieves all tenant IDs
func GetAllTenantIDs(ctx context.Context, companyStorage *companyconfig.Storage) ([]uuid.UUID, error) {
	tenantIDs := make([]uuid.UUID, 0)
	pager, err := companyconfig.NewTenantInternalPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	for {
		tis, respFields, err := companyStorage.ListTenantInternalsPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		for _, ti := range tis {
			tenantIDs = append(tenantIDs, ti.ID)

		}
		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}
	return tenantIDs, nil
}

// GetTenantURL returns the tenant URL with the region added if useRegionalURLs is true
func GetTenantURL(ctx context.Context, tenantURL string, useRegionalURLs, useEKS bool) (string, error) {
	if !(useRegionalURLs || useEKS) {
		return tenantURL, nil
	}
	if !strings.HasSuffix(tenantURL, ".userclouds.com") {
		return "", ucerr.Errorf("Can't add region to a non *.userclouds.com url: %v", tenantURL)

	}
	if !companyconfig.CanUseRegionWithTenantURL(tenantURL) {
		return "", ucerr.Errorf("Can't find '.tenant.' in tenant url (%v), can't add region", tenantURL)
	}
	return companyconfig.GetTenantRegionalURL(tenantURL, region.Current(), useEKS), nil
}

// GetTokenSourceForTenant returns a token source for the tenant
func GetTokenSourceForTenant(ctx context.Context, storage *companyconfig.Storage, tenant *companyconfig.Tenant, tenantURL string) (jsonclient.Option, error) {
	tenantDB, _, _, err := tenantdb.Connect(ctx, storage, tenant.ID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	defer func() {
		if err := tenantDB.Close(ctx); err != nil {
			uclog.Warningf(ctx, "error closing tenantDB: %v", err)
		}
	}()
	mgr := manager.NewFromDB(tenantDB, nil)
	plex, err := mgr.GetTenantPlex(ctx, tenant.ID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if len(plex.PlexConfig.PlexMap.Apps) < 1 {
		return nil, ucerr.Errorf("No login apps defined for tenant id %v [%v] at %v", tenant.Name, tenant.ID, tenant.TenantURL)
	}
	app, err := getDefaultApp(ctx, tenant, plex)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	clientSecret, err := app.ClientSecret.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	tokenSource, err := jsonclient.ClientCredentialsForURL(tenantURL, app.ClientID, clientSecret, nil)
	if err != nil {
		return nil, ucerr.Wrap(err)

	}
	return tokenSource, nil
}

func getDefaultApp(ctx context.Context, tenant *companyconfig.Tenant, plex *tenantplex.TenantPlex) (tenantplex.App, error) {
	if len(plex.PlexConfig.PlexMap.Apps) < 1 {
		return tenantplex.App{}, ucerr.Errorf("No login apps defined for tenant id %v [%v] at %v", tenant.Name, tenant.ID, tenant.TenantURL)
	}
	for _, app := range plex.PlexConfig.PlexMap.Apps {
		if strings.Contains(app.Name, "Default App") {
			return app, nil
		}
	}
	app := plex.PlexConfig.PlexMap.Apps[0]
	uclog.Warningf(ctx, "Can't find default login app for tenant: %v [%v] using: %v", tenant.Name, tenant.ID, app.Name)
	return app, nil
}
