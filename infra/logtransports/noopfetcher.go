package logtransports

import (
	"github.com/gofrs/uuid"

	"userclouds.com/infra/uclog"
)

type noOpFetcher struct{}

func newNoOpFetcher() *noOpFetcher {
	return &noOpFetcher{}
}

func (f *noOpFetcher) Init(updateHandler func(updatedMap *uclog.EventMetadataMap, tenantID uuid.UUID) error) error {
	return nil
}
func (f *noOpFetcher) Close()                                         {}
func (f *noOpFetcher) FetchEventMetadataForTenant(tenantID uuid.UUID) {}
