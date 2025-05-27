package ucsentry

// Config represents sentry config
type Config struct {
	Dsn              string  `yaml:"dsn" json:"dsn" validate:"notempty"`
	TracesSampleRate float64 `yaml:"traces_sample_rate" json:"traces_sample_rate"`
}

//go:generate genvalidate Config
