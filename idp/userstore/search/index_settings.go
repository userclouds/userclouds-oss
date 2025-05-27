package search

import "userclouds.com/infra/ucerr"

// IndexSettings represents settings for configuring an index
type IndexSettings struct {
	Ngram *NgramIndexSettings `json:"ngram,omitempty" validate:"allownil"`
}

// NewNgramIndexSettings creates an IndexSettings appropriate for the Ngram index type
func NewNgramIndexSettings(minNgram int, maxNgram int) IndexSettings {
	return IndexSettings{
		Ngram: &NgramIndexSettings{MinNgram: minNgram, MaxNgram: maxNgram},
	}
}

// Equals returns true if the two settings are identical
func (is IndexSettings) Equals(o IndexSettings) bool {
	if is.Ngram != nil {
		if o.Ngram != nil {
			return *is.Ngram == *o.Ngram
		}
		return false
	}

	return o.Ngram == nil
}

//go:generate gendbjson IndexSettings

//go:generate genvalidate IndexSettings

// NgramIndexSettings represents settings for configuring an ngram index
type NgramIndexSettings struct {
	MinNgram int `json:"min_ngram"`
	MaxNgram int `json:"max_ngram"`
}

func (nis NgramIndexSettings) extraValidate() error {
	if nis.MinNgram <= 0 || nis.MinNgram > nis.MaxNgram {
		return ucerr.Friendlyf(nil, "MinNgram must be greater than zero and less than or equal to MaxNgram")
	}
	return nil
}

//go:generate genvalidate NgramIndexSettings
