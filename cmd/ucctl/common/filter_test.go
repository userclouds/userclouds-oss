package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatFilterString(t *testing.T) {
	validKeys := []string{"id", "type_name", "organization_id", "created", "updated"}

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		// Basic operators
		{
			name:     "exact match with = operator",
			input:    "type_name=user",
			expected: "('type_name',EQ,'user')",
			wantErr:  false,
		},
		{
			name:     "pattern match with =~ operator",
			input:    "type_name=~user%",
			expected: "('type_name',LK,'user%')",
			wantErr:  false,
		},
		{
			name:     "not equal with != operator",
			input:    "type_name!=admin",
			expected: "('type_name',NE,'admin')",
			wantErr:  false,
		},
		{
			name:     "not like with !~ operator",
			input:    "type_name!~test%",
			expected: "('type_name',NL,'test%')",
			wantErr:  false,
		},

		// UUID exact matches
		{
			name:     "UUID field with exact match",
			input:    "id=550e8400-e29b-41d4-a716-446655440000",
			expected: "('id',EQ,'550e8400-e29b-41d4-a716-446655440000')",
			wantErr:  false,
		},
		{
			name:     "UUID field with pattern match",
			input:    "id=~550e8400%",
			expected: "('id',LK,'550e8400%')",
			wantErr:  false,
		},

		// AND operations
		{
			name:     "AND with two exact matches",
			input:    "type_name=user&id=123",
			expected: "(('type_name',EQ,'user'),AND,('id',EQ,'123'))",
			wantErr:  false,
		},
		{
			name:     "AND with mixed operators",
			input:    "type_name=~user%&id=123",
			expected: "(('type_name',LK,'user%'),AND,('id',EQ,'123'))",
			wantErr:  false,
		},
		{
			name:     "AND with three conditions",
			input:    "type_name=user&id=123&created=2024-01-01",
			expected: "(('type_name',EQ,'user'),AND,(('id',EQ,'123'),AND,('created',EQ,'2024-01-01')))",
			wantErr:  false,
		},

		// OR operations
		{
			name:     "OR with two exact matches",
			input:    "type_name=user|type_name=admin",
			expected: "(('type_name',EQ,'user'),OR,('type_name',EQ,'admin'))",
			wantErr:  false,
		},
		{
			name:     "OR with pattern matches",
			input:    "type_name=~user%|type_name=~admin%",
			expected: "(('type_name',LK,'user%'),OR,('type_name',LK,'admin%'))",
			wantErr:  false,
		},
		{
			name:     "OR with negation",
			input:    "type_name!=test|type_name!=demo",
			expected: "(('type_name',NE,'test'),OR,('type_name',NE,'demo'))",
			wantErr:  false,
		},

		// Grouping
		{
			name:     "simple grouping with OR inside AND",
			input:    "(type_name=user|type_name=admin)&organization_id=456",
			expected: "((('type_name',EQ,'user'),OR,('type_name',EQ,'admin')),AND,('organization_id',EQ,'456'))",
			wantErr:  false,
		},
		{
			name:     "complex nested grouping",
			input:    "((type_name=user|type_name=admin)&organization_id=456)|created=2024",
			expected: "(((('type_name',EQ,'user'),OR,('type_name',EQ,'admin')),AND,('organization_id',EQ,'456')),OR,('created',EQ,'2024'))",
			wantErr:  false,
		},
		{
			name:     "multiple grouped OR conditions",
			input:    "(type_name=user|type_name=admin)&(id=123|id=456)",
			expected: "((('type_name',EQ,'user'),OR,('type_name',EQ,'admin')),AND,(('id',EQ,'123'),OR,('id',EQ,'456')))",
			wantErr:  false,
		},

		// Whitespace handling
		{
			name:     "whitespace around operators",
			input:    "type_name = user",
			expected: "('type_name',EQ,'user')",
			wantErr:  false,
		},
		{
			name:     "whitespace in complex expression",
			input:    "type_name = user & id = 123",
			expected: "(('type_name',EQ,'user'),AND,('id',EQ,'123'))",
			wantErr:  false,
		},
		{
			name:     "whitespace in grouped expression",
			input:    "(type_name=user|type_name=admin)&organization_id=456",
			expected: "((('type_name',EQ,'user'),OR,('type_name',EQ,'admin')),AND,('organization_id',EQ,'456'))",
			wantErr:  false,
		},

		// Edge cases
		{
			name:     "empty string",
			input:    "",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "value with special characters",
			input:    "type_name=user-admin_test",
			expected: "('type_name',EQ,'user-admin_test')",
			wantErr:  false,
		},
		{
			name:     "pattern with multiple wildcards",
			input:    "type_name=~%user%admin%",
			expected: "('type_name',LK,'%user%admin%')",
			wantErr:  false,
		},

		// Error cases
		{
			name:     "invalid field name",
			input:    "invalid_field=value",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "missing value",
			input:    "type_name=",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "missing key",
			input:    "=value",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "no operator",
			input:    "type_name",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "unclosed parenthesis",
			input:    "(type_name=user",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "unexpected closing parenthesis",
			input:    "type_name=user)",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "characters before opening parenthesis",
			input:    "foo(type_name=user)",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatFilterString(tt.input, validKeys)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseFieldToken(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedOp  string
		expectedKey string
		expectedVal string
		wantErr     bool
	}{
		{
			name:        "exact match operator",
			input:       "type_name=user",
			expectedOp:  "EQ",
			expectedKey: "type_name",
			expectedVal: "user",
			wantErr:     false,
		},
		{
			name:        "pattern match operator",
			input:       "type_name=~user%",
			expectedOp:  "LK",
			expectedKey: "type_name",
			expectedVal: "user%",
			wantErr:     false,
		},
		{
			name:        "not equal operator",
			input:       "type_name!=admin",
			expectedOp:  "NE",
			expectedKey: "type_name",
			expectedVal: "admin",
			wantErr:     false,
		},
		{
			name:        "not like operator",
			input:       "type_name!~test%",
			expectedOp:  "NL",
			expectedKey: "type_name",
			expectedVal: "test%",
			wantErr:     false,
		},
		{
			name:        "whitespace trimming",
			input:       "  type_name  =  user  ",
			expectedOp:  "EQ",
			expectedKey: "type_name",
			expectedVal: "user",
			wantErr:     false,
		},
		{
			name:        "UUID value",
			input:       "id=550e8400-e29b-41d4-a716-446655440000",
			expectedOp:  "EQ",
			expectedKey: "id",
			expectedVal: "550e8400-e29b-41d4-a716-446655440000",
			wantErr:     false,
		},
		{
			name:        "value with special chars",
			input:       "name=user-admin_123",
			expectedOp:  "EQ",
			expectedKey: "name",
			expectedVal: "user-admin_123",
			wantErr:     false,
		},
		{
			name:    "empty key",
			input:   "=value",
			wantErr: true,
		},
		{
			name:    "empty value",
			input:   "key=",
			wantErr: true,
		},
		{
			name:    "no operator",
			input:   "keyvalue",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := parseFieldToken(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "field", token.typ)
				assert.Equal(t, tt.expectedOp, token.operator)
				// Verify key and value are in the token.value (normalized format)
				assert.Contains(t, token.value, tt.expectedKey)
				assert.Contains(t, token.value, tt.expectedVal)
			}
		})
	}
}

func TestTokenizeFilter(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedTypes []string
		wantErr       bool
	}{
		{
			name:          "single field",
			input:         "type_name=user",
			expectedTypes: []string{"field"},
			wantErr:       false,
		},
		{
			name:          "field with AND",
			input:         "type_name=user&id=123",
			expectedTypes: []string{"field", "and", "field"},
			wantErr:       false,
		},
		{
			name:          "field with OR",
			input:         "type_name=user|type_name=admin",
			expectedTypes: []string{"field", "or", "field"},
			wantErr:       false,
		},
		{
			name:          "grouped expression",
			input:         "(type_name=user)",
			expectedTypes: []string{"lparen", "field", "rparen"},
			wantErr:       false,
		},
		{
			name:          "complex grouped expression",
			input:         "(type_name=user|type_name=admin)&id=123",
			expectedTypes: []string{"lparen", "field", "or", "field", "rparen", "and", "field"},
			wantErr:       false,
		},
		{
			name:          "pattern match operator",
			input:         "type_name=~user%",
			expectedTypes: []string{"field"},
			wantErr:       false,
		},
		{
			name:          "not equal operator",
			input:         "type_name!=admin",
			expectedTypes: []string{"field"},
			wantErr:       false,
		},
		{
			name:          "not like operator",
			input:         "type_name!~test%",
			expectedTypes: []string{"field"},
			wantErr:       false,
		},
		{
			name:    "characters before opening paren",
			input:   "foo(type_name=user)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizeFilter(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expectedTypes), len(tokens))

				for i, expectedType := range tt.expectedTypes {
					assert.Equal(t, expectedType, tokens[i].typ, "token %d type mismatch", i)
				}
			}
		})
	}
}

func TestFilterOperatorPrecedence(t *testing.T) {
	validKeys := []string{"a", "b", "c", "d"}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "AND before OR without parens",
			input:    "a=1|b=2&c=3",
			expected: "(('a',EQ,'1'),OR,(('b',EQ,'2'),AND,('c',EQ,'3')))",
		},
		{
			name:     "multiple ANDs",
			input:    "a=1&b=2&c=3",
			expected: "(('a',EQ,'1'),AND,(('b',EQ,'2'),AND,('c',EQ,'3')))",
		},
		{
			name:     "multiple ORs",
			input:    "a=1|b=2|c=3",
			expected: "(('a',EQ,'1'),OR,(('b',EQ,'2'),OR,('c',EQ,'3')))",
		},
		{
			name:     "parens override precedence",
			input:    "(a=1|b=2)&c=3",
			expected: "((('a',EQ,'1'),OR,('b',EQ,'2')),AND,('c',EQ,'3'))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatFilterString(tt.input, validKeys)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterRealWorldScenarios(t *testing.T) {
	validKeys := []string{"id", "alias", "organization_id", "type_id", "created", "updated"}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "find objects by type UUID",
			input:    "type_id=550e8400-e29b-41d4-a716-446655440000",
			expected: "('type_id',EQ,'550e8400-e29b-41d4-a716-446655440000')",
		},
		{
			name:     "find objects by type and org",
			input:    "type_id=550e8400-e29b-41d4-a716-446655440000&organization_id=660e8400-e29b-41d4-a716-446655440001",
			expected: "(('type_id',EQ,'550e8400-e29b-41d4-a716-446655440000'),AND,('organization_id',EQ,'660e8400-e29b-41d4-a716-446655440001'))",
		},
		{
			name:     "find objects with alias prefix",
			input:    "alias=~admin%",
			expected: "('alias',LK,'admin%')",
		},
		{
			name:     "exclude test objects",
			input:    "alias!~test%",
			expected: "('alias',NL,'test%')",
		},
		{
			name:     "find multiple specific aliases",
			input:    "alias=admin|alias=superuser|alias=root",
			expected: "(('alias',EQ,'admin'),OR,(('alias',EQ,'superuser'),OR,('alias',EQ,'root')))",
		},
		{
			name:     "complex filter: type and (admin or superuser) aliases",
			input:    "type_id=550e8400-e29b-41d4-a716-446655440000&(alias=admin|alias=superuser)",
			expected: "(('type_id',EQ,'550e8400-e29b-41d4-a716-446655440000'),AND,(('alias',EQ,'admin'),OR,('alias',EQ,'superuser')))",
		},
		{
			name:     "exclude system and test objects",
			input:    "alias!=system&alias!~test%",
			expected: "(('alias',NE,'system'),AND,('alias',NL,'test%'))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatFilterString(tt.input, validKeys)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
