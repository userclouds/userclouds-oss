package search

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/userstore"
	"userclouds.com/infra/namespace/region"
)

// UserSearchIndexAccessor represents an accessor and query
// type bound to an opensearch index
type UserSearchIndexAccessor struct {
	Accessor  userstore.ResourceID `json:"accessor"`
	QueryType QueryType            `json:"query_type"`
}

// UserSearchIndex represents an opensearch index for users
type UserSearchIndex struct {
	ID                               uuid.UUID                       `json:"id"`
	Name                             string                          `json:"name"`
	Description                      string                          `json:"description"`
	DataLifeCycleState               userstore.DataLifeCycleState    `json:"data_life_cycle_state"`
	Type                             IndexType                       `json:"type"`
	Settings                         IndexSettings                   `json:"settings"`
	Accessors                        []UserSearchIndexAccessor       `json:"accessors"`
	Columns                          []userstore.ResourceID          `json:"columns"`
	LastRegionalBootstrappedValueIDs map[region.DataRegion]uuid.UUID `json:"last_regional_bootstrapped_value_ids"`
	IndexNameSuffix                  string                          `json:"index_name_suffix"`
	Bootstrapped                     time.Time                       `json:"bootstrapped"`
	Enabled                          time.Time                       `json:"enabled"`
	Searchable                       time.Time                       `json:"searchable"`
}

// Disable updates the user search index to be disabled
func (usi *UserSearchIndex) Disable() {
	usi.Searchable = time.Time{}
	usi.Bootstrapped = time.Time{}
	usi.Enabled = time.Time{}
	usi.LastRegionalBootstrappedValueIDs = map[region.DataRegion]uuid.UUID{}
}

// GetIndexName returns the index name for the specified tenant ID
func (usi UserSearchIndex) GetIndexName(tenantID uuid.UUID) string {
	return fmt.Sprintf("%v%s", tenantID, usi.IndexNameSuffix)
}
