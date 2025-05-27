package main

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantdb"
)

func main() {
	ctx := context.Background()

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "queryrunner")
	defer logtransports.Close()

	if len(os.Args) < 2 {
		uclog.Debugf(ctx, "Usage: queryrunner [query]")
		uclog.Debugf(ctx, "UC_UNIVERSE and UC_REGION environment variables must be set")
		uclog.Fatalf(ctx, "Expected a query arg, instead got %d: %v", len(os.Args), os.Args)
	}

	q := os.Args[1]
	if !strings.HasPrefix(strings.ToUpper(q), "SELECT") && !universe.Current().IsDev() {
		uclog.Fatalf(ctx, "Only SELECT queries are supported outside of dev at this time.")
	}

	ccs := cmdline.GetCompanyStorage(ctx)

	pager, err := companyconfig.NewTenantPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit),
	)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to apply pagination options: %v", err)
	}

	tenantsChan := make(chan uuid.UUID)
	allDone := sync.WaitGroup{}

	for i := range 10 {
		go worker(ctx, ccs, q, i, tenantsChan, &allDone)
	}

	for {
		tenants, pr, err := ccs.ListTenantsPaginated(ctx, *pager)
		if err != nil {
			uclog.Fatalf(ctx, "Failed to list tenants: %v", err)
		}

		for _, ten := range tenants {
			allDone.Add(1)
			tenantsChan <- ten.ID
		}

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	allDone.Wait()
}

func worker(ctx context.Context, ccs *companyconfig.Storage, q string, workerID int, tenantsChan chan uuid.UUID, allDone *sync.WaitGroup) {
	for {
		select {
		case <-ctx.Done():
			return
		case tenantID := <-tenantsChan:
			queryTenant(ctx, ccs, q, workerID, tenantID)
			allDone.Done()
		}
	}
}

func queryTenant(ctx context.Context, ccs *companyconfig.Storage, q string, workerID int, tenantID uuid.UUID) {
	tdb, _, _, err := tenantdb.Connect(ctx, ccs, tenantID)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to connect to tenant %s: %v", tenantID, err)
	}

	rows, err := tdb.QueryContext(ctx, "QueryRunner", q)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to run query on tenant %v: %v", tenantID, err)
	}

	// figure out how many columns in the results
	cols, err := rows.Columns()
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get column names on tenant %v: %v", tenantID, err)
	}

	for rows.Next() {
		// TODO (sgarrity 8/23): there is probably a better way to do this, but we don't
		// know result types (luckily strings will always work), and we need allocated
		// space to scan them into
		data := make([]string, len(cols))
		targets := make([]any, len(cols))
		for i := range cols {
			targets[i] = &data[i]
		}

		err = rows.Scan(targets...)
		if err != nil {
			uclog.Fatalf(ctx, "Failed to scan query results on %v: %v", tenantID, err)
		}

		uclog.Debugf(ctx, "Tenant %s (from %d): %v", tenantID, workerID, data)
	}
}
