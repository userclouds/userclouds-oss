package validation

import (
	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
)

// ValidateDurationID returns true if the duration ID is valid
func ValidateDurationID(
	durationID uuid.UUID,
	durationIDRequired bool,
) error {
	if durationIDRequired && durationID.IsNil() {
		return ucerr.Friendlyf(nil, "duration ID must be non-nil")
	}

	return nil
}
