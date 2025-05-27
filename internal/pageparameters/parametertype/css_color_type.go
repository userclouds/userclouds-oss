package parametertype

import "regexp"

// CSSColor is a parameter type representing a color used for CSS
const CSSColor Type = "css_color"

const transparentCSSColor = "transparent"

func init() {
	colorHexPattern := regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`)
	validator := func(v string) bool {
		return v == transparentCSSColor || colorHexPattern.MatchString(v)
	}

	if err := registerParameterType(CSSColor, validator); err != nil {
		panic(err)
	}
}
