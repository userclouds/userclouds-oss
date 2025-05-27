package tenant_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
	"userclouds.com/worker"
	workertesthelpers "userclouds.com/worker/internal/testhelpers"
)

func TestCreate(t *testing.T) {
	ctx := context.Background()
	testClient := workerclient.NewTestClient()

	wh, ccs, consoleTenant, consoleTenantDB := workertesthelpers.SetupWorkerForTest(ctx, t, testClient)
	// test creation in UserClouds company and non-UC company (affects authz creds),
	// as well as with & without orgs enabled
	//
	// we use a waitgroup here since otherwise this main test method terminates,
	// and the test servers shut down :/
	newco := testhelpers.NewCompanyForTest(ctx, t, ccs, consoleTenantDB, consoleTenant.ID, consoleTenant.CompanyID)
	var wg sync.WaitGroup
	for _, coID := range []uuid.UUID{consoleTenant.CompanyID, newco.ID} {
		for _, useOrgs := range []bool{true, false} {
			// set up new servers since we need a new URL to create each new tenant
			// TODO (sgarrity 6/23): there has to be a better way?
			srvNew := workertesthelpers.CreateTestHTTPServer(ctx, t, ccs, consoleTenant.ID)

			// set up the tenant we want to create
			ten := companyconfig.Tenant{
				BaseModel:        ucdb.NewBase(),
				Name:             "async",
				CompanyID:        coID,
				TenantURL:        workertesthelpers.GetTenantURLForServerURL(t, srvNew.URL, "async", newco.Name),
				UseOrganizations: useOrgs,
			}
			msg := worker.NewCreateTenantMessage(ten, uuid.Must(uuid.NewV4()))
			msg.SourceRegion = region.Current()
			r := httptest.NewRequest(http.MethodPost, "/", uctest.IOReaderFromJSONStruct(t, msg))

			wg.Add(1)
			go func() {
				// thanks to ChatGPT, trying this approach to run tests in parallel
				// without this goroutine, the tests would deadlock when using t.Parallel()
				// along with the waitgroup. This seems to work.
				defer wg.Done()

				t.Run(fmt.Sprintf("%v/%v", coID == consoleTenant.CompanyID, useOrgs), func(t *testing.T) {
					t.Parallel()

					rr := httptest.NewRecorder()
					wh.ServeHTTP(rr, r)

					assert.Equal(t, rr.Code, http.StatusOK)

					currentTenantState := companyconfig.TenantStateCreating
					maxTries := 120 // Two minutes since we check roughly every second
					for range maxTries {
						got, err := ccs.GetTenant(ctx, ten.ID)

						// the first few seconds we might actually get sql.ErrNoRows
						if !errors.Is(err, sql.ErrNoRows) {
							assert.NoErr(t, err)
						}

						// and likewise with sql.ErrNoRows, we'll get a nil tenant
						if got != nil {
							currentTenantState = got.State
							if currentTenantState == companyconfig.TenantStateActive {
								break
							}

							if !assert.Equal(t,
								got.State,
								companyconfig.TenantStateCreating,
								assert.Errorf("unexpected tenant state %v during creation", got.State)) {
								break
							}
						}
						// TODO (sgarrity 6/23): we should build some signal into tenant creation being completed someday
						time.Sleep(time.Second)
					}

					assert.Equal(t, currentTenantState, companyconfig.TenantStateActive, assert.Errorf("tenant creation did not complete successfully"))
				})
			}()
		}
	}

	// don't eg. shut down HTTP servers until tests have finished
	wg.Wait()
}

func TestRequeueCreateTenant(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testClient := workerclient.NewTestClient()
	assert.Equal(t, region.Current(), region.MachineRegion("mars"), assert.Errorf("expected region to be mars under CI or tests. got %v", region.Current()), assert.Must())

	wh, ccs, consoleTenant, consoleTenantDB := workertesthelpers.SetupWorkerForTest(ctx, t, testClient)
	testhelpers.NewCompanyForTest(ctx, t, ccs, consoleTenantDB, consoleTenant.ID, consoleTenant.CompanyID)
	newTenant := companyconfig.Tenant{BaseModel: ucdb.NewBase(), Name: "jerry", CompanyID: consoleTenant.CompanyID, TenantURL: "No soup for you", UseOrganizations: false}
	msg := worker.NewCreateTenantMessage(newTenant, uuid.Must(uuid.NewV4()))
	msg.SourceRegion = region.MachineRegion("themoon") // so a re-queue is triggered
	r := httptest.NewRequest(http.MethodPost, "/", uctest.IOReaderFromJSONStruct(t, msg))
	rr := httptest.NewRecorder()
	wh.ServeHTTP(rr, r)
	assert.Equal(t, rr.Code, http.StatusOK)
	queuedMsg := testClient.WaitForMessages(t, 1, 5*time.Second)[0]
	assert.Equal(t, queuedMsg.SourceRegion, region.MachineRegion("themoon"))
	assert.Equal(t, queuedMsg.Task, worker.TaskCreateTenant)
	assert.NotNil(t, queuedMsg.CreateTenant)
	assert.Equal(t, queuedMsg.CreateTenant.Tenant, newTenant)
}
