package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchParametersBuilder_ConvertTerm(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		term         string
		wantTerm     string
		wantFullTerm string
		wantTrunc    bool
	}{
		{
			name:         "basic term",
			term:         "%test%",
			wantTerm:     "test",
			wantFullTerm: "test",
			wantTrunc:    false,
		},
		{
			name:         "long term gets truncated",
			term:         "%verylongstringhere%",
			wantTerm:     "verylongst",
			wantFullTerm: "verylongstringhere",
			wantTrunc:    true,
		},
		{
			name:         "placeholder term",
			term:         "?",
			wantTerm:     "?",
			wantFullTerm: "?",
			wantTrunc:    false,
		},
		{
			name:         "invalid wildcard pattern",
			term:         "test%",
			wantTerm:     "",
			wantFullTerm: "",
			wantTrunc:    false,
		},
		{
			name:         "Don't allow *",
			term:         "%jerr*y%",
			wantTerm:     "",
			wantFullTerm: "",
			wantTrunc:    false,
		},
		{
			name:         "Don't allow %",
			term:         "%jerr%y%",
			wantTerm:     "",
			wantFullTerm: "",
			wantTrunc:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spb := searchParametersBuilder{}
			term, fullTerm, truncated := spb.convertTerm(ctx, tt.term)
			assert.Equal(t, tt.wantTerm, term)
			assert.Equal(t, tt.wantFullTerm, fullTerm)
			assert.Equal(t, tt.wantTrunc, truncated)
		})
	}
}
