package ucopensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gofrs/uuid"
	opensearch "github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	requestsigner "github.com/opensearch-project/opensearch-go/v4/signer/awsv2"

	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const (
	refreshInterval = 1 * time.Minute

	defaultMaxSearchResults = 1000
)

// Client is a wrapper around the opensearch client
type Client struct {
	client     *opensearchapi.Client
	MaxResults int
}

// Per docs, it is beneficial to create an OpenSearch.Client instance once and reuse it for all OpenSearch operations. The client is thread safe,
// so the same instance can be shared by multiple threads
var clientCache = make(map[Config]*opensearchapi.Client)
var clientCacheLock = &sync.Mutex{}

// StopConnectionRefresh is a channel to stop the connection refresh
var StopConnectionRefresh = make(chan bool)

// NewClientWithRetry creates a new opensearch client wrapper given a config and enables retry on network timeouts
func NewClientWithRetry(ctx context.Context, searchCfg *Config) (*Client, error) {
	awsCfg, err := ucaws.NewConfigWithRegion(ctx, region.GetAWSRegion(searchCfg.Region))
	if err != nil {
		return nil, ucerr.Friendlyf(err, "error creating AWS config")
	}
	return newClient(ctx, searchCfg, true, awsCfg)
}

// NewClient creates a new opensearch client wrapper given a config
func NewClient(ctx context.Context, searchCfg *Config) (*Client, error) {
	awsCfg, err := ucaws.NewConfigWithRegion(ctx, region.GetAWSRegion(searchCfg.Region))
	if err != nil {
		return nil, ucerr.Friendlyf(err, "error creating AWS config")
	}
	return newClient(ctx, searchCfg, false, awsCfg)
}

// NewClientForLocalTool creates a new opensearch client wrapper given a config for use locally on a dev machine
func NewClientForLocalTool(ctx context.Context, searchCfg *Config) (*Client, error) {
	awsCfg, err := ucaws.NewConfigForProfile(ctx, region.GetAWSRegion(searchCfg.Region), universe.Current())
	if err != nil {
		return nil, ucerr.Friendlyf(err, "error creating AWS config")
	}
	return newClient(ctx, searchCfg, false, awsCfg)
}

func newClient(ctx context.Context, searchCfg *Config, retryOnTimeout bool, awsCfg aws.Config) (*Client, error) {
	if searchCfg == nil {
		return nil, ucerr.Errorf("Config is nil")
	}

	clientCacheLock.Lock()
	defer clientCacheLock.Unlock()

	client, ok := clientCache[*searchCfg]
	if !ok {
		cfg, err := newSearchClientConfig(ctx, searchCfg, retryOnTimeout, awsCfg)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		client, err = opensearchapi.NewClient(*cfg)
		if err != nil {
			return nil, ucerr.Friendlyf(err, "error creating opensearch client")
		}
		clientCache[*searchCfg] = client
	}

	maxSearchResults := defaultMaxSearchResults
	if searchCfg.MaxResults > 0 {
		maxSearchResults = searchCfg.MaxResults
	}
	uclog.Debugf(ctx, "New OpenSearch Client %s", searchCfg.URL)
	return &Client{
		client:     client,
		MaxResults: maxSearchResults,
	}, nil
}

func newSearchClientConfig(ctx context.Context, searchCfg *Config, retryOnTimeout bool, awsCfg aws.Config) (*opensearchapi.Config, error) {
	signer, err := requestsigner.NewSignerWithService(awsCfg, "es")
	if err != nil {
		return nil, ucerr.Friendlyf(err, "error creating signer")
	}

	return &opensearchapi.Config{Client: opensearch.Config{
		Addresses:            []string{searchCfg.URL},
		Signer:               signer,
		EnableRetryOnTimeout: retryOnTimeout,
	}}, nil
}

// SearchRequest sends a search request to opensearch and returns the response body
func (c *Client) SearchRequest(ctx context.Context, indexName, searchQuery string) ([]byte, error) {
	resp, err := c.client.Search(ctx, &opensearchapi.SearchReq{Indices: []string{indexName}, Body: strings.NewReader(searchQuery)})
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	rawResp := resp.Inspect().Response
	if rawResp.IsError() {
		return nil, ucerr.Errorf("opensearch Search request failed: '%s'", rawResp.String())
	}

	respBody, err := io.ReadAll(rawResp.Body)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return respBody, nil
}

// IndexExists checks if an index exists in opensearch
func (c *Client) IndexExists(ctx context.Context, indexName string) (bool, error) {
	req := opensearchapi.IndicesExistsReq{Indices: []string{indexName}}
	resp, err := c.client.Indices.Exists(ctx, req)
	if err != nil {
		return false, ucerr.Wrap(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, ucerr.Errorf("unexpected status code: %d", resp.StatusCode)
}

// CreateIndex sends a create index request to opensearch
func (c *Client) CreateIndex(ctx context.Context, indexName string, indexDoc string) (string, error) {
	create := opensearchapi.IndicesCreateReq{
		Index: indexName,
		Body:  strings.NewReader(indexDoc),
	}
	createResponse, err := c.client.Indices.Create(ctx, create)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	rawResp := createResponse.Inspect().Response
	if rawResp.IsError() {
		return "", ucerr.Errorf("error calling IndicesCreate: '%s'", rawResp.String())
	}
	bs, err := io.ReadAll(rawResp.Body)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	return string(bs), nil
}

// IndicesStatsRequest sends an indices stats request to opensearch
func (c *Client) IndicesStatsRequest(ctx context.Context, indexName string) (string, error) {
	req := &opensearchapi.IndicesStatsReq{Indices: []string{indexName}}
	resp, err := c.client.Indices.Stats(ctx, req)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	return getResponseBody("indices stats", resp.Inspect())
}

// BulkRequest sends a bulk request to opensearch
func (c *Client) BulkRequest(ctx context.Context, body []byte) ([]byte, error) {
	resp, err := c.client.Bulk(ctx, opensearchapi.BulkReq{Body: bytes.NewBuffer(body)})
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	rawResp := resp.Inspect().Response
	if rawResp.IsError() {
		return nil, ucerr.Errorf("opensearch Bulk request failed: '%s'", rawResp.String())
	}
	respBody, err := io.ReadAll(rawResp.Body)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return respBody, nil
}

// IndexMetadata gets the metadata for an index
func (c *Client) IndexMetadata(ctx context.Context, indexName string) (string, error) {
	resp, err := c.client.Indices.Get(ctx, opensearchapi.IndicesGetReq{Indices: []string{indexName}})
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	return getResponseBody("index metadata", resp.Inspect())
}

// ListIndices lists all indices in opensearch
func (c *Client) ListIndices(ctx context.Context) (string, error) {
	resp, err := c.client.Cat.Indices(ctx, &opensearchapi.CatIndicesReq{})
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	return getResponseBody("cat indices", resp.Inspect())
}

// TenantIndexMapGetter is the function to get current tenant index map
type TenantIndexMapGetter func(ctx context.Context) (map[uuid.UUID][]QueryableIndex, error)

// InitializeConnections keeps sending search requests to the opensearch cluster to keep the connections alive
func InitializeConnections(ctx context.Context, searchCfg *Config, mapGetter TenantIndexMapGetter) error {
	if searchCfg == nil {
		return nil
	}

	searchClient, err := NewClient(ctx, searchCfg)
	if err != nil {
		return ucerr.Wrap(err)
	}

	uclog.Infof(ctx, "Initialized opensearch connection to %v", *searchCfg)
	tenantIndexMap, err := mapGetter(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	uclog.Infof(ctx, "Got tenant map %v", tenantIndexMap)
	for tenantID, indices := range tenantIndexMap {
		for _, index := range indices {
			if err := makePingSearchRequest(ctx, "foobar", index, tenantID, searchClient); err != nil {
				return ucerr.Errorf("Failed to init connection for tenant %s: %w", tenantID, err)
			}
		}
	}

	// Initialize a background thread to keep the OS connections alive.
	refreshTicker := *time.NewTicker(refreshInterval)

	go func() {
		for {
			select {
			case <-StopConnectionRefresh:
				return
			case <-refreshTicker.C:
				tenantIndexMap, err := mapGetter(ctx)
				if err != nil {
					uclog.Errorf(ctx, "Failed to connection keep alive search: %v", err)
					continue
				}

				for tenantID, indices := range tenantIndexMap {
					for _, index := range indices {
						if err := makePingSearchRequest(ctx, "foobar", index, tenantID, searchClient); err != nil {
							uclog.Errorf(ctx, "Failed to connection keep alive search: %v", err)
						}
					}
				}
			}
		}
	}()
	return nil
}

func makePingSearchRequest(ctx context.Context, query string, index QueryableIndex, tenantID uuid.UUID, searchClient *Client) error {
	searchQuery, err := index.GetIndexQuery(query, 5)
	if err != nil {
		return ucerr.Wrap(err)
	}
	indexName := index.GetIndexName(tenantID)
	if _, err := searchClient.SearchRequest(ctx, indexName, searchQuery); err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Verbosef(ctx, "Sent ping search request to %v for %v", indexName, tenantID)
	return nil
}

// IsNetworkTimeoutError checks if the error is a network timeout error
func IsNetworkTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}

func getResponseBody(apiName string, inspectedResp opensearchapi.Inspect) (string, error) {
	rawResp := inspectedResp.Response
	if rawResp.IsError() {
		return "", ucerr.Friendlyf(nil, "%s error: '%s'", apiName, rawResp.String())
	}
	bs, err := io.ReadAll(rawResp.Body)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	return string(bs), nil
}

// CountDocumentsByIDs retrieves multiple documents by their IDs from the specified index and returns the count of documents found
func (c *Client) CountDocumentsByIDs(ctx context.Context, indexName string, docIDs []string) (int, error) {
	jsonBody, err := json.Marshal(map[string]any{"ids": docIDs})
	if err != nil {
		return -1, ucerr.Wrap(err)
	}

	resp, err := c.client.MGet(ctx, opensearchapi.MGetReq{Index: indexName, Body: bytes.NewReader(jsonBody)})
	if err != nil {
		return -1, ucerr.Wrap(err)
	}
	rawResp := resp.Inspect().Response
	if rawResp.IsError() {
		return -1, ucerr.Errorf("opensearch MGet request failed: '%s'", rawResp.String())
	}
	found := 0
	for _, doc := range resp.Docs {
		if doc.Found {
			found++
		}
	}
	return found, nil
}
