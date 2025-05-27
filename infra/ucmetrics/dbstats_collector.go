// The code below is adapted from the Prometheus dbstats collector:
// https://github.com/prometheus/client_golang/blob/v1.18.0/prometheus/collectors/dbstats_collector.go
// https://pkg.go.dev/github.com/prometheus/client_golang@v1.18.0/prometheus/collectors#NewDBStatsCollector
// This collector fetches metrics from the golang sql.DB.Stats() method. The
// unaltered version from the Prometheus library works great except for when we
// have multiple DB instances with the same name. (Ideally this shouldn't
// happen, but it does happen notably in provisioning e.g. with companyconfig,
// where we load companyconfig in one place to fetch tenant info, and in
// another place when we actually provision the companyconfig db. It happens in
// parallel test execution as well). In that situation, we get a "duplicate
// metrics collector registration attempted" error because we are registering
// multiple metrics with the same database name label (and there are no other
// labels to distinguish the data point values). This modified collector simply
// adds a monotonically increasing "pool_nonce" label so that we never have
// duplicate metrics, even with the same db name.

// Copyright 2021 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ucmetrics

import (
	"database/sql"
	"fmt"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"

	"userclouds.com/infra/ucerr"
)

type dbStatsCollector struct {
	db *sql.DB

	maxOpenConnections *prometheus.Desc

	openConnections  *prometheus.Desc
	inUseConnections *prometheus.Desc
	idleConnections  *prometheus.Desc

	waitCount         *prometheus.Desc
	waitDuration      *prometheus.Desc
	maxIdleClosed     *prometheus.Desc
	maxIdleTimeClosed *prometheus.Desc
	maxLifetimeClosed *prometheus.Desc
}

var lastPoolNonce atomic.Uint64

// NewDBStatsCollector returns a collector that exports metrics about the given *sql.DB.
// See https://golang.org/pkg/database/sql/#DBStats for more information on stats.
func NewDBStatsCollector(db *sql.DB, dbName string) prometheus.Collector {
	fqName := func(name string) string {
		return "go_sql_" + name
	}
	nonce := fmt.Sprintf("%d", lastPoolNonce.Add(1))
	return &dbStatsCollector{
		db: db,
		maxOpenConnections: prometheus.NewDesc(
			fqName("max_open_connections"),
			"Maximum number of open connections to the database.",
			nil, prometheus.Labels{"db_name": dbName, "pool_nonce": nonce},
		),
		openConnections: prometheus.NewDesc(
			fqName("open_connections"),
			"The number of established connections both in use and idle.",
			nil, prometheus.Labels{"db_name": dbName, "pool_nonce": nonce},
		),
		inUseConnections: prometheus.NewDesc(
			fqName("in_use_connections"),
			"The number of connections currently in use.",
			nil, prometheus.Labels{"db_name": dbName, "pool_nonce": nonce},
		),
		idleConnections: prometheus.NewDesc(
			fqName("idle_connections"),
			"The number of idle connections.",
			nil, prometheus.Labels{"db_name": dbName, "pool_nonce": nonce},
		),
		waitCount: prometheus.NewDesc(
			fqName("wait_count_total"),
			"The total number of connections waited for.",
			nil, prometheus.Labels{"db_name": dbName, "pool_nonce": nonce},
		),
		waitDuration: prometheus.NewDesc(
			fqName("wait_duration_seconds_total"),
			"The total time blocked waiting for a new connection.",
			nil, prometheus.Labels{"db_name": dbName, "pool_nonce": nonce},
		),
		maxIdleClosed: prometheus.NewDesc(
			fqName("max_idle_closed_total"),
			"The total number of connections closed due to SetMaxIdleConns.",
			nil, prometheus.Labels{"db_name": dbName, "pool_nonce": nonce},
		),
		maxIdleTimeClosed: prometheus.NewDesc(
			fqName("max_idle_time_closed_total"),
			"The total number of connections closed due to SetConnMaxIdleTime.",
			nil, prometheus.Labels{"db_name": dbName, "pool_nonce": nonce},
		),
		maxLifetimeClosed: prometheus.NewDesc(
			fqName("max_lifetime_closed_total"),
			"The total number of connections closed due to SetConnMaxLifetime.",
			nil, prometheus.Labels{"db_name": dbName, "pool_nonce": nonce},
		),
	}
}

// Describe implements Collector.
func (c *dbStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.maxOpenConnections
	ch <- c.openConnections
	ch <- c.inUseConnections
	ch <- c.idleConnections
	ch <- c.waitCount
	ch <- c.waitDuration
	ch <- c.maxIdleClosed
	ch <- c.maxLifetimeClosed
	ch <- c.maxIdleTimeClosed
}

// Collect implements Collector.
func (c *dbStatsCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.db.Stats()
	ch <- prometheus.MustNewConstMetric(c.maxOpenConnections, prometheus.GaugeValue, float64(stats.MaxOpenConnections))
	ch <- prometheus.MustNewConstMetric(c.openConnections, prometheus.GaugeValue, float64(stats.OpenConnections))
	ch <- prometheus.MustNewConstMetric(c.inUseConnections, prometheus.GaugeValue, float64(stats.InUse))
	ch <- prometheus.MustNewConstMetric(c.idleConnections, prometheus.GaugeValue, float64(stats.Idle))
	ch <- prometheus.MustNewConstMetric(c.waitCount, prometheus.CounterValue, float64(stats.WaitCount))
	ch <- prometheus.MustNewConstMetric(c.waitDuration, prometheus.CounterValue, stats.WaitDuration.Seconds())
	ch <- prometheus.MustNewConstMetric(c.maxIdleClosed, prometheus.CounterValue, float64(stats.MaxIdleClosed))
	ch <- prometheus.MustNewConstMetric(c.maxLifetimeClosed, prometheus.CounterValue, float64(stats.MaxLifetimeClosed))
	ch <- prometheus.MustNewConstMetric(c.maxIdleTimeClosed, prometheus.CounterValue, float64(stats.MaxIdleTimeClosed))
}

// Collector is an object that collects existing metrics to be exposed by Prometheus.
type Collector struct {
	collector prometheus.Collector
}

// RegisterDBStatsCollector registers a collector for the given DB, so Prometheus will scrape DB metrics. See collector docs:
// https://pkg.go.dev/github.com/prometheus/client_golang@v1.18.0/prometheus/collectors#NewDBStatsCollector
// And golang DB stats docs:
// https://pkg.go.dev/database/sql#DBStats
func RegisterDBStatsCollector(db *sql.DB, dbName string) (Collector, error) {
	c := NewDBStatsCollector(db, dbName)
	if err := prometheus.Register(c); err != nil {
		return Collector{}, ucerr.Wrap(err)
	}
	return Collector{
		collector: c,
	}, nil
}

// UnregisterDBStatsCollector unregisters the given collector, so Prometheus will no longer scrape DB metrics for that database.
func UnregisterDBStatsCollector(c *Collector) {
	if c != nil {
		prometheus.Unregister(c.collector)
	}
}
