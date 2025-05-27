package userstore

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

// purposeCache is a convenience class that loads all purposes and can
// convert a ResourceID, which may be specified by ID or by Name, into
// a valid purpose ID

type purposeCache struct {
	purposeIDs       set.Set[uuid.UUID]
	purposeIDsByName map[string]uuid.UUID
}

func newPurposeCache(ctx context.Context, s *storage.Storage) (*purposeCache, error) {
	allPurposes, err := s.ListPurposesNonPaginated(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	purposeIDs := set.NewUUIDSet()
	purposeIDsByName := map[string]uuid.UUID{}
	for _, purpose := range allPurposes {
		purposeIDs.Insert(purpose.ID)
		purposeIDsByName[strings.ToLower(purpose.Name)] = purpose.ID
	}
	return &purposeCache{purposeIDs, purposeIDsByName}, nil
}

func (pc *purposeCache) getPurposeID(resource userstore.ResourceID) (uuid.UUID, error) {
	if resource.ID != uuid.Nil {
		if !pc.purposeIDs.Contains(resource.ID) {
			return uuid.Nil, ucerr.Friendlyf(nil, "ID %v does not match a valid purpose", resource.ID)
		}

		return resource.ID, nil
	}

	if resource.Name != "" {
		purposeID, found := pc.purposeIDsByName[strings.ToLower(resource.Name)]
		if !found {
			return uuid.Nil, ucerr.Friendlyf(nil, "Name %s does not match a valid purpose", resource.Name)
		}

		return purposeID, nil
	}

	return uuid.Nil, ucerr.Friendlyf(nil, "ID or Name must refer to a valid purpose")
}
