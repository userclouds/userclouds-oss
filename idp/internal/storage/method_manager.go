package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
)

// MethodManager wraps the logic and data related to adding/removing accessors and mutators
type MethodManager struct {
	allColumnsByID map[uuid.UUID]Column
	cm             *ColumnManager
	dtm            *DataTypeManager
	initialized    bool
	methLock       sync.RWMutex
	s              *Storage
}

// NewMethodManager creates a MethodManager
func NewMethodManager(ctx context.Context, s *Storage) *MethodManager {
	return &MethodManager{s: s}
}

// initMethodManager initializes MethodManager with current DB column table state for given DB connection.
func (mm *MethodManager) initMethodManager(ctx context.Context) error {
	if mm.initialized {
		return nil
	}

	mm.methLock.Lock()
	defer mm.methLock.Unlock()

	dtm, err := NewDataTypeManager(ctx, mm.s)
	if err != nil {
		return ucerr.Wrap(err)
	}
	mm.dtm = dtm

	cm, err := NewUserstoreColumnManager(ctx, mm.s)
	if err != nil {
		return ucerr.Wrap(err)
	}
	mm.cm = cm

	pager, err := NewColumnPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return ucerr.Wrap(err)
	}
	allColumns := make([]Column, 0)
	for {
		objRead, respFields, err := mm.s.ListColumnsPaginated(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}
		allColumns = append(allColumns, objRead...)
		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	mm.allColumnsByID = make(map[uuid.UUID]Column, len(allColumns))
	for _, col := range allColumns {
		mm.allColumnsByID[col.ID] = col
	}

	mm.initialized = true

	return nil
}

func (mm *MethodManager) getAccessorByID(ctx context.Context, id uuid.UUID) (*Accessor, int, error) {
	a, err := mm.s.GetLatestAccessor(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), ucerr.Wrap(err)
	}

	return a, http.StatusOK, nil
}

func (mm *MethodManager) getAccessorByName(ctx context.Context, name string) (*Accessor, int, error) {
	a, err := mm.s.GetAccessorByName(ctx, name)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), ucerr.Wrap(err)
	}

	return a, http.StatusOK, nil
}

func (mm *MethodManager) getMutatorByID(ctx context.Context, id uuid.UUID) (*Mutator, int, error) {
	m, err := mm.s.GetLatestMutator(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), ucerr.Wrap(err)
	}

	return m, http.StatusOK, nil
}

func (mm *MethodManager) getMutatorByName(ctx context.Context, name string) (*Mutator, int, error) {
	m, err := mm.s.GetMutatorByName(ctx, name)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), ucerr.Wrap(err)
	}

	return m, http.StatusOK, nil
}

func (mm *MethodManager) getSearchColumnID(ctx context.Context, selectorConfig userstore.UserSelectorConfig) uuid.UUID {
	if sp, err := getSearchParameters(ctx, mm.cm, selectorConfig); err == nil && sp != nil {
		return sp.Column.ID
	}

	return uuid.Nil
}

// SaveAccessor creates an accessor if it doesn't exist or updates an existing accessor if it does
func (mm *MethodManager) SaveAccessor(ctx context.Context, updated *Accessor) (code int, err error) {
	if err := mm.initMethodManager(ctx); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	updated.SearchColumnID = mm.getSearchColumnID(ctx, updated.SelectorConfig)
	current, code, err := mm.getAccessorByID(ctx, updated.ID)
	switch code {
	case http.StatusOK:
		code, err = mm.updateAccessor(ctx, current, updated)
	case http.StatusNotFound:
		code, err = mm.createAccessor(ctx, updated)
	}
	return code, ucerr.Wrap(err)
}

// CreateAccessorFromClient creates an accessor on basis of client input
func (mm *MethodManager) CreateAccessorFromClient(ctx context.Context, clientAccessor *userstore.Accessor) (code int, err error) {
	if err := mm.initMethodManager(ctx); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if clientAccessor.IsSystem {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "the IsSystem attribute cannot be set by the client")
	}

	columnIDs := make([]uuid.UUID, 0, len(clientAccessor.Columns))
	transformerIDs := make([]uuid.UUID, 0, len(clientAccessor.Columns))
	tokenAccessPolicyIDs := make([]uuid.UUID, 0, len(clientAccessor.Columns))

	for _, ct := range clientAccessor.Columns {
		columnIDs = append(columnIDs, ct.Column.ID)
		transformerIDs = append(transformerIDs, ct.Transformer.ID)
		tokenAccessPolicyIDs = append(tokenAccessPolicyIDs, ct.TokenAccessPolicy.ID)
	}

	purposeIDs := make([]uuid.UUID, 0, len(clientAccessor.Purposes))
	for _, p := range clientAccessor.Purposes {
		purposeIDs = append(purposeIDs, p.ID)
	}

	updated := Accessor{
		SystemAttributeBaseModel:          ucdb.NewSystemAttributeBase(),
		Name:                              clientAccessor.Name,
		Description:                       clientAccessor.Description,
		DataLifeCycleState:                column.DataLifeCycleStateFromClient(clientAccessor.DataLifeCycleState),
		ColumnIDs:                         columnIDs,
		TransformerIDs:                    transformerIDs,
		TokenAccessPolicyIDs:              tokenAccessPolicyIDs,
		AccessPolicyID:                    clientAccessor.AccessPolicy.ID,
		SelectorConfig:                    clientAccessor.SelectorConfig,
		PurposeIDs:                        purposeIDs,
		Version:                           0,
		IsAuditLogged:                     clientAccessor.IsAuditLogged,
		IsAutogenerated:                   clientAccessor.IsAutogenerated,
		AreColumnAccessPoliciesOverridden: clientAccessor.AreColumnAccessPoliciesOverridden,
		SearchColumnID:                    mm.getSearchColumnID(ctx, clientAccessor.SelectorConfig),
		UseSearchIndex:                    clientAccessor.UseSearchIndex,
	}

	if clientAccessor.ID != uuid.Nil {
		updated.ID = clientAccessor.ID
		a, code, err := mm.getAccessorByID(ctx, updated.ID)
		if a != nil {
			if updated.Equals(a, false) {
				return http.StatusConflict,
					ucerr.Wrap(
						ucerr.WrapWithFriendlyStructure(
							jsonclient.Error{StatusCode: http.StatusConflict},
							jsonclient.SDKStructuredError{
								Error:     "This accessor already exists",
								ID:        a.ID,
								Identical: true,
							},
						),
					)
			}
			return http.StatusConflict,
				ucerr.Wrap(
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error: fmt.Sprintf("An accessor with ID %v already exists", updated.ID),
							ID:    a.ID,
						},
					),
				)
		} else if code != http.StatusNotFound {
			return code, ucerr.Wrap(err)
		}
	} else {
		clientAccessor.ID = updated.ID
	}

	code, err = mm.createAccessor(ctx, &updated)
	return code, ucerr.Wrap(err)
}

func (mm *MethodManager) createAccessor(ctx context.Context, updated *Accessor) (code int, err error) {
	if code, err := mm.validateAccessorSettings(ctx, updated); err != nil {
		return code, ucerr.Wrap(err)
	}

	a, code, err := mm.getAccessorByName(ctx, updated.Name)
	if a != nil {
		updated.ID = a.ID // Ignore mismatched ids for purposes of comparison
		if a.Equals(updated, false) {
			return http.StatusConflict,
				ucerr.Wrap(
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error:     "This accessor already exists",
							ID:        a.ID,
							Identical: true,
						},
					),
				)
		}

		return http.StatusConflict,
			ucerr.Wrap(
				ucerr.WrapWithFriendlyStructure(
					jsonclient.Error{StatusCode: http.StatusConflict},
					jsonclient.SDKStructuredError{
						Error: fmt.Sprintf(`An accessor with the name '%s' already exists`, updated.Name),
						ID:    a.ID,
					},
				),
			)
	} else if code != http.StatusNotFound {
		return code, ucerr.Wrap(err)
	}

	if err := updated.Validate(); err != nil {
		return http.StatusBadRequest, ucerr.Wrap(err)
	}

	if err := mm.s.SaveAccessor(ctx, updated); err != nil {
		return uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
	}

	return http.StatusCreated, nil
}

// UpdateAccessorFromClient updates an accessor on basis of client input
func (mm *MethodManager) UpdateAccessorFromClient(ctx context.Context, clientAccessor *userstore.Accessor) (code int, err error) {
	if err := mm.initMethodManager(ctx); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	current, code, err := mm.getAccessorByID(ctx, clientAccessor.ID)
	if current == nil {
		return code, ucerr.Wrap(err)
	}
	if current.IsSystem {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "system accessors cannot be updated")
	}
	if current.IsSystem != clientAccessor.IsSystem {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "the IsSystem attribute cannot be changed")
	}

	columnIDs := []uuid.UUID{}
	transformerIDs := []uuid.UUID{}
	tokenAccessPolicyIDs := []uuid.UUID{}

	for _, ct := range clientAccessor.Columns {
		columnIDs = append(columnIDs, ct.Column.ID)
		transformerIDs = append(transformerIDs, ct.Transformer.ID)
		tokenAccessPolicyIDs = append(tokenAccessPolicyIDs, ct.TokenAccessPolicy.ID)
	}

	purposeIDs := []uuid.UUID{}
	for _, p := range clientAccessor.Purposes {
		purposeIDs = append(purposeIDs, p.ID)
	}

	updated := Accessor{
		SystemAttributeBaseModel:          ucdb.NewSystemAttributeBaseWithID(clientAccessor.ID),
		Name:                              clientAccessor.Name,
		Description:                       clientAccessor.Description,
		DataLifeCycleState:                column.DataLifeCycleStateFromClient(clientAccessor.DataLifeCycleState),
		ColumnIDs:                         columnIDs,
		TransformerIDs:                    transformerIDs,
		TokenAccessPolicyIDs:              tokenAccessPolicyIDs,
		AccessPolicyID:                    clientAccessor.AccessPolicy.ID,
		SelectorConfig:                    clientAccessor.SelectorConfig,
		PurposeIDs:                        purposeIDs,
		Version:                           clientAccessor.Version,
		IsAuditLogged:                     clientAccessor.IsAuditLogged,
		IsAutogenerated:                   false,
		AreColumnAccessPoliciesOverridden: clientAccessor.AreColumnAccessPoliciesOverridden,
		SearchColumnID:                    mm.getSearchColumnID(ctx, clientAccessor.SelectorConfig),
		UseSearchIndex:                    clientAccessor.UseSearchIndex,
	}

	if code, err := mm.updateAccessor(ctx, current, &updated); err != nil {
		return code, ucerr.Wrap(err)
	}

	// Update the client version if needed
	clientAccessor.Version = updated.Version

	return http.StatusOK, nil
}

func (mm *MethodManager) updateAccessor(ctx context.Context, current *Accessor, updated *Accessor) (code int, err error) {
	// ignore if the current accessor is identical to the updated one ignoring the version
	if current.Equals(updated, false) {
		updated.Version = current.Version
		return http.StatusNotModified, nil
	}

	if code, err := mm.validateAccessorSettings(ctx, updated); err != nil {
		return code, ucerr.Wrap(err)
	}

	updateNames := false
	if current.Name != updated.Name {
		// check that the updated name is not already in use by a different accessor
		a, code, err := mm.getAccessorByName(ctx, updated.Name)
		if a != nil {
			if a.ID != updated.ID {
				return http.StatusConflict,
					ucerr.Wrap(
						ucerr.WrapWithFriendlyStructure(
							jsonclient.Error{StatusCode: http.StatusConflict},
							jsonclient.SDKStructuredError{
								Error: fmt.Sprintf(`An accessor with the name '%s' already exists`, updated.Name),
								ID:    a.ID,
							},
						),
					)
			}
		} else if code != http.StatusNotFound {
			return code, ucerr.Wrap(err)
		}
		updateNames = true
	}

	updated.Version = current.Version + 1

	if err := updated.Validate(); err != nil {
		return http.StatusBadRequest, ucerr.Wrap(err)
	}

	if err := mm.s.SaveAccessor(ctx, updated); err != nil {
		return uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
	}

	if updateNames {
		// change the accessor name for all versions
		versions, err := mm.s.GetAllAccessorVersions(ctx, updated.ID)
		if err != nil {
			return http.StatusInternalServerError, ucerr.Wrap(err)
		}
		for _, a := range versions {
			if a.Name != updated.Name {
				a.Name = updated.Name
				if err := mm.s.PriorVersionSaveAccessor(ctx, &a); err != nil {
					return uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
				}
			}
		}
	}

	return http.StatusOK, nil
}

// DeleteAccessorFromClient deletes a mutator for a client, disallowing deletion of system accessors
func (mm *MethodManager) DeleteAccessorFromClient(ctx context.Context, id uuid.UUID) (version int, code int, err error) {
	if err := mm.initMethodManager(ctx); err != nil {
		return 0, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	current, code, err := mm.getAccessorByID(ctx, id)
	if current == nil {
		return 0, code, ucerr.Wrap(err)
	}

	if current.IsSystem {
		return 0, http.StatusBadRequest, ucerr.Friendlyf(nil, "system accessors cannot be deleted")
	}

	if err := mm.DeleteAccessor(ctx, id); err != nil {
		return 0, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if ap, err := mm.s.GetLatestAccessPolicy(ctx, current.AccessPolicyID); err == nil {
		if ap.IsAutogenerated {
			if errDelAP := mm.s.DeleteAllAccessPolicyVersions(ctx, current.AccessPolicyID); errDelAP != nil {
				uclog.Debugf(ctx, "Failed to delete access policy %v for accessor %v: %v", current.AccessPolicyID, id, errDelAP)
			}
		}
	} else {
		uclog.Debugf(ctx, "Failed to get access policy %v for accessor %v during deletion: %v", current.AccessPolicyID, id, err)
	}

	return current.Version, http.StatusOK, nil
}

// DeleteAccessor deletes given accessor and returns error if it doesn't exists
func (mm *MethodManager) DeleteAccessor(ctx context.Context, id uuid.UUID) error {
	if err := mm.initMethodManager(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	return ucerr.Wrap(mm.s.DeleteAllAccessorVersions(ctx, id))
}

func (mm *MethodManager) validateAccessorSettings(ctx context.Context, updated *Accessor) (code int, err error) {
	// validate each column transformer

	transformerIDs := []uuid.UUID{}
	for _, t := range updated.TransformerIDs {
		if !t.IsNil() {
			transformerIDs = append(transformerIDs, t)
		}
	}

	transformerMap, err := GetTransformerMapForIDs(ctx, mm.s, false, transformerIDs...)
	if err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	for i, columnID := range updated.ColumnIDs {
		col, found := mm.allColumnsByID[columnID]
		if !found {
			return http.StatusBadRequest, ucerr.Friendlyf(nil, "column with ID %v does not exist", columnID)
		}

		transformerID := updated.TransformerIDs[i]
		if transformerID.IsNil() {
			continue
		}

		transformer, err := transformerMap.ForID(transformerID)
		if err != nil {
			return http.StatusBadRequest, ucerr.Friendlyf(err, "transformer with ID %v does not exist", transformerID)
		}

		if transformer.TransformType != transformTypePassThrough {
			dt := mm.dtm.GetDataTypeByID(col.DataTypeID)
			if dt == nil {
				return http.StatusInternalServerError,
					ucerr.Errorf(
						"column '%s' has unrecognized data type '%v'",
						col.Name,
						col.DataTypeID,
					)
			}

			if !transformer.CanInput(*dt) {
				return http.StatusBadRequest,
					ucerr.Friendlyf(
						nil,
						"column '%s' data type '%v' cannot be represented by transformer '%s' input data type '%v'",
						col.Name,
						col.DataTypeID,
						transformer.Name,
						transformer.InputDataTypeID,
					)
			}
		}

		if transformer.TransformType == transformTypeTokenizeByReference ||
			transformer.TransformType == transformTypeTokenizeByValue {

			tokenAccessPolicyID := updated.TokenAccessPolicyIDs[i]
			if tokenAccessPolicyID.IsNil() {
				return http.StatusBadRequest,
					ucerr.Friendlyf(nil, "token resolution policy required for tokenization")
			}
			if _, err := mm.s.GetLatestAccessPolicy(ctx, tokenAccessPolicyID); err != nil {
				return http.StatusBadRequest, ucerr.Friendlyf(err, "invalid token resolution policy")
			}
		}

		if transformer.TransformType == transformTypeTokenizeByReference && col.Table != userTableName {
			return http.StatusBadRequest,
				ucerr.Friendlyf(nil, "tokenization by reference is only supported for columns in the user table")
		}
	}

	sim, err := NewSearchIndexManager(ctx, mm.s)
	if err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if err := sim.CheckAccessorCompatible(ctx, *updated); err != nil {
		return http.StatusBadRequest, ucerr.Wrap(err)
	}

	return http.StatusOK, nil
}

// SaveMutator creates an mutator if it doesn't exist or updates an existing mutator if it does
func (mm *MethodManager) SaveMutator(ctx context.Context, updated *Mutator) (code int, err error) {
	if err := mm.initMethodManager(ctx); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	current, code, err := mm.getMutatorByID(ctx, updated.ID)
	switch code {
	case http.StatusOK:
		code, err = mm.updateMutator(ctx, current, updated)
	case http.StatusNotFound:
		code, err = mm.createMutator(ctx, updated)
	}
	return code, ucerr.Wrap(err)
}

// CreateMutatorFromClient creates an mutator on basis of client input
func (mm *MethodManager) CreateMutatorFromClient(ctx context.Context, clientMutator *userstore.Mutator) (code int, err error) {
	if err := mm.initMethodManager(ctx); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if clientMutator.IsSystem {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "the IsSystem attribute cannot be set by the client")
	}

	columnIDs := []uuid.UUID{}
	normalizerIDs := []uuid.UUID{}

	for _, cv := range clientMutator.Columns {
		columnIDs = append(columnIDs, cv.Column.ID)
		normalizerIDs = append(normalizerIDs, cv.Normalizer.ID)
	}

	updated := Mutator{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
		Name:                     clientMutator.Name,
		Description:              clientMutator.Description,
		ColumnIDs:                columnIDs,
		NormalizerIDs:            normalizerIDs,
		AccessPolicyID:           clientMutator.AccessPolicy.ID,
		SelectorConfig:           clientMutator.SelectorConfig,
		Version:                  0,
	}

	if clientMutator.ID != uuid.Nil {
		updated.ID = clientMutator.ID
		m, code, err := mm.getMutatorByID(ctx, updated.ID)
		if m != nil {
			if updated.Equals(m, false) {
				return http.StatusConflict,
					ucerr.Wrap(
						ucerr.WrapWithFriendlyStructure(
							jsonclient.Error{StatusCode: http.StatusConflict},
							jsonclient.SDKStructuredError{
								Error:     "This mutator already exists",
								ID:        m.ID,
								Identical: true,
							},
						),
					)
			}
			return http.StatusConflict,
				ucerr.Wrap(
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error: fmt.Sprintf("A mutator with ID %v already exists", updated.ID),
							ID:    m.ID,
						},
					),
				)
		} else if code != http.StatusNotFound {
			return code, ucerr.Wrap(err)
		}
	} else {
		clientMutator.ID = updated.ID
	}

	code, err = mm.createMutator(ctx, &updated)
	return code, ucerr.Wrap(err)
}

func (mm *MethodManager) createMutator(ctx context.Context, updated *Mutator) (code int, err error) {
	if code, err := mm.validateMutatorColumns(ctx, updated); err != nil {
		return code, ucerr.Wrap(err)
	}

	m, code, err := mm.getMutatorByName(ctx, updated.Name)
	if m != nil {
		updated.ID = m.ID // Ignore mismatched ids for purposes of comparison
		if m.Equals(updated, false) {
			return http.StatusConflict,
				ucerr.Wrap(
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error:     "This mutator already exists",
							ID:        m.ID,
							Identical: true,
						},
					),
				)
		}

		return http.StatusConflict,
			ucerr.Wrap(
				ucerr.WrapWithFriendlyStructure(
					jsonclient.Error{StatusCode: http.StatusConflict},
					jsonclient.SDKStructuredError{
						Error: fmt.Sprintf(`A mutator with the name '%s' already exists`, updated.Name),
						ID:    m.ID,
					},
				),
			)
	} else if code != http.StatusNotFound {
		return code, ucerr.Wrap(err)
	}

	if err := updated.Validate(); err != nil {
		return http.StatusBadRequest, ucerr.Wrap(err)
	}

	if err := mm.s.SaveMutator(ctx, updated); err != nil {
		return uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
	}

	return http.StatusCreated, nil
}

// UpdateMutatorFromClient updates an mutator on basis of client input
func (mm *MethodManager) UpdateMutatorFromClient(ctx context.Context, clientMutator *userstore.Mutator) (code int, err error) {
	if err := mm.initMethodManager(ctx); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	current, code, err := mm.getMutatorByID(ctx, clientMutator.ID)
	if err != nil {
		return code, ucerr.Wrap(err)
	}
	if current.IsSystem {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "system mutators cannot be updated")
	}
	if current.IsSystem != clientMutator.IsSystem {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "the IsSystem attribute cannot be changed")
	}

	columnIDs := []uuid.UUID{}
	normalizerIDs := []uuid.UUID{}

	for _, cv := range clientMutator.Columns {
		columnIDs = append(columnIDs, cv.Column.ID)
		normalizerIDs = append(normalizerIDs, cv.Normalizer.ID)
	}

	updated := Mutator{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(clientMutator.ID),
		Name:                     clientMutator.Name,
		Description:              clientMutator.Description,
		ColumnIDs:                columnIDs,
		NormalizerIDs:            normalizerIDs,
		AccessPolicyID:           clientMutator.AccessPolicy.ID,
		SelectorConfig:           clientMutator.SelectorConfig,
		Version:                  clientMutator.Version,
	}

	if code, err := mm.updateMutator(ctx, current, &updated); err != nil {
		return code, ucerr.Wrap(err)
	}

	// Update the client version if needed
	clientMutator.Version = updated.Version

	return http.StatusOK, nil
}

func (mm *MethodManager) updateMutator(ctx context.Context, current *Mutator, updated *Mutator) (code int, err error) {
	// ignore if the current mutator is identical to the updated one ignoring the version
	if current.Equals(updated, false) {
		updated.Version = current.Version
		return http.StatusNotModified, nil
	}

	if code, err := mm.validateMutatorColumns(ctx, updated); err != nil {
		return code, ucerr.Wrap(err)
	}

	updateName := false
	if current.Name != updated.Name {
		// check that the updated name is not already in use by a different mutator
		m, code, err := mm.getMutatorByName(ctx, updated.Name)
		if m != nil {
			if m.ID != updated.ID {
				return http.StatusConflict,
					ucerr.Wrap(
						ucerr.WrapWithFriendlyStructure(
							jsonclient.Error{StatusCode: http.StatusConflict},
							jsonclient.SDKStructuredError{
								Error: fmt.Sprintf(`A mutator with the name '%s' already exists`, updated.Name),
								ID:    m.ID,
							},
						),
					)
			}
		} else if code != http.StatusNotFound {
			return code, ucerr.Wrap(err)
		}
		updateName = true
	}

	updated.Version = current.Version + 1

	if err := updated.Validate(); err != nil {
		return http.StatusBadRequest, ucerr.Wrap(err)
	}

	if err := mm.s.SaveMutator(ctx, updated); err != nil {
		return uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
	}

	if updateName {
		// change all other versions of the mutator to have the updated name
		versions, err := mm.s.GetAllMutatorVersions(ctx, updated.ID)
		if err != nil {
			return http.StatusInternalServerError, ucerr.Wrap(err)
		}
		for _, m := range versions {
			if m.Name != updated.Name {
				m.Name = updated.Name
				if err := mm.s.PriorVersionSaveMutator(ctx, &m); err != nil {
					return uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
				}
			}
		}
	}

	return http.StatusOK, nil
}

// DeleteMutatorFromClient deletes a mutator for a client, disallowing deletion of system mutators
func (mm *MethodManager) DeleteMutatorFromClient(ctx context.Context, id uuid.UUID) (version int, code int, err error) {
	if err := mm.initMethodManager(ctx); err != nil {
		return 0, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	current, code, err := mm.getMutatorByID(ctx, id)
	if current == nil {
		return 0, code, ucerr.Wrap(err)
	}
	if current.IsSystem {
		return 0, http.StatusBadRequest, ucerr.Friendlyf(nil, "system mutators cannot be deleted")
	}

	if err := mm.DeleteMutator(ctx, id); err != nil {
		return 0, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if ap, err := mm.s.GetLatestAccessPolicy(ctx, current.AccessPolicyID); err == nil {
		if ap.IsAutogenerated {
			if errDelAP := mm.s.DeleteAllAccessPolicyVersions(ctx, current.AccessPolicyID); errDelAP != nil {
				uclog.Debugf(ctx, "Failed to delete access policy %v for mutator %v: %v", current.AccessPolicyID, id, errDelAP)
			}
		}
	} else {
		uclog.Debugf(ctx, "Failed to get access policy %v for mutator %v during deletion: %v", current.AccessPolicyID, id, err)
	}

	return current.Version, http.StatusOK, nil
}

// DeleteMutator deletes given mutator and returns error if it doesn't exists
func (mm *MethodManager) DeleteMutator(ctx context.Context, id uuid.UUID) error {
	if err := mm.initMethodManager(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	return ucerr.Wrap(mm.s.DeleteAllMutatorVersions(ctx, id))
}

func (mm *MethodManager) validateMutatorColumns(ctx context.Context, updated *Mutator) (code int, err error) {
	// verify that all columns and normalizers referenced by the mutator exist, that each
	// normalizer is of a supported transform type, and that the type of each column matches
	// the output type of the associated normalizer

	normalizerMap := map[uuid.UUID]*Transformer{}
	for _, normalizerID := range updated.NormalizerIDs {
		if _, ok := normalizerMap[normalizerID]; !ok {
			normalizer, err := mm.s.GetLatestTransformer(ctx, normalizerID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return http.StatusBadRequest, ucerr.Friendlyf(nil, "normalizer with ID %v does not exist", normalizerID)
				}
				return http.StatusInternalServerError, ucerr.Wrap(err)
			}
			normalizerMap[normalizer.ID] = normalizer
		}
	}

	for i, columnID := range updated.ColumnIDs {
		col, found := mm.allColumnsByID[columnID]
		if !found {
			return http.StatusBadRequest, ucerr.Friendlyf(nil, "column with ID %v does not exist", columnID)
		}

		normalizerID := updated.NormalizerIDs[i]
		normalizer := normalizerMap[normalizerID]

		switch normalizer.TransformType {
		case transformTypePassThrough:
		case transformTypeTransform:
			dt := mm.dtm.GetDataTypeByID(col.DataTypeID)
			if dt == nil {
				return http.StatusInternalServerError,
					ucerr.Errorf(
						"column '%s' has unrecognized data type '%v'",
						col.Name,
						col.DataTypeID,
					)
			}

			if !normalizer.CanOutput(*dt) {
				return http.StatusBadRequest,
					ucerr.Friendlyf(nil,
						"column '%s' data type '%v' does not match normalizer '%s' output data type '%v'",
						col.Name,
						col.DataTypeID,
						normalizer.Name,
						normalizer.OutputDataTypeID,
					)
			}
		default:
			return http.StatusBadRequest,
				ucerr.Friendlyf(
					nil,
					"cannot use normalizer %v with transform type %v",
					normalizerID,
					normalizer.TransformType.ToClient(),
				)
		}
	}

	return http.StatusOK, nil
}
