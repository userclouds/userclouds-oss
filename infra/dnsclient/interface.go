package dnsclient

import "context"

// Client defines a DNS client so we can mock in tests
type Client interface {
	LookupCNAME(ctx context.Context, host string) ([]string, error)
	LookupTXT(ctx context.Context, host string) ([][]string, error)
}

// NewFromConfig returns a DNS Client
func NewFromConfig(cfg *Config) Client {
	return &client{config: cfg}
}

// Config configures a DNS client
type Config struct {
	HostAndPort string `yaml:"host_and_port" json:"host_and_port" validate:"notempty"`
}

//go:generate genvalidate Config
