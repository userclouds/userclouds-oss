package ucopensearch

import (
	"context"
)

// Status is the result of checking the connection to the OpenSearch cluster.
type Status struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}

// GetOpenSearchStatus returns the status of the OpenSearch cluster.
func GetOpenSearchStatus(ctx context.Context, cfg *Config) Status {
	if cfg == nil {
		return Status{Ok: false, Error: "OpenSearch disabled"}
	}
	client, err := NewClient(ctx, cfg)
	if err != nil {
		return Status{Ok: false, Error: err.Error()}
	}
	if _, err = client.client.Info(ctx, nil); err != nil {
		return Status{Ok: false, Error: err.Error()}
	}
	return Status{Ok: true}
}
