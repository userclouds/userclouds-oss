package ucopensearch

import (
	"github.com/gofrs/uuid"
)

// NameableIndex is an interface for generating identifying names for the index
type NameableIndex interface {
	GetID() uuid.UUID
	GetIndexName(tenantID uuid.UUID) string
	GetTableName() string
}

// DefinableIndex is an interface that can produce an opensearch index definition
type DefinableIndex interface {
	NameableIndex
	GetIndexDefinition() string
}

// QueryableIndex is an interface that can produce an appropriate opensearch query
type QueryableIndex interface {
	NameableIndex
	GetIndexQuery(term string, numResults int) (string, error)
}
