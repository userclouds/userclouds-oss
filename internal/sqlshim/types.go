package sqlshim

import (
	"context"

	"github.com/gofrs/uuid"
)

// Column represents a column in a table of a SQL database
type Column struct {
	Schema string
	Table  string
	Name   string
	Type   string
	Length int
}

// HandleQueryResponse is an enum for the possible responses to HandleQuery
type HandleQueryResponse int

const (
	// Passthrough indicates that the query should be passed through to the SQL database, and the results passed back without modification
	Passthrough HandleQueryResponse = iota

	// TransformResponse indicates that the query should be passed through to the SQL database, and the results should be transformed before being passed back
	TransformResponse

	// AccessDenied indicates that the query should be blocked and an access denied error returned
	AccessDenied
)

// Observer is an interface for handling SQL queries
type Observer interface {
	NotifySchemaSelected(ctx context.Context, schema string)
	HandleQuery(ctx context.Context, dbt DatabaseType, query string, tableSchema string, connectionID uuid.UUID) (HandleQueryResponse, any, string, error)
	CleanupTransformerExecution(transformInfo any)
	TransformDataRow(ctx context.Context, colNames []string, values [][]byte, transformInfo any, cumulativeRows int) (bool, error)
	TransformSummary(ctx context.Context, transformInfo any, numSelectorRows, numReturned, numDenied int)
}
