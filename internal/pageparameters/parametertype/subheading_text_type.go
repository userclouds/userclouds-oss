package parametertype

// SubheadingText is a string parameter type
const SubheadingText Type = "subheading_text"

func init() {
	validator := func(v string) bool {
		return true
	}

	if err := registerParameterType(SubheadingText, validator); err != nil {
		panic(err)
	}
}
