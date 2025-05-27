package instancebackend

import (
	"context"
	"encoding/json"
	"net/url"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// ActivityRecord contains information about a "running" instance of service that has
// registered with the server
type ActivityRecord struct {
	InstanceID           uuid.UUID            `json:"instance_id" yaml:"instance_id"`
	LastTenantID         uuid.UUID            `json:"last_tenant_id" yaml:"last_tenant_id"`
	Multitenant          bool                 `json:"multitenant" yaml:"multitenant"`
	Service              service.Service      `json:"service" yaml:"service"`
	Region               region.MachineRegion `json:"region" yaml:"region"`
	Hostname             string               `json:"hostname" yaml:"hostname"`
	CodeVersion          string               `json:"code_version" yaml:"code_version"`
	StartupTime          time.Time            `json:"startup_time" yaml:"startup_time"`
	EventMetadataVersion int                  `json:"event_metadata_version" yaml:"event_metadata_version"`
	LastActivity         time.Time            `json:"last_activity" yaml:"last_activity"`
	CallCount            int                  `json:"call_count" yaml:"call_count"`
	EventCount           int                  `json:"event_count" yaml:"event_count"`
	ErrorInternalCount   int                  `json:"error_internal_count" yaml:"error_internal_count"`
	ErrorInputCount      int                  `json:"error_input_count" yaml:"error_input_count"`
	SendRawData          bool                 `json:"send_raw_data" yaml:"send_raw_data"`
	LogLevel             uclog.LogLevel       `json:"log_level" yaml:"log_level"`
	MessageInterval      int                  `json:"message_interval" yaml:"message_interval"`
	CountersInterval     int                  `json:"counters_interval" yaml:"counters_interval"`
	ProcessedStartup     bool                 `json:"processed_startup" yaml:"processed_startup"`
	mutex                sync.Mutex
}

// InstanceActivityStore exposed the backend for keeping track of real time activity
type InstanceActivityStore struct {
	activityData      map[uuid.UUID]*ActivityRecord
	activityDataMutex sync.Mutex
	// This is only useful while we have < 100,000 of tenants but should help with debugging for a while
	tenantInstances      map[uuid.UUID]map[uuid.UUID]bool
	tenantInstancesMutex sync.Mutex
}

// NewInstanceActivityStore gets an instance of InstanceActivityStore
func NewInstanceActivityStore(ctx context.Context) (*InstanceActivityStore, error) {
	var i InstanceActivityStore
	// Create a cache for instance data. TODO this emulates a global in memory key/value pair store
	// (like memcache) that should be same across multiple instances of LogServer
	i.activityData = make(map[uuid.UUID]*ActivityRecord)
	i.tenantInstances = make(map[uuid.UUID]map[uuid.UUID]bool)

	i.activityDataMutex = sync.Mutex{}
	i.tenantInstancesMutex = sync.Mutex{}

	return &i, nil
}

// UpdateActivityStore updates the activity store on basis of the call
func (i *InstanceActivityStore) UpdateActivityStore(url *url.URL) (*ActivityRecord, error) {
	instanceString := url.Query().Get("instance_id")

	var instanceID uuid.UUID
	if err := instanceID.UnmarshalText([]byte(instanceString)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	r := i.getOrCreateActivityRecord(instanceID)
	r.mutex.Lock()
	r.LastActivity = time.Now().UTC()
	r.CallCount = r.CallCount + 1
	// In case this node doesn't receive the startup event, update basic info
	i.updateBaseInfoCreateActivityRecord(r, url)
	r.mutex.Unlock()
	return r, nil
}

// UpdateEventCount updates the count of event for the instance in activity store
func (i *InstanceActivityStore) UpdateEventCount(r *ActivityRecord, eventCount int, errorInputCount int, errorInternalCount int) {
	r.mutex.Lock()
	r.EventCount = r.EventCount + eventCount
	r.ErrorInputCount = r.ErrorInputCount + errorInputCount
	r.ErrorInternalCount = r.ErrorInternalCount + errorInternalCount
	r.mutex.Unlock()
}

// ProcessStartupEvent updates the activity store with information from service startup event
func (i *InstanceActivityStore) ProcessStartupEvent(r *ActivityRecord, service service.Service, startupTime time.Time, payload string) {
	r.mutex.Lock()
	r.Service = service
	r.StartupTime = startupTime
	r.ProcessedStartup = true

	var serviceInfo uclog.ServiceStartupInfo
	if err := json.Unmarshal([]byte(payload), &serviceInfo); err == nil {
		r.Region = serviceInfo.Region
		r.CodeVersion = serviceInfo.CodeVersion
		r.Hostname = serviceInfo.Hostname
		r.EventMetadataVersion = 0
		r.EventCount = 0
	}
	r.mutex.Unlock()
}

// UpdateTenant - updates tenant information in the record
func (i *InstanceActivityStore) UpdateTenant(r *ActivityRecord, tenantID uuid.UUID) {

	if tenantID != uuid.Nil && tenantID != r.LastTenantID {
		r.mutex.Lock()
		if r.LastTenantID != uuid.Nil && r.LastTenantID != tenantID {
			r.Multitenant = true
		}
		r.LastTenantID = tenantID
		r.mutex.Unlock()

		i.tenantInstancesMutex.Lock()
		if _, ok := i.tenantInstances[tenantID]; !ok {
			i.tenantInstances[tenantID] = make(map[uuid.UUID]bool)
		}
		i.tenantInstances[tenantID][r.InstanceID] = true
		i.tenantInstancesMutex.Unlock()
	}
}

// UpdateEventMapVersion - updates tenant information in the record
func (i *InstanceActivityStore) UpdateEventMapVersion(r *ActivityRecord, newMapVersion int) {
	r.mutex.Lock()
	r.EventMetadataVersion = newMapVersion
	r.mutex.Unlock()
}

func (i *InstanceActivityStore) getOrCreateActivityRecord(id uuid.UUID) *ActivityRecord {
	i.activityDataMutex.Lock()
	defer i.activityDataMutex.Unlock()

	r, ok := i.activityData[id]
	if !ok {
		// First time we are seeing this instance so create the basic record
		r = &ActivityRecord{InstanceID: id, mutex: sync.Mutex{}}
		i.activityData[id] = r
	}
	return r
}

func (i *InstanceActivityStore) updateBaseInfoCreateActivityRecord(r *ActivityRecord, url *url.URL) {
	svc := service.Service(url.Query().Get("service"))

	if r.Service == "" && service.IsValid(svc) {
		r.Service = svc
	}

	if r.StartupTime.IsZero() {
		r.StartupTime = time.Now().UTC()
	}
}

// GetActivityRecordsForTenant - updates tenant information in the record
func (i *InstanceActivityStore) GetActivityRecordsForTenant(tenantID uuid.UUID) []*ActivityRecord {
	i.tenantInstancesMutex.Lock()
	t, ok := i.tenantInstances[tenantID]
	if !ok {
		t = make(map[uuid.UUID]bool)
	}
	a := make([]*ActivityRecord, 0, len(t))
	i.activityDataMutex.Lock()
	for r := range t {
		a = append(a, i.activityData[r])
	}
	i.activityDataMutex.Unlock()
	i.tenantInstancesMutex.Unlock()
	return a
}
