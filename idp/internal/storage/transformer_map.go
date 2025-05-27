package storage

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucerr"
)

// TransformerMap is a map of transformers by ID and name
type TransformerMap struct {
	idMap   map[uuid.UUID]Transformer
	nameMap map[string]Transformer
}

func newTransformerMap() TransformerMap {
	transformerMap := TransformerMap{}
	transformerMap.idMap = map[uuid.UUID]Transformer{}
	transformerMap.nameMap = map[string]Transformer{}
	return transformerMap
}

func (tm TransformerMap) addTransformers(transformers ...Transformer) {
	for _, tf := range transformers {
		tm.idMap[tf.ID] = tf
		tm.nameMap[strings.ToLower(tf.Name)] = tf
	}
}

// ForID returns the transformer for the specified id, returning an error if not found
func (tm *TransformerMap) ForID(id uuid.UUID) (*Transformer, error) {
	t, found := tm.idMap[id]
	if !found {
		return nil, ucerr.Errorf("could not find transformer for id %v", id)
	}

	return &t, nil
}

// ForName returns the transformer for the specified name, returning an error if not found
func (tm *TransformerMap) ForName(name string) (*Transformer, error) {
	t, found := tm.nameMap[strings.ToLower(name)]
	if !found {
		return nil, ucerr.Errorf("could not find transformer for name %s", name)
	}

	return &t, nil
}

// GetTransformerMapForIDs returns a TransformerMap for the given transformer IDs
func GetTransformerMapForIDs(ctx context.Context, s *Storage, errorOnMissing bool, ids ...uuid.UUID) (*TransformerMap, error) {
	var resourceIDs []userstore.ResourceID
	for _, id := range ids {
		resourceIDs = append(resourceIDs, userstore.ResourceID{ID: id})
	}

	tm, err := GetTransformerMapForResourceIDs(ctx, s, errorOnMissing, resourceIDs...)
	return tm, ucerr.Wrap(err)
}

// GetTransformerMapForResourceIDs returns a TransformerMap for the given transformer RIDs
func GetTransformerMapForResourceIDs(ctx context.Context, s *Storage, errorOnMissing bool, transformers ...userstore.ResourceID) (*TransformerMap, error) {
	transformerMap := newTransformerMap()
	var nonNativeTransformersRIDs []userstore.ResourceID
	for _, transformer := range transformers {
		nt, found := nativeTransformers[transformer.ID]
		if found {
			transformerMap.addTransformers(nt.t)
		} else {
			nonNativeTransformersRIDs = append(nonNativeTransformersRIDs, transformer)
		}
	}
	if len(nonNativeTransformersRIDs) == 0 {
		return &transformerMap, nil
	}

	nonNativeTransformers, err := s.GetTransformersForResourceIDs(ctx, errorOnMissing, nonNativeTransformersRIDs...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	transformerMap.addTransformers(nonNativeTransformers...)
	return &transformerMap, nil
}
