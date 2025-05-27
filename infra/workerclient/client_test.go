package workerclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/workerclient"
	"userclouds.com/worker"
)

func TestDevClient(t *testing.T) {
	ctx := context.Background()
	sentMsg := worker.CreateCheckTenantCNameMessage(uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()))
	devServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodPost)
		var receivedMsg worker.Message
		assert.NoErr(t, jsonapi.Unmarshal(r, &receivedMsg))
		assert.Equal(t, worker.TaskCheckTenantCNAME, receivedMsg.Task)
		assert.Equal(t, receivedMsg.SourceRegion, region.Current())
	}))
	t.Cleanup(devServer.Close)
	qc := workerclient.NewHTTPClient(devServer.URL)
	assert.NoErr(t, qc.Send(ctx, sentMsg))
}
