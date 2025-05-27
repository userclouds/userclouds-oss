package search

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/config"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/worker"
)

const defaultValuesPerBootstrapBatch = 10000

// Bootstrapper is responsible for managing the bootstrapping of an enabled but not
// bootstrapped search index.
type Bootstrapper struct {
	workerClient  workerclient.Client
	tenantState   *tenantmap.TenantState
	searchUpdater *storage.SearchUpdater
	batchSize     int
}

// NewBootstrapper creates an object capable of bootstrapping a search index
func NewBootstrapper(ctx context.Context, searchUpdateConfig *config.SearchUpdateConfig, workerClient workerclient.Client, tenantState *tenantmap.TenantState, r region.DataRegion, batchSize int) (*Bootstrapper, error) {
	s := storage.NewFromTenantState(ctx, tenantState)
	cm, err := storage.NewUserstoreColumnManager(ctx, s)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	sim, err := storage.NewSearchIndexManager(ctx, s)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	regDB, ok := tenantState.UserRegionDbMap[r]
	if !ok {
		return nil, ucerr.Errorf("db for region %s not found", r)
	}
	us := storage.NewUserStorage(ctx, regDB, r, tenantState.ID)

	su, err := storage.NewSearchUpdater(ctx, us, cm, sim, searchUpdateConfig, true)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if batchSize == 0 {
		uclog.Infof(ctx, "Batch size not set, using default value of %v", defaultValuesPerBootstrapBatch)
		batchSize = defaultValuesPerBootstrapBatch
	}
	return &Bootstrapper{
		workerClient:  workerClient,
		tenantState:   tenantState,
		searchUpdater: su,
		batchSize:     batchSize,
	}, nil
}

// BootstrapIndex will populate into the associated opensearch index for an enabled but not
// bootstrapped index the terms for up to bs.batchSize number of values, starting
// after the specified last value that was bootstrapped. If all values have been bootstrapped,
// the associated search index will be marked as bootstrapped. If there remain additional
// values to bootstrap, a task will be enqueued to the worker with the index ID.
func (bs *Bootstrapper) BootstrapIndex(ctx context.Context, indexID, lastBootstrappedValueID uuid.UUID, r region.DataRegion) error {
	lastBootstrappedValueID, err := bs.searchUpdater.ProcessIndex(ctx, indexID, lastBootstrappedValueID, bs.batchSize)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if !lastBootstrappedValueID.IsNil() {
		uclog.Infof(ctx, "sending continue bootstrapping for tenant '%v' index '%v' lastBootstrappedValueID '%v'", bs.tenantState.ID, indexID, lastBootstrappedValueID)
		msg := worker.BootstrapTenantOpenSearchIndexMessage(bs.tenantState.ID, indexID, lastBootstrappedValueID, r, bs.batchSize)
		if err := bs.workerClient.Send(ctx, msg); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
