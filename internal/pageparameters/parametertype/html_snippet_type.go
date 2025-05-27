package parametertype

import (
	"regexp"

	"github.com/microcosm-cc/bluemonday"
)

// HTMLSnippet is a parameter type representing an HTML snippet
const HTMLSnippet Type = "html_snippet"

func init() {
	// We only allow anchor tags in the snippet, and the only allowed attributes
	// within an anchor are href, rel, target, and title.
	snippetPolicy := bluemonday.StrictPolicy()
	snippetPolicy.AllowStandardURLs()
	snippetPolicy.RequireNoFollowOnLinks(false)
	snippetPolicy.AllowAttrs("href", "rel", "target", "title").OnElements("a")

	// bluemonday assumes the html is properly formed, so perform an additional
	// check to ensure any specified anchors have proper start and end tags
	balancedAnchorPattern := regexp.MustCompile("<[Aa].+?>.+?</[Aa]>")
	startAnchorPattern := regexp.MustCompile("<[Aa].+?>")
	endAnchorPattern := regexp.MustCompile("</[Aa]>")

	validator := func(snippet string) bool {
		if sanitized := snippetPolicy.Sanitize(snippet); sanitized != snippet {
			return false
		}

		snippetBytes := []byte(snippet)
		numBalancedAnchors := len(balancedAnchorPattern.FindAllIndex(snippetBytes, -1))
		if numBalancedAnchors != len(startAnchorPattern.FindAllIndex(snippetBytes, -1)) ||
			numBalancedAnchors != len(endAnchorPattern.FindAllIndex(snippetBytes, -1)) {
			return false
		}

		return true
	}

	if err := registerParameterType(HTMLSnippet, validator); err != nil {
		panic(err)
	}
}
