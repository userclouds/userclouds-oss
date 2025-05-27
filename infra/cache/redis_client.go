package cache

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"sync"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/cryptobyte"
	cryptobyte_asn1 "golang.org/x/crypto/cryptobyte/asn1"

	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

var redisClientsMutex = sync.RWMutex{}
var redisClients = map[string]*redis.Client{}

var awsCertPool *x509.CertPool

// RedisStatus is the result of checking the connection to to a redis server.
type RedisStatus struct {
	Ok    bool   `json:"ok" yaml:"ok"`
	Error string `json:"error" yaml:"error"`
}

const (
	certLocation = "/etc/ssl/certs/ca-certificates.crt"
)

// InitRedisCertForCloud - For cloud envs, where we use TLS to connect to redis, we need to load the AWS root certs
func InitRedisCertForCloud() error {
	f, err := os.ReadFile(certLocation)
	if err != nil {
		return ucerr.Wrap(err)
	}

	cp := x509.NewCertPool()
	var b *pem.Block
	rest := f
	for {
		b, rest = pem.Decode(rest)

		// TODO (sgarrity 10/23): This is a hack to get around the fact that
		// for some reason I haven't yet figured out, the root certs listed on EC2
		// have some x509v3 extensions that are not supported by the Go x509 parser.
		// They load fine using openssl, but not Go.  So we just trim off the extensions
		// that go's ASN1 parser doesn't grok, and continue. This is probably not an
		// awesome answer, but the alternatives I see are either to hard code a given
		// root cert into the code (that is currently used by our redis clusters), or
		// just bypass cert validation altogether. I think this is least bad?
		input := cryptobyte.String(b.Bytes)
		if !input.ReadASN1Element(&input, cryptobyte_asn1.SEQUENCE) {
			return ucerr.New("failed to parse cert")
		}

		cert, err := x509.ParseCertificate(input)
		if err != nil {
			return ucerr.Errorf("failed to parse cert: %w", err)
		}
		cp.AddCert(cert)

		if len(rest) == 0 {
			break
		}
	}

	awsCertPool = cp

	return nil
}

// GetRedisClient returns a redis client for the given config either returning one from the cache or creating it.
func GetRedisClient(ctx context.Context, cfg *RedisConfig) (*redis.Client, error) {
	if cfg == nil {
		return nil, ucerr.New("Redis config is nil")
	}

	// Remove password from config before using it as a key
	cfgKey := cfg.CacheKey()

	redisClientsMutex.RLock()
	existingClient := redisClients[cfgKey]
	redisClientsMutex.RUnlock()

	if existingClient != nil {
		return existingClient, nil
	}

	rc, err := NewRedisClient(ctx, cfg)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	uclog.Infof(ctx, "Created new RedisClient [%v]", cfg)

	if err := rc.Ping(ctx).Err(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	// Save a local copy of the client before taking the lock
	lrc := rc
	redisClientsMutex.Lock()
	if redisClients[cfgKey] == nil {
		redisClients[cfgKey] = rc
	} else {
		rc = redisClients[cfgKey]
	}
	redisClientsMutex.Unlock()
	// There was a race and another thread already set the client
	if lrc != rc {
		if err := lrc.Close(); err != nil {
			uclog.Errorf(ctx, "Failed to close duplicate RedisClient [%v] - %v", cfg, err)
		}
	}

	return rc, nil
}

// NewRedisClient returns a new redis client based on the given config and applies AWS cert (for TLS) if needed.
func NewRedisClient(ctx context.Context, redisCfg *RedisConfig) (*redis.Client, error) {
	if redisCfg == nil {
		return nil, ucerr.Errorf("redis config is nil")
	}
	options := redis.Options{
		Addr: fmt.Sprintf("%v:%v", redisCfg.Host, redisCfg.Port),
		DB:   int(redisCfg.DBName),
	}
	if redisCfg.Username != "" {
		pw, err := redisCfg.Password.Resolve(ctx)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		options.Username = redisCfg.Username
		options.Password = pw
	}
	if awsCertPool != nil {
		options.TLSConfig = &tls.Config{
			RootCAs: awsCertPool,
			// AWS Elasticache Redis only support TLS 1.2 and below
			MinVersion: tls.VersionTLS12,
		}
	}
	rc := redis.NewClient(&options)
	if rc == nil {
		return nil, ucerr.Errorf("failed to create redis client")
	}
	return rc, nil
}

// NewLocalRedisClient returns a new redis client that connects to locally running redis server.
func NewLocalRedisClient() *redis.Client {
	return NewLocalRedisClientForDB(0)
}

// NewLocalRedisClientForDB returns a new redis client that connects to locally running redis server connected to the given DB.
func NewLocalRedisClientForDB(redisDB int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       redisDB,
	})
}

// NewLocalRedisClientConfigForTests return a new redis config that connects to locally running redis server.
func NewLocalRedisClientConfigForTests() *RegionalRedisConfig {
	return &RegionalRedisConfig{
		RedisConfig: RedisConfig{
			Host:     "localhost",
			Port:     6379,
			DBName:   0,
			Username: "default",
		},
		Region: region.Current(),
	}
}

// GetRedisStatus returns the status of the local (regional) redis server.
func GetRedisStatus(ctx context.Context, cfg *Config) RedisStatus {
	if cfg == nil {
		return RedisStatus{
			Ok:    true, // We validate config is not nil in config objects. so if it nil here that is ok and not an error
			Error: "cache not configured",
		}
	}
	localCfg := cfg.GetLocalRedis()
	if localCfg == nil {
		return RedisStatus{
			Ok:    false,
			Error: "No local redis config found",
		}
	}
	rc, err := GetRedisClient(ctx, localCfg)
	if err != nil {
		uclog.Errorf(ctx, "Failed to get redis client: %v", err)
		return RedisStatus{
			Ok:    false,
			Error: "Failed to get redis client",
		}
	}
	if err := rc.Ping(ctx).Err(); err != nil {
		uclog.Errorf(ctx, "Failed to ping local redis at %s: %v", localCfg.Host, err)
		return RedisStatus{
			Ok:    false,
			Error: "Failed to ping redis",
		}
	}
	return RedisStatus{
		Ok: true,
	}
}
