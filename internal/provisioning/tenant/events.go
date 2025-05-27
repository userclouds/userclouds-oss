package tenant

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	idpprov "userclouds.com/idp/provisioning"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/logdb"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/logserver/provisioning"
)

const maxParallelProvisioners = 5

// EventProvisioner is a Provisionable object used to set up metric metadata for all custom events in a tenant.
type EventProvisioner struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	tenantInternal *companyconfig.TenantInternal
}

// Provision implements Provisionable and creates or updates metric metadata for a
// tenant in the tenant's dedicated log DB.
func (t *EventProvisioner) Provision(ctx context.Context) error {
	return ucerr.Wrap(executeProvisioningForCustomEvents(ctx, &t.tenantInternal.TenantDBConfig, &t.tenantInternal.LogConfig.LogDB, t.tenantInternal.ID, []types.ProvisionOperation{types.Provision, types.Validate}))
}

// Validate implements Provisionable and validates metric metadata for a
// tenant in the tenant's dedicated log DB.
func (t *EventProvisioner) Validate(ctx context.Context) error {
	return ucerr.Wrap(executeProvisioningForCustomEvents(ctx, &t.tenantInternal.TenantDBConfig, &t.tenantInternal.LogConfig.LogDB, t.tenantInternal.ID, []types.ProvisionOperation{types.Validate}))
}

// Cleanup implements Provisionable and clenas up metric metadata for a
// tenant in the tenant's dedicated log DB.
func (t *EventProvisioner) Cleanup(ctx context.Context) error {
	return ucerr.Wrap(executeProvisioningForCustomEvents(ctx, &t.tenantInternal.TenantDBConfig, &t.tenantInternal.LogConfig.LogDB, t.tenantInternal.ID, []types.ProvisionOperation{types.Cleanup}))
}

// Because provisioning connections are not reused (instead multitenant cache will open new ones), we need to close them so they don't
// hang around either during running "provision" or in a long running console process
// closeDBConnection closes the DB connection and logs any errors
func closeDBConnection(ctx context.Context, db *ucdb.DB) {
	if err := db.Close(ctx); err != nil {
		uclog.Errorf(ctx, "Failed to close provisioning db connection %v", *db)
	}
}

// initCustomEventProvisioner initializes an event provisioner for custom events
func initCustomEventProvisioner(ctx context.Context, tenantDBCfg *ucdb.Config, logDB *ucdb.DB, tenantID uuid.UUID) (provisioning.EventProvisioner, error) {
	// We expect tenant and log DBs to be fully migrated before we provision the events metadata
	tenantDB, err := ucdb.New(ctx, tenantDBCfg, migrate.SchemaValidator(tenantdb.Schema))
	if err != nil {
		return provisioning.EventProvisioner{}, ucerr.Wrap(err)
	}
	defer closeDBConnection(ctx, tenantDB)
	uclog.Infof(ctx, "Connected to tenant db %v", tenantID)
	mT, err := idpprov.GetAllCustomEvents(ctx, tenantDB, tenantID, nil)
	if err != nil {
		return provisioning.EventProvisioner{}, ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "Found %d custom event types to provision", len(mT))
	provisioner := provisioning.NewEventProvisioner("ProvCustomEventsCMD", logDB, mT)
	return *provisioner, nil
}

// executeProvisioningForCustomEvents executes provisioning operations for custom events
func executeProvisioningForCustomEvents(ctx context.Context, tenantDBCfg *ucdb.Config, logDBCfg *ucdb.Config, tenantID uuid.UUID, ops []types.ProvisionOperation) error {
	logDB, err := ucdb.New(ctx, logDBCfg, migrate.SchemaValidator(logdb.Schema))
	if err != nil {
		return ucerr.Wrap(err)
	}
	defer closeDBConnection(ctx, logDB)

	uclog.Infof(ctx, "Connected to tenant log db %v", tenantID)
	provisioner, err := initCustomEventProvisioner(ctx, tenantDBCfg, logDB, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(provisioner.ExecuteOperations(ctx, ops, "custom"))
}

func getTenantConnectionData(ctx context.Context, companyStorage *companyconfig.Storage, tenantID uuid.UUID) ([]*companyconfig.TenantInternal, error) {
	if !tenantID.IsNil() {
		tenantInternal, err := companyStorage.GetTenantInternal(ctx, tenantID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		return []*companyconfig.TenantInternal{tenantInternal}, nil
	}

	tenants := []companyconfig.Tenant{}
	pager, err := companyconfig.NewTenantPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	for {
		ts, respFields, err := companyStorage.ListTenantsPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		tenants = append(tenants, ts...)

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	tenantsInternal := make([]*companyconfig.TenantInternal, 0, len(tenants))
	for _, t := range tenants {
		tenantInternal, err := companyStorage.GetTenantInternal(ctx, t.ID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		tenantsInternal = append(tenantsInternal, tenantInternal)
	}
	return tenantsInternal, nil
}

// ExecuteProvisioningForEvents provisions the static and custom events for the tenants in the companyconfig DB
func ExecuteProvisioningForEvents(ctx context.Context, companyConfigDBCfg *ucdb.Config, companyStorage *companyconfig.Storage, tenantID uuid.UUID, ops []types.ProvisionOperation) error {
	// First provision the static events in companyconfig DB
	if err := provisioning.ExecuteProvisioningForStaticEvents(ctx, companyConfigDBCfg, ops); err != nil {
		return ucerr.Wrap(err)
	}

	// Get the DB info for the tenants for which we need to do custom events provisioning
	tenantsInternal, err := getTenantConnectionData(ctx, companyStorage, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	uclog.Infof(ctx, "Found %d tenants to %v custom events", len(tenantsInternal), ops)

	// Next provision the custom events in tenant DB by first reading objects from the tenantDB
	// and then verifying that they exist in event_metadata table in log DB

	// Wrap each tenant DB info in a provisioner
	provs := make([]types.Provisionable, 0, len(tenantsInternal))
	for _, ti := range tenantsInternal {
		provs = append(provs, &EventProvisioner{
			Named:          types.NewNamed(fmt.Sprintf("TenantEventProvisioner:%s", ti.ID)),
			Parallelizable: types.NewParallelizable(types.Provision, types.Validate, types.Cleanup),
			tenantInternal: ti,
		})
	}

	// Execute the provisioners in parallel but in batches to control how many new connections we open to the databases
	p, err := types.NewBatchProvisioner("EventsCMD", provs, maxParallelProvisioners)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := p.Execute(ctx, ops); err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "Provisioning for events complete")
	return nil
}
