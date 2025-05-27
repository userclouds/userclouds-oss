package userstore

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/config"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucerr"
)

// userValueUpdater is used to aggregate all changes required for a User
// in response to a set of column mutations. A single userValueUpdater
// should be created for an ExecuteMutator request, and setUser should
// be called for each User that may be affected by the request, with
// applyMutations called to apply each column mutation for the request,
// and finally saveChanges to persist all aggregated changes to the database.

type userValueUpdater struct {
	configStorage            *storage.Storage
	userStorage              *storage.UserStorage
	columnManager            *storage.ColumnManager
	searchIndexManager       *storage.SearchIndexManager
	searchUpdateConfig       *config.SearchUpdateConfig
	baseTime                 time.Time
	rc                       retentionCache
	baseUser                 *storage.BaseUser
	existingValues           storage.ColumnConsentedValues
	liveNewValues            []storage.ColumnConsentedValue
	liveUpdatedValues        map[uuid.UUID]storage.ColumnConsentedValue
	liveRemovedValues        map[uuid.UUID]storage.ColumnConsentedValue
	softDeletedNewValues     map[int][]storage.ColumnConsentedValue
	softDeletedUpdatedValues map[uuid.UUID]storage.ColumnConsentedValue
	softDeletedRemovedValues map[uuid.UUID]storage.ColumnConsentedValue
}

func newUserValueUpdater(
	ctx context.Context,
	configStorage *storage.Storage,
	userStorage *storage.UserStorage,
	searchUpdateConfig *config.SearchUpdateConfig,
	baseTime time.Time,
	tenantID uuid.UUID,
	regionID uuid.UUID,
) (*userValueUpdater, error) {
	columnManager, err := storage.NewUserstoreColumnManager(ctx, configStorage)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	sim, err := storage.NewSearchIndexManager(ctx, configStorage)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	uvu := &userValueUpdater{
		configStorage:      configStorage,
		userStorage:        userStorage,
		columnManager:      columnManager,
		searchIndexManager: sim,
		searchUpdateConfig: searchUpdateConfig,
		baseTime:           baseTime,
		rc:                 newRetentionCache(configStorage, tenantID, regionID, baseTime),
	}
	uvu.setUser(nil)
	return uvu, nil
}

func (uvu *userValueUpdater) setUser(baseUser *storage.BaseUser) {
	uvu.baseUser = baseUser
	uvu.existingValues = storage.ColumnConsentedValues{}
	uvu.liveNewValues = []storage.ColumnConsentedValue{}
	uvu.liveUpdatedValues = map[uuid.UUID]storage.ColumnConsentedValue{}
	uvu.liveRemovedValues = map[uuid.UUID]storage.ColumnConsentedValue{}
	uvu.softDeletedNewValues = map[int][]storage.ColumnConsentedValue{}
	uvu.softDeletedUpdatedValues = map[uuid.UUID]storage.ColumnConsentedValue{}
	uvu.softDeletedRemovedValues = map[uuid.UUID]storage.ColumnConsentedValue{}
}

func (uvu *userValueUpdater) applyMutations(ctx context.Context, user *storage.User, columnMutations []columnMutation) (int, error) {
	uvu.existingValues = user.ColumnValues
	for _, cm := range columnMutations {
		am, code, err := applyMutation(ctx, user, cm)
		if err != nil {
			return code, ucerr.Wrap(err)
		}

		// remove all deleted values, retaining any purposes that have a non-immediate deletion timeout

		for _, removedValue := range am.removedValues {
			uvu.liveRemovedValues[removedValue.ID] = removedValue

			retainedDeletedPurposes := []storage.ConsentedPurpose{}
			for _, removedPurpose := range removedValue.ConsentedPurposes {
				deletionTimeout, err := uvu.rc.getDeletionTimeout(ctx, cm.column.ID, removedPurpose.Purpose)
				if err != nil {
					return http.StatusInternalServerError, ucerr.Wrap(err)
				}

				if deletionTimeout != userstore.GetRetentionTimeoutImmediateDeletion() {
					retainedDeletedPurposes = append(
						retainedDeletedPurposes,
						storage.ConsentedPurpose{
							Purpose:          removedPurpose.Purpose,
							RetentionTimeout: deletionTimeout,
						},
					)
				}
			}

			if len(retainedDeletedPurposes) > 0 {
				values := uvu.softDeletedNewValues[removedValue.Ordering]
				values = append(
					values,
					storage.ColumnConsentedValue{
						ID:                removedValue.ID,
						Version:           removedValue.Version,
						ColumnName:        cm.column.Name,
						Value:             removedValue.Value,
						Ordering:          removedValue.Ordering,
						ConsentedPurposes: retainedDeletedPurposes,
					},
				)
				uvu.softDeletedNewValues[removedValue.Ordering] = values
			}
		}

		// add any new values, applying all updated purposes to the values with appropriate retention timeouts

		if len(am.addedValues) > 0 {
			for _, addedValue := range am.addedValues {
				newValue := addedValue.value
				for _, addedPurpose := range addedValue.purposes.Items() {
					retentionTimeout, err :=
						uvu.rc.getRetentionTimeout(ctx, cm.column.ID, addedPurpose)
					if err != nil {
						return http.StatusInternalServerError, ucerr.Wrap(err)
					}

					newValue.ConsentedPurposes = append(
						newValue.ConsentedPurposes,
						storage.ConsentedPurpose{
							Purpose:          addedPurpose,
							RetentionTimeout: retentionTimeout,
						},
					)
				}

				uvu.liveNewValues = append(uvu.liveNewValues, newValue)
			}
		}

		// update any current values that had their purposes or orderings changed

		for valueID, currentValue := range am.currentValues {
			updatedPurposes := am.updatedPurposesByID[valueID]

			if am.currentPurposesByID[valueID].Equal(updatedPurposes) {
				// current value has no purpose changes, but should still be updated if it has an ordering change

				if am.updatedOrderingValueIDs.Contains(valueID) {
					uvu.liveUpdatedValues[valueID] = currentValue
				}

				continue
			}

			// current value has purpose changes, so we update all current values, adding new purposes and associated
			// retention timeouts if necessary, removing any deleted purposes, and retaining any deleted purposes
			// that have a non-immediate deletion timeout

			consentedPurposes := []storage.ConsentedPurpose{}
			retainedDeletedPurposes := []storage.ConsentedPurpose{}

			for _, currentPurpose := range currentValue.ConsentedPurposes {
				if updatedPurposes.Contains(currentPurpose.Purpose) {
					consentedPurposes = append(consentedPurposes, currentPurpose)
					updatedPurposes.Evict(currentPurpose.Purpose)
				} else {
					deletionTimeout, err := uvu.rc.getDeletionTimeout(ctx, cm.column.ID, currentPurpose.Purpose)
					if err != nil {
						return http.StatusInternalServerError, ucerr.Wrap(err)
					}

					if deletionTimeout != userstore.GetRetentionTimeoutImmediateDeletion() {
						retainedDeletedPurposes = append(
							retainedDeletedPurposes,
							storage.ConsentedPurpose{
								Purpose:          currentPurpose.Purpose,
								RetentionTimeout: deletionTimeout,
							},
						)
					}
				}
			}

			for _, updatedPurpose := range updatedPurposes.Items() {
				retentionTimeout, err := uvu.rc.getRetentionTimeout(ctx, cm.column.ID, updatedPurpose)
				if err != nil {
					return http.StatusInternalServerError, ucerr.Wrap(err)
				}

				consentedPurposes = append(
					consentedPurposes,
					storage.ConsentedPurpose{
						Purpose:          updatedPurpose,
						RetentionTimeout: retentionTimeout,
					},
				)
			}

			uvu.liveUpdatedValues[currentValue.ID] =
				storage.ColumnConsentedValue{
					ID:                currentValue.ID,
					Version:           currentValue.Version,
					ColumnName:        cm.column.Name,
					Value:             currentValue.Value,
					Ordering:          currentValue.Ordering,
					ConsentedPurposes: consentedPurposes,
				}

			if len(retainedDeletedPurposes) > 0 {
				values := uvu.softDeletedNewValues[currentValue.Ordering]
				values = append(
					values,
					storage.ColumnConsentedValue{
						ID:                currentValue.ID,
						Version:           currentValue.Version,
						ColumnName:        cm.column.Name,
						Value:             currentValue.Value,
						Ordering:          currentValue.Ordering,
						ConsentedPurposes: retainedDeletedPurposes,
					},
				)
				uvu.softDeletedNewValues[currentValue.Ordering] = values
			}
		}
	}

	if uvu.hasChanges() {
		if err := uvu.validateOrdering(user); err != nil {
			return http.StatusInternalServerError, ucerr.Wrap(err)
		}
	}

	return http.StatusOK, nil
}

// String implements the stringer interface
func (uvu userValueUpdater) String() string {
	if uvu.baseUser == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString("{")
	if len(uvu.existingValues) > 0 {
		b.WriteString(" EXISTING:[")
		for _, values := range uvu.existingValues {
			for _, v := range values {
				fmt.Fprintf(&b, " %s", v.GetFriendlyDescription())
			}
		}
		b.WriteString(" ]")
	}
	if len(uvu.liveNewValues) > 0 {
		b.WriteString(" LIVE ADDED:[")
		for _, v := range uvu.liveNewValues {
			fmt.Fprintf(&b, " %s", v.GetFriendlyDescription())
		}
		b.WriteString(" ]")
	}
	if len(uvu.liveUpdatedValues) > 0 {
		b.WriteString(" LIVE UPDATED:[")
		for _, v := range uvu.liveUpdatedValues {
			fmt.Fprintf(&b, " %s", v.GetFriendlyDescription())
		}
		b.WriteString(" ]")
	}
	if len(uvu.liveRemovedValues) > 0 {
		b.WriteString(" LIVE REMOVED:[")
		for _, v := range uvu.liveRemovedValues {
			fmt.Fprintf(&b, " %s", v.GetFriendlyDescription())
		}
		b.WriteString(" ]")
	}
	if len(uvu.softDeletedNewValues) > 0 {
		b.WriteString(" SOFT-DELETED ADDED:[")
		for _, values := range uvu.softDeletedNewValues {
			for _, v := range values {
				fmt.Fprintf(&b, " %s", v.GetFriendlyDescription())
			}
		}
		b.WriteString(" ]")
	}
	if len(uvu.softDeletedUpdatedValues) > 0 {
		b.WriteString(" SOFT-DELETED UPDATED:[")
		for _, v := range uvu.softDeletedUpdatedValues {
			fmt.Fprintf(&b, " %s", v.GetFriendlyDescription())
		}
		b.WriteString(" ]")
	}
	if len(uvu.softDeletedRemovedValues) > 0 {
		b.WriteString(" SOFT-DELETED REMOVED:[")
		for _, v := range uvu.softDeletedRemovedValues {
			fmt.Fprintf(&b, " %s", v.GetFriendlyDescription())
		}
		b.WriteString(" ]")
	}
	b.WriteString(" }")
	return b.String()
}

func (uvu *userValueUpdater) removeExpiredPurposes(
	ctx context.Context,
	liveValues storage.ColumnConsentedValues,
	softDeletedValues storage.ColumnConsentedValues,
) error {
	for columnName, values := range liveValues {
		col := uvu.columnManager.GetUserColumnByName(columnName)
		if col == nil {
			return ucerr.Errorf("could not find column '%s'", columnName)
		}

		if col.Attributes.System {
			continue
		}

		for _, value := range values {
			anyExpiredPurposes := false
			unexpiredPurposes := []storage.ConsentedPurpose{}
			retainedExpiredPurposes := []storage.ConsentedPurpose{}
			for _, purpose := range value.ConsentedPurposes {
				if purpose.RetentionTimeout == userstore.GetRetentionTimeoutIndefinite() ||
					purpose.RetentionTimeout.Before(uvu.baseTime) {
					unexpiredPurposes = append(unexpiredPurposes, purpose)
				} else {
					anyExpiredPurposes = true

					deletionTimeout, err := uvu.rc.getDeletionTimeout(ctx, col.ID, purpose.Purpose)
					if err != nil {
						return ucerr.Wrap(err)
					}

					if deletionTimeout != userstore.GetRetentionTimeoutImmediateDeletion() {
						retainedExpiredPurposes = append(
							retainedExpiredPurposes,
							storage.ConsentedPurpose{
								Purpose:          purpose.Purpose,
								RetentionTimeout: deletionTimeout,
							},
						)
					}
				}
			}

			if anyExpiredPurposes {
				if len(unexpiredPurposes) > 0 {
					value.ConsentedPurposes = unexpiredPurposes
					uvu.liveUpdatedValues[value.ID] = value
				} else {
					uvu.liveRemovedValues[value.ID] = value
				}

				if len(retainedExpiredPurposes) > 0 {
					values := uvu.softDeletedNewValues[value.Ordering]
					values = append(
						values,
						storage.ColumnConsentedValue{
							ID:                value.ID,
							Version:           value.Version,
							ColumnName:        value.ColumnName,
							Value:             value.Value,
							Ordering:          value.Ordering,
							ConsentedPurposes: retainedExpiredPurposes,
						},
					)
					uvu.softDeletedNewValues[value.Ordering] = values
				}
			}
		}
	}

	for columnName, values := range softDeletedValues {
		col := uvu.columnManager.GetUserColumnByName(columnName)
		if col == nil {
			return ucerr.Errorf("could not find column '%s'", columnName)
		}

		if col.Attributes.System {
			continue
		}

		for _, value := range values {
			anyExpiredPurposes := false
			unexpiredPurposes := []storage.ConsentedPurpose{}
			for _, purpose := range value.ConsentedPurposes {
				if purpose.RetentionTimeout.Before(uvu.baseTime) {
					unexpiredPurposes = append(unexpiredPurposes, purpose)
				} else {
					anyExpiredPurposes = true
				}
			}

			if anyExpiredPurposes {
				if len(unexpiredPurposes) > 0 {
					value.ConsentedPurposes = unexpiredPurposes
					uvu.softDeletedUpdatedValues[value.ID] = value
				} else {
					uvu.softDeletedRemovedValues[value.ID] = value
				}
			}
		}
	}

	return nil
}

func (uvu userValueUpdater) hasChanges() bool {
	return len(uvu.liveNewValues) > 0 ||
		len(uvu.liveUpdatedValues) > 0 ||
		len(uvu.liveRemovedValues) > 0 ||
		len(uvu.softDeletedNewValues) > 0 ||
		len(uvu.softDeletedUpdatedValues) > 0 ||
		len(uvu.softDeletedRemovedValues) > 0
}

func (uvu *userValueUpdater) newUserColumnLiveValue(consentedValue storage.ColumnConsentedValue) (*storage.UserColumnLiveValue, error) {
	col := uvu.columnManager.GetUserColumnByName(consentedValue.ColumnName)
	if col == nil {
		return nil, ucerr.Errorf("could not find column '%s'", consentedValue.ColumnName)
	}

	v, err := storage.NewUserColumnLiveValue(uvu.baseUser.ID, col, &consentedValue)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return v, nil
}

func (uvu *userValueUpdater) newUserColumnSoftDeletedValue(consentedValue storage.ColumnConsentedValue) (*storage.UserColumnSoftDeletedValue, error) {
	col := uvu.columnManager.GetUserColumnByName(consentedValue.ColumnName)
	if col == nil {
		return nil, ucerr.Errorf("could not find column '%s'", consentedValue.ColumnName)
	}

	v, err := storage.NewUserColumnSoftDeletedValue(uvu.baseUser.ID, col, &consentedValue)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return v, nil
}

func (uvu *userValueUpdater) saveChanges(ctx context.Context) error {
	liveNewValues := []storage.UserColumnLiveValue{}
	liveRemovedValues := []storage.UserColumnLiveValue{}
	liveUpdatedValues := []storage.UserColumnLiveValue{}
	softDeletedNewValues := []storage.UserColumnSoftDeletedValue{}
	softDeletedRemovedValues := []storage.UserColumnSoftDeletedValue{}
	softDeletedUpdatedValues := []storage.UserColumnSoftDeletedValue{}

	for _, value := range uvu.liveNewValues {
		v, err := uvu.newUserColumnLiveValue(value)
		if err != nil {
			return ucerr.Wrap(err)
		}

		liveNewValues = append(liveNewValues, *v)
	}

	for _, value := range uvu.liveRemovedValues {
		v, err := uvu.newUserColumnLiveValue(value)
		if err != nil {
			return ucerr.Wrap(err)
		}

		liveRemovedValues = append(liveRemovedValues, *v)
	}

	for _, value := range uvu.liveUpdatedValues {
		v, err := uvu.newUserColumnLiveValue(value)
		if err != nil {
			return ucerr.Wrap(err)
		}

		liveUpdatedValues = append(liveUpdatedValues, *v)
	}

	if len(uvu.softDeletedNewValues) > 0 {
		orderings := []int{}
		for ordering := range uvu.softDeletedNewValues {
			orderings = append(orderings, ordering)
		}
		sort.Ints(orderings)

		for _, ordering := range orderings {
			for _, value := range uvu.softDeletedNewValues[ordering] {
				v, err := uvu.newUserColumnSoftDeletedValue(value)
				if err != nil {
					return ucerr.Wrap(err)
				}

				softDeletedNewValues = append(softDeletedNewValues, *v)
			}
		}
	}

	for _, value := range uvu.softDeletedRemovedValues {
		v, err := uvu.newUserColumnSoftDeletedValue(value)
		if err != nil {
			return ucerr.Wrap(err)
		}

		softDeletedRemovedValues = append(softDeletedRemovedValues, *v)
	}

	for _, value := range uvu.softDeletedUpdatedValues {
		v, err := uvu.newUserColumnSoftDeletedValue(value)
		if err != nil {
			return ucerr.Wrap(err)
		}

		softDeletedUpdatedValues = append(softDeletedUpdatedValues, *v)
	}

	if err := uvu.userStorage.InsertUserColumnLiveValues(ctx, uvu.columnManager, uvu.searchIndexManager, uvu.searchUpdateConfig, liveNewValues); err != nil {
		return ucerr.Wrap(err)
	}

	if err := uvu.userStorage.UpdateUserColumnLiveValues(ctx, liveUpdatedValues); err != nil {
		return ucerr.Wrap(err)
	}

	if err := uvu.userStorage.DeleteUserColumnLiveValues(ctx, liveRemovedValues); err != nil {
		return ucerr.Wrap(err)
	}

	if err := uvu.userStorage.InsertUserColumnSoftDeletedValues(ctx, softDeletedNewValues); err != nil {
		return ucerr.Wrap(err)
	}

	if err := uvu.userStorage.UpdateUserColumnSoftDeletedValues(ctx, softDeletedUpdatedValues); err != nil {
		return ucerr.Wrap(err)
	}

	if err := uvu.userStorage.DeleteUserColumnSoftDeletedValues(ctx, softDeletedRemovedValues); err != nil {
		return ucerr.Wrap(err)
	}

	if err := uvu.userStorage.MarkUserUpdated(ctx, uvu.baseUser); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

func (uvu userValueUpdater) validateOrdering(user *storage.User) error {
	ov := storage.NewOrderingValidator(column.DataLifeCycleStateLive)
	for _, existingValues := range user.ColumnValues {
		for _, existingValue := range existingValues {
			if _, found := uvu.liveUpdatedValues[existingValue.ID]; found {
				continue
			}
			if _, found := uvu.liveRemovedValues[existingValue.ID]; found {
				continue
			}
			ov.AddValue(existingValue)
		}
	}

	for _, newValue := range uvu.liveNewValues {
		ov.AddValue(newValue)
	}

	for _, updatedValue := range uvu.liveUpdatedValues {
		ov.AddValue(updatedValue)
	}

	return ucerr.Wrap(ov.Validate())
}
