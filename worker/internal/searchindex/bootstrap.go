package searchindex

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/config"
	"userclouds.com/idp/search"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/tenantmap"
)

// Bootstrap bootstraps a portion of a tenant's OpenSearch index
func Bootstrap(ctx context.Context, searchUpdateCfg *config.SearchUpdateConfig, workerClient workerclient.Client,
	ts *tenantmap.TenantState, indexID, lastBootstrappedValueID uuid.UUID, r region.DataRegion, batchSize int) error {
	bs, err := search.NewBootstrapper(ctx, searchUpdateCfg, workerClient, ts, r, batchSize)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := bs.BootstrapIndex(ctx, indexID, lastBootstrappedValueID, r); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
