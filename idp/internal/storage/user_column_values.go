package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"

	"userclouds.com/idp/config"
	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucdb/errorcode"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/infra/uctypes/timestamparray"
	ucuuid "userclouds.com/infra/uctypes/uuid"
	"userclouds.com/infra/uctypes/uuidarray"
)

const userColumnValuesPerBatch = 100

// ColumnValueType is an enum indicating the sql value type of a column value
type ColumnValueType int

const (
	columnValueTypeUnknown       ColumnValueType = 0
	columnValueTypeVarchar       ColumnValueType = 1
	columnValueTypeUniqueVarchar ColumnValueType = 2
	columnValueTypeBoolean       ColumnValueType = 3
	columnValueTypeInteger       ColumnValueType = 4
	columnValueTypeUniqueInteger ColumnValueType = 5
	columnValueTypeTimestamp     ColumnValueType = 6
	columnValueTypeUUID          ColumnValueType = 7
	columnValueTypeUniqueUUID    ColumnValueType = 8
	columnValueTypeJSONB         ColumnValueType = 9
)

// BaseUserColumnValue common parameters that apply both for a live and soft-deleted user column value row
type BaseUserColumnValue struct {
	ucdb.VersionBaseModel
	ValueType           ColumnValueType               `db:"value_type"`
	ColumnID            uuid.UUID                     `db:"column_id" validate:"notnil"`
	Column              *Column                       `db:"-"`
	IsNew               bool                          `db:"-"`
	UserID              uuid.UUID                     `db:"user_id" validate:"notnil"`
	Ordering            int                           `db:"ordering"`
	ConsentedPurposeIDs uuidarray.UUIDArray           `db:"consented_purpose_ids" validate:"skip"`
	RetentionTimeouts   timestamparray.TimestampArray `db:"retention_timeouts" validate:"skip"`
}

func (v BaseUserColumnValue) extraValidate() error {
	if len(v.ConsentedPurposeIDs) == 0 || len(v.ConsentedPurposeIDs) != len(v.RetentionTimeouts) {
		return ucerr.New("ConsentedPurposeIDs and RetentionTimeouts must be non-empty and equal in length")
	}

	consentedPurposeIDSet := set.NewUUIDSet(v.ConsentedPurposeIDs...)
	if consentedPurposeIDSet.Contains(uuid.Nil) {
		return ucerr.New("ConsentedPurposeIDs contains a nil PurposeID")
	}

	if consentedPurposeIDSet.Size() != len(v.ConsentedPurposeIDs) {
		return ucerr.New("ConsentedPurposeIDs contains duplicate entries")
	}

	return nil
}

//go:generate genvalidate BaseUserColumnValue

func (v *BaseUserColumnValue) apply(valueID uuid.UUID, userID uuid.UUID, c *Column, ccv *ColumnConsentedValue) {
	if valueID != uuid.Nil {
		v.VersionBaseModel = ucdb.NewVersionBaseWithID(valueID)
		v.Version = ccv.Version
		v.IsNew = false
	} else {
		v.VersionBaseModel = ucdb.NewVersionBase()
		v.IsNew = true
	}
	v.ColumnID = c.ID
	v.Column = c
	v.UserID = userID
	v.Ordering = ccv.Ordering

	for _, consentedPurpose := range ccv.ConsentedPurposes {
		v.ConsentedPurposeIDs = append(v.ConsentedPurposeIDs, consentedPurpose.Purpose)
		v.RetentionTimeouts = append(v.RetentionTimeouts, consentedPurpose.RetentionTimeout)
	}
}

func (v *BaseUserColumnValue) hasPurpose(purposeID uuid.UUID) bool {
	return slices.Contains(v.ConsentedPurposeIDs, purposeID)
}

// UserColumnLiveValue represents the parameters that apply for a live user column value row
type UserColumnLiveValue struct {
	BaseUserColumnValue
	VarcharValue       *string    `db:"varchar_value" validate:"allownil"`
	VarcharUniqueValue *string    `db:"varchar_unique_value" validate:"allownil"`
	BooleanValue       *bool      `db:"boolean_value" validate:"allownil"`
	IntValue           *int       `db:"int_value" validate:"allownil"`
	IntUniqueValue     *int       `db:"int_unique_value" validate:"allownil"`
	TimestampValue     *time.Time `db:"timestamp_value" validate:"allownil"`
	UUIDValue          *uuid.UUID `db:"uuid_value" validate:"allownil"`
	UUIDUniqueValue    *uuid.UUID `db:"uuid_unique_value" validate:"allownil"`
	JSONBValue         any        `db:"jsonb_value" validate:"allownil"`
}

//go:generate genvalidate UserColumnLiveValue

// NewUserColumnLiveValue creates a new UserColumnLiveValue for the ColumnConsentedValue
func NewUserColumnLiveValue(
	userID uuid.UUID,
	c *Column,
	ccv *ColumnConsentedValue,
) (*UserColumnLiveValue, error) {
	var v UserColumnLiveValue
	v.BaseUserColumnValue.apply(ccv.ID, userID, c, ccv)

	ok := false
	switch c.IndexType {
	case columnIndexTypeNone, columnIndexTypeIndexed:
		switch c.GetConcreteDataTypeID() {
		case datatype.Boolean.ID:
			var booleanVal bool
			if booleanVal, ok = ccv.Value.(bool); ok {
				v.ValueType = columnValueTypeBoolean
				v.BooleanValue = &booleanVal
			}
		case datatype.Integer.ID:
			var intVal int
			if intVal, ok = ccv.Value.(int); ok {
				v.ValueType = columnValueTypeInteger
				v.IntValue = &intVal
			}
		case datatype.String.ID:
			var varcharVal string
			if varcharVal, ok = ccv.Value.(string); ok {
				v.ValueType = columnValueTypeVarchar
				v.VarcharValue = &varcharVal
			}
		case datatype.Timestamp.ID, datatype.Date.ID:
			var timestampVal time.Time
			if timestampVal, ok = ccv.Value.(time.Time); ok {
				v.ValueType = columnValueTypeTimestamp
				v.TimestampValue = &timestampVal
			}
		case datatype.UUID.ID:
			var uuidVal uuid.UUID
			if uuidVal, ok = ccv.Value.(uuid.UUID); ok {
				v.ValueType = columnValueTypeUUID
				v.UUIDValue = &uuidVal
			}
		case datatype.Composite.ID:
			var compositeVal userstore.CompositeValue
			if compositeVal, ok = ccv.Value.(userstore.CompositeValue); ok {
				v.ValueType = columnValueTypeJSONB
				v.JSONBValue = &compositeVal
			}
		}
	case columnIndexTypeUnique:
		switch c.GetConcreteDataTypeID() {
		case datatype.Integer.ID:
			var intUniqueVal int
			if intUniqueVal, ok = ccv.Value.(int); ok {
				v.ValueType = columnValueTypeUniqueInteger
				v.IntUniqueValue = &intUniqueVal
			}
		case datatype.String.ID:
			var varcharUniqueVal string
			if varcharUniqueVal, ok = ccv.Value.(string); ok {
				v.ValueType = columnValueTypeUniqueVarchar
				v.VarcharUniqueValue = &varcharUniqueVal
			}
		case datatype.UUID.ID:
			var uuidUniqueVal uuid.UUID
			if uuidUniqueVal, ok = ccv.Value.(uuid.UUID); ok {
				v.ValueType = columnValueTypeUniqueUUID
				v.UUIDUniqueValue = &uuidUniqueVal
			}
		}
	}

	if !ok {
		return nil, ucerr.Errorf(
			"unsupported column data type/index type/value: '%v/%v/%v'",
			c.DataTypeID,
			c.IndexType,
			ccv.Value,
		)
	}

	return &v, nil
}

// UserColumnLiveValues is a collection of live values
type UserColumnLiveValues []UserColumnLiveValue

func (uclv UserColumnLiveValues) getExternalAliasValue() *UserColumnLiveValue {
	for i := range uclv {
		if uclv[i].Column.ID == column.ExternalAliasColumnID {
			return &uclv[i]
		}
	}

	return nil
}

// NB: the quotes around the column_id value are optional because some dbs include them, but postgres does not
var conflictMatcher = regexp.MustCompile(fmt.Sprintf(`Key \(column_id, ?\w+\)=\('?(%s)'?, ?(.+)\) already exists.`, ucuuid.UUIDPattern))

func (uclv UserColumnLiveValues) classifyError(err error) error {
	var pqError *pq.Error
	if ok := errors.As(err, &pqError); ok {
		if pqError.Code == errorcode.UniqueViolation() {
			matches := conflictMatcher.FindStringSubmatch(pqError.Detail)
			if len(matches) == 3 && matches[0] == pqError.Detail {
				columnID := matches[1]
				value := matches[2]
				if !strings.HasPrefix(value, `'`) {
					value = fmt.Sprintf(`'%s'`, value)
				}
				for i := range uclv {
					if uclv[i].Column.ID.String() == columnID {
						return ucerr.WrapWithFriendlyStructure(
							err,
							jsonclient.SDKStructuredError{
								Error: fmt.Sprintf("user '%v' cannot update unique column '%s' - value %s is already in use",
									uclv[i].UserID,
									uclv[i].Column.Name,
									value,
								),
								ID:          uclv[i].UserID,
								SecondaryID: uclv[i].Column.ID,
							},
						)
					}
				}
			} else {
				userIDs := set.NewUUIDSet()
				for _, update := range uclv {
					userIDs.Insert(update.UserID)
				}

				if userIDs.Size() == 1 {
					return ucerr.Friendlyf(err, "user '%v' update failed due to uniqueness constraint: '%s'", uclv[0].UserID, pqError.Detail)
				}

				return ucerr.Friendlyf(err, "user update failed due to uniqueness constraint: '%s'", pqError.Detail)
			}
		}
	}

	return ucerr.Wrap(err)
}

// UserColumnSoftDeletedValue represents the parameters that apply for a soft-deleted user column value row
type UserColumnSoftDeletedValue struct {
	BaseUserColumnValue
	VarcharValue   *string    `db:"varchar_value" validate:"allownil"`
	BooleanValue   *bool      `db:"boolean_value" validate:"allownil"`
	IntValue       *int       `db:"int_value" validate:"allownil"`
	TimestampValue *time.Time `db:"timestamp_value" validate:"allownil"`
	UUIDValue      *uuid.UUID `db:"uuid_value" validate:"allownil"`
	JSONBValue     any        `db:"jsonb_value" validate:"allownil"`
}

func (v *UserColumnSoftDeletedValue) extraValidate() error {
	if slices.Contains(v.RetentionTimeouts, userstore.GetRetentionTimeoutIndefinite()) {
		return ucerr.New("soft-deleted value purposes cannot be retained indefinitely")
	}

	return nil
}

//go:generate genvalidate UserColumnSoftDeletedValue

// NewUserColumnSoftDeletedValue creates a new UserColumnSoftDeletedValue for the ColumnConsentedValue
func NewUserColumnSoftDeletedValue(
	userID uuid.UUID,
	c *Column,
	ccv *ColumnConsentedValue,
) (*UserColumnSoftDeletedValue, error) {
	var v UserColumnSoftDeletedValue
	// we always create a new soft-deleted value
	v.BaseUserColumnValue.apply(uuid.Nil, userID, c, ccv)

	ok := false
	switch c.IndexType {
	case columnIndexTypeNone, columnIndexTypeIndexed:
		switch c.GetConcreteDataTypeID() {
		case datatype.Boolean.ID:
			var booleanVal bool
			if booleanVal, ok = ccv.Value.(bool); ok {
				v.ValueType = columnValueTypeBoolean
				v.BooleanValue = &booleanVal
			}
		case datatype.Integer.ID:
			var intVal int
			if intVal, ok = ccv.Value.(int); ok {
				v.ValueType = columnValueTypeInteger
				v.IntValue = &intVal
			}
		case datatype.String.ID:
			var varcharVal string
			if varcharVal, ok = ccv.Value.(string); ok {
				v.ValueType = columnValueTypeVarchar
				v.VarcharValue = &varcharVal
			}
		case datatype.Timestamp.ID, datatype.Date.ID:
			var timestampVal time.Time
			if timestampVal, ok = ccv.Value.(time.Time); ok {
				v.ValueType = columnValueTypeTimestamp
				v.TimestampValue = &timestampVal
			}
		case datatype.UUID.ID:
			var uuidVal uuid.UUID
			if uuidVal, ok = ccv.Value.(uuid.UUID); ok {
				v.ValueType = columnValueTypeUUID
				v.UUIDValue = &uuidVal
			}
		case datatype.Composite.ID:
			var compositeVal userstore.CompositeValue
			if compositeVal, ok = ccv.Value.(userstore.CompositeValue); ok {
				v.ValueType = columnValueTypeJSONB
				v.JSONBValue = &compositeVal
			}
		}
	case columnIndexTypeUnique:
		switch c.GetConcreteDataTypeID() {
		case datatype.Integer.ID:
			var intVal int
			if intVal, ok = ccv.Value.(int); ok {
				v.ValueType = columnValueTypeInteger
				v.IntValue = &intVal
			}
		case datatype.String.ID:
			var varcharVal string
			if varcharVal, ok = ccv.Value.(string); ok {
				v.ValueType = columnValueTypeVarchar
				v.VarcharValue = &varcharVal
			}
		case datatype.UUID.ID:
			var uuidVal uuid.UUID
			if uuidVal, ok = ccv.Value.(uuid.UUID); ok {
				v.ValueType = columnValueTypeUUID
				v.UUIDValue = &uuidVal
			}
		}
	}

	if !ok {
		return nil, ucerr.Errorf(
			"unsupported column data type/index type/value: '%v/%v/%v'",
			c.DataTypeID,
			c.IndexType,
			ccv.Value,
		)
	}

	return &v, nil
}

// DeleteUserColumnLiveValues deletes a collection of user column live values
func (s *UserStorage) DeleteUserColumnLiveValues(ctx context.Context, values UserColumnLiveValues) error {
	if len(values) == 0 {
		return nil
	}

	deletedIDs := make(uuidarray.UUIDArray, len(values))

	for i, v := range values {
		deletedIDs[i] = v.ID
	}

	const q = "DELETE FROM user_column_pre_delete_values WHERE id = ANY($1);"

	if _, err := s.db.ExecContext(ctx, "DeleteUserColumnLiveValues", q, deletedIDs); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}
	if externalAliasValue := values.getExternalAliasValue(); externalAliasValue != nil {
		if externalAliasValue.hasPurpose(constants.OperationalPurposeID) {
			if err := s.DeleteAlias(ctx, externalAliasValue.UserID); err != nil {
				return ucerr.Wrap(err)
			}
		}
	}

	return nil
}

const insertUserColumnLiveValuesQueryTemplate = `
/* lint-sql-ok */
INSERT INTO user_column_pre_delete_values (
id,
updated,
deleted,
_version,
value_type,
column_id,
user_id,
consented_purpose_ids,
retention_timeouts,
ordering,
varchar_value,
varchar_unique_value,
boolean_value,
int_value,
int_unique_value,
timestamp_value,
uuid_value,
uuid_unique_value,
jsonb_value
)
VALUES %s;
`

var insertUserColumnLiveValuesParamTypes = []string{
	"UUID",
	"TIMESTAMP",
	"TIMESTAMP",
	"INT",
	"INT",
	"UUID",
	"UUID",
	"UUID[]",
	"VARCHAR[]",
	"INT",
	"VARCHAR",
	"VARCHAR",
	"BOOL",
	"INT",
	"INT",
	"TIMESTAMP",
	"UUID",
	"UUID",
	"JSONB",
}

// InsertUserColumnLiveValues inserts a collection of user column live values
func (s *UserStorage) InsertUserColumnLiveValues(
	ctx context.Context,
	cm *ColumnManager,
	sim *SearchIndexManager,
	searchUpdateCfg *config.SearchUpdateConfig,
	values UserColumnLiveValues,
) error {
	if len(values) == 0 {
		return nil
	}

	bu, err := ucdb.NewBatchUpdater(userColumnValuesPerBatch, insertUserColumnLiveValuesParamTypes)
	if err != nil {
		return ucerr.Wrap(err)
	}

	su, err := NewSearchUpdater(ctx, s, cm, sim, searchUpdateCfg, false)
	if err != nil {
		return ucerr.Wrap(err)
	}

	updateTime := time.Now().UTC()

	for _, v := range values {
		v.Updated = updateTime
		v.Version = 1

		if err := v.Validate(); err != nil {
			return ucerr.Wrap(err)
		}

		if err := bu.ScheduleUpdate(
			v.ID,
			v.Updated,
			v.Deleted,
			v.Version,
			v.ValueType,
			v.ColumnID,
			v.UserID,
			v.ConsentedPurposeIDs,
			v.RetentionTimeouts,
			v.Ordering,
			v.VarcharValue,
			v.VarcharUniqueValue,
			v.BooleanValue,
			v.IntValue,
			v.IntUniqueValue,
			v.TimestampValue,
			v.UUIDValue,
			v.UUIDUniqueValue,
			v.JSONBValue,
		); err != nil {
			return ucerr.Wrap(err)
		}

		if err := su.ProcessColumnValue(ctx, v); err != nil {
			return ucerr.Wrap(err)
		}
	}

	if err := bu.ApplyUpdates(ctx, s.db, "InsertUserColumnLiveValues", insertUserColumnLiveValuesQueryTemplate); err != nil {
		return ucerr.Wrap(values.classifyError(err))
	}

	if externalAliasValue := values.getExternalAliasValue(); externalAliasValue != nil {
		if externalAliasValue.hasPurpose(constants.OperationalPurposeID) {
			if err := s.SaveAlias(ctx, externalAliasValue.UserID, *externalAliasValue.VarcharUniqueValue); err != nil {
				return ucerr.Wrap(err)
			}
		}
	}
	su.SendUpdatesIfNeeded(ctx)
	return nil
}

const updateUserColumnLiveValuesQueryTemplate = `
UPDATE
user_column_pre_delete_values
SET
updated = v.updated,
_version = v._version,
consented_purpose_ids = v.consented_purpose_ids,
retention_timeouts = v.retention_timeouts,
ordering = v.ordering,
varchar_value = v.varchar_value,
varchar_unique_value = v.varchar_unique_value,
boolean_value = v.boolean_value,
int_value = v.int_value,
int_unique_value = v.int_unique_value,
timestamp_value = v.timestamp_value,
uuid_value = v.uuid_value,
uuid_unique_value = v.uuid_unique_value,
jsonb_value = v.jsonb_value
FROM
(VALUES %s)
AS v (
id,
updated,
_version,
consented_purpose_ids,
retention_timeouts,
ordering,
varchar_value,
varchar_unique_value,
boolean_value,
int_value,
int_unique_value,
timestamp_value,
uuid_value,
uuid_unique_value,
jsonb_value,
current_version
)
WHERE user_column_pre_delete_values.id = v.id
AND user_column_pre_delete_values._version = v.current_version;
`

var updateUserColumnLiveValuesParamTypes = []string{
	"UUID",
	"TIMESTAMP",
	"INT",
	"UUID[]",
	"VARCHAR[]",
	"INT",
	"VARCHAR",
	"VARCHAR",
	"BOOL",
	"INT",
	"INT",
	"TIMESTAMP",
	"UUID",
	"UUID",
	"JSONB",
	"INT",
}

// UpdateUserColumnLiveValues updates a collection of user column live values
func (s *UserStorage) UpdateUserColumnLiveValues(ctx context.Context, values UserColumnLiveValues) error {
	if len(values) == 0 {
		return nil
	}

	bu, err := ucdb.NewBatchUpdater(userColumnValuesPerBatch, updateUserColumnLiveValuesParamTypes)
	if err != nil {
		return ucerr.Wrap(err)
	}

	updateTime := time.Now().UTC()

	for _, v := range values {
		currentVersion := v.Version

		v.Updated = updateTime
		v.Version++

		if err := v.Validate(); err != nil {
			return ucerr.Wrap(err)
		}

		if err := bu.ScheduleUpdate(
			v.ID,
			v.Updated,
			v.Version,
			v.ConsentedPurposeIDs,
			v.RetentionTimeouts,
			v.Ordering,
			v.VarcharValue,
			v.VarcharUniqueValue,
			v.BooleanValue,
			v.IntValue,
			v.IntUniqueValue,
			v.TimestampValue,
			v.UUIDValue,
			v.UUIDUniqueValue,
			v.JSONBValue,
			currentVersion,
		); err != nil {
			return ucerr.Wrap(err)
		}
	}

	if err := bu.ApplyUpdates(ctx, s.db, "UpdateUserColumnLiveValues", updateUserColumnLiveValuesQueryTemplate); err != nil {
		return ucerr.Wrap(values.classifyError(err))
	}

	if externalAliasValue := values.getExternalAliasValue(); externalAliasValue != nil {
		if externalAliasValue.hasPurpose(constants.OperationalPurposeID) {
			if err := s.SaveAlias(ctx, externalAliasValue.UserID, *externalAliasValue.VarcharUniqueValue); err != nil {
				return ucerr.Wrap(err)
			}
		} else if err := s.DeleteAlias(ctx, externalAliasValue.UserID); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

// DeleteUserColumnSoftDeletedValues deletes a collection of user column soft-deleted values
func (s *UserStorage) DeleteUserColumnSoftDeletedValues(ctx context.Context, values []UserColumnSoftDeletedValue) error {
	if len(values) == 0 {
		return nil
	}

	var deletedIDs uuidarray.UUIDArray

	for _, value := range values {
		deletedIDs = append(deletedIDs, value.ID)
	}

	const q = "DELETE FROM user_column_post_delete_values WHERE id = ANY($1);"

	if _, err := s.db.ExecContext(ctx, "DeleteUserColumnSoftDeletedValues", q, deletedIDs); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}
	return nil
}

const insertUserColumnSoftDeletedValuesQueryTemplate = `
/* lint-sql-ok */
INSERT INTO user_column_post_delete_values (
id,
updated,
deleted,
_version,
value_type,
column_id,
user_id,
consented_purpose_ids,
retention_timeouts,
ordering,
varchar_value,
boolean_value,
int_value,
timestamp_value,
uuid_value,
jsonb_value
)
VALUES %s;
`

var insertUserColumnSoftDeletedValuesParamTypes = []string{
	"UUID",
	"TIMESTAMP",
	"TIMESTAMP",
	"INT",
	"INT",
	"UUID",
	"UUID",
	"UUID[]",
	"VARCHAR[]",
	"INT",
	"VARCHAR",
	"BOOL",
	"INT",
	"TIMESTAMP",
	"UUID",
	"JSONB",
}

// InsertUserColumnSoftDeletedValues inserts a collection of user column soft-deleted values
func (s *UserStorage) InsertUserColumnSoftDeletedValues(ctx context.Context, values []UserColumnSoftDeletedValue) error {
	// TODO: should we use an async job for this?
	if len(values) == 0 {
		return nil
	}

	bu, err := ucdb.NewBatchUpdater(userColumnValuesPerBatch, insertUserColumnSoftDeletedValuesParamTypes)
	if err != nil {
		return ucerr.Wrap(err)
	}

	updateTime := time.Now().UTC()

	for _, v := range values {
		v.Updated = updateTime
		v.Version = 1

		if err := v.Validate(); err != nil {
			return ucerr.Wrap(err)
		}

		if err := bu.ScheduleUpdate(
			v.ID,
			v.Updated,
			v.Deleted,
			v.Version,
			v.ValueType,
			v.ColumnID,
			v.UserID,
			v.ConsentedPurposeIDs,
			v.RetentionTimeouts,
			v.Ordering,
			v.VarcharValue,
			v.BooleanValue,
			v.IntValue,
			v.TimestampValue,
			v.UUIDValue,
			v.JSONBValue,
		); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return ucerr.Wrap(bu.ApplyUpdates(ctx, s.db, "InsertUserColumnSoftDeletedValues", insertUserColumnSoftDeletedValuesQueryTemplate))
}

const updateUserColumnSoftDeletedValuesQueryTemplate = `
UPDATE
user_column_post_delete_values
SET
updated = v.updated,
_version = v._version,
consented_purpose_ids = v.consented_purpose_ids,
retention_timeouts = v.retention_timeouts,
ordering = v.ordering,
varchar_value = v.varchar_value,
boolean_value = v.boolean_value,
int_value = v.int_value,
timestamp_value = v.timestamp_value,
uuid_value = v.uuid_value,
jsonb_value = v.jsonb_value
FROM
(VALUES %s)
AS v (
id,
updated,
_version,
consented_purpose_ids,
retention_timeouts,
ordering,
varchar_value,
boolean_value,
int_value,
timestamp_value,
uuid_value,
jsonb_value,
current_version
)
WHERE user_column_post_delete_values.id = v.id
AND user_column_post_delete_values._version = v.current_version;
`

var updateUserColumnSoftDeletedValuesParamTypes = []string{
	"UUID",
	"TIMESTAMP",
	"INT",
	"UUID[]",
	"VARCHAR[]",
	"INT",
	"VARCHAR",
	"BOOL",
	"INT",
	"TIMESTAMP",
	"UUID",
	"JSONB",
	"INT",
}

// UpdateUserColumnSoftDeletedValues updates a collection of user column soft-deleted values
func (s *UserStorage) UpdateUserColumnSoftDeletedValues(ctx context.Context, values []UserColumnSoftDeletedValue) error {
	if len(values) == 0 {
		return nil
	}

	bu, err := ucdb.NewBatchUpdater(userColumnValuesPerBatch, updateUserColumnSoftDeletedValuesParamTypes)
	if err != nil {
		return ucerr.Wrap(err)
	}

	updateTime := time.Now().UTC()

	for _, v := range values {
		currentVersion := v.Version

		v.Updated = updateTime
		v.Version++

		if err := v.Validate(); err != nil {
			return ucerr.Wrap(err)
		}

		if err := bu.ScheduleUpdate(
			v.ID,
			v.Updated,
			v.Version,
			v.ConsentedPurposeIDs,
			v.RetentionTimeouts,
			v.Ordering,
			v.VarcharValue,
			v.BooleanValue,
			v.IntValue,
			v.TimestampValue,
			v.UUIDValue,
			v.JSONBValue,
			currentVersion,
		); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return ucerr.Wrap(bu.ApplyUpdates(ctx, s.db, "UpdateUserColumnSoftDeletedValues", updateUserColumnSoftDeletedValuesQueryTemplate))
}
