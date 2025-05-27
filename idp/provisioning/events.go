package provisioning

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/events"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/logserver/client"
)

// GetAllCustomEvents returns all custom events for the idp items (accessors, mutators, access policies, access policy templates, transformers)
func GetAllCustomEvents(ctx context.Context, tenantDB *ucdb.DB, tenantID uuid.UUID, cacheCfg *cache.Config) ([]client.MetricMetadata, error) {
	s := storage.New(ctx, tenantDB, tenantID, cacheCfg)
	allEvents := make([]client.MetricMetadata, 0)

	accessors, err := s.ListAccessorsNonPaginated(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	for _, a := range accessors {
		allEvents = append(allEvents, events.GetEventsForAccessor(a.ID, a.Version)...)
	}
	mutators, err := s.ListMutatorsNonPaginated(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	for _, m := range mutators {
		allEvents = append(allEvents, events.GetEventsForMutator(m.ID, m.Version)...)
	}
	accessPolicies, err := s.ListAccessPoliciesNonPaginated(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	for _, ap := range accessPolicies {
		allEvents = append(allEvents, events.GetEventsForAccessPolicy(ap.ID, ap.Version)...)
	}

	accessPolicyTemplates, err := s.ListAccessPolicyTemplatesNonPaginated(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	for _, apt := range accessPolicyTemplates {
		allEvents = append(allEvents, events.GetEventsForAccessPolicyTemplate(apt.ID, apt.Version)...)
	}

	transformers, err := s.ListTransformersNonPaginated(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	for _, t := range transformers {
		allEvents = append(allEvents, events.GetEventsForTransformer(t.ID, t.Version)...)
	}

	return allEvents, nil
}
