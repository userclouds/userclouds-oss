package types

import (
	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/companyconfig"
)

// ProvisionerState holds a bunch of attributes that are passed around during provisioning
type ProvisionerState struct {
	Simulate           bool
	Deep               bool
	Operation          string
	Target             string
	ResourceType       string
	OwnerUserID        uuid.UUID
	CompanyStorage     *companyconfig.Storage
	CompanyConfigDBCfg *ucdb.Config
	StatusDBCfg        *ucdb.Config
}

// IsTargetAll returns true if the target is "all"
func (ps *ProvisionerState) IsTargetAll() bool {
	return ps.Target == "all"
}

// GetUUIDTarget returns the UUID of the target if there is only one target, otherwise returns uuid.Nil
func (ps *ProvisionerState) GetUUIDTarget() uuid.UUID {
	return uuid.FromStringOrNil(ps.Target)
}
