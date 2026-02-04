package cache

import (
	"fmt"

	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
)

// RedisConfig holds redis connection info
type RedisConfig struct {
	Host      string        `json:"host" yaml:"host" validate:"notempty"`
	Port      int           `json:"port" yaml:"port" validate:"notzero"`
	DBName    uint8         `json:"dbname" yaml:"dbname"`
	Username  string        `json:"username" yaml:"username"`
	Password  secret.String `json:"password" yaml:"password"`
	EnableTLS bool          `json:"enable_tls" yaml:"enable_tls"`
}

func (cfg *RedisConfig) extraValidate() error {
	uv := universe.Current()
	if !uv.IsCloud() {
		return nil
	}
	if cfg.Username == "" {
		return ucerr.Friendlyf(nil, "RedisConfig.Username can't be empty in %v", uv)
	}
	if cfg.Password.IsEmpty() {
		return ucerr.Friendlyf(nil, "RedisConfig.Password can't be empty in %v", uv)
	}
	return nil

}

// CacheKey generates a string with fields of the config except for the password
func (cfg RedisConfig) CacheKey() string {
	if cfg.Username == "" {
		return fmt.Sprintf("%s-%d-%d", cfg.Host, cfg.Port, cfg.DBName)
	}
	return fmt.Sprintf("%s-%d-%s-%d", cfg.Host, cfg.Port, cfg.Username, cfg.DBName)
}

//go:generate genvalidate RedisConfig

// RegionalRedisConfig holds redis connection info for a specific region
type RegionalRedisConfig struct {
	RedisConfig `yaml:",inline" json:",inline"` // embedding this to make it easy to manage
	Region      region.MachineRegion `json:"region" yaml:"region"`
}

//go:generate genvalidate RegionalRedisConfig

// Config represents config currently just for redis caches, but the name should be provider agnostic etc
type Config struct {
	RedisCacheConfig []RegionalRedisConfig `yaml:"redis_caches,omitempty" json:"redis_caches,omitempty"`
}

//go:generate genvalidate Config

// GetLocalRedis returns our local redis cluster
func (n Config) GetLocalRedis() *RedisConfig {
	for i, r := range n.RedisCacheConfig {
		if r.Region == region.Current() {
			return &n.RedisCacheConfig[i].RedisConfig
		}
	}
	return nil
}

// GetRemoteRedisClusters returns all the remote clusters (for eg publishing invalidation messages)
func (n Config) GetRemoteRedisClusters() []RedisConfig {
	var clusters []RedisConfig
	for i, r := range n.RedisCacheConfig {
		if r.Region != region.Current() {
			clusters = append(clusters, n.RedisCacheConfig[i].RedisConfig)
		}
	}

	return clusters
}
