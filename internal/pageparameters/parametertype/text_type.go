package parametertype

// Text is a string parameter type
const Text Type = "text"

func init() {
	validator := func(v string) bool {
		return v != ""
	}

	if err := registerParameterType(Text, validator); err != nil {
		panic(err)
	}
}
