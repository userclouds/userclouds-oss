package stringutils

import (
	"regexp"
	"strings"

	"github.com/iancoleman/strcase"
)

var r = regexp.MustCompile(`(Id|Uuid|Api)($|[A-Z0-9])`)

// ToUpperCamel converts/normalizes a string to a CamelCase string.
func ToUpperCamel(s string) string {
	camel := strcase.ToCamel(s)
	nm := r.ReplaceAllStringFunc(camel, func(m string) string {
		parts := r.FindStringSubmatch(m)
		return strings.ToUpper(parts[1]) + parts[2]
	})
	return strings.Replace(nm, "Ids", "IDs", 1)
}
