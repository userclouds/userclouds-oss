package ucaws

// Config defines configuration for UserClouds AWS access.
type Config struct {
	AccessKeyID     string `yaml:"access_key" json:"access_key" validate:"notempty"`
	AccessKeySecret string `yaml:"secret_key" json:"secret_key" validate:"notempty"` // TODO (sgarrity 1/24): convert to secret.String without cycle
	Region          string `yaml:"region" json:"region" validate:"notempty"`
}

// UseAccessKey returns true if the Config has an access key.
func (c *Config) UseAccessKey() bool {
	return c.AccessKeyID != "" && c.AccessKeySecret != ""
}

//go:generate genvalidate Config
