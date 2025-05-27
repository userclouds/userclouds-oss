package skeleton

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/cache"
	serviceNamespace "userclouds.com/infra/namespace/service"
)

func TestHealthCheck(t *testing.T) {
	// Create a mock server
	mockService := serviceNamespace.IDP
	startupTime := time.Now().UTC()

	server := Server{
		service:     mockService,
		startupTime: startupTime,
		cacheConfig: &cache.Config{RedisCacheConfig: []cache.RegionalRedisConfig{*cache.NewLocalRedisClientConfigForTests()}},
	}

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/healthcheck", nil)
	w := httptest.NewRecorder()

	// Call the healthCheck handler
	server.healthCheck(w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	// With a proper Redis client config for tests, the status code should be OK
	// since Cache.Ok will be true
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	// Decode the response
	var hcs HealthCheckStatus
	err := json.NewDecoder(resp.Body).Decode(&hcs)
	assert.NoErr(t, err)

	// Verify the service field
	assert.Equal(t, hcs.Service, mockService)
	assert.Equal(t, hcs.StartupTime, startupTime)
}
