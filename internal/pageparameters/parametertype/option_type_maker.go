package parametertype

import (
	"regexp"
	"strings"

	"userclouds.com/infra/ucerr"
)

type optionTypeRestriction int

const (
	requireAll optionTypeRestriction = iota
	requireAtLeastOne
	requireExactlyOne
	requireAny
)

const matchAllOptionTypes = "*"

const regexpPrefix = "regexp:"

// GetOptions will return a slice of all options in a comma-delimited string
func GetOptions(value string) (options []string) {
	if value == "" {
		return options
	}

	return strings.Split(value, ",")
}

func makeOptionValidator(otr optionTypeRestriction, optionTypes string) (pv parameterValidator, err error) {
	var optionMatcher *regexp.Regexp
	if strings.HasPrefix(optionTypes, regexpPrefix) {
		optionMatcher = regexp.MustCompile(strings.TrimPrefix(optionTypes, regexpPrefix))
		optionTypes = matchAllOptionTypes
	} else {
		optionMatcher = regexp.MustCompile(".")
	}

	allowedElements := strings.Split(optionTypes, ",")
	uniqueAllowedElements := map[string]bool{}
	for _, ae := range allowedElements {
		if len(strings.TrimSpace(ae)) != len(ae) {
			return pv, ucerr.New("allowedElements cannot have leading or trailing spaces")
		}
		if ae != "" {
			uniqueAllowedElements[ae] = true
		}
	}
	if len(allowedElements) == 0 || len(allowedElements) != len(uniqueAllowedElements) {
		return pv, ucerr.New("allowedElements must be non-empty and unique")
	}

	return func(v string) bool {
		if otr == requireAny && strings.TrimSpace(v) == "" {
			return true
		}

		elements := strings.Split(v, ",")
		uniqueElements := map[string]bool{}
		for _, e := range elements {
			e = strings.TrimSpace(e)
			if !optionMatcher.MatchString(e) {
				return false
			}
			if optionTypes == matchAllOptionTypes {
				if e == "" || uniqueElements[e] {
					return false
				}
			} else if !uniqueAllowedElements[e] || uniqueElements[e] {
				return false
			}
			uniqueElements[e] = true
		}

		switch otr {
		case requireExactlyOne:
			return len(uniqueElements) == 1
		case requireAtLeastOne:
			return len(uniqueElements) > 0
		case requireAll:
			return len(uniqueElements) == len(uniqueAllowedElements)
		case requireAny:
			return true
		default:
			return false
		}
	}, nil
}
