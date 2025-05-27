package ucerr

// Warning is simpy a wrapper around an error that will be logged as a warning instead of an error
type Warning struct {
	error
	text string
}

// NewWarning creates a new error object that will be automatically logged as a
// warning (instead of an error) by things like uchttp.Error() and jsonapi.MarshalError()
// This is useful for errors that are "expected" like invalid tenant names (caused by
// invalid external requests that we can't resolve, want to propagate as golang errors, but
// shouldn't trigger error alarms etc)
func NewWarning(text string) error {
	return Warning{
		error: New(text),
		text:  text,
	}
}
