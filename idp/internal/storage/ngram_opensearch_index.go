package storage

import (
	"encoding/json"
	"fmt"
	"strings"

	"userclouds.com/idp/userstore/search"
	"userclouds.com/infra/ucerr"
)

const ngramUserSearchIndexMaxResults = 1001

// NgramUserSearchIndex is a single column user search index that indexes the column value using generated ngrams of the value.
type NgramUserSearchIndex struct {
	queryableUserSearchIndex
}

func newDefineableNgramUserSearchIndex(usi UserSearchIndex) (*NgramUserSearchIndex, error) {
	index := NgramUserSearchIndex{
		queryableUserSearchIndex: queryableUserSearchIndex{
			UserSearchIndex: usi,
		},
	}
	if err := index.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &index, nil
}

func newQueryableNgramUserSearchIndex(usi UserSearchIndex, qt search.QueryType) (*NgramUserSearchIndex, error) {
	index := NgramUserSearchIndex{
		queryableUserSearchIndex: queryableUserSearchIndex{
			UserSearchIndex: usi,
			queryType:       &qt,
		},
	}
	if err := index.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &index, nil
}

func (i NgramUserSearchIndex) extraValidate() error {
	if i.Settings.Ngram == nil {
		return ucerr.Friendlyf(nil, "Settings.Ngram must be specified")
	}

	if i.queryType != nil {
		switch *i.queryType {
		case search.QueryTypeTerm, search.QueryTypeWildcard:
		default:
			return ucerr.Friendlyf(nil, "query type '%v' is unsupported", i.queryType)
		}
	}

	if len(i.ColumnIDs) != 1 {
		return ucerr.Friendlyf(nil, "only one column ID supported")
	}

	return nil
}

//go:generate genvalidate NgramUserSearchIndex

// GetIndexDefinition is part of the ucopensearch.DefineableIndex interface
// and returns a string that represents the index definition
func (i NgramUserSearchIndex) GetIndexDefinition() string {
	return fmt.Sprintf(
		`
{
  "settings": {
    "index": {
      "max_ngram_diff": %d
    },
  "analysis": {
    "filter": {
      "ngram_filter": {
        "type": "ngram",
        "min_gram": %d,
        "max_gram": %d
      }
    },
    "analyzer": {
      "ngram_analyzer": {
        "type": "custom",
        "tokenizer": "keyword",
        "filter": ["lowercase", "ngram_filter"]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "%v": {
        "type": "text",
        "analyzer": "ngram_analyzer"
      }
    }
  }
}`,
		i.Settings.Ngram.MaxNgram-i.Settings.Ngram.MinNgram,
		i.Settings.Ngram.MinNgram,
		i.Settings.Ngram.MaxNgram,
		i.ColumnIDs[0],
	)
}

// GetIndexQuery is part of the ucopensearch.QueryableIndex interface and returns a
// string that can be used to query for the requested term and number of results.
func (i NgramUserSearchIndex) GetIndexQuery(
	term string,
	numResults int,
) (string, error) {
	if i.queryType == nil {
		return "", ucerr.Friendlyf(nil, "queryType must be specified")
	}

	if numResults <= 0 || numResults > ngramUserSearchIndexMaxResults {
		return "", ucerr.Friendlyf(nil, "numResults must be a positive number <= %d", ngramUserSearchIndexMaxResults)
	}

	if len(term) < i.Settings.Ngram.MinNgram || len(term) > i.Settings.Ngram.MaxNgram {
		return "", ucerr.Friendlyf(nil, "length of term must satisfy %d <= len(term) <= %d", i.Settings.Ngram.MinNgram, i.Settings.Ngram.MaxNgram)
	}

	columnID := i.ColumnIDs[0].String()
	queryObject := make(map[string]any)
	searchTerm := strings.ToLower(term)
	switch *i.queryType {
	case search.QueryTypeTerm:
		queryObject["term"] = map[string]any{columnID: searchTerm}
	case search.QueryTypeWildcard:
		queryObject["wildcard"] = map[string]any{columnID: fmt.Sprintf("*%s*", searchTerm)}
	default:
		return "", ucerr.Errorf("unsupported query type '%v'", i.queryType)
	}
	jsonBytes, err := json.Marshal(map[string]any{"size": numResults, "query": queryObject})
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	return string(jsonBytes), nil
}
