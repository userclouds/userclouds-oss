package acme

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"userclouds.com/infra/acme"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/acmestorage"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/testkeys"
	"userclouds.com/test/testlogtransport"
)

func TestDebounce(t *testing.T) {
	ctx := context.Background()
	testlogtransport.InitLoggerAndTransportsForTests(t)

	tdb := testdb.New(t, migrate.NewTestSchema(tenantdb.Schema))

	order := &acmestorage.Order{
		BaseModel: ucdb.NewBase(),
		Status:    acmestorage.OrderStatusPending,
		Token:     "testtoken",
	}

	s := acmestorage.New(tdb)
	assert.NoErr(t, s.SaveOrder(ctx, order))

	ctx = multitenant.SetTenantState(ctx, &tenantmap.TenantState{TenantDB: tdb})

	cfg := &acme.Config{
		DirectoryURL: "unused",
		AccountURL:   "unused",
		PrivateKey:   testkeys.Config.PrivateKey,
	}
	qc, err := workerclient.NewClientFromConfig(ctx, &workerclient.Config{Type: workerclient.TypeTest})
	assert.NoErr(t, err)
	h := NewHandler(cfg, qc)
	rr := httptest.NewRecorder()

	// NB: this URL would normally be /.well-known/acme-challenge/testtoken, but we are testing
	// the acme-challenge handler directly we just need the remainder of the path
	r := httptest.NewRequest(http.MethodGet, "/testtoken", nil).WithContext(ctx)

	h.ServeHTTP(rr, r)
	assert.Equal(t, rr.Code, http.StatusOK)

	bs, err := io.ReadAll(rr.Body)
	assert.NoErr(t, err)
	assert.Equal(t, string(bs), "testtoken.scmo4HYdhOMaadwjmmBqjP5qesWfUNkXugua9jdFh6k")
}
func TestDebounceNoWorkerClient(t *testing.T) {
	ctx := context.Background()
	ctx = multitenant.SetTenantState(ctx, &tenantmap.TenantState{TenantDB: testdb.New(t, migrate.NewTestSchema(tenantdb.Schema))})
	cfg := &acme.Config{DirectoryURL: "unused", AccountURL: "unused", PrivateKey: testkeys.Config.PrivateKey}
	h := NewHandler(cfg, nil)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/testtoken", nil).WithContext(ctx)
	h.ServeHTTP(rr, r)
	assert.Equal(t, rr.Code, http.StatusNotImplemented)
}

func TestThumbprint(t *testing.T) {
	ctx := context.Background()
	cfg := &acme.Config{
		PrivateKey: testkeys.Config.PrivateKey,
	}
	pk, err := cfg.PrivateKey.Resolve(ctx)
	assert.NoErr(t, err)

	tb, err := acme.ComputeThumbprint(ctx, pk)
	assert.NoErr(t, err)
	assert.Equal(t, tb, "scmo4HYdhOMaadwjmmBqjP5qesWfUNkXugua9jdFh6k")
}
