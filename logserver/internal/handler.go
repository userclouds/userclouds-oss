package internal

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/logeventmetadata"
	"userclouds.com/logserver/client"
	"userclouds.com/logserver/config"
	events "userclouds.com/logserver/events"
	"userclouds.com/logserver/internal/countersbackend"
	"userclouds.com/logserver/internal/instancebackend"
	"userclouds.com/logserver/internal/kinesisbackend"
	"userclouds.com/logserver/internal/middleware"
	"userclouds.com/logserver/internal/storage"
)

type handler struct {
	cfg         *config.Config
	tenantCache *storage.TenantCache
	kinesisBE   *kinesisbackend.KinesisConnections
	counterBE   *countersbackend.CountersStore
	activityBE  *instancebackend.InstanceActivityStore
}

// CounterRow provides data for row in the event log
type CounterRow struct {
	ID        uint64          `json:"id" yaml:"id"`
	EventName string          `json:"event_name" yaml:"event_name"`
	EventType string          `json:"event_type" yaml:"event_type"`
	Service   service.Service `json:"service" yaml:"service"`
	Timestamp int64           `json:"timestamp" yaml:"timestamp"`
	Count     int             `json:"count" yaml:"count"`
}

// NewHandler returns a LogServer handler.
func NewHandler(cfg *config.Config, tenantCache *storage.TenantCache, kinesisBE *kinesisbackend.KinesisConnections,
	counterBE *countersbackend.CountersStore, activityBE *instancebackend.InstanceActivityStore) (*uchttp.ServeMux, *uchttp.ServeMux) {
	h := &handler{cfg: cfg, tenantCache: tenantCache, kinesisBE: kinesisBE, counterBE: counterBE, activityBE: activityBE}

	authedHB := builder.NewHandlerBuilder()
	if h.kinesisBE != nil {
		// Handles uploads of data to pass through to the kinesis stream
		authedHB.HandleFunc("/logserver/kinesis", h.kinesisHandler)
	}
	// Handles upload of counter data to be aggregated and recorded
	authedHB.HandleFunc("/logserver/counters", h.countersHandler)
	// Handles download of recent activity
	authedHB.HandleFunc("/logserver/activity", h.activityHandler)
	// Handles download of infra instance for tenant
	authedHB.HandleFunc("/logserver/instances", h.instanceHandler)
	// Handles queries of metrics for charts
	authedHB.HandleFunc("/logserver/chart", h.chartHandler)
	// Handles queries of event count
	authedHB.HandleFunc("/logserver/query", h.queryHandler)
	// Handles management of event types
	authedHB.CollectionHandler("/logserver/eventmetadata").
		Delete(h.deleteEventTypeHandler).
		DeleteAll(h.deleteEventsTypeHandler).
		GetAll(h.getEventTypesHandler).
		GetOne(h.getEventTypeHandler).
		Post(h.createEventTypeHandler).
		WithAuthorizer(uchttp.NewAllowAllAuthorizer())

	// Handles download of event metadata to service running LogServerTransport
	openHB := builder.NewHandlerBuilder().HandleFunc("/default", h.eventdataHandler)

	return authedHB.Build(), openHB.Build()
}

// kinesisHandler passes the data through to the right kinesis stream
func (h *handler) kinesisHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.kinesisBE == nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "Kinesis backend not configured"), jsonapi.Code(http.StatusServiceUnavailable))
		return
	}

	activityRecord, err := h.activityBE.UpdateActivityStore(r.URL)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.New("Invalid or missing instance id"))
		return
	}

	// Read the destination from the query string
	svc := service.Service(r.URL.Query().Get("service"))
	if !service.IsValid(svc) {
		jsonapi.MarshalError(ctx, w, ucerr.New("Invalid service specified"))
		return
	}

	uclog.Debugf(ctx, "Kinesis data post for service %s", svc)

	var messagesMap map[uuid.UUID][][]byte
	if err := jsonapi.Unmarshal(r, &messagesMap); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	for tenantID, messages := range messagesMap {
		uclog.Debugf(ctx, "Processing %d updates for tenant %s service %s", len(messages), tenantID.String(), svc)
		h.activityBE.UpdateTenant(activityRecord, tenantID)

		if !middleware.TenantMatchesOrIsConsole(ctx, tenantID) {
			jsonapi.MarshalError(ctx, w, ucerr.Errorf("Token used to authorize the call %v doesn't match tenant ID %v in the body of request", middleware.GetTenantID(ctx), tenantID),
				jsonapi.Code(http.StatusForbidden))
			return
		}

		// Post the bytes into the correct kinesis stream
		if err := h.kinesisBE.Write(ctx, tenantID, svc, messages); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	settings := h.getUpdatedSettings()
	jsonapi.Marshal(w, settings)
}

func (h *handler) countersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	activityRecord, err := h.activityBE.UpdateActivityStore(r.URL)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.New("Invalid or missing instance id"))
		return
	}

	// Read the destination from the query string
	svc := service.Service(r.URL.Query().Get("service"))
	if !service.IsValid(svc) {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("Invalid service specified: %s", svc))
		return
	}

	baseTimeString := r.URL.Query().Get("base_time")

	var baseTime time.Time
	if err = baseTime.UnmarshalText([]byte(baseTimeString)); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var countersMap map[uuid.UUID]map[string]int
	if err := jsonapi.Unmarshal(r, &countersMap); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var eventCount int
	var errorInternalCount int
	var errorInputCount int
	for tenantID, counters := range countersMap {
		uclog.Debugf(ctx, "Processing %d updates at %s for tenant %s service %s", len(counters), baseTime.String(), tenantID, svc)
		h.activityBE.UpdateTenant(activityRecord, tenantID)

		if !middleware.TenantMatchesOrIsConsole(ctx, tenantID) {
			jsonapi.MarshalError(ctx, w, ucerr.New("Token used to authorize the call doesn't match tenant ID in the body of request"), jsonapi.Code(http.StatusForbidden))
			return
		}

		em, err := h.tenantCache.GetEventMapForTenant(ctx, tenantID)
		if err != nil {
			continue
		}

		mapRefreshed := false
		for key, count := range counters {
			keys := strings.Split(key, "_")

			// The longest key name is EventCode_Offset_EventName_Payload and the shortest (and most common) is
			// EventCode_Offset
			if len(keys) < 2 || len(keys) > 4 {
				jsonapi.MarshalError(ctx, w, err)
				return
			}

			// Parse out the event type
			// TODO: why are we parsing this when it was all structured? rewrite
			eci, err := strconv.Atoi(keys[0])
			if err != nil {
				jsonapi.MarshalError(ctx, w, err)
				return
			}
			eventCode := uclog.EventCode(eci)

			// Parse out the offset in time from baseline time of this set of events
			offset, err := strconv.Atoi(keys[1])
			if err != nil {
				jsonapi.MarshalError(ctx, w, err)
				return
			}

			nextFragment := 2
			// If the client didn't have this event in its map - resolve it server side
			if eventCode == uclog.EventCodeUnknown {
				if len(keys) == 2 {
					jsonapi.MarshalError(ctx, w, ucerr.New("Missing name for client side unresolved event"))
					return
				}
				eventInfo, ok := em.StringIDMap[keys[2]]

				// Try to refresh the eventmetadata cache and see if the event can be resolved
				if !ok && !mapRefreshed {
					mapRefreshed = true
					em, err = h.tenantCache.RefreshEventMapForTenant(ctx, tenantID)
					if err != nil {
						continue
					}
					eventInfo, ok = em.StringIDMap[keys[2]]
				}

				if !ok || eventInfo.Code == uclog.EventCodeUnknown {
					// This event is not known to the server either
					// TODO we should add a new row to the map
					uclog.Warningf(ctx, "Unknown event %s. Please run provisioning", keys[2])
				} else {
					// We found the event in the map so look up the code and if it should be ignored
					eventCode = eventInfo.Code
					if eventInfo.Ignore || eventCode == uclog.EventCodeNone {
						continue
					}
				}
				nextFragment++
			}
			// Get payload if included
			payload := ""
			if len(keys) > nextFragment {
				payload = keys[nextFragment]
				uclog.Debugf(ctx, "Payload %s count %d event %d offset %d \n", payload, count, eventCode, offset)
			}
			eventTime := baseTime.Add(time.Duration(offset * int(time.Second)))

			// Check if this event is expected from the service that is posting it
			eventInfo, ok := em.CodeMap[eventCode]
			if ok && eventInfo.Service != "" && eventInfo.Service != svc {
				uclog.Warningf(ctx, "Expected event %d from service %v. Please update the map in logserver/events/events.go", eventCode, svc)
			}

			if events.IsStartUpEvent(eventCode) {
				h.activityBE.ProcessStartupEvent(activityRecord, svc, eventTime, payload)
			}
			if err := h.counterBE.UpdateCounters(ctx, tenantID, svc, eventCode, eventTime, count); err != nil {
				uclog.Errorf(ctx, "updatecounters: %v", err)
			}

			if !events.IsDuration(eventCode, &em.CodeMap) {
				eventCount += count
			}

			if events.IsInputError(eventCode, &em.CodeMap) {
				errorInputCount += count
			}
			if events.IsInternalError(eventCode, &em.CodeMap) {
				errorInternalCount += count
			}
		}
	}
	h.activityBE.UpdateEventCount(activityRecord, eventCount, errorInputCount, errorInternalCount)
	settings := h.getUpdatedSettings()
	jsonapi.Marshal(w, settings)
}

func (h *handler) instanceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	uclog.Debugf(ctx, "Infra instance get ")

	tenantString := r.URL.Query().Get("tenant_id")
	tenantID, err := uuid.FromString(tenantString)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.New("Invalid or missing tenant id specified"))
		return
	}

	if !middleware.TenantMatchesOrIsConsole(ctx, tenantID) {
		jsonapi.MarshalError(ctx, w, ucerr.New("Token used to authorize the call doesn't match tenant ID in the body of request"), jsonapi.Code(http.StatusForbidden))
		return
	}

	// Get activity records for this tenant
	activityRecords := h.activityBE.GetActivityRecordsForTenant(tenantID)
	// Get the last few events

	// TODO - restrict to only our console domain
	w.Header().Add("Access-Control-Allow-Origin", "*")
	jsonapi.Marshal(w, activityRecords)
}

func (h *handler) activityHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantString := r.URL.Query().Get("tenant_id")
	tenantID, err := uuid.FromString(tenantString)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.New("Invalid or missing tenant id specified"))
		return
	}

	if !middleware.TenantMatchesOrIsConsole(ctx, tenantID) {
		jsonapi.MarshalError(ctx, w, ucerr.New("Token used to authorize the call doesn't match tenant ID in the body of request"), jsonapi.Code(http.StatusForbidden))
		return
	}

	var limit int
	if limitString := r.URL.Query().Get("event_count"); limitString != "" {
		var err error
		limit, err = strconv.Atoi(limitString)
		if err != nil || limit <= 0 || limit > 1000 {
			jsonapi.MarshalError(ctx, w, ucerr.New("Invalid number of events specified"))
			return
		}
	}

	// TODO: this is some really funky behavior ... huh?
	svc := service.Plex
	if serviceString := r.URL.Query().Get("service"); serviceString != "" {
		// TODO support multiple services here
		if svc = service.Service(serviceString); !service.IsValid(svc) {
			jsonapi.MarshalError(ctx, w, ucerr.New("Invalid service specified"))
			return
		}
	}

	uclog.Debugf(ctx, "Activity Get for tenant %v", tenantID)
	l, err := h.counterBE.QueryCountersLog(ctx, tenantID, []service.Service{svc}, limit)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Resolve the event names for the log
	em, err := h.tenantCache.GetEventMapForTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrTenantBeingProvisioned) {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusNotFound))
			return
		}
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var count int
	var le = make([]CounterRow, len(*l))
	for _, e := range *l {
		eventInfo, ok := em.CodeMap[e.EventCode]
		if ok {
			le[count] = CounterRow{ID: e.ID, EventType: string(eventInfo.Category), EventName: eventInfo.Name,
				Service: eventInfo.Service, Timestamp: e.Timestamp, Count: e.Count}
			count++
		}
	}
	// Trim unused space at the end of the array
	le = le[:count]
	// Sort the events according to the time stamps
	sort.SliceStable(le, func(i, j int) bool {
		return le[i].Timestamp > le[j].Timestamp
	})

	jsonapi.Marshal(w, le)
}
func (h *handler) queryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var queryReq client.CountQueryRequest
	var query countersbackend.CountQuery

	if err := jsonapi.Unmarshal(r, &queryReq); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	tenantID := queryReq.TenantID

	if !middleware.TenantMatchesOrIsConsole(ctx, tenantID) {
		jsonapi.MarshalError(ctx, w, ucerr.New("Token used to authorize the call doesn't match tenant ID in the body of request"), jsonapi.Code(http.StatusForbidden))
		return
	}

	em, err := h.tenantCache.GetEventMapForTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Map EventStrings to ucLog EventCodes
	var codes []uclog.EventCode
	for _, eventString := range queryReq.EventStrings {
		if queryReq.Service != em.StringIDMap[eventString].Service {
			continue
		}

		code := em.StringIDMap[eventString].Code
		codes = append(codes, code)
	}

	query = countersbackend.CountQuery{
		TenantID:  queryReq.TenantID,
		Service:   queryReq.Service,
		EventCode: codes,
		Start:     queryReq.Start,
		End:       queryReq.End,
	}

	// Validate query
	if err := query.Validate(); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Fetch the counter data
	c, err := h.counterBE.QueryCount(ctx, query)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, &c)
}

func (h *handler) chartHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var queries []countersbackend.CounterQuery

	if err := jsonapi.Unmarshal(r, &queries); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Ensure that at least one query is provided
	if len(queries) == 0 {
		jsonapi.MarshalError(ctx, w, ucerr.New("No queries in the body of the request"))
		return
	}

	// Validate the data in the queries
	for i := range queries {
		if err := queries[i].Validate(); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}

		uclog.Debugf(ctx, "Query post for tenant %v for service %s events %v", queries[i].TenantID, queries[i].Service, queries[i].EventCode)

		if !middleware.TenantMatchesOrIsConsole(ctx, queries[i].TenantID) {
			jsonapi.MarshalError(ctx, w, ucerr.New("Token used to authorize the call doesn't match tenant ID in the body of request"), jsonapi.Code(http.StatusForbidden))
			return
		}
	}

	// Fetch the counter data
	mR, err := h.counterBE.QueryCounters(ctx, queries)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Generate the charts.
	var charts RechartsResponse
	if mR != nil && len(*mR) > 0 {
		for _, queryData := range *mR {
			// Reinitialize querySets for each queryData iteration.
			var querySets []RechartsColumn

			// Get the rows from the queryData
			rows := queryData.Rows

			// Loop over the "Counts", i.e. the values from the query data to create a new chart
			for i := len(rows[0].Counts) - 1; i >= 0; i-- {
				chartData := RechartsData{}
				chartData.Values = make(map[string]int)
				chartData.XAxis = strconv.Itoa(len(rows[0].Counts) - 1 - i)

				// For each row in the chartData, get the value from the row.Counts[i]
				for _, row := range rows {
					key := strconv.Itoa(int(row.EventCode))
					chartData.Values[key] = row.Counts[i]
				}
				querySets = append(querySets, RechartsColumn{
					Column: []RechartsData{chartData},
				})
			}

			charts.Charts = append(charts.Charts, RechartsChart{
				Chart: querySets,
			})
		}
	}

	jsonapi.Marshal(w, charts)
}

func (h *handler) getUpdatedSettings() client.LogTransportSettings {
	// TODO implement back off logic on basis of load or of the debugging flags for particular instance
	return client.LogTransportSettings{}
}

// This handler doesn't require AuthN to prevent start up sequencing between services
func (h *handler) eventdataHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	activityRecord, err := h.activityBE.UpdateActivityStore(r.URL)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.New("Invalid or missing instance id"))
		return
	}

	// TODO encode this in the URL
	tenantString := r.URL.Query().Get("tenant_id")
	tenantID, err := uuid.FromString(tenantString)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.New("Invalid or missing tenant id specified"))
		return
	}

	uclog.Debugf(ctx, "Got request for event metadata download for %v", tenantID)

	em, err := h.tenantCache.GetEventMapForTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrTenantBeingProvisioned) {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusNotFound))
			return
		}
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	eventdata := events.GetEventMetaDataMap(&em.StringIDMap)

	h.activityBE.UpdateEventMapVersion(activityRecord, eventdata.Version)
	jsonapi.Marshal(w, eventdata)
}

func (h *handler) deleteEventTypeHandler(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	tenantString := r.URL.Query().Get("tenant_id")
	tenantID, err := uuid.FromString(tenantString)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.New("Invalid or missing tenant id specified"))
		return
	}

	s, err := h.tenantCache.GetEventMetadataStorageForTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := s.DeleteMetricMetadata(ctx, id); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validateEventTypeSelectorQuery(query string) bool {
	// Validate the query string TODO - this code should be part of URL generation
	validReferenceURLPrefixes := []string{
		"/tokenizer/policies/transformation/",
		"/tokenizer/policies/access/",
		"/userstore/api/accessors/",
		"/userstore/api/mutators/",
	}

	foundPrefix := false
	queryTrimmed := query
	for _, pref := range validReferenceURLPrefixes {
		if strings.HasPrefix(query, pref) {
			foundPrefix = true
			queryTrimmed = strings.TrimPrefix(query, pref)
		}
	}
	if !foundPrefix {
		return false
	}
	qs := strings.Split(queryTrimmed, "/")
	if len(qs) != 2 {
		return false
	}
	if _, err := strconv.Atoi(qs[1]); err != nil {
		return false
	}
	if _, err := uuid.FromString(qs[0]); err != nil {
		return false
	}
	return true
}

func (h *handler) deleteEventsTypeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantString := r.URL.Query().Get("tenant_id")
	tenantID, err := uuid.FromString(tenantString)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.New("Invalid or missing tenant id specified"))
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		jsonapi.MarshalError(ctx, w, ucerr.New("Query parameter is required"))
		return
	}
	if !validateEventTypeSelectorQuery(query) {
		jsonapi.MarshalError(ctx, w, ucerr.New("Query parameter is invalid"))
		return
	}

	s, err := h.tenantCache.GetEventMetadataStorageForTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := s.DeleteMetricsMetadataByURL(ctx, query); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) getEventTypesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantString := r.URL.Query().Get("tenant_id")
	tenantID, err := uuid.FromString(tenantString)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.New("Invalid or missing tenant id specified"))
		return
	}

	query := r.URL.Query().Get("query")
	if query != "" && !validateEventTypeSelectorQuery(query) {
		jsonapi.MarshalError(ctx, w, ucerr.New("Query parameter is invalid"))
		return
	}

	s, err := h.tenantCache.GetEventMetadataStorageForTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrTenantBeingProvisioned) {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusNotFound))
			return
		}
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// TODO: do we want to paginate getEventTypesHandler?
	var allMetrics []logeventmetadata.MetricMetadata

	pager, err := logeventmetadata.NewMetricMetadataPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit),
	)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	for {
		pageMetrics, respFields, err := s.ListMetricMetadatasPaginated(ctx, *pager)
		if err != nil {
			jsonapi.MarshalSQLError(ctx, w, err)
			return
		}

		if query != "" {
			for _, metric := range pageMetrics {
				if metric.ReferenceURL == query {
					allMetrics = append(allMetrics, metric)
				}
			}
		} else {
			allMetrics = append(allMetrics, pageMetrics...)
		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	jsonapi.Marshal(w, allMetrics)
}

func (h *handler) getEventTypeHandler(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	tenantString := r.URL.Query().Get("tenant_id")
	tenantID, err := uuid.FromString(tenantString)
	if err != nil {
		if errors.Is(err, storage.ErrTenantBeingProvisioned) {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusNotFound))
			return
		}
		jsonapi.MarshalError(ctx, w, ucerr.New("Invalid or missing tenant id specified"))
		return
	}

	s, err := h.tenantCache.GetEventMetadataStorageForTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	typ, err := s.GetMetricMetadata(ctx, id)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, typ)
}

func (h *handler) createEventTypeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantString := r.URL.Query().Get("tenant_id")
	tenantID, err := uuid.FromString(tenantString)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.New("Invalid or missing tenant id specified"))
		return
	}

	s, err := h.tenantCache.GetEventMetadataStorageForTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var metricTypes []logeventmetadata.MetricMetadata
	if err := jsonapi.Unmarshal(r, &metricTypes); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := storage.SaveMetricsMetaDataArray(ctx, metricTypes, s); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Update the cache if needed
	h.tenantCache.AddMetricsTypesForTenant(ctx, tenantID, &metricTypes)

	jsonapi.Marshal(w, metricTypes)
}
