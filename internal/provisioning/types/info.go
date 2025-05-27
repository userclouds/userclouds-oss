package types

import (
	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/companyconfig"
)

// ProvisionInfo contains the information we pass around to provisioning functions
type ProvisionInfo struct {
	CompanyStorage *companyconfig.Storage
	TenantDB       *ucdb.DB
	LogDB          *ucdb.DB
	CacheCfg       *cache.Config
	TenantID       uuid.UUID
}
