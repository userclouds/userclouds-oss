package types

import (
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantplex"
)

// TenantFile data we load from a file in order to provision a tenant
type TenantFile struct {
	Tenant     companyconfig.Tenant    `json:"tenant"`
	PlexConfig tenantplex.TenantConfig `json:"plex_config"`

	// TODO: These only exist for validation because the JSON specifies full
	// URLs, so it's oddly duplicative in the JSON.
	// In Console, these are part of the service's universe-specific Config
	// (`tenant_protocol`, `tenant_sub_domain`) which makes a little more sense.
	Protocol  string `json:"protocol" validate:"notempty"`
	SubDomain string `json:"sub_domain" validate:"notempty"`

	// Optionally the file can specify a different DB configs for where its tenant DB is provisioned
	TenantDBCfg *ucdb.Config `json:"tenant_db" validate:"allownil"`
}

func (t TenantFile) extraValidate() error {
	for _, p := range t.PlexConfig.PlexMap.Providers {
		if p.Type == tenantplex.ProviderTypeUC &&
			p.UC.IDPURL != t.Tenant.TenantURL {
			return ucerr.Errorf("in provisioning, tenant %v URL %s should match plex UC IDP URL %s", t.Tenant.ID, t.Tenant.TenantURL, p.UC.IDPURL)
		}
	}
	return nil
}

//go:generate genvalidate TenantFile
