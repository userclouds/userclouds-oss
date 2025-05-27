package search

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/ucopensearch"
	"userclouds.com/worker"
)

// Enabler is responsible for enabling a search index configured for a tenant, which
// involves updating the state of the search index, creating the external opensearch
// index, and sending the bootstrap task to populating that index.
type Enabler struct {
	ctx          context.Context
	searchClient *ucopensearch.Client
	workerClient workerclient.Client
	ts           *tenantmap.TenantState
	sim          *storage.SearchIndexManager
}

// NewEnabler creates an object capable of enabling a search index
func NewEnabler(
	ctx context.Context,
	searchCfg *ucopensearch.Config,
	workerClient workerclient.Client,
	ts *tenantmap.TenantState,
) (*Enabler, error) {
	s := storage.NewFromTenantState(ctx, ts)
	sim, err := storage.NewSearchIndexManager(ctx, s)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	searchClient, err := ucopensearch.NewClient(ctx, searchCfg)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	enabler := &Enabler{
		ctx:          ctx,
		searchClient: searchClient,
		workerClient: workerClient,
		ts:           ts,
		sim:          sim,
	}

	return enabler, nil
}

// EnableIndex sets up the associated opensearch index, enables the index,
// and initiates a backfill.
func (e *Enabler) EnableIndex(indexID uuid.UUID) error {
	index := e.sim.GetIndexByID(indexID)
	if index == nil {
		return ucerr.Friendlyf(nil, "index id '%v' is unrecognized", indexID)
	}

	definableIndex, err := index.GetDefinableIndex()
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := e.createOpenSearchIndex(definableIndex); err != nil {
		return ucerr.Wrap(err)
	}

	index.Enabled = time.Now().UTC()

	if _, err := e.sim.UpdateIndex(e.ctx, index); err != nil {
		return ucerr.Wrap(err)
	}

	for r := range e.ts.UserRegionDbMap {
		msg := worker.BootstrapTenantOpenSearchIndexMessage(e.ts.ID, indexID, uuid.Nil, r, 0)
		if err := e.workerClient.Send(e.ctx, msg); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

func (e *Enabler) createOpenSearchIndex(index ucopensearch.DefinableIndex) error {
	indexName := index.GetIndexName(e.ts.ID)
	if exists, err := e.searchClient.IndexExists(e.ctx, indexName); err != nil {
		return ucerr.Wrap(err)
	} else if exists {
		uclog.Warningf(e.ctx, "Index '%v' already exists for tenant '%v'", index.GetID(), e.ts.ID)
		return nil
	}
	resp, err := e.searchClient.CreateIndex(e.ctx, indexName, index.GetIndexDefinition())
	if err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(e.ctx, "CreateIndex '%s' response: %s", indexName, resp)
	return nil
}
