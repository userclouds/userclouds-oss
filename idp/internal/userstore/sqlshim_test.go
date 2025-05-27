package userstore_test

import (
	"context"
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"userclouds.com/idp/helpers"
	"userclouds.com/idp/internal"
	"userclouds.com/idp/internal/sqlshim"
	"userclouds.com/idp/internal/sqlshim/msqlshim"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/userstore"
	"userclouds.com/infra/assert"
	cacheTestHelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	sqlshimtypes "userclouds.com/internal/sqlshim"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
	"userclouds.com/tools/generate/genschemas"
)

func TestSqlShim(t *testing.T) {
	ctx := context.Background()

	company, tenant, ccs, tenantDB, _, tm := testhelpers.CreateTestServer(ctx, t)
	ts := tenantmap.NewTenantState(tenant, company,
		uctest.MustParseURL(tenant.TenantURL), tenantDB, nil, nil,
		region.DefaultUserDataRegionForUniverse(universe.Current()), ccs, false, nil, cacheTestHelpers.NewRedisConfigForTests())
	ctx = multitenant.SetTenantState(ctx, ts)

	jwtVerifier := uctest.JWTVerifier{}
	wc := workerclient.NewTestClient()
	// need Mysql to be running for this test (Yext uses mysql not postgres)
	_, _, name, dbPort := genschemas.StartTemporaryMysql(ctx, "mysql-proxy-test", 336)
	defer func() {
		assert.NoErr(t, exec.Command("docker", "rm", "-f", name).Run())
	}()

	proxyPort := 12300 + rand.Intn(100)

	// create the test database
	db, err := sqlx.Open("mysql", fmt.Sprintf("root:mysecretpassword@tcp(127.0.0.1:%d)/sys", dbPort))
	assert.NoErr(t, err)
	_, err = db.Exec("CREATE DATABASE test;")
	assert.NoErr(t, err)
	assert.NoErr(t, db.Close())

	// set up the test database (we need a new connection to that db specifically)
	db, err = sqlx.Open("mysql", fmt.Sprintf("root:mysecretpassword@tcp(127.0.0.1:%d)/test", dbPort))
	assert.NoErr(t, err)
	db.MustExec("CREATE TABLE test (id INT PRIMARY KEY, name VARCHAR(255))")
	db.MustExec("INSERT INTO test (id, name) VALUES (1, 'test')")
	db.MustExec("INSERT INTO test (id, name) VALUES (2, 'test2')")
	assert.NoErr(t, db.Close())

	// create the proxy data structures
	dbID := uuid.Must(uuid.NewV4())
	assert.NoErr(t, ccs.SaveSQLShimProxy(ctx, &companyconfig.SQLShimProxy{
		BaseModel:  ucdb.NewBase(),
		Host:       "127.0.0.1",
		Port:       proxyPort,
		TenantID:   tenant.ID,
		DatabaseID: dbID,
	}))
	s := storage.New(ctx, tenantDB, tenant.ID, nil)
	assert.NoErr(t, s.SaveSQLShimDatabase(ctx, &storage.SQLShimDatabase{
		BaseModel: ucdb.NewBaseWithID(dbID),
		Name:      "test",
		Type:      sqlshimtypes.DatabaseTypeMySQL,
		Host:      "127.0.0.1",
		Port:      dbPort,
		Username:  "root",
		Password:  secret.NewTestString("mysecretpassword"),
		Schemas:   pq.StringArray{"test"},
	}))

	// update the schema after creating it above
	assert.NoErr(t, helpers.IngestSqlshimDatabaseSchemas(ctx, ts, dbID))

	// start the actual proxy process
	proxy := sqlshim.NewProxy(proxyPort,
		tm,
		nil,
		jwtVerifier,
		ccs,
		wc,
		nil,
		msqlshim.ConnectionFactory{},
		userstore.ProxyHandlerFactory{})
	assert.NoErr(t, proxy.Start(ctx))

	// connect to the proxy
	dsn := fmt.Sprintf("root:mysecretpassword@tcp(127.0.0.1:%d)/test?parseTime=true", proxyPort)
	db, err = sqlx.Open("mysql", dsn)
	assert.NoErr(t, err)
	defer func() {
		assert.NoErr(t, db.Close())
	}()

	// test random query that broke once (go-mysql-server 1.10 to 1.11)
	_, err = db.ExecContext(ctx, "SET NAMES utf8mb4")
	assert.NoErr(t, err)

	entries := getAuditLogEntries(ctx, t, tenantDB)
	assert.Equal(t, len(entries), 1, assert.Must())
	assert.Equal(t, entries[0].Type, internal.AuditLogEventTypeSqlshimUnhandledQuery)
	assert.Equal(t, entries[0].Payload["Reason"], "did not parse query")

	// test a normal query
	var res []int
	assert.NoErr(t, db.SelectContext(ctx, &res, "SELECT id FROM test WHERE id IN (1)"))
	assert.Equal(t, res, []int{1})

	entries = getAuditLogEntries(ctx, t, tenantDB)
	assert.Equal(t, len(entries), 4, assert.Must())
	assert.Equal(t, entries[3].Type, internal.AuditLogEventTypeExecuteAccessor)
	assert.Equal(t, entries[3].Payload["Name"], "SELECT_id_FROM_testtest_WHERE_id")

	accessor, err := s.GetAccessorByName(ctx, "SELECT_id_FROM_testtest_WHERE_id")
	assert.NoErr(t, err)
	assert.Equal(t, accessor.Name, "SELECT_id_FROM_testtest_WHERE_id")

	// test a very long query
	wq := strings.Repeat("id = 1 AND ", 50)
	wq = wq[:len(wq)-5] // remove the trailing AND
	assert.NoErr(t, db.SelectContext(ctx, &res, "SELECT id FROM test WHERE "+wq))
	assert.NoErr(t, err)
	assert.Equal(t, res, []int{1})

	entries = getAuditLogEntries(ctx, t, tenantDB)
	assert.Equal(t, len(entries), 7, assert.Must())
	assert.Equal(t, entries[6].Type, internal.AuditLogEventTypeExecuteAccessor)
	assert.Equal(t, entries[6].Payload["Name"], "SELECT_id_FROM_testtest_WHERE_idANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDi")

	// test another query that used to map to the same accessor name as above
	wq = fmt.Sprintf("%s AND id = 1", wq)
	assert.NoErr(t, db.SelectContext(ctx, &res, "SELECT id FROM test WHERE "+wq))
	assert.Equal(t, res, []int{1})

	entries = getAuditLogEntries(ctx, t, tenantDB)
	assert.Equal(t, len(entries), 10, assert.Must())
	assert.Equal(t, entries[9].Type, internal.AuditLogEventTypeExecuteAccessor)
	assert.True(t, strings.HasPrefix(
		entries[9].Payload["Name"].(string),
		"SELECT_id_FROM_testtest_WHERE_idANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDidANDi_"),
		assert.Errorf("expected prefix, got %s", entries[9].Payload["Name"]))

	complexQuery := "SELECT id FROM test WHERE id IS NULL;"
	var complexRes []string
	err = db.SelectContext(ctx, &complexRes, complexQuery)
	assert.NoErr(t, err)

	entries = getAuditLogEntries(ctx, t, tenantDB)
	assert.Equal(t, len(entries), 13, assert.Must())
	assert.Equal(t, entries[12].Type, internal.AuditLogEventTypeExecuteAccessor)
	assert.Equal(t, entries[12].Payload["Name"], "SELECT_id_FROM_testtest_WHERE_idISNULL")

}

func getAuditLogEntries(ctx context.Context, t *testing.T, tenantDB *ucdb.DB) []auditlog.Entry {
	time.Sleep(time.Second) // flush the auditlog -- is there a better way?

	as := auditlog.NewStorage(tenantDB)
	pager, err := auditlog.NewEntryPaginatorFromOptions(pagination.SortKey("created,id"), pagination.SortOrder(pagination.OrderAscending))
	assert.NoErr(t, err)
	entries, _, err := as.ListEntriesPaginated(ctx, *pager)
	assert.NoErr(t, err)

	return entries
}
