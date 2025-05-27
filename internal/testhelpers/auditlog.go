package testhelpers

import (
	"context"
	"testing"
	"time"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
)

// CheckAuditLog checks the audit log for a specific event type and username, it retries for up to 10 seconds to account for async writes
func CheckAuditLog(ctx context.Context, t *testing.T, tenantDB *ucdb.DB, et auditlog.EventType, username string, minEventTime time.Time, expectedCount int) []auditlog.Entry {
	als := auditlog.NewStorage(tenantDB)
	startTime := time.Now().UTC()
	for time.Now().UTC().Sub(startTime) < 5*time.Second {
		relevantEntries := make([]auditlog.Entry, 0, expectedCount)

		pager, err := auditlog.NewEntryPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
		assert.NoErr(t, err)

		total := 0
		for {
			es, pr, err := als.ListEntriesPaginated(ctx, *pager)
			assert.NoErr(t, err)
			total += len(es)
			for _, e := range es {
				if !minEventTime.IsZero() && e.Created.Before(minEventTime) {
					continue
				}
				if e.Type == et && e.Actor == username {
					relevantEntries = append(relevantEntries, e)
				}
			}

			if !pager.AdvanceCursor(*pr) {
				break
			}
		}

		if len(relevantEntries) == expectedCount {
			uclog.Debugf(ctx, "Found expected %d relevant %v entries (out of %d entries)", len(relevantEntries), et, total)
			return relevantEntries
		}
		uclog.Debugf(ctx, "Found %d (out of %d) %v entries, waiting for %d", len(relevantEntries), total, et, expectedCount)
		time.Sleep(100 * time.Millisecond)
	}
	assert.FailContinue(t, "Did not find %d expected %v for %v audit log entries", expectedCount, et, username)
	return nil
}
