package dnsclient

import (
	"context"
	"sync"

	"github.com/miekg/dns"
)

var clients = map[string]*TestClient{}
var clientMu sync.Mutex

// NewTestClient returns a dns client for testing
func NewTestClient(cfg *Config) *TestClient {
	clientMu.Lock()
	defer clientMu.Unlock()

	if clients[cfg.HostAndPort] == nil {
		clients[cfg.HostAndPort] = &TestClient{answers: map[string]map[string]string{}}
	}

	return clients[cfg.HostAndPort]
}

// TestClient represents a DNS client for testing
type TestClient struct {
	answers map[string]map[string]string
}

// LookupCNAME implements dnsclient.Client
func (tc TestClient) LookupCNAME(ctx context.Context, host string) ([]string, error) {
	return []string{tc.answers[host]["CNAME"]}, nil
}

// LookupTXT implements dnsclient.Client
func (tc TestClient) LookupTXT(ctx context.Context, host string) ([][]string, error) {
	return [][]string{{tc.answers[host]["TXT"]}}, nil
}

// SetAnswer sets an answer for a query in test
func (tc TestClient) SetAnswer(query, typ, answer string) {
	if typ == "CNAME" {
		answer = dns.Fqdn(answer)
	}
	if tc.answers[query] == nil {
		tc.answers[query] = map[string]string{}
	}
	tc.answers[query][typ] = answer
}
