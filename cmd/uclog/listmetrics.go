package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	logServerClient "userclouds.com/logserver/client"
)

var durations = map[string]int{}
var counts = map[string]int{}
var serviceNames = []string{"plex", "idp", "authz", "tokenizer", "console"}

// ListMetricsForService lists all the log messages in the kinesis stream for [tenant, service, region]
func ListMetricsForService(ctx context.Context, tenantID uuid.UUID, tenantURL string, clientID string, clientSecret string, service string,
	timeWindow string, serviceFilter string, callsummary bool, summary bool) error {
	ts := jsonclient.ClientCredentialsTokenSource(tenantURL+"/oidc/token", clientID, clientSecret, nil)
	consoleLogServerClient, err := logServerClient.NewClientForTenant(tenantURL, tenantID, ts)
	if err != nil {
		return ucerr.Wrap(err)
	}

	for _, s := range serviceNames {
		// If service is specified only fetch events for that service
		if service != "" && service != "all" && s != service {
			continue
		}

		recs, err := consoleLogServerClient.ListCounterRecordsForTenant(ctx, s, 999, tenantID)
		if err != nil {
			return ucerr.Wrap(err)
		}

		for _, r := range *recs {

			t := time.Unix(r.Timestamp, 0)

			key := fmt.Sprintf("%s: [%s] %s", t.Format("01-02-2006"), s, r.EventName)

			if _, ok := durations[key]; !ok {
				durations[key] = 0
				counts[key] = 0
			}
			if r.EventType == "Duration" {
				durations[key] = durations[key] + r.Count
			} else if r.EventName != "HTTP Request" {
				counts[key] = counts[key] + r.Count
			}
		}
	}
	keys := make([]string, 0, len(counts))

	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, n := range keys {
		if c := counts[n]; c > 0 {
			uclog.Debugf(ctx, "%s calls: %d average duration: %v", n, c, float64(durations[n])/float64(c))
		}
	}

	return nil
}
