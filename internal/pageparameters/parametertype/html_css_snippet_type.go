package parametertype

import (
	"regexp"
)

// HTMLCSSSnippet is a parameter type representing an HTML snippet that may contain CSS (but not JavaScript)
const HTMLCSSSnippet Type = "html_css_snippet"

func init() {
	scriptTags := regexp.MustCompile("<script>")

	validator := func(snippet string) bool {
		return scriptTags.FindAll([]byte(snippet), -1) == nil
	}

	if err := registerParameterType(HTMLCSSSnippet, validator); err != nil {
		panic(err)
	}
}
