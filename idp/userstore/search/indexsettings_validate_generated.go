// NOTE: automatically generated file -- DO NOT EDIT

package search

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o IndexSettings) Validate() error {
	if o.Ngram != nil {
		if err := o.Ngram.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
