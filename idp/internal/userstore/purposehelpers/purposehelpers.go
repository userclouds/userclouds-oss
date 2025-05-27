package purposehelpers

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

// CheckPurposeIDs checks if the given purpose IDs are consented to for the given value, and returns the value if so
func CheckPurposeIDs(
	ctx context.Context,
	c *storage.Column,
	dt *column.DataType,
	purposeIDs set.Set[uuid.UUID],
	value any,
	consentRecord map[string][]storage.ConsentedPurposeIDs,
) (bool, any, error) {
	var consentedValue any

	if c.IsArray {
		if value == nil {
			return true, nil, nil
		}

		// For arrays, if any of the values have a <consented purpose IDs array> containing the <purpose ID>, we consider the purpose met and
		// return the subset of the values that match
		var cv column.Value
		if err := cv.Set(*dt, c.Attributes.Constraints, c.IsArray, value); err != nil {
			return false, nil, ucerr.Wrap(err)
		}

		valuesArray := cv.GetAnyArray(ctx)
		consentsArray := consentRecord[c.Name]

		if len(valuesArray) != len(consentsArray) {
			return false, nil, ucerr.New("array length mismatch between column values and consent records")
		}

		consentedArrayValues := []any{}

		for i := range valuesArray {
			consentedPurposeIDs := set.NewUUIDSet(consentsArray[i]...)
			if consentedPurposeIDs.IsSupersetOf(purposeIDs) {
				consentedArrayValues = append(consentedArrayValues, valuesArray[i])
			}
		}

		if len(consentedArrayValues) == 0 {
			return false, nil, nil
		}

		consentedValue = consentedArrayValues

	} else {
		// For non-array columns, we just check if the <consented purpose IDs array> contains all of the <purpose IDs>

		var consentedPurposeIDs set.Set[uuid.UUID]
		if len(consentRecord[c.Name]) > 0 {
			consentedPurposeIDs = set.NewUUIDSet(consentRecord[c.Name][0]...)
		}

		if !consentedPurposeIDs.IsSupersetOf(purposeIDs) {
			return false, nil, nil
		}

		consentedValue = value
	}

	return true, consentedValue, nil
}
