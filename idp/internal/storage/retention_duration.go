package storage

import (
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// RetentionDurationImmediateDeletion is used when a deleted column
// value should not be retained, and is the default deletion duration.
var RetentionDurationImmediateDeletion = idp.RetentionDuration{Unit: idp.DurationUnitYear, Duration: 0}

// RetentionDurationIndefinite is used when a column value should
// be retained indefinitely, and is the default retention duration.
var RetentionDurationIndefinite = idp.RetentionDuration{Unit: idp.DurationUnitIndefinite, Duration: 0}

// RetentionDurationMax is the maximum specifiable retention duration.
// Since live data can have indefinite retention, it seems appropriate
// to have a maximum allowed specified retention time, which would
// primarily be relevant for deleted data.
var RetentionDurationMax = idp.RetentionDuration{Unit: idp.DurationUnitYear, Duration: 10}

// TODO: should this maximum be configurable by tenant and region?

// GetDefaultRetentionDuration returns the appropriate default retention duration
func GetDefaultRetentionDuration(durationType column.DataLifeCycleState) idp.RetentionDuration {
	if durationType == column.DataLifeCycleStateSoftDeleted {
		return RetentionDurationImmediateDeletion
	}

	return RetentionDurationIndefinite
}

// ColumnValueRetentionDuration represents a configured retention duration for
// a userstore column value purpose.
//
// DurationType signifies whether this is a duration for a live column value
// or for a soft-deleted column value.
//
// Based on whether a column id or purpose id is specified, a retention duration
// can be specified that applies for all column value purposes in a tenant; for
// all column value purposes in a tenant for a specific purpose; or for all column
// value purposes in a tenant for a specific purpose and column. The most specific
// configured duration will be used for a given tenant, column, and purpose.
//
// The order of precedence, from lowest to highest, is:
//
// column_id  purpose_id  description
// ---------  ----------  ---------------------------------------------------------------------
// nil        nil         A configured duration for any column and purpose in the tenant
//
// nil        specified   A configured duration for a specified purpose in the tenant
//
// specified  specified   A configured duration for a specific column and purpose in the tenant
//
// We also have a global default duration, defined in code above, that is used when
// there is no configured duration for a given column and purpose.
type ColumnValueRetentionDuration struct {
	ucdb.VersionBaseModel
	ColumnID     uuid.UUID                 `db:"column_id" json:"column_id"`
	PurposeID    uuid.UUID                 `db:"purpose_id" json:"purpose_id"`
	DurationType column.DataLifeCycleState `db:"duration_type" json:"duration_type"`
	DurationUnit column.DurationUnit       `db:"duration_unit" json:"duration_unit"`
	Duration     int                       `db:"duration" json:"duration"`
}

func (cvrd ColumnValueRetentionDuration) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	switch key {
	case "column_id,purpose_id,id":
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"column_id:%v,purpose_id:%v,id:%v",
				cvrd.ColumnID,
				cvrd.PurposeID,
				cvrd.ID,
			),
		)
	case "purpose_id,id":
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"purpose_id:%v,id:%v",
				cvrd.PurposeID,
				cvrd.ID,
			),
		)
	}
}

func (ColumnValueRetentionDuration) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"column_id":     pagination.UUIDKeyType,
		"purpose_id":    pagination.UUIDKeyType,
		"duration_type": pagination.IntKeyType,
	}
}

//go:generate genpageable ColumnValueRetentionDuration

func (cvrd *ColumnValueRetentionDuration) extraValidate() error {
	if cvrd.ColumnID != uuid.Nil && cvrd.PurposeID.IsNil() {
		return ucerr.New("ColumnID cannot be specified if PurposeID is not specified")
	}

	d := idp.RetentionDuration{Unit: cvrd.DurationUnit.ToClient(), Duration: cvrd.Duration}
	if err := d.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if RetentionDurationMax.LessThan(d) {
		return ucerr.Errorf("Duration cannot be more than %v", RetentionDurationMax)
	}

	return nil
}

//go:generate genvalidate ColumnValueRetentionDuration

//go:generate genorm ColumnValueRetentionDuration --cache --followerreads column_value_retention_durations tenantdb
