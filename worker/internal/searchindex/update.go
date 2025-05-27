package searchindex

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/ucopensearch"
	"userclouds.com/worker"
)

const maxRetries = 10
const resendDelay = 20 * time.Second

// Update updates the search index with the given updateData
func Update(ctx context.Context, tenantID uuid.UUID, searchCfg *ucopensearch.Config, updateData []byte, wc workerclient.Client, attempt int) error {
	sc, err := ucopensearch.NewClient(ctx, searchCfg)
	if err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "bulk insert tenant: %v  %d bytes, attempt [%d/%d]", tenantID, len(updateData), attempt, maxRetries)
	resp, err := sc.BulkRequest(ctx, updateData)
	if err != nil {
		if ucopensearch.IsNetworkTimeoutError(err) {
			uclog.Errorf(ctx, "bulk insert [%d bytes] network timeout: %s", len(updateData), err)
			resendUpdate(ctx, tenantID, updateData, wc, attempt)
		}
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "bulk insert (%d bytes) tenant %v attempt %d response: %s", len(updateData), tenantID, attempt, string(resp[:300]))
	return nil
}

func resendUpdate(ctx context.Context, tenantID uuid.UUID, updateData []byte, wc workerclient.Client, attempt int) {
	if attempt > maxRetries {
		uclog.Errorf(ctx, "Failed to update search index after %d attempts (%d)", maxRetries, attempt)
		return
	}
	// This is less than ideal. but we have no other way to to a delayed retry
	// Ideally, we can implement something in the message level that will have a scheduled time for a message
	// and have the worker re-queue messages that are not yet due
	time.Sleep(resendDelay)

	uclog.Infof(ctx, "Resending update to search index for tenant %s, attempt %d", tenantID, attempt)
	msg := worker.CreateUpdateTenantOpenSearchIndexMessage(tenantID, updateData, attempt)
	if err := wc.Send(ctx, msg); err != nil {
		uclog.Errorf(ctx, "Failed to send message to worker %s: '%v'", wc, err)
	}
}
