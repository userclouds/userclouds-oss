package main

import (
	"context"

	tenantdb "userclouds.com/idp/migration"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/dbdata"
)

func getDatabaseData(ctx context.Context, uv universe.Universe, service string) (*migrate.ServiceData, error) {
	if service == "tenantdb" {
		sd, err := tenantdb.GetServiceData(ctx, uv)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if err := sd.Migrations.Validate(); err != nil {
			return nil, ucerr.Wrap(err)
		}
		return sd, nil
	}
	return dbdata.GetDatabaseData(ctx, service)
}

func getMigrationURL(ctx context.Context, uv universe.Universe, dbName string) (string, error) {
	svc := service.Undefined
	if dbName == "tenantdb" {
		svc = service.IDP
	} else if dbName == "logdb" || dbName == "status" {
		svc = service.LogServer
	} else if dbName == "companyconfig" {
		svc = service.Console
	}
	if !svc.IsUndefined() {
		return cmdline.GetURLForUniverse(uv, "/migrations", svc)
	}
	// Not supported yet:
	// else if dbName == "rootdb" || dbName == "rootdbstatus" {
	return "", nil
}
