package watchdog

import (
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
)

// SlowProvisionWatchdog checks for tenants that have been in the provisioning state for too long
func SlowProvisionWatchdog(ccs *companyconfig.Storage) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		pager, err := companyconfig.NewTenantPaginatorFromOptions()
		if err != nil {
			uchttp.Error(ctx, w, err, http.StatusInternalServerError)
			return
		}

		for {
			tenants, pr, err := ccs.ListTenantsPaginated(ctx, *pager)
			if err != nil {
				uchttp.Error(ctx, w, err, http.StatusInternalServerError)
				return
			}

			for _, tenant := range tenants {
				// for now, we're just checking for tenants stuck in provisioning
				if tenant.State != companyconfig.TenantStateCreating {
					continue
				}

				if tenant.Created.Add(10 * time.Minute).Before(time.Now().UTC()) {
					err := ucerr.Errorf("Tenant %s has been in provisioning for more than 10 minutes", tenant.ID)
					uclog.Errorf(ctx, "watchdog: %v", err)
					sentry.CaptureException(err)
				}
			}

			if !pager.AdvanceCursor(*pr) {
				break
			}
		}
	})
}
