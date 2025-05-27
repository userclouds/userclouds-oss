package main

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/timer"
)

type accessorsTest struct {
	idp           *idp.Client
	accessorID    uuid.UUID
	searchStrings []string
	stopOnError   bool
}

func newAccessorsTest(tenantURL string, tokenSource jsonclient.Option, accessorID uuid.UUID, stopOnError bool, searchStrings []string) (*accessorsTest, error) {
	idpc, err := idp.NewClient(tenantURL, idp.JSONClient(tokenSource))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &accessorsTest{
		idp:           idpc,
		accessorID:    accessorID,
		searchStrings: searchStrings,
		stopOnError:   stopOnError,
	}, nil
}

func runAccessor(ctx context.Context, tenantURL string, tokenSource jsonclient.Option, iterations int, accessorID uuid.UUID, stopOnError bool, searchStrings []string) error {
	act, err := newAccessorsTest(tenantURL, tokenSource, accessorID, stopOnError, searchStrings)
	if err != nil {
		return ucerr.Wrap(err)
	}

	tmr := timer.Start()
	for i := 1; i < iterations+1; i++ {
		uclog.Infof(ctx, "Run Accessors Completed for %v (%d/%d)", tenantURL, i, iterations)

		if err := act.runAccessors(ctx); err != nil {
			return ucerr.Wrap(err)
		}
		uclog.Infof(ctx, "Run Accessors Completed for in %v (%d/%d)", tmr.Reset(), i, iterations)
	}
	return nil
}

func (act *accessorsTest) runAccessors(ctx context.Context) error {
	for _, searchString := range act.searchStrings {
		fieldsStr, err := act.idp.ExecuteAccessor(ctx, act.accessorID, nil, []any{searchString})
		if err != nil {
			if act.stopOnError {
				return ucerr.Wrap(err)
			}
			uclog.Warningf(ctx, "ExecuteAccessor for %v for '%v' returned error: %v", act.accessorID, searchString, err)
			continue

		}
		if len(fieldsStr.Data) == 0 {
			uclog.Warningf(ctx, "ExecuteAccessor %v for '%v' returned no data: %v", act.accessorID, searchString, fieldsStr)
			continue
		}
		fields := map[string]string{}
		for _, field := range fieldsStr.Data {
			if err = json.Unmarshal([]byte(field), &fields); err != nil {
				return ucerr.Wrap(err)
			}
			uclog.Infof(ctx, " ExecuteAccessor %v for '%v' returned %v", act.accessorID, searchString, fields)
		}
	}
	return nil
}
