package ucerr

// ErrorWithName gives a name to the error
type ErrorWithName interface {
	error
	Name() string
}

type errorWithName struct {
	error
	name string
}

func (e errorWithName) Name() string {
	return e.name
}

// WrapWithName wraps an error with a name
func WrapWithName(err error, name string) ErrorWithName {
	return errorWithName{
		error: Wrap(err, ExtraSkip()),
		name:  name,
	}
}
