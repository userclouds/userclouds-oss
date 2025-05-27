package search

// QueryType is an enum for supported search query types
type QueryType string

// supported QueryTypes
const (
	QueryTypeTerm     QueryType = "term"
	QueryTypeWildcard QueryType = "wildcard"
)

//go:generate genconstant QueryType
