package uiinitdata

// Config for UI init data
type Config struct {
	StatsigAPIKey string       `yaml:"statsigapikey" json:"statsigapikey"`
	Sentry        SentryConfig `yaml:"sentry" json:"sentry"`
}

// SentryConfig for UI init data
type SentryConfig struct {
	Dsn string `yaml:"dsn" json:"dsn"`
}

//go:generate genvalidate Config
//go:generate genvalidate SentryConfig
