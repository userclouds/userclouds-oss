package dbdata

import (
	"context"

	"userclouds.com/infra/migrate"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/logdb"
	"userclouds.com/internal/rootdb"
	"userclouds.com/internal/rootdbstatus"
	"userclouds.com/logserver"
)

// GetDatabaseData returns information about this "service" used for DB migrations.
func GetDatabaseData(ctx context.Context, service string) (*migrate.ServiceData, error) {
	var sd *migrate.ServiceData
	var err error
	switch service {
	case "companyconfig":
		sd, err = companyconfig.GetServiceData(ctx)
	case "status":
		sd, err = logdb.GetServiceData(ctx)
	case "rootdb":
		sd, err = rootdb.GetServiceData(ctx)
	case "rootdbstatus":
		sd, err = rootdbstatus.GetServiceData(ctx)
	case "logserver":
		sd, err = logserver.GetServiceData(ctx)
	case "plex", "dataprocessor":
		break
	default:
		return nil, ucerr.Errorf("migrate doesn't recognize service: %s", service)
	}

	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if err := sd.Migrations.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}
	if sd.DBCfg == nil {
		return nil, ucerr.Errorf("service %s has no databases to migrate", service)
	}
	return sd, nil
}
