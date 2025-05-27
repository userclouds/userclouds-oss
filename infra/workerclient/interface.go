package workerclient

import (
	"context"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/worker"
)

// Client defines an interface for clients that send messages to various queues
type Client interface {
	Send(context.Context, worker.Message) error
}

// NewClientFromConfig returns a new Client based on the given config
func NewClientFromConfig(ctx context.Context, cfg *Config) (Client, error) {
	client, err := newClient(ctx, cfg)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "Initialized worker client [%v]: %s", cfg.Type, client)
	return client, nil
}

func newClient(ctx context.Context, cfg *Config) (Client, error) {
	switch cfg.Type {
	case TypeSQS:
		return newSQSClient(ctx, cfg)
	case TypeHTTP:
		url := cfg.GetURL()
		if url == "" {
			return nil, ucerr.Errorf("missing URL for HTTP worker client")
		}
		return NewHTTPClient(url), nil
	case TypeTest:
		if !universe.Current().IsTestOrCI() {
			return nil, ucerr.Errorf("cannot use test worker client outside of test/CI universe")
		}
		return NewTestClient(), nil
	default:
		return nil, ucerr.Errorf("unknown queue type: %s", cfg.Type)
	}
}
