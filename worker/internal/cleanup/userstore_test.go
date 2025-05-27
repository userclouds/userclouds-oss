package cleanup

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/testhelpers"
)

// provisionCompanyWithTenants creates a company of the specified type and provisions
// the requested number of tenants, returning the list of tenant IDs
func provisionCompanyWithTenants(ctx context.Context, t *testing.T, storage *companyconfig.Storage,
	companyConfigDBCfg *ucdb.Config, logDBCfg *ucdb.Config, companyType companyconfig.CompanyType, tenantCount int) []uuid.UUID {
	company := testhelpers.ProvisionTestCompanyWithoutACL(ctx, t, storage)
	company.Type = companyType
	assert.NoErr(t, storage.SaveCompany(ctx, company))
	tenantIDs := make([]uuid.UUID, 0, tenantCount)
	for range tenantCount {
		tenant, _ := testhelpers.ProvisionTestTenant(ctx, t, storage, companyConfigDBCfg, logDBCfg, company.ID)
		tenantIDs = append(tenantIDs, tenant.ID)
	}
	return tenantIDs
}

func TestCleanUserStoreForAllTenantsHandler(t *testing.T) {
	ctx := context.Background()
	companyConfigDBCfg, logDBCfg, storage := testhelpers.NewTestStorage(t)
	customerTenantIDs := provisionCompanyWithTenants(ctx, t, storage, companyConfigDBCfg, logDBCfg, companyconfig.CompanyTypeCustomer, 2)
	internalTenantIDs := provisionCompanyWithTenants(ctx, t, storage, companyConfigDBCfg, logDBCfg, companyconfig.CompanyTypeInternal, 1)
	allTenantIDs := append(customerTenantIDs, internalTenantIDs...)

	tests := []struct {
		name                  string
		url                   string
		expectedStatus        int
		expectedDryRun        bool
		expectedMaxCandidates int
		expectedTenantIDs     []uuid.UUID
	}{
		{
			name:                  "default parameters",
			url:                   "/cleanup",
			expectedStatus:        http.StatusOK,
			expectedDryRun:        true,
			expectedMaxCandidates: defaultMaxCandidates,
			expectedTenantIDs:     allTenantIDs,
		},
		{
			name:                  "with explicit parameters",
			url:                   "/cleanup?no-dry-run=true&max-candidates=50&company-types=customer",
			expectedStatus:        http.StatusOK,
			expectedDryRun:        false,
			expectedMaxCandidates: 50,
			expectedTenantIDs:     customerTenantIDs,
		},
		{
			name:                  "no companies of a given type",
			url:                   "/cleanup?no-dry-run=true&max-candidates=50&company-types=demo,prospect",
			expectedStatus:        http.StatusOK,
			expectedDryRun:        false,
			expectedMaxCandidates: 50,
			expectedTenantIDs:     nil,
		},
		{
			name:              "invalid company type",
			url:               "/cleanup?company-types=invalid",
			expectedStatus:    http.StatusBadRequest,
			expectedTenantIDs: nil, // No messages expected
		},
		{
			name:              "invalid max candidates",
			url:               "/cleanup?max-candidates=abc",
			expectedStatus:    http.StatusBadRequest,
			expectedTenantIDs: nil, // No messages expected
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wc := workerclient.NewTestClient()
			w := httptest.NewRecorder()
			handler := CleanUserStoreForAllTenantsHandler(storage, wc)
			handler.ServeHTTP(w, httptest.NewRequest("GET", tc.url, nil))
			assert.Equal(t, tc.expectedStatus, w.Code)
			sentMessages := wc.GetMessages()
			assert.Equal(t, len(tc.expectedTenantIDs), len(sentMessages))
			if tc.expectedStatus != http.StatusOK || len(sentMessages) == 0 {
				return
			}
			for _, msg := range sentMessages {
				msgParams := msg.UserStoreDataCleanup
				assert.NotNil(t, msgParams, assert.Must())
				assert.Equal(t, tc.expectedDryRun, msgParams.DryRun)
				assert.Equal(t, tc.expectedMaxCandidates, msgParams.MaxCandidates)
				assert.True(t,
					slices.Contains(tc.expectedTenantIDs, msg.TenantID),
					assert.Errorf("expected tenant IDs: %v, actual tenant ID: %v", tc.expectedTenantIDs, msg.TenantID))
			}
		})
	}
}
