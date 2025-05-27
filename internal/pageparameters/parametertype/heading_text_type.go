package parametertype

// HeadingText is a string parameter type
const HeadingText Type = "heading_text"

func init() {
	validator := func(v string) bool {
		return v != ""
	}

	if err := registerParameterType(HeadingText, validator); err != nil {
		panic(err)
	}
}
