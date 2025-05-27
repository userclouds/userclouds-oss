package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/events"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
	logServerClient "userclouds.com/logserver/client"
)

// GetCountsRequest is the request for a event count
type GetCountsRequest struct {
	ObjectIDs         []uuid.UUID     `json:"object_ids"`
	Start             time.Time       `json:"start"`
	End               time.Time       `json:"end"`
	Service           service.Service `json:"service"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	EventSuffixFilter []string        `json:"event_suffix_filter"`
}

// GetCountsResponse is the response for a event count
type GetCountsResponse struct {
	ID    uuid.UUID `json:"id"`
	Count int       `json:"count"`
}

func (h *handler) listCounterRecords(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var limit = 30
	if limitString := r.URL.Query().Get("event_count"); limitString != "" {
		var err error
		limit, err = strconv.Atoi(limitString)
		if err != nil || limit <= 0 || limit > 1000 {
			jsonapi.MarshalError(ctx, w, ucerr.New("Invalid number of events specified"))
			return
		}
	}

	var svc = "plex"
	if serviceString := r.URL.Query().Get("service"); serviceString != "" {
		svc = serviceString
	}

	cRecords, err := h.consoleLogServerClient.ListCounterRecordsForTenant(ctx, svc, limit, id)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, cRecords)
}

func (h *handler) listCounterSources(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	sources, err := h.consoleLogServerClient.GetSourcesForTenant(ctx, id)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, sources)
}

func (h *handler) getCharts(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var queries []logServerClient.CounterQuery
	if err := jsonapi.Unmarshal(r, &queries); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	charts, err := h.consoleLogServerClient.GetCharts(ctx, &queries)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, charts)
}

func (h *handler) getCounts(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var query GetCountsRequest
	var counts []GetCountsResponse
	if err := jsonapi.Unmarshal(r, &query); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	idpClient, err := h.newIDPClient(ctx, query.TenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// For each objectId request count of events
	// TODO: support other object types (e.g. transformers, mutators)
	for _, objectID := range query.ObjectIDs {
		// Get accessor by the objectID to get the accessor version
		var accessor *userstore.Accessor
		accessor, err := idpClient.GetAccessor(ctx, objectID)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}

		// Get event strings for the ObjectID
		eventStrings := []string{}
		for _, event := range events.GetEventsForAccessor(objectID, accessor.Version) {

			// If no suffix filter, add all events, otherwise filter by suffix
			if len(query.EventSuffixFilter) == 0 {
				eventStrings = append(eventStrings, event.StringID)
			} else {
				for _, suffix := range query.EventSuffixFilter {
					if strings.HasSuffix(event.StringID, suffix) {
						eventStrings = append(eventStrings, event.StringID)
					}
				}
			}
		}

		q := logServerClient.CountQueryRequest{
			EventStrings: eventStrings,
			Start:        query.Start,
			End:          query.End,
			TenantID:     query.TenantID,
			Service:      query.Service,
		}

		res, err := h.consoleLogServerClient.GetCounts(ctx, &q)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}

		count := GetCountsResponse{
			ID:    objectID,
			Count: res.Count,
		}

		counts = append(counts, count)
	}

	jsonapi.Marshal(w, counts)
}
