package parametertype

// Bool is a bool parameter type
const Bool Type = "bool"

func init() {
	validator := func(v string) bool {
		return v == "true" || v == "false"
	}

	if err := registerParameterType(Bool, validator); err != nil {
		panic(err)
	}
}
