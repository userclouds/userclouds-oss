package provisioning_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/gofrs/uuid"

	idpEvents "userclouds.com/idp/events"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/logdb"
	"userclouds.com/internal/logeventmetadata"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/logserver/client"
	"userclouds.com/logserver/events"
	"userclouds.com/logserver/provisioning"
)

type EventControlSource struct {
	e []client.MetricMetadata
}

func (cs *EventControlSource) GetData(context.Context) (any, error) {
	return cs.e, nil
}

func TestEventProvisioning(t *testing.T) {
	ctx := context.Background()
	_, logDB, _ := testhelpers.NewTestStorage(t)
	testhelpers.CreateTestServer(ctx, t)
	db, err := ucdb.NewWithLimits(ctx, logDB, migrate.SchemaValidator(logdb.Schema), 10, 10)
	assert.NoErr(t, err)

	t.Run("ProvisionEmpty", func(t *testing.T) {
		eP := provisioning.NewEventProvisioner("", db, []client.MetricMetadata{})
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))
	})

	t.Run("ProvisionSingle", func(t *testing.T) {
		e := idpEvents.GetEventsForTransformer(uuid.Must(uuid.NewV4()), 0)
		eP := provisioning.NewEventProvisioner("ProvisionSingle", db, []client.MetricMetadata{e[0]})

		err := eP.Provision(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, eP.Name(), "ProvisionSingle:Event")

		err = eP.Validate(ctx)
		assert.NoErr(t, err)

		// Should fail
		eP = provisioning.NewEventProvisioner("ProvisionSingle", db, []client.MetricMetadata{e[1]})
		err = eP.Validate(ctx)
		assert.NotNil(t, err)
	})

	t.Run("ProvisionArrayTokenizer", func(t *testing.T) {
		e := idpEvents.GetEventsForTransformer(uuid.Must(uuid.NewV4()), 0)
		eP := provisioning.NewEventProvisioner("ProvisionArray", db, e)
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))
	})

	t.Run("ProvisionArrayIDP", func(t *testing.T) {
		e := idpEvents.GetEventsForAccessor(uuid.Must(uuid.NewV4()), 0)

		eP := provisioning.NewEventProvisioner("ProvisionArray", db, e)
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))

		e = idpEvents.GetEventsForMutator(uuid.Must(uuid.NewV4()), 0)

		eP = provisioning.NewEventProvisioner("ProvisionArray", db, e)
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))
	})

	t.Run("ProvisionViaControlSource", func(t *testing.T) {
		aID := uuid.Must(uuid.NewV4())
		cs := EventControlSource{e: idpEvents.GetEventsForAccessor(aID, 0)}
		eP := provisioning.NewEventProvisioner("ProvisionArray", db, nil, provisioning.ControlSource(&cs))
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))

		cs = EventControlSource{e: idpEvents.GetEventsForAccessor(aID, 1)}
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))

		// Double check that the validate method ran against the correct control source
		checkForEventMetadata(ctx, t, db, idpEvents.GetEventsForAccessor(aID, 1), nil)
	})

	t.Run("ProvisionDuplicate", func(t *testing.T) {
		gpID := uuid.Must(uuid.NewV4())
		e := idpEvents.GetEventsForTransformer(gpID, 0)

		eP := provisioning.NewEventProvisioner("ProvisionDuplicate", db, e)
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))

		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))
	})

	t.Run("ProvisionMultiThreaded", func(t *testing.T) {
		wg := sync.WaitGroup{}
		gpIDs := []uuid.UUID{}
		apIDs := []uuid.UUID{}

		for i := range 20 {
			wg.Add(1)
			gpID := uuid.Must(uuid.NewV4())
			apID := uuid.Must(uuid.NewV4())

			gpIDs = append(gpIDs, gpID)
			apIDs = append(apIDs, apID)

			go func(threadID int, gpID uuid.UUID, apID uuid.UUID) {
				eGP := idpEvents.GetEventsForTransformer(gpID, 0)
				eP := provisioning.NewEventProvisioner(fmt.Sprintf("Thread(%d)", threadID), db, eGP)
				assert.NoErr(t, eP.Provision(ctx))
				assert.NoErr(t, eP.Validate(ctx))

				eAP := idpEvents.GetEventsForAccessPolicy(apID, 0)
				eP = provisioning.NewEventProvisioner(fmt.Sprintf("Thread(%d)", threadID), db, eAP)
				assert.NoErr(t, eP.Provision(ctx))
				assert.NoErr(t, eP.Validate(ctx))

				wg.Done()
			}(i, gpID, apID)
		}

		wg.Wait()

		// Now check that all expected events were provisioned by reading them from DB. Note that Validate() has already done that on
		// individual basis so we are checking for some multi-threading issues
		evs := make([]client.MetricMetadata, 0, len(gpIDs)+len(apIDs))
		for _, gpID := range gpIDs {
			es := idpEvents.GetEventsForTransformer(gpID, 0)
			evs = append(evs, es...)
		}

		for _, apID := range apIDs {
			es := idpEvents.GetEventsForAccessPolicy(apID, 0)
			evs = append(evs, es...)
		}
		checkForEventMetadata(ctx, t, db, evs, nil)

		// Now clean up the events for all the policies
		for i := range 20 {
			wg.Add(1)
			gpID := uuid.Must(uuid.NewV4())
			apID := uuid.Must(uuid.NewV4())

			gpIDs = append(gpIDs, gpID)
			apIDs = append(apIDs, apID)

			go func(threadID int, gpID uuid.UUID, apID uuid.UUID) {
				eGP := idpEvents.GetEventsForTransformer(gpID, 0)
				eP := provisioning.NewEventProvisioner(fmt.Sprintf("Thread(%d)", threadID), db, eGP)
				assert.NoErr(t, eP.Cleanup(ctx))
				assert.NotNil(t, eP.Validate(ctx))

				eAP := idpEvents.GetEventsForAccessPolicy(apID, 0)
				eP = provisioning.NewEventProvisioner(fmt.Sprintf("Thread(%d)", threadID), db, eAP)
				assert.NoErr(t, eP.Cleanup(ctx))
				assert.NotNil(t, eP.Validate(ctx))

				wg.Done()
			}(i, gpIDs[i], apIDs[i])
		}
		wg.Wait()

		// Recheck that events are gone
		checkForEventMetadata(ctx, t, db, nil, evs)
	})

	t.Run("ProvisionStaticEvents", func(t *testing.T) {
		allEventTypes := events.GetLogEventTypes()
		allMetrics := make([]client.MetricMetadata, 0, len(allEventTypes))

		var maxCode uclog.EventCode
		for s, e := range allEventTypes {
			le := client.MetricMetadata{
				BaseModel:    ucdb.NewBase(),
				Name:         e.Name,
				StringID:     s,
				Code:         e.Code,
				Service:      e.Service,
				ReferenceURL: e.URL,
				Category:     e.Category,
				Attributes:   client.MetricAttributes{Ignore: e.Ignore},
			}
			allMetrics = append(allMetrics, le)
			if maxCode < e.Code {
				maxCode = e.Code
			}
		}
		eP := provisioning.NewEventProvisioner("ProvisionStaticEvents", db, allMetrics)
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))

		// Duplicate check
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))

		assert.NoErr(t, eP.Cleanup(ctx))
		assert.NotNil(t, eP.Validate(ctx))

		// Reprovision and validate post cleanup
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))
	})

	t.Run("ProvisionStaticEventsBoundary", func(t *testing.T) {
		metrics := []client.MetricMetadata{{
			BaseModel:    ucdb.NewBase(),
			Name:         "Call API1",
			StringID:     "API1",
			Code:         30000,
			Service:      service.Console,
			ReferenceURL: "",
			Category:     uclog.EventCategoryCall,
			Attributes:   client.MetricAttributes{Ignore: true},
		}}

		eP := provisioning.NewEventProvisioner("ProvisionStaticEventsBoundary", db, metrics)
		// Provision the event the first time
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))

		// Provision the event the second time (should be a nop)
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))

		// Change the name and try to re-provision (should fail for now)
		metrics[0].Name = "Call Change AP1"
		assert.NotNil(t, eP.Provision(ctx))
		assert.NotNil(t, eP.Validate(ctx))

		// Trigger the soft delete and update
		metrics[0].Name = "Call API1"
		metrics[0].Code = 30001

		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))

	})

	t.Run("ProvisionTestValidate", func(t *testing.T) {
		metrics := []client.MetricMetadata{{
			BaseModel:    ucdb.NewBase(),
			Name:         "Call API2",
			StringID:     "API2",
			Code:         20000,
			Service:      service.Console,
			ReferenceURL: "",
			Category:     uclog.EventCategoryCall,
			Attributes:   client.MetricAttributes{Ignore: true},
		}}
		eP := provisioning.NewEventProvisioner("ProvisionTestValidate", db, metrics)
		// Provision the event the first time
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))

		// Provision the event the second time (should be a nop)
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))

		// Change the name and try to validate
		metrics[0].Name = "Call Change API2"
		assert.NotNil(t, eP.Validate(ctx))
		metrics[0].Name = "Call API2"

		// Change the code and try to validate
		metrics[0].Code = 20001
		assert.NotNil(t, eP.Validate(ctx))
		metrics[0].Code = 20000

		// Change the service and try to validate
		metrics[0].Service = service.AuthZ
		assert.NotNil(t, eP.Validate(ctx))
		metrics[0].Service = service.Console

		// Change the reference url and try to validate
		metrics[0].ReferenceURL = "!"
		assert.NotNil(t, eP.Validate(ctx))
		metrics[0].ReferenceURL = ""

		// Change the category and try to validate
		metrics[0].Category = uclog.EventCategoryDuration
		assert.NotNil(t, eP.Validate(ctx))
		metrics[0].Category = uclog.EventCategoryCall

		// Change the attributes and try to validate
		metrics[0].Attributes = client.MetricAttributes{Ignore: false}
		assert.NotNil(t, eP.Validate(ctx))
		metrics[0].Attributes = client.MetricAttributes{Ignore: true}

		// Unmodified event should still be there
		assert.NoErr(t, eP.Validate(ctx))
	})

	t.Run("Deprovision", func(t *testing.T) {
		metrics := []client.MetricMetadata{{
			BaseModel:    ucdb.NewBase(),
			Name:         "Call API3",
			StringID:     "API3",
			Code:         10000,
			Service:      service.Console,
			ReferenceURL: "",
			Category:     uclog.EventCategoryCall,
			Attributes:   client.MetricAttributes{Ignore: true},
		}}

		eP := provisioning.NewEventProvisioner("Deprovision", db, metrics)
		// Provision the event the first time
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))

		// Deprovision the event the second time (validate should fail)
		assert.NoErr(t, eP.Cleanup(ctx))
		assert.NotNil(t, eP.Validate(ctx))

		// Reprovision
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))
	})

	t.Run("ProvisionTestDuplicates", func(t *testing.T) {
		metrics := []client.MetricMetadata{{
			BaseModel:    ucdb.NewBase(),
			Name:         "Call API2",
			StringID:     "DupName",
			Code:         30000,
			Service:      service.Console,
			ReferenceURL: "",
			Category:     uclog.EventCategoryCall,
			Attributes:   client.MetricAttributes{Ignore: true},
		}, {
			BaseModel:    ucdb.NewBase(),
			Name:         "Call API2",
			StringID:     "DupName",
			Code:         30001,
			Service:      service.Console,
			ReferenceURL: "",
			Category:     uclog.EventCategoryCall,
			Attributes:   client.MetricAttributes{Ignore: true},
		}}

		eP := provisioning.NewEventProvisioner("ProvisionTestDuplicates", db, metrics)
		// Try provisioning with duplicate names
		assert.NotNil(t, eP.Provision(ctx))
		assert.NotNil(t, eP.Validate(ctx))

		// Try provisioning with duplicate codes
		metrics[0].StringID = "NonDupeName"
		metrics[0].Code = 30001
		assert.NotNil(t, eP.Provision(ctx))
		assert.NotNil(t, eP.Validate(ctx))
	})

	t.Run("ProvisionTestThreeWayConflict", func(t *testing.T) {
		metrics := []client.MetricMetadata{{
			BaseModel:    ucdb.NewBase(),
			Name:         "Call API2",
			StringID:     "ThreeWayConflictA",
			Code:         30002,
			Service:      service.Console,
			ReferenceURL: "",
			Category:     uclog.EventCategoryCall,
			Attributes:   client.MetricAttributes{Ignore: true},
		}}
		eP := provisioning.NewEventProvisioner("ProvisionTestThreeWayConflict", db, metrics)
		// Provision first event
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))

		metrics[0].StringID = "ThreeWayConflictB"

		metrics = append(metrics, client.MetricMetadata{
			BaseModel:    ucdb.NewBase(),
			Name:         "Call API2",
			StringID:     "ThreeWayConflictA",
			Code:         30003,
			Service:      service.Console,
			ReferenceURL: "",
			Category:     uclog.EventCategoryCall,
			Attributes:   client.MetricAttributes{Ignore: true},
		})

		eP = provisioning.NewEventProvisioner("ProvisionTestThreeWayConflict", db, metrics)
		// Try provisioning with three way conflict
		types.ConfirmOperation = func(p string) bool { return true }
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))
		types.ConfirmOperation = func(p string) bool { return false }
	})
}

func checkForEventMetadata(ctx context.Context, t *testing.T, db *ucdb.DB, eventsExpected []client.MetricMetadata, eventsNotExpected []client.MetricMetadata) {
	_, stringIDMap, _ := getAllEventMetadata(ctx, t, db)

	for _, e := range eventsExpected {
		em, ok := stringIDMap[e.StringID]
		assert.True(t, ok)
		assert.True(t, em.EqualLogClientMetricMetadata(e, true, true))
	}

	for _, e := range eventsNotExpected {
		_, ok := stringIDMap[e.StringID]
		assert.False(t, ok)
	}
}

func getAllEventMetadata(ctx context.Context, t *testing.T, db *ucdb.DB) ([]logeventmetadata.MetricMetadata, map[string]logeventmetadata.MetricMetadata,
	map[uclog.EventCode]logeventmetadata.MetricMetadata) {
	s := logeventmetadata.NewStorage(db)

	var allMetrics []logeventmetadata.MetricMetadata

	pager, err := logeventmetadata.NewMetricMetadataPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	assert.NoErr(t, err)

	for {
		pageMetrics, respFields, err := s.ListMetricMetadatasPaginated(ctx, *pager)
		assert.NoErr(t, err)

		allMetrics = append(allMetrics, pageMetrics...)

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	stringIDMap := make(map[string]logeventmetadata.MetricMetadata)
	codeMap := make(map[uclog.EventCode]logeventmetadata.MetricMetadata)
	for _, m := range allMetrics {
		stringIDMap[m.StringID] = m
		// Check that all codes are unique - DB enforces it but just in case
		_, ok := codeMap[m.Code]
		assert.False(t, ok)
		codeMap[m.Code] = m
	}
	return allMetrics, stringIDMap, codeMap
}
