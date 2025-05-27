package userstore

import (
	"strings"
	"testing"

	"userclouds.com/infra/assert"
)

func TestKeyValueRegex(t *testing.T) {
	testCases := []struct {
		name     string
		comment  string
		expected map[string]string
	}{
		{
			name:    "with whitespace",
			comment: "/* location='US', org='50', user='830474' */",
			expected: map[string]string{
				"location": "US",
				"org":      "50",
				"user":     "830474",
			},
		},
		{
			name:    "without whitespace",
			comment: "/*location='US',org='50',user='830474'*/",
			expected: map[string]string{
				"location": "US",
				"org":      "50",
				"user":     "830474",
			},
		},
		{
			name:    "mixed whitespace",
			comment: "/*location='US', org='50',user='830474' */",
			expected: map[string]string{
				"location": "US",
				"org":      "50",
				"user":     "830474",
			},
		},
		{
			name:    "with special characters in values",
			comment: "/* location='US, East', org='dept=finance', user='user, with, commas' */",
			expected: map[string]string{
				"location": "US, East",
				"org":      "dept=finance",
				"user":     "user, with, commas",
			},
		},
		{
			name:    "with double quotes",
			comment: `/* location="US, East", org="dept=finance", user="user, with, commas" */`,
			expected: map[string]string{
				"location": `US, East`,
				"org":      `dept=finance`,
				"user":     `user, with, commas`,
			},
		},
		{
			name:    "with mixed quotes",
			comment: `/* location="US, East", org='dept=finance', user="user, with, commas" */`,
			expected: map[string]string{
				"location": `US, East`,
				"org":      `dept=finance`,
				"user":     `user, with, commas`,
			},
		},
		{
			name:    "with unquoted values",
			comment: "/* foo=bar baz=tab */",
			expected: map[string]string{
				"foo": "bar",
				"baz": "tab",
			},
		},
		{
			name:    "with mixed quoted and unquoted values",
			comment: "/* location='US' org=50 user='830474' */",
			expected: map[string]string{
				"location": "US",
				"org":      "50",
				"user":     "830474",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Extract comments
			comments := commentRegex.FindAllStringSubmatch(tc.comment, -1)
			assert.Equal(t, len(comments), 1, assert.Must())

			// Extract content between comment markers
			comment := comments[0][0]
			// Remove the comment markers and any whitespace after/before them
			commentContent := strings.TrimSuffix(strings.TrimPrefix(comment, "/*"), "*/")

			result := make(map[string]string)

			match, err := keyValueRegex.FindStringMatch(commentContent)
			assert.NoErr(t, err)
			for match != nil {
				key := match.GroupByNumber(1).String()
				val := match.GroupByNumber(3).String()

				result[key] = val

				match, err = keyValueRegex.FindNextMatch(match)
				if err != nil {
					assert.NoErr(t, err)
				}
			}

			assert.Equal(t, result, tc.expected)
		})
	}
}
