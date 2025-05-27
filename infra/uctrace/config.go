package uctrace

// Config represents tracing config
type Config struct {
	CollectorHost string `yaml:"collector_host" json:"collector_host" validate:"notempty"`
}

//go:generate genvalidate Config
