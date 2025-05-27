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

func searchCompanies(ctx context.Context, ccs *companyconfig.Storage, query string) (*ucdb.Config, error) {
	pager, err := companyconfig.NewCompanyPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit),
		pagination.Filter(fmt.Sprintf("('name',IL,'%%%s%%')", query)),
	)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	companies := []companyconfig.Company{}
	for {
		cos, pr, err := ccs.ListCompaniesPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		if len(cos) == 0 {
			return nil, ucerr.New("no companies found")
		}

		if len(cos) == 1 {
			uclog.Debugf(ctx, "found 1 company: %s (%v)", cos[0].Name, cos[0].ID)
			return listTenantsForCompany(ctx, ccs, cos[0])
		}

		companies = append(companies, cos...)

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	for i, co := range companies {
		uclog.Debugf(ctx, "%d: %s", i, co.Name)
	}
	choice := cmdline.ReadConsole(ctx, "Enter company number: ")
	c, err := strconv.Atoi(choice)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if c < 0 || c >= len(companies) {
		return nil, ucerr.Errorf("invalid choice %d, must be [0, %d]", c, len(companies)-1)
	}

	return listTenantsForCompany(ctx, ccs, companies[c])
}

func listTenantsForCompany(ctx context.Context, ccs *companyconfig.Storage, company companyconfig.Company) (*ucdb.Config, error) {
	tens, err := ccs.ListTenantsForCompany(ctx, company.ID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	cos := make(map[uuid.UUID]companyconfig.Company)
	cos[company.ID] = company
	return chooseTenantWithCompanies(ctx, ccs, tens, cos)
}
