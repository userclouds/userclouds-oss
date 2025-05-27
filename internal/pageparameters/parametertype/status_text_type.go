package parametertype

// StatusText is a string parameter type
const StatusText Type = "status_text"

func init() {
	validator := func(v string) bool {
		return v != ""
	}

	if err := registerParameterType(StatusText, validator); err != nil {
		panic(err)
	}
}
