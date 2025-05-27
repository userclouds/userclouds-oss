package search

// IndexType is an enum for supported search index types
type IndexType string

// supported IndexTypes
const (
	IndexTypeDeprecated IndexType = "deprecated"
	IndexTypeNgram      IndexType = "ngram"
)

//go:generate genconstant IndexType

// IsDeprecated returns true if this is the deprecated index type
func (it IndexType) IsDeprecated() bool {
	return it == IndexTypeDeprecated
}
