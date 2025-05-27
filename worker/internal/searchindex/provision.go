package searchindex

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/search"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/ucopensearch"
)

// Provision a tenant's OpenSearch index
func Provision(ctx context.Context, searchCfg *ucopensearch.Config, workerClient workerclient.Client, ts *tenantmap.TenantState, indexID uuid.UUID) error {
	e, err := search.NewEnabler(ctx, searchCfg, workerClient, ts)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if err := e.EnableIndex(indexID); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
