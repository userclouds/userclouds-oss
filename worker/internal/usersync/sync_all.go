package usersync

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/plex/manager"
	"userclouds.com/plex/worker/usersync"
	"userclouds.com/worker"
	"userclouds.com/worker/storage"
)

// All handles the sync-all message and kicks off any required sync tasks
func All(companyConfigStorage *companyconfig.Storage, tm *tenantmap.StateMap, wc workerclient.Client) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		uclog.SetHandlerName(ctx, "syncall")
		uclog.Debugf(ctx, "syncall called")
		if err := dispatchSyncUsers(ctx, companyConfigStorage, tm, wc); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		jsonapi.Marshal(w, nil) // be explicit about HTTP 200 OK - done with this SQS message
	})
}

func dispatchSyncUsers(ctx context.Context, companyConfigStorage *companyconfig.Storage, tm *tenantmap.StateMap, wc workerclient.Client) error {
	pager, err := companyconfig.NewTenantPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return ucerr.Wrap(err)
	}
	msgCount := 0
	tenantsCount := 0
	// page until we're done
	for {
		tenantsPaged, pr, err := companyConfigStorage.ListTenantsPaginated(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}
		tenantsCount += len(tenantsPaged)
		for _, tenant := range tenantsPaged {
			if !tenant.SyncUsers {
				continue
			}

			uclog.Debugf(ctx, "syncall checking %v", tenant.ID)
			// use the tenant state map to get a cached tenantdb connection
			ts, err := tm.GetTenantStateForID(ctx, tenant.ID)
			if err != nil {
				// don't fail whole sync because of this
				uclog.Errorf(ctx, "failed to connect to tenantdb: %v", err)
				continue
			}

			mgr := manager.NewFromDB(ts.TenantDB, ts.CacheConfig)
			tp, err := mgr.GetTenantPlex(ctx, tenant.ID)
			if err != nil {
				// don't fail whole sync because of this
				uclog.Errorf(ctx, "failed to get tenant plex config: %v", err)
				continue
			}

			// check tenant providers
			// TODO: refactor this to be shared logic with console/internal/api/tenant.go

			// can't sync a single provider (sync to what?)
			if len(tp.PlexConfig.PlexMap.Providers) < 2 {
				continue
			}

			ap, err := tp.PlexConfig.PlexMap.GetActiveProvider()
			if err != nil {
				// don't fail whole sync because of this
				uclog.Errorf(ctx, "failed to get active provider: %v", err)
				continue
			}

			if ap.CanSyncUsers() {
				uclog.Debugf(ctx, "syncall sending message for %v", tenant.ID)
				msg := worker.CreateSyncAllUsersMessage(tenant.ID)
				if err := wc.Send(ctx, msg); err != nil {
					// TODO: should we retry? for now, will just get picked up next time
					// don't break the rest of the syncs because of this, thought
					uclog.Errorf(ctx, "failed to send message to queue: %v", err)
				} else {
					msgCount++
				}
			}
		}

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}
	uclog.Infof(ctx, "syncall sent %d messages for %d tenants", msgCount, tenantsCount)
	return nil

}

// SyncTenantUsers syncs all users for a tenant
func SyncTenantUsers(ctx context.Context, tenantID uuid.UUID, tenantDB *ucdb.DB, cacheCfg *cache.Config) error {
	mgr := manager.NewFromDB(tenantDB, cacheCfg)
	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		return ucerr.Errorf("failed to get tenant plex config: %w", err)
	}
	s := storage.New(tenantDB)
	return ucerr.Wrap(usersync.AllUsers(ctx, tenantID, &tp.PlexConfig, s))
}
