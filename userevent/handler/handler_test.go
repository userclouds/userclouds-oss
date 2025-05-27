package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
	"userclouds.com/userevent"
)

type testFixture struct {
	t           *testing.T
	handler     http.Handler
	email       *uctest.EmailClient
	tenant      *companyconfig.Tenant
	bearerToken string
}

func newTestFixture(t *testing.T) *testFixture {
	// Set up test company & tenant
	ctx := context.Background()
	_, tenant, _, _, handler, _ := testhelpers.CreateTestServer(ctx, t)
	jwt := uctest.CreateJWT(t, oidc.UCTokenClaims{}, fmt.Sprintf("http://jerry.%s", testhelpers.TestTenantSubDomain))
	return &testFixture{
		t:           t,
		handler:     handler,
		email:       &uctest.EmailClient{}, // Stub out email dispatch so we can retrieve MFA codes.
		tenant:      tenant,
		bearerToken: fmt.Sprintf("Bearer %s", jwt),
	}
}

func reportEvent(tf *testFixture, events []userevent.UserEvent) *http.Response {
	reader := uctest.IOReaderFromJSONStruct(tf.t, userevent.ReportEventsRequest{Events: events})
	req := httptest.NewRequest(http.MethodPost, tf.tenant.TenantURL+"/userevent/events", reader)
	req.Header.Add("Authorization", tf.bearerToken)
	w := httptest.NewRecorder()
	tf.handler.ServeHTTP(w, req)
	return w.Result()
}

func listEvents(tf *testFixture) []userevent.UserEvent {
	req := httptest.NewRequest(http.MethodGet, tf.tenant.TenantURL+"/userevent/events", nil)
	req.Header.Add("Authorization", tf.bearerToken)
	w := httptest.NewRecorder()
	tf.handler.ServeHTTP(w, req)
	assert.Equal(tf.t, w.Result().StatusCode, http.StatusOK, assert.Errorf("expected code %d, got %d", http.StatusOK, w.Result().StatusCode), assert.Must())
	var resp userevent.ListEventsResponse
	assert.NoErr(tf.t, json.NewDecoder(w.Result().Body).Decode(&resp))
	return resp.Data
}

func TestUserEvent(t *testing.T) {
	tf := newTestFixture(t)

	// Test ingesting multiple events
	// NOTE: keep event Types alpha-sorted to make validation later easier.
	events := []userevent.UserEvent{
		{BaseModel: ucdb.NewBase(), Type: "some_event1", UserAlias: "user1", Payload: userevent.Payload{"key1": "val1"}},
		{BaseModel: ucdb.NewBase(), Type: "some_event2", UserAlias: "user2", Payload: userevent.Payload{"key1": "val1"}},
	}
	resp := reportEvent(tf, events)
	assert.Equal(t, resp.StatusCode, http.StatusNoContent)

	eventsResp := listEvents(tf)
	assert.Equal(t, len(eventsResp), 2)

	// Alpha sort events to make comparison easier
	sort.Slice(eventsResp, func(i, j int) bool {
		return eventsResp[i].Type < eventsResp[j].Type
	})

	// Validate event contents
	for i, ev := range events {
		evResp := eventsResp[i]
		assert.Equal(t, evResp.Type, ev.Type)
		assert.Equal(t, evResp.UserAlias, ev.UserAlias)
		assert.Equal(t, evResp.Payload, ev.Payload)
	}

	// Test appending a new event (NOTE: event Type alpha sorts to the end)
	newEvents := []userevent.UserEvent{
		{
			BaseModel: ucdb.NewBase(),
			Type:      "some_event3",
			UserAlias: "foo",
			Payload:   userevent.Payload{},
		},
	}

	resp = reportEvent(tf, newEvents)
	assert.Equal(t, resp.StatusCode, http.StatusNoContent)

	eventsResp = listEvents(tf)
	assert.Equal(t, len(eventsResp), 3, assert.Must())

	sort.Slice(eventsResp, func(i, j int) bool {
		return eventsResp[i].Type < eventsResp[j].Type
	})

	concatEvents := append(events, newEvents...)
	for i := range concatEvents {
		assert.Equal(t, eventsResp[i].Type, concatEvents[i].Type)
		assert.Equal(t, eventsResp[i].UserAlias, concatEvents[i].UserAlias)
		assert.Equal(t, eventsResp[i].Payload, concatEvents[i].Payload)
	}
}

func TestBadEventName(t *testing.T) {
	tf := newTestFixture(t)

	// Invalid identifier name
	resp := reportEvent(tf, []userevent.UserEvent{{BaseModel: ucdb.NewBase(), Type: "!invalid", UserAlias: "user1"}})
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// Missing user alias
	resp = reportEvent(tf, []userevent.UserEvent{{BaseModel: ucdb.NewBase(), Type: "missing_user_id"}})
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
}
