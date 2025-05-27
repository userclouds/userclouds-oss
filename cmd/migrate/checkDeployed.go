package main

import (
	"context"
	"fmt"
	"net/url"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

func checkDeployedVersion(ctx context.Context, dbName string, serviceData migrate.ServiceData, migrationURL string) error {
	if err := serviceData.Migrations.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	localMax := serviceData.Migrations.GetMaxAvailable()

	migURL, err := url.Parse(migrationURL)
	if err != nil {
		return ucerr.Wrap(err)
	}
	baseURL := fmt.Sprintf("%s://%s", migURL.Scheme, migURL.Host)
	client := jsonclient.New(baseURL)

	var prodMaxes []int
	if err := client.Get(ctx, migURL.Path, &prodMaxes); err != nil {
		return ucerr.Errorf("failed to get current migration status for %v (%v%v): %v", dbName, baseURL, migURL.Path, err)
	}

	for _, m := range prodMaxes {
		if m != localMax {
			return ucerr.Errorf("'%s' universe max migration %d for service '%s' doesn't match (expected %d, got %v)", universe.Current(), m, dbName, localMax, prodMaxes)
		}
	}
	uclog.Infof(ctx, "'%s' universe migrations for %s in sync with local code (%d)", universe.Current(), dbName, localMax)
	return nil
}
