package generate

import "strings"

// GetPluralName returns the plural form of the given name
func GetPluralName(name string) string {
	if strings.HasSuffix(name, "y") && !strings.HasSuffix(name, "Key") {
		return name[:len(name)-1] + "ies"
	}
	if strings.HasSuffix(name, "x") {
		return name + "es"
	}
	return name + "s"
}
