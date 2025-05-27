package ucimage

// Config holds the configuration required for storing and retrieving images from aws
type Config struct {
	Host     string `yaml:"host" json:"host" validate:"notempty"`
	S3Bucket string `yaml:"s3_bucket" json:"s3_bucket" validate:"notempty"`
}

//go:generate genvalidate Config
