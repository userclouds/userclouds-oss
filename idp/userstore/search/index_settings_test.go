package search_test

import (
	"testing"

	"userclouds.com/idp/userstore/search"
	"userclouds.com/infra/assert"
)

func TestIndexSettingsEquals(t *testing.T) {
	var settings1 search.IndexSettings
	var settings2 search.IndexSettings
	assert.True(t, settings1.Equals(settings2))

	settings1.Ngram = &search.NgramIndexSettings{MinNgram: 3, MaxNgram: 10}
	settings2.Ngram = &search.NgramIndexSettings{MinNgram: 3, MaxNgram: 10}
	assert.True(t, settings1.Equals(settings2))
}
