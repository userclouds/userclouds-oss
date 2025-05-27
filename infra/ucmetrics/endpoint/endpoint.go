package endpoint

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"userclouds.com/infra/uchttp"
)

const (
	metricsPath = "/metrics"
)

// AddMetricsEndpoint adds a /metrics endpoint to the given ServeMux
func AddMetricsEndpoint(mux *uchttp.ServeMux) {
	mux.Handle(metricsPath, promhttp.Handler())
}
