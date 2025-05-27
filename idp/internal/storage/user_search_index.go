package storage

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/search"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/infra/uctypes/uuidarray"
	"userclouds.com/internal/ucopensearch"
)

// AccessorSearchIndex maps an accessor to a searchable user search index with the specified query type
type AccessorSearchIndex struct {
	ucdb.BaseModel // ID is the Accessor ID

	UserSearchIndexID uuid.UUID        `db:"user_search_index_id" validate:"notnil"`
	QueryType         search.QueryType `db:"query_type"`
}

func (AccessorSearchIndex) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"query_type":           pagination.StringKeyType,
		"user_search_index_id": pagination.TimestampKeyType,
	}
}

// Equals checks whether two accessor search indices are equal
func (asi AccessorSearchIndex) Equals(o *AccessorSearchIndex) bool {
	return asi.ID == o.ID &&
		asi.UserSearchIndexID == o.UserSearchIndexID &&
		asi.QueryType == o.QueryType
}

//go:generate genpageable AccessorSearchIndex

//go:generate genvalidate AccessorSearchIndex

//go:generate genorm --cache --followerreads --nonpaginatedlist AccessorSearchIndex accessor_search_indices tenantdb

// RegionalBootstrappedValueIDMap maps regional data regions to the last bootstrapped value id
type RegionalBootstrappedValueIDMap map[region.DataRegion]uuid.UUID

//go:generate gendbjson RegionalBootstrappedValueIDMap

// UserSearchIndex represents an opensearch index for user ids
type UserSearchIndex struct {
	ucdb.BaseModel

	Name                             string                         `db:"name" validate:"notempty"`
	Description                      string                         `db:"description"`
	DataLifeCycleState               column.DataLifeCycleState      `db:"data_life_cycle_state"`
	Type                             search.IndexType               `db:"type"`
	Settings                         search.IndexSettings           `db:"settings" validate:"skip"`
	ColumnIDs                        uuidarray.UUIDArray            `db:"column_ids" validate:"skip"`
	LastRegionalBootstrappedValueIDs RegionalBootstrappedValueIDMap `db:"last_regional_bootstrapped_value_ids"`
	Bootstrapped                     time.Time                      `db:"bootstrapped"`
	Enabled                          time.Time                      `db:"enabled"`
	Searchable                       time.Time                      `db:"searchable"`
}

// NewUserSearchIndexFromClient creates a new user search index from client counterpart
func NewUserSearchIndexFromClient(cusi search.UserSearchIndex) UserSearchIndex {
	usi := UserSearchIndex{
		Name:                             cusi.Name,
		Description:                      cusi.Description,
		DataLifeCycleState:               column.DataLifeCycleStateFromClient(cusi.DataLifeCycleState),
		Type:                             cusi.Type,
		Settings:                         cusi.Settings,
		LastRegionalBootstrappedValueIDs: cusi.LastRegionalBootstrappedValueIDs,
		Bootstrapped:                     cusi.Bootstrapped,
		Enabled:                          cusi.Enabled,
		Searchable:                       cusi.Searchable,
	}

	for _, c := range cusi.Columns {
		usi.ColumnIDs = append(usi.ColumnIDs, c.ID)
	}

	if cusi.ID.IsNil() {
		usi.BaseModel = ucdb.NewBase()
	} else {
		usi.BaseModel = ucdb.NewBaseWithID(cusi.ID)
	}

	return usi
}

func (usi UserSearchIndex) extraValidate() error {
	if usi.Type.IsDeprecated() {
		return ucerr.Friendlyf(nil, "deprecated search index is not supported")
	}

	if usi.DataLifeCycleState != column.DataLifeCycleStateLive {
		return ucerr.Friendlyf(nil, "Only live data can be search-indexed right now")
	}

	uniqueColumnIDs := set.NewUUIDSet(usi.ColumnIDs...)
	if uniqueColumnIDs.Size() != len(usi.ColumnIDs) {
		return ucerr.Friendlyf(nil, "ColumnIDs must be unique")
	}

	if usi.IsEnabled() {
		if len(usi.ColumnIDs) == 0 {
			return ucerr.Friendlyf(nil, "ColumnIDs cannot be empty if Enabled is set")
		}

		if _, err := usi.GetDefinableIndex(); err != nil {
			return ucerr.Wrap(err)
		}
	} else {
		for _, val := range usi.LastRegionalBootstrappedValueIDs {
			if val.IsNil() {
				return ucerr.Friendlyf(nil, "LastRegionalBootstrappedValueIDs must all be nil if Enabled is unset")
			}
		}
		if usi.IsBootstrapped() {
			return ucerr.Friendlyf(nil, "Bootstrapped cannot be set if Enabled is unset")
		}
	}

	if usi.IsBootstrapped() {
		for _, val := range usi.LastRegionalBootstrappedValueIDs {
			if val.IsNil() {
				return ucerr.Friendlyf(nil, "LastRegionalBootstrappedValueIDs must all be nil if Bootstrapped is set")
			}
		}
	} else if usi.IsSearchable() {
		return ucerr.Friendlyf(nil, "Searchable cannot be set if Bootstrapped is unset")
	}

	return nil
}

func (usi UserSearchIndex) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "name,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"name:%v,id:%v",
				usi.Name,
				usi.ID,
			),
		)
	}
}

func (UserSearchIndex) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"bootstrapped":       pagination.TimestampKeyType,
		"change_feed_job_id": pagination.UUIDKeyType,
		"column_ids":         pagination.UUIDArrayKeyType,
		"created":            pagination.TimestampKeyType,
		"enabled":            pagination.TimestampKeyType,
		"name":               pagination.StringKeyType,
		"searchable":         pagination.TimestampKeyType,
		"type":               pagination.StringKeyType,
		"updated":            pagination.TimestampKeyType,
	}
}

// CheckAccessorCompatible returns an error if the accessor and query type are not compatible with the user search index
func (usi UserSearchIndex) CheckAccessorCompatible(a Accessor, qt search.QueryType) error {
	if a.SearchColumnID.IsNil() {
		return ucerr.Friendlyf(nil, "Accessor '%v' does not have a search supported selector", a.ID)
	}

	if !usi.IsSearchable() {
		return ucerr.Friendlyf(nil, "Accessor '%v' cannot be associated with non-searchable UserSearchIndex '%v'", a.ID, usi.ID)
	}

	if _, err := usi.GetQueryableIndex(qt); err != nil {
		return ucerr.Wrap(err)
	}

	if slices.Contains(usi.ColumnIDs, a.SearchColumnID) {
		return nil
	}

	return ucerr.Friendlyf(nil, "Accessor '%v' search Column '%v' is not compatible with UserSearchIndex '%v'", a.ID, a.SearchColumnID, usi.ID)
}

// CheckColumnUnIndexed returns an error if the user search index is enabled and indexes the column
func (usi UserSearchIndex) CheckColumnUnIndexed(c Column) error {
	if !usi.IsEnabled() {
		return nil
	}

	if slices.Contains(usi.ColumnIDs, c.ID) {
		return ucerr.Friendlyf(nil, "column %v is part of enabled UserSearchIndex %v and must be SearchIndexed", c.ID, usi.ID)
	}

	return nil
}

// Equals checks whether two user search indices are equal
func (usi UserSearchIndex) Equals(o *UserSearchIndex) bool {
	if usi.EquivalentTo(o) &&
		usi.ID == o.ID &&
		usi.Bootstrapped == o.Bootstrapped &&
		usi.Description == o.Description &&
		usi.Enabled == o.Enabled &&
		usi.Searchable == o.Searchable {
		for r, val := range usi.LastRegionalBootstrappedValueIDs {
			if val != o.LastRegionalBootstrappedValueIDs[r] {
				return false
			}
		}
		return true
	}
	return false
}

// EquivalentTo checks whether two user search indexes are logically equivalent
func (usi UserSearchIndex) EquivalentTo(o *UserSearchIndex) bool {
	if len(usi.ColumnIDs) != len(o.ColumnIDs) {
		return false
	}

	columnIDs := set.NewUUIDSet(usi.ColumnIDs...)
	for _, columnID := range o.ColumnIDs {
		if !columnIDs.Contains(columnID) {
			return false
		}
	}

	return strings.EqualFold(usi.Name, o.Name) &&
		usi.Settings.Equals(o.Settings) &&
		usi.Type == o.Type
}

// GetDefinableIndex returns an appropriate DefinableIndex for the user search index
func (usi UserSearchIndex) GetDefinableIndex() (ucopensearch.DefinableIndex, error) {
	switch usi.Type {
	case search.IndexTypeNgram:
		di, err := newDefineableNgramUserSearchIndex(usi)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		return di, nil
	}

	return nil, ucerr.Friendlyf(nil, "IndexType '%v' is unsupported", usi.Type)
}

// GetID is part of the ucopensearch.NameableIndex interface
func (usi UserSearchIndex) GetID() uuid.UUID {
	return usi.ID
}

// GetIndexName is part of the ucopensearch.NameableIndex interface
func (usi UserSearchIndex) GetIndexName(tenantID uuid.UUID) string {
	return usi.ToClientModel().GetIndexName(tenantID)
}

func (usi UserSearchIndex) getIndexNameSuffix() string {
	return fmt.Sprintf("_%s_%v", usi.GetTableName(), usi.ID)
}

// GetQueryableIndex returns an appropriate QueryableIndex for the user search index and specified query type
func (usi UserSearchIndex) GetQueryableIndex(qt search.QueryType) (ucopensearch.QueryableIndex, error) {
	switch usi.Type {
	case search.IndexTypeNgram:
		qi, err := newQueryableNgramUserSearchIndex(usi, qt)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		return qi, nil
	}

	return nil, ucerr.Friendlyf(nil, "index type '%v' is unsupported", usi.Type)
}

// GetTableName is part of the ucopensearch.NameableIndex interface
func (usi UserSearchIndex) GetTableName() string {
	if usi.DataLifeCycleState == column.DataLifeCycleStateSoftDeleted {
		return "user_column_post_delete_values"
	}

	return "user_column_pre_delete_values"
}

// IsBootstrapped returns whether the index has been bootstrapped
func (usi UserSearchIndex) IsBootstrapped() bool {
	return !usi.Bootstrapped.IsZero()
}

// IsEnabled returns whether the index is enabled
func (usi UserSearchIndex) IsEnabled() bool {
	return !usi.Enabled.IsZero()
}

// IsSearchable returns whether the index is searchable
func (usi UserSearchIndex) IsSearchable() bool {
	return !usi.Searchable.IsZero()
}

func (usi UserSearchIndex) supportsColumnID(id uuid.UUID) bool {
	return slices.Contains(usi.ColumnIDs, id)
}

// ToClientModel translates the user search index to the client model
func (usi UserSearchIndex) ToClientModel() search.UserSearchIndex {
	cusi := search.UserSearchIndex{
		ID:                               usi.ID,
		Name:                             usi.Name,
		Description:                      usi.Description,
		DataLifeCycleState:               usi.DataLifeCycleState.ToClient(),
		Type:                             usi.Type,
		Settings:                         usi.Settings,
		LastRegionalBootstrappedValueIDs: usi.LastRegionalBootstrappedValueIDs,
		IndexNameSuffix:                  usi.getIndexNameSuffix(),
		Bootstrapped:                     usi.Bootstrapped,
		Enabled:                          usi.Enabled,
		Searchable:                       usi.Searchable,
	}

	for _, columnID := range usi.ColumnIDs {
		cusi.Columns = append(cusi.Columns, userstore.ResourceID{ID: columnID})
	}

	return cusi
}

//go:generate genpageable UserSearchIndex

//go:generate genvalidate UserSearchIndex

//go:generate genorm --cache --followerreads --nonpaginatedlist UserSearchIndex user_search_indices tenantdb

// queryableUserSearchIndex is a queryable UserSearchIndex
type queryableUserSearchIndex struct {
	UserSearchIndex

	queryType *search.QueryType
}

// Validate implements the Validateable interface
func (qusi queryableUserSearchIndex) Validate() error {
	// NOTE: we intentionally do not revalidate UserSearchIndex here
	if qusi.queryType != nil {
		if err := qusi.queryType.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
