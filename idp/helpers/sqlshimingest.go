package helpers

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql" // mysql driver
	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // also registers Postgres driver

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/sqlshim"
	"userclouds.com/internal/tenantmap"
)

var ucDataTypesByDatabaseTypeSQLDataType = map[sqlshim.DatabaseType]map[string]userstore.ResourceID{
	sqlshim.DatabaseTypePostgres: {
		"bigint":                      datatype.Integer,
		"boolean":                     datatype.Boolean,
		"char":                        datatype.String,
		"character":                   datatype.String,
		"character varying":           datatype.String,
		"date":                        datatype.Date,
		"integer":                     datatype.Integer,
		"numeric":                     datatype.String,
		"smallint":                    datatype.Integer,
		"text":                        datatype.String,
		"timestamp":                   datatype.Timestamp,
		"timestamp with time zone":    datatype.Timestamp,
		"timestamp without time zone": datatype.Timestamp,
		"uuid":                        datatype.UUID,
		"varchar":                     datatype.String,
		// TODO: we represent psql jsonb columns as a string for now, but
		//       may eventually want to derive and create a composite data
		//       type so we have structured types for the fields
		"jsonb": datatype.String,
	},
	sqlshim.DatabaseTypeMySQL: {
		"bigint":     datatype.Integer,
		"bit":        datatype.String,
		"blob":       datatype.String,
		"char":       datatype.String,
		"date":       datatype.Date,
		"datetime":   datatype.Timestamp,
		"decimal":    datatype.String,
		"double":     datatype.String,
		"enum":       datatype.String,
		"float":      datatype.String,
		"int":        datatype.Integer,
		"json":       datatype.String,
		"longblob":   datatype.String,
		"longtext":   datatype.String,
		"mediumint":  datatype.Integer,
		"mediumtext": datatype.String,
		"smallint":   datatype.Integer,
		"text":       datatype.String,
		"time":       datatype.Timestamp,
		"timestamp":  datatype.Timestamp,
		"tinyint":    datatype.Integer,
		"varchar":    datatype.String,
	},
}

func ucDataTypeFromSQLDataType(dbt sqlshim.DatabaseType, dt string) (userstore.ResourceID, error) {
	ucDataTypesBySQLDataType, found := ucDataTypesByDatabaseTypeSQLDataType[dbt]
	if !found {
		return userstore.ResourceID{}, ucerr.Friendlyf(nil, "unsupported database type '%v'", dbt)
	}

	ucDataType, found := ucDataTypesBySQLDataType[dt]
	if !found {
		return userstore.ResourceID{}, ucerr.Friendlyf(nil, "unsupported data type '%v' for database type '%v'", dt, dbt)
	}

	return ucDataType, nil
}

// LoadColumnsIntoUserstore loads columns into the userstore
func LoadColumnsIntoUserstore(ctx context.Context, s *storage.Storage, databaseID uuid.UUID, dbt sqlshim.DatabaseType, columns []sqlshim.Column) error {
	cm, err := storage.NewColumnManager(ctx, s, databaseID)
	if err != nil {
		uclog.Errorf(ctx, "failed to create column manager: %v", err)
		return ucerr.Wrap(err)
	}

	for _, c := range columns {
		dataType, err := ucDataTypeFromSQLDataType(dbt, c.Type)
		if err != nil {
			uclog.Errorf(ctx, "failed to derive UC column type - defaulting to String: %v", err)
			// log but default to string
			dataType = datatype.String
		}

		if col := cm.GetColumnByTableAndName(c.Table, c.Name); col == nil {
			uclog.Infof(ctx, "creating column: %v", c)
			newColumn := &userstore.Column{
				Table:              c.Table,
				Name:               c.Name,
				DataType:           dataType,
				IsArray:            false,
				IndexType:          userstore.ColumnIndexTypeNone,
				AccessPolicy:       userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID, Name: policy.AccessPolicyAllowAll.Name},
				DefaultTransformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID, Name: policy.TransformerPassthrough.Name},
			}
			if _, err := cm.CreateColumnFromClient(ctx, newColumn); err != nil {
				uclog.Errorf(ctx, "failed to save new column %v: %v", newColumn, err)
				return ucerr.Wrap(err)
			}
		} else if col.DataTypeID != dataType.ID {
			uclog.Errorf(
				ctx,
				"UC column data type (%v) doesn't match type sql column type (%v) for column: %s",
				col.DataTypeID,
				dataType.ID,
				c.Name,
			)
		}
	}

	return nil
}

// IngestSqlshimDatabaseSchemas ingests the schemas of a sqlshim database into the userstore
func IngestSqlshimDatabaseSchemas(ctx context.Context, ts *tenantmap.TenantState, databaseID uuid.UUID) error {
	uclog.Infof(ctx, "Ingesting sqlshim database schema for tenant %v", ts.ID)

	s := storage.NewFromTenantState(ctx, ts)
	db, err := s.GetSQLShimDatabase(ctx, databaseID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	password, err := db.Password.Resolve(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	var allColumns []sqlshim.Column
	if db.Type == sqlshim.DatabaseTypeMySQL {
		for _, schema := range db.Schemas {
			columns, err := extractColumnsFromMysqlSchema(ctx, db.Host, db.Port, db.Username, password, schema)
			if err != nil {
				return ucerr.Wrap(err)
			}
			allColumns = append(allColumns, columns...)
		}
	} else if db.Type == sqlshim.DatabaseTypePostgres {
		for _, schema := range db.Schemas {
			columns, err := extractColumnsFromPsqlSchema(ctx, db.Host, db.Port, db.Username, password, schema)
			if err != nil {
				return ucerr.Wrap(err)
			}
			allColumns = append(allColumns, columns...)
		}
	} else {
		return ucerr.Errorf("unsupported database type: %v", db.Type)
	}

	if err := LoadColumnsIntoUserstore(ctx, s, databaseID, db.Type, allColumns); err != nil {
		return ucerr.Wrap(err)
	}

	db.SchemasUpdated = time.Now().UTC()
	if err := s.SaveSQLShimDatabase(ctx, db); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

type mysqlSchemaColumn struct {
	TableSchema string `db:"TABLE_SCHEMA"`
	TableName   string `db:"TABLE_NAME"`
	ColumnName  string `db:"COLUMN_NAME"`
	DataType    string `db:"DATA_TYPE"`
}

var typeRegExp = regexp.MustCompile(`(.*)\((\d+)\)`)

func extractColumnsFromMysqlSchema(ctx context.Context, host string, port int, username, password, schema string) ([]sqlshim.Column, error) {

	db, err := sqlx.Connect("mysql", fmt.Sprintf("%s:%s@(%s:%d)/%s", username, password, host, port, schema))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			uclog.Errorf(ctx, "failed to close database connection: %v", err)
		}
	}()

	var msqlColumns []mysqlSchemaColumn
	if err := db.SelectContext(ctx, &msqlColumns, "SELECT TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME, DATA_TYPE FROM information_schema.columns WHERE TABLE_SCHEMA=?; /* lint-deleted bypass-known-table-check */", schema); err != nil {
		return nil, ucerr.Wrap(err)
	}

	columns := []sqlshim.Column{}
	for _, column := range msqlColumns {
		t := column.DataType
		l := 0
		if m := typeRegExp.FindStringSubmatch(column.DataType); len(m) == 3 {
			t = m[1]
			l, err = strconv.Atoi(m[2])
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
		}
		columns = append(columns, sqlshim.Column{
			Table:  schema + "." + column.TableName,
			Name:   column.ColumnName,
			Type:   t,
			Length: l,
		})
	}

	return columns, nil
}

type postgresSchemaColumn struct {
	TableName  string        `db:"table_name"`
	ColumnName string        `db:"column_name"`
	DataType   string        `db:"data_type"`
	CharMaxLen sql.NullInt32 `db:"character_maximum_length"`
}

func extractColumnsFromPsqlSchema(ctx context.Context, host string, port int, username, password, schema string) ([]sqlshim.Column, error) {
	db, err := sqlx.Connect("postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, username, password, schema))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			uclog.Errorf(ctx, "failed to close database connection: %v", err)
		}
	}()

	var psqlColumns []postgresSchemaColumn
	if err := db.SelectContext(ctx, &psqlColumns, "SELECT table_name, column_name, data_type, character_maximum_length FROM information_schema.columns WHERE table_schema <> 'pg_catalog' AND table_schema <> 'information_schema'; /* lint-sql-unsafe-columns lint-system-table deleted */"); err != nil {
		return nil, ucerr.Wrap(err)
	}

	columns := []sqlshim.Column{}
	for _, column := range psqlColumns {
		l := 0
		if column.CharMaxLen.Valid {
			l = int(column.CharMaxLen.Int32)
		}

		columns = append(columns, sqlshim.Column{
			Table:  schema + "." + column.TableName,
			Name:   column.ColumnName,
			Type:   column.DataType,
			Length: l,
		})
	}

	return columns, nil
}
