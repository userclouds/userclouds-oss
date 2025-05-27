package provisioning

import (
	"context"

	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/logserver/client"
	"userclouds.com/logserver/events"
)

func getStaticEventTypes() []client.MetricMetadata {
	allEventTypes := events.GetLogEventTypes()
	allMetrics := make([]client.MetricMetadata, 0, len(allEventTypes))

	for s, e := range allEventTypes {
		allMetrics = append(allMetrics, client.MetricMetadata{
			BaseModel:    ucdb.NewBase(),
			Name:         e.Name,
			StringID:     s,
			Code:         e.Code,
			Service:      e.Service,
			ReferenceURL: e.URL,
			Category:     e.Category,
			Attributes:   client.MetricAttributes{Ignore: e.Ignore, System: true, AnyService: (e.Service == "")},
		})

	}
	return allMetrics
}

func initStaticEventProvisioner(ctx context.Context, companyConfigDBCfg *ucdb.Config, mT []client.MetricMetadata) EventProvisioner {
	// We expect the DB to be fully migrated before we provision the events metadata
	db, err := ucdb.New(ctx, companyConfigDBCfg, migrate.SchemaValidator(companyconfig.Schema))
	if err != nil {
		uclog.Fatalf(ctx, "Couldn't connect to the companyconfig DB %v with error %v", companyConfigDBCfg.DBName, err)
	}
	uclog.Infof(ctx, "Connected to companyconfig db")
	return *NewEventProvisioner("ProvStaticEventsCMD", db, mT)
}

// ExecuteProvisioningForStaticEvents provisions, validates, or cleans up static events
func ExecuteProvisioningForStaticEvents(ctx context.Context, companyConfigDBCfg *ucdb.Config, ops []types.ProvisionOperation) error {
	allMetrics := getStaticEventTypes()
	eP := initStaticEventProvisioner(ctx, companyConfigDBCfg, allMetrics)
	uclog.Infof(ctx, "Found %d system event types in the files to %v", len(allMetrics), ops)
	return ucerr.Wrap(eP.ExecuteOperations(ctx, ops, "static"))
}

// NukeStaticEvents nukes all static events
func NukeStaticEvents(ctx context.Context, companyConfigDBCfg *ucdb.Config) error {
	uv := universe.Current()
	if !uv.IsDev() {
		return ucerr.Errorf("can't nuke in non-dev universes: %v", uv)
	}
	allMetrics := getStaticEventTypes()
	eP := initStaticEventProvisioner(ctx, companyConfigDBCfg, allMetrics)

	if err := eP.Nuke(ctx); err != nil {
		return ucerr.Errorf("Failed to nuke non custom events: %v", err)
	}
	return nil
}
