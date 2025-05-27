package parametertype

// ButtonText is a string parameter type
const ButtonText Type = "button_text"

func init() {
	validator := func(v string) bool {
		return v != ""
	}

	if err := registerParameterType(ButtonText, validator); err != nil {
		panic(err)
	}
}
