package provisioning

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/google/go-cmp/cmp"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/logeventmetadata"
	"userclouds.com/internal/provisioning/types"
	logServerClient "userclouds.com/logserver/client"
	"userclouds.com/logserver/internal/storage"
)

var suppressConfirmation *bool
var suppressMutex sync.Mutex

type options struct {
	cs          types.ControlSource
	parallelOps []types.ProvisionOperation
}

// Option makes EventProvisioner extensible
type Option interface {
	apply(*options)
}

type optFunc func(*options)

func (o optFunc) apply(opts *options) {
	o(opts)
}

// ControlSource returns an Option that will cause EventProvisioner to get the events metadata from the control source instead of arguments
func ControlSource(controlSource types.ControlSource) Option {
	return optFunc(func(opts *options) {
		opts.cs = controlSource
	})
}

// ParallelOperations returns an Option that will cause EventProvisioner to allow the specified operations to be run in parallel
func ParallelOperations(pos ...types.ProvisionOperation) Option {
	return optFunc(func(opts *options) {
		opts.parallelOps = pos
	})
}

// EventProvisioner is a Provisionable object used to set up metric metadata.
type EventProvisioner struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	logDB *ucdb.DB
	mT    []logServerClient.MetricMetadata
	cs    types.ControlSource
}

// NewEventProvisioner initializes and return Provisionable object for a set of event metadata entries
func NewEventProvisioner(name string, logDB *ucdb.DB, mT []logServerClient.MetricMetadata, opts ...Option) *EventProvisioner {

	var options options
	for _, opt := range opts {
		opt.apply(&options)
	}

	eventProvisioner := EventProvisioner{
		Named:          types.NewNamed(name + ":Event"),
		Parallelizable: types.NewParallelizable(options.parallelOps...),
		logDB:          logDB,
		mT:             mT,
		cs:             options.cs,
	}
	return &eventProvisioner
}
func (t *EventProvisioner) suppressConfirm() bool {
	return suppressConfirmation != nil && *suppressConfirmation && universe.Current().IsDev()
}

func (t *EventProvisioner) initSuppressConfirm() {
	if suppressConfirmation == nil && universe.Current().IsDev() {
		val := types.ConfirmOperation("Skip confirmations for other event type overrides")
		suppressConfirmation = &val
	}
}

func (t *EventProvisioner) initEventsFromControlSource(ctx context.Context) error {
	// Check if we were initialized with an control source that should provide the events array
	if t.cs != nil {
		data, err := t.cs.GetData(ctx)
		if err != nil {
			return ucerr.Wrap(err)
		}
		var ok bool

		if t.mT, ok = data.([]logServerClient.MetricMetadata); !ok {
			return ucerr.Errorf("Expected an []MetricMetadata instead gotv %v from control source")
		}
	}
	return nil
}

// Provision implements Provisionable and creates or updates metric metadata for a
// tenant in the tenant's dedicated log DB.
func (t *EventProvisioner) Provision(ctx context.Context) error {
	// Connect to Tenant's tokenizer DB
	s := logeventmetadata.NewStorage(t.logDB)

	if err := t.initEventsFromControlSource(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	var metricTypes []logeventmetadata.MetricMetadata
	codeSpecified := false
	incomingCodeMap := make(map[uclog.EventCode]logeventmetadata.MetricMetadata)
	incomingStringIDMap := make(map[string]logeventmetadata.MetricMetadata)
	for i := range t.mT {
		metricTypes = append(metricTypes, logeventmetadata.NewMetricMetadataFromLogClientMetricMetadata(t.mT[i]))

		if t.mT[i].Code > 0 {
			codeSpecified = true

			// Check for duplicate codes
			ee, ok := incomingCodeMap[t.mT[i].Code]
			if ok {
				return ucerr.Errorf("Two events %v and %v have same code", ee, t.mT[i])
			}
			incomingCodeMap[t.mT[i].Code] = metricTypes[i]
		}
		// Check for duplicate string_ids
		ee, ok := incomingStringIDMap[t.mT[i].StringID]
		if ok {
			return ucerr.Errorf("Two events %v and %v have same string_id", ee, t.mT[i])
		}
		incomingStringIDMap[t.mT[i].StringID] = metricTypes[i]
	}

	// Filter out already provisioned events
	cpMetricTypes := []logeventmetadata.MetricMetadata{}
	upMetricTypes := []logeventmetadata.MetricMetadata{}
	delMetricTypes := map[uclog.EventCode]logeventmetadata.MetricMetadata{}
	if types.ProvisionMode == types.OfflineProvisionMode || codeSpecified {
		allMetrics, err := storage.GetMetricsMetaDataArray(ctx, s)
		if err != nil {
			return ucerr.Wrap(err)
		}

		stringIDMap := make(map[string]logeventmetadata.MetricMetadata)
		for _, m := range *allMetrics {
			stringIDMap[m.StringID] = m
		}

		// See if the metric metadata is already exists for metric being provisioned
		for _, m := range metricTypes {
			if em, ok := stringIDMap[m.StringID]; ok {
				if m.Code == 0 {
					m.Code = em.Code
				}

				// Check if the stored data is same as data being provisioned and skip
				if em.Equal(m, true) {
					continue
				}
				// Check if the stored data is same as data being provisioned and remove the entry in that case
				if em.Code == m.Code {
					suppressMutex.Lock()
					if !t.suppressConfirm() && !types.ConfirmOperation(fmt.Sprintf("Are you sure want to update existing metric  %s", cmp.Diff(em, m))) {
						suppressMutex.Unlock()
						return ucerr.Errorf("Attempted to updated existing metric in provisioning %s", em.StringID)
					}
					t.initSuppressConfirm()
					suppressMutex.Unlock()
					m.ID = em.ID
					upMetricTypes = append(upMetricTypes, m)
					continue
				}
				// Save list of metrics we will soft deleted
				delMetricTypes[em.Code] = em
				// After delete this will be a create
				cpMetricTypes = append(cpMetricTypes, m)
			} else {
				// Check if the code is already used
				if em, err := s.GetMetricMetadataByCode(ctx, m.Code); err == nil {
					// Make sure the code is valid
					if m.Code == 0 {
						return ucerr.Errorf("Unexpected event with code 0 - %s", em.StringID)
					}

					suppressMutex.Lock()
					if !t.suppressConfirm() && !types.ConfirmOperation(fmt.Sprintf("Are you sure want to change the string_id on existing metric  %s", cmp.Diff(*em, m))) {
						suppressMutex.Unlock()
						return ucerr.Errorf("Attempted to change string_id from %s to %s on code %d", em.StringID, m.StringID, m.Code)
					}
					t.initSuppressConfirm()
					suppressMutex.Unlock()
					m.ID = em.ID
					upMetricTypes = append(upMetricTypes, m)
				} else { // Brand new metric
					cpMetricTypes = append(cpMetricTypes, m)
				}
			}
		}
	} else {
		cpMetricTypes = metricTypes
	}

	// Check for three way conflict: type A (string X, code 1) in DB and incoming type B (stringID X, code 2) and incoming type C (stringID Y, code 1)
	abortOnError := universe.Current().IsProdOrStaging()
	for {
		conflict := false
		for i, um := range upMetricTypes {
			if dm, ok := delMetricTypes[um.Code]; ok {
				if abortOnError {
					return ucerr.Errorf("Three way conflict unexpected on staging or prod, leads to data loss incoming %v and existing %v", um, dm)
				}
				cpMetricTypes = append(cpMetricTypes, um)
				upMetricTypes = slices.Delete(upMetricTypes, i, i+1)
				conflict = true
				break
			}
		}
		if !conflict {
			break
		}
	}

	uclog.Infof(ctx, "Found %d existing event types that are changing code from the DB and will be soft deleted", len(delMetricTypes))
	for _, dm := range delMetricTypes {
		// Soft delete the existing entry to preserve the old code
		if err := s.DeleteMetricMetadata(ctx, dm.ID); err != nil {
			return ucerr.Wrap(err)
		}
	}
	uclog.Infof(ctx, "Found %d new event types that are missing in the DB and will be created", len(cpMetricTypes))
	if err := storage.SaveMetricsMetaDataArray(ctx, cpMetricTypes, s); err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "Found %d existing event types that will be updated in the DB", len(upMetricTypes))
	if err := storage.UpdateMetricsMetaDataArray(ctx, upMetricTypes, s); err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "Provisioned %d event types", len(cpMetricTypes)+len(upMetricTypes))
	return nil
}

// Validate validates that a tenant has the appropriate events provisioned.
func (t *EventProvisioner) Validate(ctx context.Context) error {
	// Connect to Tenant's log DB
	s := logeventmetadata.NewStorage(t.logDB)

	if err := t.initEventsFromControlSource(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	stringIDMap := make(map[string]logServerClient.MetricMetadata, len(t.mT))
	countFound := 0

	for _, m := range t.mT {
		stringIDMap[m.StringID] = m
	}

	pager, err := logeventmetadata.NewMetricMetadataPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit),
	)
	if err != nil {
		return ucerr.Wrap(err)
	}

	for {
		pageMetrics, respFields, err := s.ListMetricMetadatasPaginated(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}

		for _, m := range pageMetrics {
			v, ok := stringIDMap[m.StringID]

			if ok {
				if m.EqualLogClientMetricMetadata(v, true, true) {
					countFound++

					delete(stringIDMap, m.StringID)
				}
			}
		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	if countFound != len(t.mT) {
		return ucerr.Errorf("Didn't find all the custom events expected during provisioning %v", stringIDMap)
	}

	return nil
}

// Cleanup removes event types from tenant's log DB
func (t *EventProvisioner) Cleanup(ctx context.Context) error {
	// Connect to Tenant's log DB
	s := logeventmetadata.NewStorage(t.logDB)

	stringIDMap := make(map[string]logServerClient.MetricMetadata, len(t.mT))
	mD := []logeventmetadata.MetricMetadata{}

	for _, m := range t.mT {
		stringIDMap[m.StringID] = m
	}

	pager, err := logeventmetadata.NewMetricMetadataPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit),
	)
	if err != nil {
		return ucerr.Wrap(err)
	}

	for {
		pageMetrics, respFields, err := s.ListMetricMetadatasPaginated(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}

		for _, m := range pageMetrics {
			v, ok := stringIDMap[m.StringID]

			if ok {
				if m.EqualLogClientMetricMetadata(v, true, true) {
					mD = append(mD, m)
				} else {
					return ucerr.Errorf("The event being deprovisioned has different data in DB (db) %v (new) %v", m, v)
				}

			}
		}

		if len(pageMetrics) < pagination.MaxLimit {
			break
		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	if err := storage.DeleteMetricsMetaDataArray(ctx, mD, s); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// Nuke removes event types from tenant's log DB
func (t *EventProvisioner) Nuke(ctx context.Context) error {
	// Connect to Tenant's log DB
	s := logeventmetadata.NewStorage(t.logDB)

	if err := s.NukeNonCustomEvent(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// ExecuteOperations executes the specified operations on the EventProvisioner
func (t *EventProvisioner) ExecuteOperations(ctx context.Context, ops []types.ProvisionOperation, eventType string) error {
	for _, op := range ops {
		if op == types.Provision {
			if err := t.Provision(ctx); err != nil {
				return ucerr.Wrap(err)
			}
		}
		if op == types.Validate {
			if err := t.Validate(ctx); err != nil {
				if len(ops) != 1 {
					return ucerr.Errorf("Failed to validate provisioned %v events: %v", eventType, err)
				}
				uclog.Errorf(ctx, "Failed to validate %v events: %v", eventType, err)
			} else {
				uclog.Infof(ctx, "Validated that DB state matches the expected %v event types", eventType)
			}
		}
		if op == types.Cleanup {
			return ucerr.Wrap(t.Cleanup(ctx))
		}
	}
	return nil
}
