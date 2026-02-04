package common

import (
	"fmt"
	"strings"
)

// Filter key constants for different resource types
var (
	EdgeFilterKeys       = []string{"id", "source_object_id", "target_object_id", "created", "updated"}
	EdgeTypeFilterKeys   = []string{"id", "type_name", "organization_id", "source_object_type_id", "target_object_type_id", "created", "updated"}
	ObjectFilterKeys     = []string{"id", "alias", "organization_id", "type_id", "created", "updated"}
	ObjectTypeFilterKeys = []string{"id", "type_name", "created", "updated"}
)

// filterToken represents a token in the filter expression
type filterToken struct {
	typ      string // "field", "and", "or", "lparen", "rparen"
	value    string
	operator string // "EQ", "LK", "NE", "NL" (only set for "field" type)
}

// tokenizeFilter breaks the filter string into tokens
// Supports operators: = (EQ), =~ (LK), != (NE), !~ (NL)
func tokenizeFilter(input string) ([]filterToken, error) {
	var tokens []filterToken
	var current strings.Builder

	for i := 0; i < len(input); i++ {
		ch := input[i]

		switch ch {
		case '&':
			if current.Len() > 0 {
				token, err := parseFieldToken(strings.TrimSpace(current.String()))
				if err != nil {
					return nil, err
				}
				tokens = append(tokens, token)
				current.Reset()
			}
			tokens = append(tokens, filterToken{typ: "and", value: "&"})

		case '|':
			if current.Len() > 0 {
				token, err := parseFieldToken(strings.TrimSpace(current.String()))
				if err != nil {
					return nil, err
				}
				tokens = append(tokens, token)
				current.Reset()
			}
			tokens = append(tokens, filterToken{typ: "or", value: "|"})

		case '(':
			if current.Len() > 0 {
				return nil, fmt.Errorf("unexpected characters before '('")
			}
			tokens = append(tokens, filterToken{typ: "lparen", value: "("})

		case ')':
			if current.Len() > 0 {
				token, err := parseFieldToken(strings.TrimSpace(current.String()))
				if err != nil {
					return nil, err
				}
				tokens = append(tokens, token)
				current.Reset()
			}
			tokens = append(tokens, filterToken{typ: "rparen", value: ")"})

		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		token, err := parseFieldToken(strings.TrimSpace(current.String()))
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// parseFieldToken parses a field expression (key<op>value) and returns a token
// Supported operators: = (EQ), =~ (LK), != (NE), !~ (NL)
func parseFieldToken(fieldExpr string) (filterToken, error) {
	// Try operators in order of length (longest first to avoid partial matches)
	operators := []struct {
		symbol string
		op     string
	}{
		{"=~", "LK"},
		{"!~", "NL"},
		{"!=", "NE"},
		{"=", "EQ"},
	}

	for _, opInfo := range operators {
		if idx := strings.Index(fieldExpr, opInfo.symbol); idx != -1 {
			key := strings.TrimSpace(fieldExpr[:idx])
			value := strings.TrimSpace(fieldExpr[idx+len(opInfo.symbol):])

			if key == "" || value == "" {
				return filterToken{}, fmt.Errorf("invalid filter format '%s': key and value cannot be empty", fieldExpr)
			}

			return filterToken{
				typ:      "field",
				value:    key + "=" + value, // Store in normalized format for error messages
				operator: opInfo.op,
			}, nil
		}
	}

	return filterToken{}, fmt.Errorf("invalid filter format '%s': expected key<operator>value (operators: =, =~, !=, !~)", fieldExpr)
}

// parseFilterExpression recursively parses filter tokens into a filter string
func parseFilterExpression(tokens []filterToken, validKeys []string) (string, int, error) {
	if len(tokens) == 0 {
		return "", 0, fmt.Errorf("unexpected end of expression")
	}

	// Parse primary expression (field or grouped expression)
	var left string
	var consumed int

	if tokens[0].typ == "lparen" {
		// Parse grouped expression
		result, n, err := parseFilterExpression(tokens[1:], validKeys)
		if err != nil {
			return "", 0, err
		}
		consumed = n + 1

		// Expect closing paren
		if consumed >= len(tokens) || tokens[consumed].typ != "rparen" {
			return "", 0, fmt.Errorf("missing closing parenthesis")
		}
		consumed++
		left = result

	} else if tokens[0].typ == "field" {
		// Parse field<op>value
		parts := strings.SplitN(tokens[0].value, "=", 2)
		if len(parts) != 2 {
			return "", 0, fmt.Errorf("invalid filter format '%s': expected key<op>value", tokens[0].value)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" || value == "" {
			return "", 0, fmt.Errorf("invalid filter format '%s': key and value cannot be empty", tokens[0].value)
		}

		// Validate the key is a supported field
		valid := false
		for _, validKey := range validKeys {
			if key == validKey {
				valid = true
				break
			}
		}
		if !valid {
			return "", 0, fmt.Errorf("unsupported filter field '%s': must be one of %v", key, validKeys)
		}

		// Use the operator from the token (EQ, LK, NE, or NL)
		operator := tokens[0].operator
		if operator == "" {
			operator = "EQ" // Default to EQ if not set
		}
		left = fmt.Sprintf("('%s',%s,'%s')", key, operator, value)
		consumed = 1

	} else {
		return "", 0, fmt.Errorf("unexpected token: %s", tokens[0].value)
	}

	// Check for binary operator (& or |)
	if consumed < len(tokens) {
		if tokens[consumed].typ == "and" {
			consumed++
			right, n, err := parseFilterExpression(tokens[consumed:], validKeys)
			if err != nil {
				return "", 0, err
			}
			consumed += n
			return fmt.Sprintf("(%s,AND,%s)", left, right), consumed, nil

		} else if tokens[consumed].typ == "or" {
			consumed++
			right, n, err := parseFilterExpression(tokens[consumed:], validKeys)
			if err != nil {
				return "", 0, err
			}
			consumed += n
			return fmt.Sprintf("(%s,OR,%s)", left, right), consumed, nil

		} else if tokens[consumed].typ == "rparen" {
			// Return without consuming the closing paren (let parent handle it)
			return left, consumed, nil
		}
	}

	return left, consumed, nil
}

// FormatFilterString constructs a filter string from a filter expression
// Supports:
//   - key=value for exact match (EQ operator)
//   - key=~value for pattern match with wildcards (LK operator)
//   - key!=value for not equal (NE operator)
//   - key!~value for negated pattern match (NL operator)
//   - & for AND operations
//   - | for OR operations
//   - () for grouping
//
// Examples:
//   - "type_name=user" - exact match
//   - "type_name=~user%" - pattern match (starts with "user")
//   - "type_id=550e8400-e29b-41d4-a716-446655440000" - exact UUID match
//   - "type_name!=admin" - not equal
//   - "type_name!~test%" - does not start with "test"
//   - "type_name=user&id=123" - AND operation
//   - "type_name=user|type_name=admin" - OR operation
//   - "(type_name=user|type_name=admin)&organization_id=456" - grouping
func FormatFilterString(filterInput string, validKeys []string) (string, error) {
	if filterInput == "" {
		return "", nil
	}

	// Tokenize the input
	tokens, err := tokenizeFilter(filterInput)
	if err != nil {
		return "", err
	}

	if len(tokens) == 0 {
		return "", nil
	}

	// Parse the expression
	result, consumed, err := parseFilterExpression(tokens, validKeys)
	if err != nil {
		return "", err
	}

	// Ensure all tokens were consumed
	if consumed != len(tokens) {
		return "", fmt.Errorf("unexpected tokens after expression")
	}

	return result, nil
}

// ValidateFilterFlags validates that filter flags are not both specified
func ValidateFilterFlags(filter, rawFilter string) error {
	if filter != "" && rawFilter != "" {
		return fmt.Errorf("cannot specify both --filter and --raw-filter")
	}
	return nil
}

// GetFilterString returns the appropriate filter string (raw or formatted)
// Returns error if both are specified or if formatting fails
func GetFilterString(filter, rawFilter string, validKeys []string) (string, error) {
	if err := ValidateFilterFlags(filter, rawFilter); err != nil {
		return "", err
	}

	if rawFilter != "" {
		return rawFilter, nil
	}

	if filter != "" {
		return FormatFilterString(filter, validKeys)
	}

	return "", nil
}
