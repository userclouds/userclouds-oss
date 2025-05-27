package migrate

import (
	"context"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
)

type schemaValidator struct {
	schema Schema
}

// Validate implements ucdb.Validator
func (sv schemaValidator) Validate(ctx context.Context, db *ucdb.DB) error {
	// skip this check in prod to speed up new DB connections
	// TODO (sgarrity 11/23): revisit this if/when we have longer-lived connections and
	// can afford the performance hit?
	if universe.Current().IsProd() {
		return nil
	}

	columnMap, err := SelectColumns(ctx, db)
	if err != nil {
		return ucerr.Wrap(err)
	}

	for requiredTable, requiredCols := range sv.schema.Columns {
		if _, ok := columnMap[requiredTable]; !ok {
			return ucerr.Errorf("database missing required table %v", requiredTable)
		}

		extant := set.NewStringSet(columnMap[requiredTable]...)
		required := set.NewStringSet(requiredCols...)

		if !extant.IsSupersetOf(required) {
			return ucerr.Errorf("database missing required columns for table %v: %v", requiredTable, required.Difference(extant))
		}
	}

	// only in dev, warn if we haven't migrated up yet (makes it easier to debug things)
	if universe.Current().IsDev() {
		maxInDB, err := GetMaxVersion(ctx, db)
		if err != nil {
			return ucerr.Wrap(err)
		}

		if maxInCode := sv.schema.Migrations.GetMaxAvailable(); maxInDB != maxInCode {
			uclog.Warningf(ctx, "migrations max version %d does not match schema max version %d", maxInDB, maxInCode)
		}
	}

	return nil
}

// SchemaValidator returns a Validator
func SchemaValidator(schema Schema) ucdb.Validator {
	return schemaValidator{schema: schema}
}
