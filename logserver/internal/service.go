package internal

import (
	"context"
	"net/http"
	"sync"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
)

type noopValidator struct{}

// Validate implements ucdb.Validator
func (n noopValidator) Validate(_ context.Context, _ *ucdb.DB) error {
	return nil
}

// CreateMigrationHandler returns an http handler function that lists the latest migration per tenant DB
// TODO: this has the potential to be pretty expensive so definitely needs to be authenticated
// TODO: can we unify this with the per-tenant DB handles we already create?
// TODO: this is basically duplicated from idp/internal/authn except for logDB
func CreateMigrationHandler(storage *companyconfig.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		pager, err := companyconfig.NewTenantPaginatorFromOptions(pagination.Limit(5))
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		// We don't always close channels since if we bail out early, we prefer to have a memory leak (which will be handled by GC) than a panic which will happen
		// when trying to write to a closed channel
		errors := make(chan error, 100)
		versionsBatches := make(chan []int, 100)
		done := make(chan bool, 1)
		wg := sync.WaitGroup{}
		for {
			tenants, respFields, err := storage.ListTenantsPaginated(ctx, *pager)
			if err != nil {
				jsonapi.MarshalError(ctx, w, err)
				return
			}
			wg.Add(1)
			go func() {
				versions, err := checkTenantsMigrations(ctx, storage, tenants)
				if err != nil {
					errors <- err
				} else {
					versionsBatches <- versions
				}
				wg.Done()
			}()
			if !pager.AdvanceCursor(*respFields) {
				break
			}
		}
		var allVersions []int
		// Running wg.Wait in a go routine so we can collect results from the versionsBatches channel w/o waiting for all go routines to complete
		go func() {
			wg.Wait()
			done <- true
		}()

		for {
			select {
			case err := <-errors:
				jsonapi.MarshalError(ctx, w, err)
				return
			case versions := <-versionsBatches:
				allVersions = append(allVersions, versions...)
			case <-done:
				uclog.Infof(ctx, "Checked migrations for %v tenants", len(allVersions))
				close(done)
				close(errors)
				close(versionsBatches)
				jsonapi.Marshal(w, allVersions)
				return
			}
		}
	}
}

func checkTenantsMigrations(ctx context.Context, storage *companyconfig.Storage, tenants []companyconfig.Tenant) ([]int, error) {
	var vs []int
	uclog.Infof(ctx, "Check migrations for %v tenants", len(tenants))
	for _, t := range tenants {
		cfg, err := storage.GetTenantInternal(ctx, t.ID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		// NB: we have to use a noopValidator here otherwise we can never return migrations != current codebase
		db, err := ucdb.New(ctx, &cfg.LogConfig.LogDB, noopValidator{})
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		v, err := migrate.GetMaxVersion(ctx, db)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		vs = append(vs, v)

		if err := db.Close(ctx); err != nil {
			uclog.Errorf(ctx, "Failed to close a log DB connection  %v", err)
		}
	}
	return vs, nil
}
