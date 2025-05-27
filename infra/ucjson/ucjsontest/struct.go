package ucjsontest

// this file exists as it's own package because easyjson won't generate the
// code if it's in a _test file (strangly it creates the file itself, but no encoder/decoder)
// this seems expected (in easyjson/parser/parser.go:excludeTestFiles) if annoying.

// EasyJSONTest is a test struct for easyjson
//
//go:generate easyjson .
//easyjson:json
type EasyJSONTest struct {
	ID string `json:"id"`
}
