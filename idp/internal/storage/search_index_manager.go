package storage

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/userstore/search"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/infra/uctypes/timestamp"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/ucopensearch"
)

// SearchIndexManager is responsible for managing user search indices
type SearchIndexManager struct {
	s                   *Storage
	tenantID            uuid.UUID
	lock                sync.RWMutex
	indicesByAccessorID map[uuid.UUID]AccessorSearchIndex
	indicesByID         map[uuid.UUID]UserSearchIndex
	indicesByName       map[string]UserSearchIndex
}

// NewSearchIndexManager creates a new SearchIndexManager
func NewSearchIndexManager(ctx context.Context, s *Storage) (*SearchIndexManager, error) {
	sim := SearchIndexManager{s: s, tenantID: s.GetTenantID()}

	if err := sim.initSearchIndexManager(ctx); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &sim, nil
}

func (sim *SearchIndexManager) initSearchIndexManager(ctx context.Context) error {
	sim.lock.Lock()
	defer sim.lock.Unlock()

	accessorIndices, err := sim.s.ListAccessorSearchIndexesNonPaginated(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	indices, err := sim.s.ListUserSearchIndexesNonPaginated(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	sim.indicesByAccessorID = map[uuid.UUID]AccessorSearchIndex{}
	sim.indicesByID = map[uuid.UUID]UserSearchIndex{}
	sim.indicesByName = map[string]UserSearchIndex{}

	for _, accessorIndex := range accessorIndices {
		sim.indicesByAccessorID[accessorIndex.ID] = accessorIndex
	}

	for _, index := range indices {
		sim.indicesByID[index.ID] = index
		sim.indicesByName[strings.ToLower(index.Name)] = index
	}

	return nil
}

// GetQueryableIndicesForTenant retrieves all indices for given tenant
func GetQueryableIndicesForTenant(ctx context.Context, ts *tenantmap.TenantState) ([]ucopensearch.QueryableIndex, error) {
	searchableIndices := make([]ucopensearch.QueryableIndex, 0)

	s := NewFromTenantState(ctx, ts)

	sim, err := NewSearchIndexManager(ctx, s)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	indices := sim.GetIndices()
	uclog.Verbosef(ctx, "Got %d candidate indices for tenant %v", len(indices), ts.ID)
	for _, index := range indices {
		if index.IsSearchable() {
			qindex, err := index.GetQueryableIndex(search.QueryTypeTerm)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			searchableIndices = append(searchableIndices, qindex)
		}
	}

	return searchableIndices, nil
}

// CheckAccessorCompatible returns an error if the accessor is not appropriately associated with a user search index
func (sim *SearchIndexManager) CheckAccessorCompatible(ctx context.Context, acc Accessor) error {
	if accessorIndex, found := sim.indicesByAccessorID[acc.ID]; found {
		index, found := sim.indicesByID[accessorIndex.UserSearchIndexID]
		if !found {
			return ucerr.Errorf(
				"Accessor '%v' paired with unrecognized UserSearchIndex '%v'",
				acc.ID,
				accessorIndex.UserSearchIndexID,
			)
		}

		if err := index.CheckAccessorCompatible(acc, accessorIndex.QueryType); err != nil {
			return ucerr.Wrap(err)
		}
	} else if acc.UseSearchIndex {
		// In container env, where we don't run OpenSearch and define indices, we short circuit this check.
		if universe.Current().IsContainer() {
			uclog.Warningf(ctx, "UseSearchIndex is true for accessor %v, but no UserSearchIndex is defined. Ignoring in container universe", acc.ID)
			return nil
		}
		return ucerr.Friendlyf(
			nil,
			"UseSearchIndex cannot be true without pairing Accessor with searchable UserSearchIndex",
		)
	}

	return nil
}

// CheckColumnUnindexed returns an error if any user search index that is enabled indexes the column
func (sim *SearchIndexManager) CheckColumnUnindexed(c Column) error {
	for _, index := range sim.GetIndices() {
		if err := index.CheckColumnUnIndexed(c); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

// CreateIndex creates a user search index based on the provided input
func (sim *SearchIndexManager) CreateIndex(ctx context.Context, created *UserSearchIndex) (int, error) {
	if created.IsEnabled() {
		return http.StatusBadRequest,
			ucerr.Friendlyf(nil, "newly created index cannot be Enabled")
	}

	if err := created.Validate(); err != nil {
		return http.StatusBadRequest, ucerr.Wrap(err)
	}

	if existing := sim.GetIndexByID(created.ID); existing != nil {
		if created.EquivalentTo(existing) {
			return http.StatusConflict,
				ucerr.Wrap(
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error:     "This index already exists",
							ID:        created.ID,
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
						Error: fmt.Sprintf(
							"An index with the ID '%v' already exists with name '%s'",
							created.ID,
							existing.Name,
						),
						ID: created.ID,
					},
				),
			)
	}

	if existing := sim.GetIndexByName(created.Name); existing != nil {
		if created.EquivalentTo(existing) {
			return http.StatusConflict,
				ucerr.Wrap(
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error:     "This index already exists",
							ID:        existing.ID,
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
						Error: fmt.Sprintf(
							"An index with the name '%s' already exists with ID '%v'",
							existing.Name,
							existing.ID,
						),
						ID: created.ID,
					},
				),
			)
	}

	cm, err := NewUserstoreColumnManager(ctx, sim.s)
	if err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	for _, columnID := range created.ColumnIDs {
		c := cm.GetColumnByID(columnID)
		if c == nil {
			return http.StatusBadRequest,
				ucerr.Friendlyf(
					nil,
					"ColumnIDs contains an unrecognized column: %v",
					columnID,
				)
		}

		if err := c.confirmSearchIndexable(); err != nil {
			return http.StatusBadRequest,
				ucerr.Friendlyf(
					err,
					"ColumnIDs contains column '%v' that cannot be indexed in search",
					columnID,
				)
		}
	}

	if err := sim.s.SaveUserSearchIndex(ctx, created); err != nil {
		return uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
	}

	sim.lock.Lock()
	sim.indicesByID[created.ID] = *created
	sim.indicesByName[strings.ToLower(created.Name)] = *created
	sim.lock.Unlock()

	return http.StatusCreated, nil
}

// DeleteIndex deletes the specified index if it exists
func (sim *SearchIndexManager) DeleteIndex(ctx context.Context, indexID uuid.UUID) (int, error) {
	deleted := sim.GetIndexByID(indexID)
	if deleted == nil {
		return http.StatusNotFound, ucerr.Friendlyf(nil, "index %v could not be found", indexID)
	}

	if deleted.IsEnabled() {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "index %v is still in use", deleted.ID)
	}

	if err := sim.s.DeleteUserSearchIndex(ctx, deleted.ID); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	sim.lock.Lock()
	delete(sim.indicesByID, deleted.ID)
	delete(sim.indicesByName, strings.ToLower(deleted.Name))
	sim.lock.Unlock()

	return http.StatusOK, nil
}

// GetAccessorsByID returns the accessors using the provided index ID
func (sim *SearchIndexManager) GetAccessorsByID(indexID uuid.UUID) []AccessorSearchIndex {
	var asis []AccessorSearchIndex
	for _, asi := range sim.indicesByAccessorID {
		if asi.UserSearchIndexID == indexID {
			asis = append(asis, asi)
		}
	}

	return asis
}

// GetIndexByID returns the user search index for the provided index ID
func (sim *SearchIndexManager) GetIndexByID(indexID uuid.UUID) *UserSearchIndex {
	usi, found := sim.indicesByID[indexID]
	if !found {
		return nil
	}
	return &usi
}

// GetIndexByName returns the user search index for the provided index name
func (sim *SearchIndexManager) GetIndexByName(indexName string) *UserSearchIndex {
	usi, found := sim.indicesByName[strings.ToLower(indexName)]
	if !found {
		return nil
	}
	return &usi
}

// GetIndices returns all user search indices
func (sim *SearchIndexManager) GetIndices() []UserSearchIndex {
	indices := make([]UserSearchIndex, 0, len(sim.indicesByID))
	for _, index := range sim.indicesByID {
		indices = append(indices, index)
	}
	return indices
}

// GetQueryableIndexForAccessorID returns the QueryableIndex for the provided accessor ID
func (sim *SearchIndexManager) GetQueryableIndexForAccessorID(accessorID uuid.UUID) (ucopensearch.QueryableIndex, error) {
	ai, found := sim.indicesByAccessorID[accessorID]
	if !found {
		return nil, nil
	}
	usi, found := sim.indicesByID[ai.UserSearchIndexID]
	if !found {
		return nil, ucerr.Errorf("accessor '%v' is paired with unrecognized user search index '%v'", ai.ID, ai.UserSearchIndexID)
	}

	queryableIndex, err := usi.GetQueryableIndex(ai.QueryType)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return queryableIndex, nil
}

// RemoveAccessor disassociates the specified accessor from any user search index
func (sim *SearchIndexManager) RemoveAccessor(
	ctx context.Context,
	accessorID uuid.UUID,
) (int, error) {
	accessor, err := sim.s.GetLatestAccessor(ctx, accessorID)
	if err != nil {
		return uchttp.SQLReadErrorMapper(err), ucerr.Wrap(err)
	}

	if _, found := sim.indicesByAccessorID[accessorID]; !found {
		return http.StatusNotModified, nil
	}

	if accessor.UseSearchIndex {
		return http.StatusBadRequest,
			ucerr.Friendlyf(
				nil,
				"Accessor '%v' with UseSearchIndex set to true must be paired with a searchable UserSearchIndex",
				accessorID,
			)
	}

	if err := sim.s.DeleteAccessorSearchIndex(ctx, accessorID); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	sim.lock.Lock()
	delete(sim.indicesByAccessorID, accessorID)
	sim.lock.Unlock()
	return http.StatusOK, nil
}

// SetAccessor pairs the specified accessor with the specified user search index and query type
func (sim *SearchIndexManager) SetAccessor(
	ctx context.Context,
	accessorID uuid.UUID,
	indexID uuid.UUID,
	queryType search.QueryType,
) (int, error) {
	updated := AccessorSearchIndex{
		BaseModel:         ucdb.NewBaseWithID(accessorID),
		UserSearchIndexID: indexID,
		QueryType:         queryType,
	}

	if sim.indicesByAccessorID[updated.ID].Equals(&updated) {
		return http.StatusNotModified, nil
	}

	accessor, err := sim.s.GetLatestAccessor(ctx, updated.ID)
	if err != nil {
		return uchttp.SQLReadErrorMapper(err), ucerr.Wrap(err)
	}

	index := sim.GetIndexByID(updated.UserSearchIndexID)
	if index == nil {
		return http.StatusBadRequest,
			ucerr.Friendlyf(
				nil,
				"Accessor %v cannot be paired with unrecognized UserSearchIndex %v",
				updated.ID,
				updated.UserSearchIndexID,
			)
	}

	if err := index.CheckAccessorCompatible(*accessor, queryType); err != nil {
		return http.StatusBadRequest, ucerr.Wrap(err)
	}

	if err := sim.s.SaveAccessorSearchIndex(ctx, &updated); err != nil {
		return uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
	}

	sim.lock.Lock()
	sim.indicesByAccessorID[updated.ID] = updated
	sim.lock.Unlock()

	return http.StatusOK, nil
}

// UpdateIndex creates a UserSearch index based on the provided input
func (sim *SearchIndexManager) UpdateIndex(ctx context.Context, updated *UserSearchIndex) (int, error) {
	existing := sim.GetIndexByID(updated.ID)
	if existing == nil {
		return http.StatusNotFound, ucerr.Errorf("Index not found: %v", updated.ID)
	}

	if existing.Equals(updated) {
		return http.StatusNotModified, nil
	}

	if err := updated.Validate(); err != nil {
		return http.StatusBadRequest, ucerr.Wrap(err)
	}

	nameChanged := !strings.EqualFold(existing.Name, updated.Name)
	existingColumnIDs := set.NewUUIDSet(existing.ColumnIDs...)
	updatedColumnIDs := set.NewUUIDSet(updated.ColumnIDs...)

	if existing.IsEnabled() && updated.IsEnabled() {
		if nameChanged {
			return http.StatusBadRequest,
				ucerr.Friendlyf(nil, "Cannot change index name when it is enabled")
		}

		if existing.Type != updated.Type {
			return http.StatusBadRequest,
				ucerr.Friendlyf(nil, "Cannot change index type when it is enabled")
		}

		if !existing.Settings.Equals(updated.Settings) {
			return http.StatusBadRequest,
				ucerr.Friendlyf(nil, "Cannot change index settings when it is enabled")
		}

		if !existingColumnIDs.Equal(updatedColumnIDs) {
			return http.StatusBadRequest,
				ucerr.Friendlyf(nil, "Cannot change index column ids when it is enabled")
		}

		if existing.IsBootstrapped() &&
			updated.IsBootstrapped() &&
			!timestamp.NormalizedEqual(existing.Bootstrapped, updated.Bootstrapped) {
			return http.StatusBadRequest,
				ucerr.Friendlyf(nil, "Already set Bootstrapped timestamp should not change")
		}

		if !timestamp.NormalizedEqual(existing.Enabled, updated.Enabled) {
			return http.StatusBadRequest,
				ucerr.Friendlyf(nil, "Already set Enabled timestamp should not change")
		}

		if existing.IsSearchable() &&
			updated.IsSearchable() &&
			!timestamp.NormalizedEqual(existing.Searchable, updated.Searchable) {
			return http.StatusBadRequest,
				ucerr.Friendlyf(nil, "Already set Searchable timestamp should not change")
		}
	}

	if existing.IsSearchable() && !updated.IsSearchable() {
		for _, index := range sim.indicesByAccessorID {
			if index.UserSearchIndexID == updated.ID {
				return http.StatusBadRequest,
					ucerr.Friendlyf(nil, "Searchable index is still associated with an accessor")
			}
		}
	}

	if nameChanged {
		if conflicting := sim.GetIndexByName(updated.Name); conflicting != nil {
			return http.StatusConflict,
				ucerr.Wrap(
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error: fmt.Sprintf(
								"An index with the name '%s' already exists with ID '%v'",
								updated.Name,
								conflicting.ID,
							),
							ID: updated.ID,
						},
					),
				)
		}
	}

	addedColumnIDs := updatedColumnIDs.Difference(existingColumnIDs)

	if (!existing.IsEnabled() && updated.IsEnabled()) || addedColumnIDs.Size() > 0 {
		cm, err := NewUserstoreColumnManager(ctx, sim.s)
		if err != nil {
			return http.StatusInternalServerError, ucerr.Wrap(err)
		}

		// confirm any added columns have a compatible data type

		for _, columnID := range addedColumnIDs.Items() {
			c := cm.GetColumnByID(columnID)
			if c == nil {
				return http.StatusBadRequest,
					ucerr.Friendlyf(
						nil,
						"ColumnIDs contains an unrecognized column: %v",
						columnID,
					)
			}

			if err := c.confirmSearchIndexable(); err != nil {
				return http.StatusBadRequest,
					ucerr.Friendlyf(
						err,
						"ColumnIDs contains column '%v' that cannot be indexed in search",
						columnID,
					)
			}
		}

		// if update is enabling the index, confirm each column is search indexed

		if updated.IsEnabled() {
			for _, columnID := range updated.ColumnIDs {
				c := cm.GetColumnByID(columnID)
				if c == nil {
					return http.StatusInternalServerError,
						ucerr.Friendlyf(
							nil,
							"ColumnIDs contains an unrecognized column: %v",
							columnID,
						)
				}

				if !c.SearchIndexed {
					return http.StatusBadRequest,
						ucerr.Friendlyf(
							nil,
							"ColumnIDs contains column '%v' that is not SearchIndexed",
							columnID,
						)
				}
			}
		}
	}

	if err := sim.s.SaveUserSearchIndex(ctx, updated); err != nil {
		return uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
	}

	sim.lock.Lock()
	sim.indicesByID[updated.ID] = *updated
	if nameChanged {
		delete(sim.indicesByName, strings.ToLower(existing.Name))
	}
	sim.indicesByName[strings.ToLower(updated.Name)] = *updated
	sim.lock.Unlock()

	return http.StatusOK, nil
}
