package acme

import "userclouds.com/infra/secret"

// Config represents our ACME server account / config
type Config struct {
	// which CA we talk to
	DirectoryURL string `yaml:"directory_url,omitempty" json:"directory_url" validate:"notempty"`

	// our account at that CA
	AccountURL string        `yaml:"account_url,omitempty" json:"account_url" validate:"notempty"`
	PrivateKey secret.String `yaml:"private_key,omitempty" json:"private_key"`
}

//go:generate genvalidate Config
