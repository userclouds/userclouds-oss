package storage

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucerr"
	uctime "userclouds.com/infra/uctypes/timestamp"
	"userclouds.com/infra/uctypes/timestamparray"
	"userclouds.com/infra/uctypes/uuidarray"
)

type userRowsTableInfo struct {
	dlcs column.DataLifeCycleState
}

func newUserRowsTableInfo(dlcs column.DataLifeCycleState) (*userRowsTableInfo, error) {
	switch dlcs {
	case column.DataLifeCycleStateLive, column.DataLifeCycleStateSoftDeleted:
		return &userRowsTableInfo{dlcs: dlcs}, nil
	default:
		return nil, ucerr.Errorf("unsupported DataLifeCycleState %v", dlcs)
	}
}

const usersNonSystemColumnLiveQueryTemplate = `
/* lint-sql-unsafe-columns bypass-known-table-check */
SELECT%s
u.id user_id,
u.created user_created,
u.updated user_updated,
u.deleted user_deleted,
u._version user_version,
u.organization_id,
ucv.id value_id,
ucv._version value_version,
ucv.value_type,
ucv.column_id,
ucv.ordering,
ucv.consented_purpose_ids,
ucv.retention_timeouts,
ucv.varchar_value,
ucv.varchar_unique_value,
ucv.boolean_value,
ucv.int_value,
ucv.int_unique_value,
ucv.timestamp_value,
ucv.uuid_value,
ucv.uuid_unique_value,
ucv.jsonb_value
FROM
users u
LEFT JOIN user_column_pre_delete_values ucv
ON ucv.user_id = u.id%s%s%s
ORDER BY%s,
ucv.column_id ASC,
ucv.ordering ASC;
`

const usersNonSystemColumnSoftDeletedQueryTemplate = `
/* lint-sql-unsafe-columns bypass-known-table-check */
SELECT%s
u.id user_id,
u.created user_created,
u.updated user_updated,
u.deleted user_deleted,
u._version user_version,
u.organization_id,
ucv.id value_id,
ucv._version value_version,
ucv.value_type,
ucv.column_id,
ucv.ordering,
ucv.consented_purpose_ids,
ucv.retention_timeouts,
ucv.varchar_value,
'' varchar_unique_value,
ucv.boolean_value,
ucv.int_value,
0 int_unique_value,
ucv.timestamp_value,
ucv.uuid_value,
'00000000-0000-0000-0000-000000000000'::UUID uuid_unique_value,
ucv.jsonb_value
FROM
users u
LEFT JOIN user_column_post_delete_values ucv
ON ucv.user_id = u.id%s%s%s
ORDER BY%s,
ucv.column_id ASC,
ucv.created ASC,
ucv.ordering ASC;
`

func (urti userRowsTableInfo) getNonSystemColumnQuery(
	nonSystemSortColumnClause string,
	columnJoinClause string,
	sortJoinClause string,
	selectorClause string,
	orderByClause string,
) string {
	if urti.dlcs == column.DataLifeCycleStateSoftDeleted {
		return fmt.Sprintf(
			usersNonSystemColumnSoftDeletedQueryTemplate,
			nonSystemSortColumnClause,
			columnJoinClause,
			sortJoinClause,
			selectorClause,
			orderByClause,
		)
	}

	return fmt.Sprintf(
		usersNonSystemColumnLiveQueryTemplate,
		nonSystemSortColumnClause,
		columnJoinClause,
		sortJoinClause,
		selectorClause,
		orderByClause,
	)
}

const usersSystemColumnQueryTemplate = `
/* lint-sql-unsafe-columns */
SELECT
u.id user_id,
u.created user_created,
u.updated user_updated,
u.deleted user_deleted,
u._version user_version,
u.organization_id,
'00000000-0000-0000-0000-000000000000'::UUID value_id,
0 value_version,
0 value_type,
'00000000-0000-0000-0000-000000000000'::UUID column_id,
0 ordering,
NULL::UUID[] consented_purpose_ids,
NULL::VARCHAR[] retention_timeouts,
'' varchar_value,
'' varchar_unique_value,
FALSE boolean_value,
0 int_value,
0 int_unique_value,
'0001-01-01 00:00:00'::TIMESTAMP timestamp_value,
'00000000-0000-0000-0000-000000000000'::UUID uuid_value,
'00000000-0000-0000-0000-000000000000'::UUID uuid_unique_value,
'{}'::JSONB jsonb_value
FROM
users u%s
ORDER BY%s;
`

func (userRowsTableInfo) getSystemColumnQuery(selectorClause string, orderByClause string) string {
	return fmt.Sprintf(usersSystemColumnQueryTemplate, selectorClause, orderByClause)
}

func (urti userRowsTableInfo) getUserColumnValuesTableName() string {
	if urti.dlcs == column.DataLifeCycleStateSoftDeleted {
		return "user_column_post_delete_values"
	}

	return "user_column_pre_delete_values"
}

// UserQueryRow represents the result for a join query against the users and pre- or post-delete user_column_values tables
type UserQueryRow struct {
	UserID              uuid.UUID                     `db:"user_id"`
	UserCreated         time.Time                     `db:"user_created"`
	UserUpdated         time.Time                     `db:"user_updated"`
	UserDeleted         time.Time                     `db:"user_deleted"`
	OrganizationID      uuid.UUID                     `db:"organization_id"`
	UserVersion         int                           `db:"user_version"`
	ValueID             *uuid.UUID                    `db:"value_id"`
	ValueVersion        *int                          `db:"value_version"`
	ValueType           *ColumnValueType              `db:"value_type"`
	ColumnID            *uuid.UUID                    `db:"column_id"`
	Ordering            *int                          `db:"ordering"`
	ConsentedPurposeIDs uuidarray.UUIDArray           `db:"consented_purpose_ids"`
	RetentionTimeouts   timestamparray.TimestampArray `db:"retention_timeouts"`
	VarcharValue        *string                       `db:"varchar_value"`
	VarcharUniqueValue  *string                       `db:"varchar_unique_value"`
	BooleanValue        *bool                         `db:"boolean_value"`
	IntValue            *int                          `db:"int_value"`
	IntUniqueValue      *int                          `db:"int_unique_value"`
	TimestampValue      *time.Time                    `db:"timestamp_value"`
	UUIDValue           *uuid.UUID                    `db:"uuid_value"`
	UUIDUniqueValue     *uuid.UUID                    `db:"uuid_unique_value"`
	JSONBValue          any                           `db:"jsonb_value"`
}

func (uqr *UserQueryRow) toUserRow() userRow {
	ur := userRow{
		userID:              uqr.UserID,
		userCreated:         uqr.UserCreated,
		userUpdated:         uqr.UserUpdated,
		userDeleted:         uqr.UserDeleted,
		organizationID:      uqr.OrganizationID,
		userVersion:         uqr.UserVersion,
		consentedPurposeIDs: uqr.ConsentedPurposeIDs,
		retentionTimeouts:   uqr.RetentionTimeouts,
		varcharValue:        uqr.VarcharValue,
		varcharUniqueValue:  uqr.VarcharUniqueValue,
		booleanValue:        uqr.BooleanValue,
		intValue:            uqr.IntValue,
		intUniqueValue:      uqr.IntUniqueValue,
		timestampValue:      uqr.TimestampValue,
		uuidValue:           uqr.UUIDValue,
		uuidUniqueValue:     uqr.UUIDUniqueValue,
		jsonbValue:          uqr.JSONBValue,
	}

	if uqr.ValueID != nil {
		ur.valueID = *uqr.ValueID
		ur.valueVersion = *uqr.ValueVersion
		ur.valueType = *uqr.ValueType
		ur.columnID = *uqr.ColumnID
		ur.ordering = *uqr.Ordering
	}

	return ur
}

type userRow struct {
	userID              uuid.UUID
	userCreated         time.Time
	userUpdated         time.Time
	userDeleted         time.Time
	organizationID      uuid.UUID
	userVersion         int
	valueID             uuid.UUID
	valueVersion        int
	valueType           ColumnValueType
	columnID            uuid.UUID
	ordering            int
	consentedPurposeIDs uuidarray.UUIDArray
	retentionTimeouts   timestamparray.TimestampArray
	varcharValue        *string
	varcharUniqueValue  *string
	booleanValue        *bool
	intValue            *int
	intUniqueValue      *int
	timestampValue      *time.Time
	uuidValue           *uuid.UUID
	uuidUniqueValue     *uuid.UUID
	jsonbValue          any
}

func (ur userRow) getSystemColumnValue(columnName string) (any, error) {
	switch columnName {
	case "id":
		return ur.userID, nil
	case "created":
		return ur.userCreated, nil
	case "updated":
		return ur.userUpdated, nil
	case "deleted":
		return ur.userDeleted, nil
	case "organization_id":
		return ur.organizationID, nil
	case "_version":
		return ur.userVersion, nil
	}

	return nil, ucerr.Errorf("system column name '%s' is unrecognized", columnName)
}

func getJSONBValue[T any](ur userRow) (T, error) {
	var t T
	if err := json.Unmarshal(
		fmt.Appendf(nil, "%s", ur.jsonbValue),
		&t,
	); err != nil {
		return t, ucerr.Wrap(err)
	}
	return t, nil
}

type columnInfo struct {
	column Column
	dlcs   column.DataLifeCycleState
}

func newColumnInfo(c Column, dlcs column.DataLifeCycleState) columnInfo {
	return columnInfo{
		column: c,
		dlcs:   dlcs,
	}
}

func (ci columnInfo) getBracketedColumnName() string {
	return fmt.Sprintf("{%s}", ci.getColumnName())
}

func (ci columnInfo) getColumnName() string {
	return ci.column.Name
}

func (ci columnInfo) getDBColumnName() string {
	if ci.isSystemColumn() {
		if ci.column.Attributes.SystemName != "" {
			return ci.column.Attributes.SystemName
		}
	}

	return ci.column.Name
}

func (ci columnInfo) getUserRowColumnName() (string, error) {
	switch ci.column.IndexType {
	case columnIndexTypeNone, columnIndexTypeIndexed:
		switch ci.column.GetConcreteDataTypeID() {
		case datatype.String.ID:
			return "varchar_value", nil
		case datatype.Boolean.ID:
			return "boolean_value", nil
		case datatype.Integer.ID:
			return "int_value", nil
		case datatype.Timestamp.ID, datatype.Date.ID:
			return "timestamp_value", nil
		case datatype.UUID.ID:
			return "uuid_value", nil
		case datatype.Composite.ID:
			return "jsonb_value", nil
		}
	case columnIndexTypeUnique:
		if !ci.column.IsArray {
			switch ci.dlcs {
			case column.DataLifeCycleStateLive:
				switch ci.column.GetConcreteDataTypeID() {
				case datatype.String.ID:
					return "varchar_unique_value", nil
				case datatype.Integer.ID:
					return "int_unique_value", nil
				case datatype.UUID.ID:
					return "uuid_unique_value", nil
				}
			case column.DataLifeCycleStateSoftDeleted:
				switch ci.column.GetConcreteDataTypeID() {
				case datatype.String.ID:
					return "varchar_value", nil
				case datatype.Integer.ID:
					return "int_value", nil
				case datatype.UUID.ID:
					return "uuid_value", nil
				}
			}
		}
	}

	return "",
		ucerr.Errorf(
			"unsupported data life cycle state '%v', column data_type_id/index_type/is_array: '%v/%v/%v'",
			ci.dlcs,
			ci.column.DataTypeID,
			ci.column.IndexType,
			ci.column.IsArray,
		)
}

func (ci columnInfo) getUserRowValue(ur userRow) (any, error) {
	switch ci.column.IndexType {
	case columnIndexTypeNone, columnIndexTypeIndexed:
		switch ci.column.GetConcreteDataTypeID() {
		case datatype.String.ID:
			return *ur.varcharValue, nil
		case datatype.Boolean.ID:
			return *ur.booleanValue, nil
		case datatype.Integer.ID:
			return *ur.intValue, nil
		case datatype.Timestamp.ID, datatype.Date.ID:
			return uctime.Normalize(*ur.timestampValue), nil
		case datatype.UUID.ID:
			return *ur.uuidValue, nil
		case datatype.Composite.ID:
			v, err := getJSONBValue[userstore.CompositeValue](ur)
			return v, ucerr.Wrap(err)
		}
	case columnIndexTypeUnique:
		if !ci.column.IsArray {
			switch ci.dlcs {
			case column.DataLifeCycleStateLive:
				switch ci.column.GetConcreteDataTypeID() {
				case datatype.String.ID:
					return *ur.varcharUniqueValue, nil
				case datatype.Integer.ID:
					return *ur.intUniqueValue, nil
				case datatype.UUID.ID:
					return *ur.uuidUniqueValue, nil
				}
			case column.DataLifeCycleStateSoftDeleted:
				switch ci.column.GetConcreteDataTypeID() {
				case datatype.String.ID:
					return *ur.varcharValue, nil
				case datatype.Integer.ID:
					return *ur.intValue, nil
				case datatype.UUID.ID:
					return *ur.uuidValue, nil
				}
			}
		}
	}

	return "",
		ucerr.Errorf(
			"unsupported data life cycle state '%v', column data_type_id/index_type/is_array: '%v/%v/%v'",
			ci.dlcs,
			ci.column.DataTypeID,
			ci.column.IndexType,
			ci.column.IsArray,
		)
}

func (ci columnInfo) isSystemColumn() bool {
	return ci.column.Attributes.System
}

func (ci columnInfo) useArray() bool {
	return ci.dlcs == column.DataLifeCycleStateSoftDeleted || ci.column.IsArray
}
