package validation

import (
	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

// RetentionDurationsResponseValidator is used to validate a retentions durations response
type RetentionDurationsResponseValidator struct {
	retentionDurationValidator
}

// NewRetentionDurationsResponseValidator creates a new validator for the specified data lifecycle state
func NewRetentionDurationsResponseValidator(dlcs column.DataLifeCycleState) RetentionDurationsResponseValidator {
	return RetentionDurationsResponseValidator{
		retentionDurationValidator{dlcs: dlcs},
	}
}

// ValidateColumnResponse will validate a retention durations response for a column
func (v RetentionDurationsResponseValidator) ValidateColumnResponse(
	columnID uuid.UUID,
	maxDuration idp.RetentionDuration,
	rds ...idp.ColumnRetentionDuration,
) error {
	uniquePurposeIDs := set.NewUUIDSet()
	for _, rd := range rds {
		if rd.ColumnID != columnID {
			return ucerr.Friendlyf(nil, "response retention duration ColumnID '%v' does not match expected '%v'", rd.ColumnID, columnID)
		}

		if rd.PurposeID.IsNil() {
			return ucerr.Friendlyf(nil, "response retention duration PurposeID must not be nil")
		}

		if uniquePurposeIDs.Contains(rd.PurposeID) {
			return ucerr.Friendlyf(nil, "response retention duration PurposeID '%v' is not unique", rd.PurposeID)
		}
		uniquePurposeIDs.Insert(rd.PurposeID)

		if rd.PurposeName == nil || *rd.PurposeName == "" {
			return ucerr.Friendlyf(nil, "response retention duration must have a PurposeName")
		}
	}

	return ucerr.Wrap(v.validateResponse(maxDuration, rds...))
}

// ValidatePurposeResponse will validate a retention duration response for a purpose
func (v RetentionDurationsResponseValidator) ValidatePurposeResponse(
	purposeID uuid.UUID,
	maxDuration idp.RetentionDuration,
	rd idp.ColumnRetentionDuration,
) error {
	if rd.ColumnID != uuid.Nil {
		return ucerr.Friendlyf(nil, "response retention duration ColumnID must be nil")
	}

	if rd.PurposeID != purposeID {
		return ucerr.Friendlyf(nil, "response retention duration PurposeID '%v' does not match expected '%v'", rd.PurposeID, purposeID)
	}

	if rd.PurposeName == nil || *rd.PurposeName == "" {
		return ucerr.Friendlyf(nil, "response retention duration must have a valid PurposeName")
	}

	return ucerr.Wrap(v.validateResponse(maxDuration, rd))
}

// ValidateTenantResponse will validate a retention duration response for the tenant
func (v RetentionDurationsResponseValidator) ValidateTenantResponse(
	maxDuration idp.RetentionDuration,
	rd idp.ColumnRetentionDuration,
) error {
	if rd.ColumnID != uuid.Nil || rd.PurposeID != uuid.Nil {
		return ucerr.Friendlyf(nil, "response retention duration ColumnID and PurposeID must be nil")
	}

	if rd.PurposeName != nil {
		return ucerr.Friendlyf(nil, "response retention duration must have an empty PurposeName")
	}

	return ucerr.Wrap(v.validateResponse(maxDuration, rd))
}

func (v RetentionDurationsResponseValidator) validateResponse(
	maxDuration idp.RetentionDuration,
	retentionDurations ...idp.ColumnRetentionDuration,
) error {
	if len(retentionDurations) == 0 {
		return ucerr.Friendlyf(nil, "response has no retention durations")
	}

	if err := maxDuration.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	uniqueDurationIDs := set.NewUUIDSet()
	for _, rd := range retentionDurations {
		if column.DataLifeCycleStateFromClient(rd.DurationType) != v.dlcs {
			return ucerr.Friendlyf(nil, "response retention duration DurationType must be %v", v.dlcs)
		}

		if rd.DefaultDuration == nil {
			return ucerr.Friendlyf(nil, "response retention duration DefaultDuration must be specified")
		}

		if err := v.validateRetentionDuration(*rd.DefaultDuration); err != nil {
			return ucerr.Wrap(err)
		}

		if maxDuration.LessThan(*rd.DefaultDuration) {
			return ucerr.Friendlyf(nil, "response MaxDuration '%v' cannot be less than retention duration DefaultDuration '%v'", maxDuration, rd.DefaultDuration)
		}

		if rd.UseDefault {
			if rd.ID != uuid.Nil {
				return ucerr.Friendlyf(nil, "response retention duration ID must be nil")
			}

			if rd.Duration != *rd.DefaultDuration {
				return ucerr.Friendlyf(nil, "response retention duration Duration must equal DefaultDuration")
			}
		} else {
			if rd.ID.IsNil() {
				return ucerr.Friendlyf(nil, "response retention duration ID must not be nil")
			}

			if uniqueDurationIDs.Contains(rd.ID) {
				return ucerr.Friendlyf(nil, "response retention duration ID '%v' is not unique", rd.ID)
			}
			uniqueDurationIDs.Insert(rd.ID)

			if err := v.validateRetentionDuration(rd.Duration); err != nil {
				return ucerr.Wrap(err)
			}

			if maxDuration.LessThan(rd.Duration) {
				return ucerr.Friendlyf(nil, "response MaxDuration '%v' cannot be less than retention duration Duration '%v'", maxDuration, rd.Duration)
			}
		}
	}

	return nil
}
