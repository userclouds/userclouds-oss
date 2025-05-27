package defaults

import (
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
)

var defaultTransformersByID = map[uuid.UUID]storage.Transformer{}
var defaultTransformers = []storage.Transformer{
	TransformerCreditCard,
	TransformerEmail,
	TransformerFullName,
	TransformerPassthrough,
	TransformerSSN,
	TransformerUUID,
}

// GetDefaultTransformers returns the default transformers
func GetDefaultTransformers() []storage.Transformer {
	var transformers []storage.Transformer
	transformers = append(transformers, defaultTransformers...)
	return transformers
}

// IsDefaultTransformer returns true if id refers to a default transformer
func IsDefaultTransformer(id uuid.UUID) bool {
	if _, found := defaultTransformersByID[id]; found {
		return true
	}

	return false
}

func init() {
	for _, dt := range defaultTransformers {
		if _, found := defaultTransformersByID[dt.ID]; found {
			panic(fmt.Sprintf("transformer %s has conflicting id %v", dt.Name, dt.ID))
		}
		defaultTransformersByID[dt.ID] = dt
	}
}
