package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/companyconfig"
)

func chooseTenant(ctx context.Context, ccs *companyconfig.Storage, tens []companyconfig.Tenant) (*ucdb.Config, error) {
	// load companies so we can give descriptive tenant choices
	pager, err := companyconfig.NewCompanyPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit),
	)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	cos, pr, err := ccs.ListCompaniesPaginated(ctx, *pager)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	companies := make(map[uuid.UUID]companyconfig.Company, len(cos)) // start with the first page length, at least
	for {
		for _, co := range cos {
			companies[co.ID] = co
		}

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	return chooseTenantWithCompanies(ctx, ccs, tens, companies)
}

// this is a simple optimization for the case we were called from searchCompanies
func chooseTenantWithCompanies(ctx context.Context,
	ccs *companyconfig.Storage,
	tens []companyconfig.Tenant,
	companies map[uuid.UUID]companyconfig.Company) (*ucdb.Config, error) {

	// display the list of tenants
	for i, ten := range tens {
		uclog.Debugf(ctx, "%d: %s (%s, %s, %v)", i, ten.Name, companies[ten.CompanyID].Name, ten.TenantURL, ten.ID)
	}
	choice := cmdline.ReadConsole(ctx, "Enter tenant number: ")
	c, err := strconv.Atoi(choice)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if c < 0 || c >= len(tens) {
		return nil, ucerr.Errorf("invalid choice %d, must be [0, %d]", c, len(tens)-1)
	}

	return getTenantDBConfigForID(ctx, ccs, tens[c].ID)
}

func searchTenants(ctx context.Context, ccs *companyconfig.Storage, query string) (*ucdb.Config, error) {
	pager, err := companyconfig.NewTenantPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit),
		pagination.Filter(fmt.Sprintf("('name',IL,'%%%s%%')", query)),
	)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	tenants := []companyconfig.Tenant{}
	for {
		tens, pr, err := ccs.ListTenantsPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		if len(tens) == 0 {
			return nil, ucerr.New("no tenants found")
		}

		if len(tens) == 1 {
			uclog.Debugf(ctx, "found 1 tenant: %s (%v)", tens[0].Name, tens[0].ID)
			return getTenantDBConfigForID(ctx, ccs, tens[0].ID)
		}

		tenants = append(tenants, tens...)

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	return chooseTenant(ctx, ccs, tenants)
}
