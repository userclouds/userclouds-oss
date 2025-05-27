package cleanup

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"userclouds.com/idp/helpers"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/worker"
)

const (
	defaultMaxCandidates = 100
)

// CleanUserStoreForTenant cleans up data for a tenant
func CleanUserStoreForTenant(ctx context.Context, ts *tenantmap.TenantState, params worker.DataCleanupParams) error {
	uclog.Infof(ctx, "Cleaning userstore data for tenant %v  max: %d dry run: %v", ts.ID, params.MaxCandidates, params.DryRun)
	remaining, err := helpers.CleanForTenant(ctx, ts, params.MaxCandidates, params.DryRun)
	if err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "Cleaned %d/%d userstore records for tenant %v", params.MaxCandidates, remaining, ts.ID)
	return nil
}

// CleanUserStoreResponse represents the response from the cleanup operation
type CleanUserStoreResponse struct {
	CompaniesCount int                         `json:"companies_count" yaml:"companies_count"`
	TenantsCount   int                         `json:"tenants_count" yaml:"tenants_count"`
	DryRun         bool                        `json:"dry_run" yaml:"dry_run"`
	MaxCandidates  int                         `json:"max_candidates" yaml:"max_candidates"`
	CompanyTypes   []companyconfig.CompanyType `json:"company_types" yaml:"company_types"`
}

// CleanUserStoreForAllTenantsHandler returns a handler that dispatches cleanup tasks for tenants
func CleanUserStoreForAllTenantsHandler(ccs *companyconfig.Storage, wc workerclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		uclog.SetHandlerName(ctx, "clean-userstore-data")
		qp := r.URL.Query()
		dryRun := qp.Get("no-dry-run") != "true"
		companyTypes, err := getCompanyTypes(qp)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return
		}

		maxCandidates, err := getMaxCandidates(qp)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return
		}

		companiesCount, tenantsCount, err := dispatchCleanUserStoreForTenant(ctx, ccs, wc, companyTypes, dryRun, maxCandidates)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusInternalServerError))
			return
		}

		response := CleanUserStoreResponse{
			CompaniesCount: companiesCount,
			TenantsCount:   tenantsCount,
			DryRun:         dryRun,
			MaxCandidates:  maxCandidates,
			CompanyTypes:   companyTypes,
		}
		jsonapi.Marshal(w, response)
	}
}

func getCompanyTypes(qp url.Values) ([]companyconfig.CompanyType, error) {
	companyTypes := qp.Get("company-types")
	if companyTypes == "" {
		return companyconfig.AllCompanyTypes, nil
	}
	companyTypesNames := set.NewStringSet(strings.Split(companyTypes, ",")...)
	selectedCompanyTypes := make([]companyconfig.CompanyType, 0, companyTypesNames.Size())
	for _, ct := range companyconfig.AllCompanyTypes {
		ctStr, err := ct.MarshalText()
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if companyTypesNames.Contains(string(ctStr)) {
			selectedCompanyTypes = append(selectedCompanyTypes, ct)
		}
	}
	if len(selectedCompanyTypes) != companyTypesNames.Size() {
		return nil, ucerr.Errorf("invalid company types: %v", companyTypesNames)
	}
	return selectedCompanyTypes, nil
}

func getMaxCandidates(qp url.Values) (int, error) {
	maxCandidatesStr := qp.Get("max-candidates")
	if maxCandidatesStr == "" {
		return defaultMaxCandidates, nil
	}
	maxCandidates, err := strconv.Atoi(maxCandidatesStr)
	if err != nil {
		return 0, ucerr.Wrap(err)
	}
	return maxCandidates, nil
}

func dispatchCleanUserStoreForTenant(ctx context.Context, ccs *companyconfig.Storage, wc workerclient.Client, companyTypes []companyconfig.CompanyType, dryRun bool, maxCandidates int) (int, int, error) {
	uclog.Infof(ctx, "dispatching userstore cleanup tasks with dry run: %v, max candidates: %d, company types: %v", dryRun, maxCandidates, companyTypes)
	pager, err := companyconfig.NewCompanyPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return 0, 0, ucerr.Wrap(err)
	}
	tenantsCount := 0
	companiesCount := 0
	for {
		companies, pr, err := ccs.ListCompaniesPaginated(ctx, *pager)
		if err != nil {
			return companiesCount, tenantsCount, ucerr.Wrap(err)
		}
		for _, company := range companies {
			if !slices.Contains(companyTypes, company.Type) {
				continue
			}
			tenants, err := ccs.ListTenantsForCompany(ctx, company.ID)
			if err != nil {
				return companiesCount, tenantsCount, ucerr.Wrap(err)
			}
			if len(tenants) == 0 {
				continue
			}
			companiesCount++
			for _, tenant := range tenants {
				msg := worker.UserStoreDataCleanupMessage(tenant.ID, maxCandidates, dryRun)
				if err := wc.Send(ctx, msg); err != nil {
					return companiesCount, tenantsCount, ucerr.Wrap(err)
				}
				tenantsCount++
			}
		}
		if !pager.AdvanceCursor(*pr) {
			break
		}
	}
	uclog.Infof(ctx, "dispatched userstore cleanup tasks for %d tenants in %d companies. dry-run=%v, max-candidates=%d ", tenantsCount, companiesCount, dryRun, maxCandidates)
	return companiesCount, tenantsCount, nil
}
