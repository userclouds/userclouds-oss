package storage

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/search"
	"userclouds.com/infra/assert"
)

func uniqueName(name string) string {
	return name + "_" + uuid.Must(uuid.NewV4()).String()
}
func TestNgramUserSearchIndex_GetIndexQuery(t *testing.T) {
	columnID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name        string
		queryType   search.QueryType
		term        string
		numResults  int
		wantErr     bool
		errContains string
		wantQuery   string
	}{
		{
			name:       "valid term query",
			queryType:  search.QueryTypeTerm,
			term:       "test",
			numResults: 10,
			wantQuery:  fmt.Sprintf(`{"query":{"term":{"%s":"test"}},"size":10}`, columnID),
		},
		{
			name:       "search term with a quote",
			queryType:  search.QueryTypeTerm,
			term:       `"jerry`,
			numResults: 10,
			wantQuery:  fmt.Sprintf(`{"query":{"term":{"%s":"\"jerry"}},"size":10}`, columnID),
		},
		{
			name:       "valid wildcard query",
			queryType:  search.QueryTypeWildcard,
			term:       "test",
			numResults: 10,
			wantQuery:  fmt.Sprintf(`{"query":{"wildcard":{"%s":"*test*"}},"size":10}`, columnID),
		},
		{
			name:        "zero results",
			queryType:   search.QueryTypeTerm,
			term:        "test",
			numResults:  0,
			wantErr:     true,
			errContains: "numResults must be a positive number",
		},
		{
			name:        "too many results",
			queryType:   search.QueryTypeTerm,
			term:        "test",
			numResults:  1002,
			wantErr:     true,
			errContains: "numResults must be a positive number",
		},
		{
			name:        "term too short",
			queryType:   search.QueryTypeTerm,
			term:        "ab",
			numResults:  10,
			wantErr:     true,
			errContains: "length of term must satisfy",
		},
		{
			name:        "term too long",
			queryType:   search.QueryTypeTerm,
			term:        "verylongstring12345",
			numResults:  10,
			wantErr:     true,
			errContains: "length of term must satisfy",
		},
		{
			name:       "max allowed results",
			queryType:  search.QueryTypeTerm,
			term:       "test",
			numResults: 1001,
			wantQuery:  fmt.Sprintf(`{"query":{"term":{"%s":"test"}},"size":1001}`, columnID),
		},
		{
			name:       "term at min length",
			queryType:  search.QueryTypeTerm,
			term:       "abc",
			numResults: 10,
			wantQuery:  fmt.Sprintf(`{"query":{"term":{"%s":"abc"}},"size":10}`, columnID),
		},
		{
			name:       "term at max length",
			queryType:  search.QueryTypeTerm,
			term:       "abcdefghij",
			numResults: 10,
			wantQuery:  fmt.Sprintf(`{"query":{"term":{"%s":"abcdefghij"}},"size":10}`, columnID),
		},
	}

	searchIndex := NewUserSearchIndexFromClient(search.UserSearchIndex{
		Name:               uniqueName("search_index"),
		DataLifeCycleState: userstore.DataLifeCycleStateLive,
		Type:               search.IndexTypeNgram,
		Settings: search.IndexSettings{
			Ngram: &search.NgramIndexSettings{MinNgram: 3, MaxNgram: 10},
		},
		Columns: []userstore.ResourceID{{ID: columnID}},
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, err := searchIndex.GetQueryableIndex(tt.queryType)
			assert.NoErr(t, err)
			got, err := idx.GetIndexQuery(tt.term, tt.numResults)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			assert.NoErr(t, err)
			assert.Equal(t, tt.wantQuery, got)
		})
	}
}
