package ucmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "uc"
)

// Subsystem is a Prometheus subsystem name (used to construct the metric name)
type Subsystem string

// CreateGauge creates a new gauge metric
func CreateGauge(subsystem Subsystem, name, help string, labels ...string) *prometheus.GaugeVec {
	return promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: string(subsystem),
		Name:      name,
		Help:      help,
	}, labels)
}

// CreateCounter creates a new counter metric
func CreateCounter(subsystem Subsystem, name, help string, labels ...string) *prometheus.CounterVec {
	return promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: string(subsystem),
		Name:      name,
		Help:      help,
	}, labels)
}

// CreateHistogram creates a new histogram metric
func CreateHistogram(subsystem Subsystem, name, help string, labels ...string) *prometheus.HistogramVec {
	return promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: string(subsystem),
		Name:      name,
		Help:      help,
		// Use native histograms, which are a new, experimental Prometheus
		// feature that enable sparse and dynamic buckets, rather than
		// requiring you to define the buckets when defining the metric. This
		// is both more efficient and provides higher resolution. The API is
		// not yet stable, but it has been out for over a year and we aren't
		// using these histograms for anything important yet, so I figure this
		// is a good time to try them out and it's okay if they break.
		// https://www.usenix.org/conference/srecon23emea/presentation/rabenstein
		// https://promcon.io/2022-munich/talks/native-histograms-in-prometheus/
		// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus?utm_source=godoc#HistogramOpts
		NativeHistogramBucketFactor: 1.1,
	}, labels)
}
