package parametertype

// LabelText is a string parameter type
const LabelText Type = "label_text"

func init() {
	validator := func(v string) bool {
		return v != ""
	}

	if err := registerParameterType(LabelText, validator); err != nil {
		panic(err)
	}
}
