package api_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/config"
	. "userclouds.com/authz/internal/api"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
	"userclouds.com/test/testlogtransport"
)

func newFakeObjectType(typeName string) *authz.ObjectType {
	return &authz.ObjectType{
		BaseModel: ucdb.NewBase(),
		TypeName:  typeName,
	}
}

func newFakeEdgeType(typeName string, srcOT, tgtOT *authz.ObjectType) *authz.EdgeType {
	return &authz.EdgeType{
		BaseModel:          ucdb.NewBase(),
		TypeName:           typeName,
		SourceObjectTypeID: srcOT.ID,
		TargetObjectTypeID: tgtOT.ID,
	}
}

type testFixture struct {
	h         http.Handler
	jwtHeader string
	tenantDB  *ucdb.DB
	t         *testing.T
}

func newTestFixture(t *testing.T) (*testFixture, context.Context) {
	ctx := context.Background()
	tdb := testdb.New(t, migrate.NewTestSchema(tenantdb.Schema))

	ctx = multitenant.SetTenantState(ctx, &tenantmap.TenantState{TenantDB: tdb})
	jwt := uctest.CreateJWT(t, oidc.UCTokenClaims{}, fmt.Sprintf("http://jerry.%s", testhelpers.TestTenantSubDomain))
	return &testFixture{
		h:         auth.Middleware(uctest.JWTVerifier{}, uuid.Nil).Apply(NewHandler(ctx, nil, config.Config{})),
		jwtHeader: fmt.Sprintf("Bearer %s", jwt),
		tenantDB:  tdb,
		t:         t,
	}, ctx
}

func (tf *testFixture) authzRequest(ctx context.Context, method, path string, reqBody any) (*httptest.ResponseRecorder, string) {
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(method, path, uctest.IOReaderFromJSONStruct(tf.t, reqBody))
	r.Header.Add("Authorization", tf.jwtHeader)
	r = r.WithContext(ctx)
	tf.h.ServeHTTP(rr, r)
	respBody, err := io.ReadAll(rr.Body)
	assert.NoErr(tf.t, err)
	return rr, string(respBody)
}
func TestConflict(t *testing.T) {
	tf, ctx := newTestFixture(t)
	ot := newFakeObjectType("test")
	ot2 := newFakeObjectType(ot.TypeName)

	rr, _ := tf.authzRequest(ctx, http.MethodPost, "/objecttypes", ot)
	assert.Equal(t, rr.Code, http.StatusCreated)

	rr, bs := tf.authzRequest(ctx, http.MethodPost, "/objecttypes", ot2)
	assert.Equal(t, rr.Code, http.StatusConflict)
	assert.Contains(t, strings.TrimSpace(bs), `This object type already exists`)

	alias := "testname"
	o := &authz.Object{
		BaseModel: ucdb.NewBase(),
		TypeID:    ot.ID,
		Alias:     &alias,
	}
	o2 := &authz.Object{
		BaseModel: ucdb.NewBase(),
		TypeID:    ot.ID,
		Alias:     o.Alias,
	}

	rr, _ = tf.authzRequest(ctx, http.MethodPost, "/objects", o)
	assert.Equal(t, rr.Code, http.StatusCreated)

	rr, bs = tf.authzRequest(ctx, http.MethodPost, "/objects", o2)
	assert.Equal(t, rr.Code, http.StatusConflict)
	assert.Contains(t, strings.TrimSpace(bs), `This object already exists`)

	et := newFakeEdgeType("test", ot, ot)
	rr, _ = tf.authzRequest(ctx, http.MethodPost, "/edgetypes", et)
	assert.Equal(t, rr.Code, http.StatusCreated, assert.Must())

	e := &authz.Edge{
		BaseModel:      ucdb.NewBase(),
		EdgeTypeID:     et.ID,
		SourceObjectID: o.ID,
		TargetObjectID: o.ID,
	}
	e2 := &authz.Edge{
		BaseModel:      ucdb.NewBase(),
		EdgeTypeID:     et.ID,
		SourceObjectID: o.ID,
		TargetObjectID: o.ID,
	}

	rr, _ = tf.authzRequest(ctx, http.MethodPost, "/edges", e)
	assert.Equal(t, rr.Code, http.StatusCreated)

	rr, bs = tf.authzRequest(ctx, http.MethodPost, "/edges", e2)
	assert.Equal(t, rr.Code, http.StatusConflict)
	assert.Contains(t, strings.TrimSpace(bs), `This edge already exists`)

}

func TestEmptyBodyObjectType(t *testing.T) {
	ctx := context.Background()
	tdb := testdb.New(t, migrate.NewTestSchema(tenantdb.Schema))

	ctx = multitenant.SetTenantState(ctx, &tenantmap.TenantState{
		TenantDB: tdb,
	})
	h := NewHandler(ctx, nil, config.Config{})

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/objecttypes", nil)
	r = r.WithContext(ctx)
	h.ServeHTTP(rr, r)
	assert.Equal(t, rr.Code, http.StatusBadRequest)
}

func TestEdgeType(t *testing.T) {
	tf, ctx := newTestFixture(t)
	tt := testlogtransport.InitLoggerAndTransportsForTestsWithLevel(t, uclog.LogLevelVerbose)
	ot := newFakeObjectType("test")
	rr, _ := tf.authzRequest(ctx, http.MethodPost, "/objecttypes", ot)
	assert.Equal(t, rr.Code, http.StatusCreated)
	et := newFakeEdgeType("test", ot, ot)

	et2 := newFakeEdgeType(et.TypeName, ot, ot)

	rr, _ = tf.authzRequest(ctx, http.MethodPost, "/edgetypes", et)
	assert.Equal(t, rr.Code, http.StatusCreated)

	rr, bs := tf.authzRequest(ctx, http.MethodPost, "/edgetypes", et2)
	assert.Equal(t, rr.Code, http.StatusConflict)
	assert.Contains(t, strings.TrimSpace(bs), `This edge type already exists`)

	rr, _ = tf.authzRequest(ctx, http.MethodPut, "/edgetypes/"+et.ID.String(), et)
	assert.Equal(t, rr.Code, http.StatusOK)

	et.TypeName = "test1"
	tt.ClearMessages()
	rr, _ = tf.authzRequest(ctx, http.MethodPut, "/edgetypes/"+et.ID.String(), et)
	assert.Equal(t, rr.Code, http.StatusOK)
	tt.AssertLogsContainString("GetContext: SaveEdgeType")

	// No-Op update, we should short-circuit and not save anything to DB and not create a audit log entry
	tt.ClearMessages()
	rr, _ = tf.authzRequest(ctx, http.MethodPut, "/edgetypes/"+et.ID.String(), et)
	assert.Equal(t, rr.Code, http.StatusOK)
	tt.AssertLogsDoesntContainString("GetContext: SaveEdgeType")
	tt.AssertLogsContainString(fmt.Sprintf("edge type %v: no-op update", et.ID))

}
