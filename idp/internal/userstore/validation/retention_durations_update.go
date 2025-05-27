package validation

import (
	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

// RetentionDurationsUpdateValidator is used to validate a retentions durations update request
type RetentionDurationsUpdateValidator struct {
	retentionDurationValidator
}

// NewRetentionDurationsUpdateValidator creates a new validator for the specified data lifecycle state
func NewRetentionDurationsUpdateValidator(dlcs column.DataLifeCycleState) RetentionDurationsUpdateValidator {
	return RetentionDurationsUpdateValidator{
		retentionDurationValidator{dlcs: dlcs},
	}
}

// ValidateColumnUpdate will validate a retention durations update request for a column
func (v RetentionDurationsUpdateValidator) ValidateColumnUpdate(
	columnID uuid.UUID,
	durationID uuid.UUID,
	durationIDExpected bool,
	rds ...idp.ColumnRetentionDuration,
) error {
	if err := v.validateUpdate(durationID, durationIDExpected, rds...); err != nil {
		return ucerr.Wrap(err)
	}

	uniqueDurationIDs := set.NewUUIDSet()
	uniquePurposeIDs := set.NewUUIDSet()
	for _, rd := range rds {
		if rd.ID != uuid.Nil {
			if uniqueDurationIDs.Contains(rd.ID) {
				return ucerr.Friendlyf(nil, "column update retention duration ID '%v' is not unique", rd.ID)
			}
			uniqueDurationIDs.Insert(rd.ID)
		}

		if rd.ColumnID != columnID {
			return ucerr.Friendlyf(nil, "column update retention duration ColumnID '%v' does not match expected '%v'", rd.ColumnID, columnID)
		}

		if rd.PurposeID.IsNil() {
			return ucerr.Friendlyf(nil, "column update retention duration PurposeID must not be nil")
		}

		if uniquePurposeIDs.Contains(rd.PurposeID) {
			return ucerr.Friendlyf(nil, "column update retention duration PurposeID '%v' is not unique", rd.PurposeID)
		}
		uniquePurposeIDs.Insert(rd.PurposeID)
	}

	return nil
}

// ValidatePurposeUpdate will validate a retention duration update request for a purpose
func (v RetentionDurationsUpdateValidator) ValidatePurposeUpdate(
	purposeID uuid.UUID,
	durationID uuid.UUID,
	durationIDExpected bool,
	rd idp.ColumnRetentionDuration,
) error {
	if err := v.validateUpdate(durationID, durationIDExpected, rd); err != nil {
		return ucerr.Wrap(err)
	}

	if durationID != rd.ID {
		return ucerr.Friendlyf(nil, "purpose update retention duration has unexpected ID '%v'", rd.ID)
	}

	if rd.ColumnID != uuid.Nil {
		return ucerr.Friendlyf(nil, "purpose update retention duration must have nil ColumnID")
	}

	if rd.PurposeID != purposeID {
		return ucerr.Friendlyf(nil, "purpose update retention duration PurposeID '%v' does not match expected '%v'", rd.PurposeID, purposeID)
	}

	if rd.UseDefault {
		return ucerr.Friendlyf(nil, "purpose update retention duration UseDefault must be false")
	}

	return nil
}

// ValidateTenantUpdate will validate a retention durations update request for the tenant
func (v RetentionDurationsUpdateValidator) ValidateTenantUpdate(
	durationID uuid.UUID,
	durationIDExpected bool,
	rd idp.ColumnRetentionDuration,
) error {
	if err := v.validateUpdate(durationID, durationIDExpected, rd); err != nil {
		return ucerr.Wrap(err)
	}

	if durationID != rd.ID {
		return ucerr.Friendlyf(nil, "tenant update retention duration has unexpected ID '%v'", rd.ID)
	}

	if rd.ColumnID != uuid.Nil || rd.PurposeID != uuid.Nil {
		return ucerr.Friendlyf(nil, "tenant update retention duration must have nil ColumnID and PurposeID")
	}

	if rd.UseDefault {
		return ucerr.Friendlyf(nil, "tenant update retention duration UseDefault must be false")
	}

	return nil
}

func (v RetentionDurationsUpdateValidator) validateUpdate(
	durationID uuid.UUID,
	durationIDExpected bool,
	rds ...idp.ColumnRetentionDuration,
) error {
	if len(rds) == 0 {
		return ucerr.Friendlyf(nil, "update must have at least one retention duration")
	}

	if err := ValidateDurationID(durationID, durationIDExpected); err != nil {
		return ucerr.Wrap(err)
	}

	if durationID != uuid.Nil {
		if len(rds) != 1 {
			return ucerr.Friendlyf(nil, "update must have exactly one retention duration")
		}

		if rds[0].ID != durationID {
			return ucerr.Friendlyf(nil, "update retention duration ID '%v' does not match '%v'", rds[0].ID, durationID)
		}

		if rds[0].UseDefault {
			return ucerr.Friendlyf(nil, "update retention duration UseDefault must be false")
		}
	}

	for _, rd := range rds {
		if column.DataLifeCycleStateFromClient(rd.DurationType) != v.dlcs {
			return ucerr.Friendlyf(nil, "update retention duration DurationType must be %v", v.dlcs)
		}

		if !rd.UseDefault {
			if err := v.validateRetentionDuration(rd.Duration); err != nil {
				return ucerr.Wrap(err)
			}

			if storage.RetentionDurationMax.LessThan(rd.Duration) {
				return ucerr.Friendlyf(nil, "update retention duration Duration '%v' cannot be greater than '%v'", rd.Duration, storage.RetentionDurationMax)
			}
		}
	}

	return nil
}
