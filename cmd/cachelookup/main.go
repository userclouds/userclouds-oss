package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"

	"userclouds.com/authz"
	"userclouds.com/idp/helpers"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/yamlconfig"
	"userclouds.com/internal/cmdline"
)

type keyInfo struct {
	key  string
	ttl  time.Duration
	data string
}

type cacheHelper struct {
	tenantID    uuid.UUID
	redisClient *redis.Client
	np          cache.KeyNameProvider
	keyID       cache.KeyNameID
}

func (ch *cacheHelper) providerInfo() string {
	return reflect.ValueOf(ch.np).Elem().Type().PkgPath()
}

// CacheOnlyConfig is a config struct for the cache helper
type CacheOnlyConfig struct {
	CacheConfig cache.Config `yaml:"cache" json:"cache"`
}

//go:generate genvalidate CacheOnlyConfig

func newCacheHelper(ctx context.Context, objectType, tenantIDOrName string) *cacheHelper {
	var cfg CacheOnlyConfig
	// The cache config is specified in the base env files (base_debug.yaml, base_prod.yaml, etc..)
	// so we load it from there, we don't need to load any service specific config
	if err := yamlconfig.LoadEnv(ctx, "fake", cfg, yamlconfig.GetLoadParams(true, true, true)); err != nil {
		uclog.Fatalf(ctx, "Failed to load idp config: %v", err)
	}
	redisClient, err := cache.GetRedisClient(ctx, cfg.CacheConfig.GetLocalRedis())
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get redis client: %v", err)
	}

	ccs := cmdline.GetCompanyStorage(ctx)
	tenant, err := cmdline.GetTenantByIDOrName(ctx, ccs, tenantIDOrName)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get tenant '%v': %v", tenantIDOrName, err)

	}
	keyID, provider := getCacheKeyIDAndNameProvider(ctx, objectType, tenant.ID, "KeyID")
	if provider == nil {
		uclog.Fatalf(ctx, "No cache key name provider found for tenant %v", tenant.ID)
	}
	return &cacheHelper{
		tenantID:    tenant.ID,
		redisClient: redisClient,
		np:          provider,
		keyID:       keyID,
	}
}

func main() {
	ctx := context.Background()

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "cachelookup")
	defer logtransports.Close()

	if len(os.Args) < 3 {
		uclog.Infof(ctx, "Usage: cachelookup <objecttype> <tenant ID or name> [object ID]")
		uclog.Infof(ctx, "object ID is optional, if not specified, all objects of the given type will be listed")
		uclog.Infof(ctx, "UC_UNIVERSE and UC_REGION environment variables must be set")
		uclog.Fatalf(ctx, "unknown usage")
	}
	objectType := os.Args[1]
	tenantIDOrName := os.Args[2]
	helper := newCacheHelper(ctx, objectType, tenantIDOrName)

	if len(os.Args) > 3 {
		// If the object ID is provided, show info for that object
		helper.lookupObject(ctx, os.Args[3])
	} else {
		// list all objects of the given type
		helper.lookupObjects(ctx)
	}
}

func (ch *cacheHelper) lookupObjects(ctx context.Context) {
	uclog.Infof(ctx, "Looking up cached objects by Key ID '%v' (from %v)", ch.keyID, ch.providerInfo())
	pattern := string(ch.np.GetKeyNameWithString(ch.keyID, "*"))
	keys, err := ch.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get keys w/ pattern '%v': %v", pattern, err)
	}
	if len(keys) == 0 {
		uclog.Infof(ctx, "No keys found (pattern: %s)", pattern)
		return
	}
	for _, key := range keys {
		if ki := ch.getInfoForKey(ctx, key, false); ki != nil {
			uclog.Infof(ctx, "Key %v TTL: %v", ki.key, ki.ttl)
		}
	}
}

func (ch *cacheHelper) lookupObject(ctx context.Context, objIDStr string) {
	objID := uuid.Must(uuid.FromString(objIDStr))
	uclog.Infof(ctx, "Looking up object %v by Key ID %v (from %v)", objID, ch.keyID, ch.providerInfo())
	key := string(ch.np.GetKeyNameWithID(ch.keyID, objID))
	if ki := ch.getInfoForKey(ctx, key, true); ki == nil {
		uclog.Infof(ctx, "Key %v not found", key)
	} else {
		uclog.Infof(ctx, "Key %v found. TTL: %v seconds", ki.key, ki.ttl)
		uclog.Infof(ctx, "\n\n\n")
		uclog.Infof(ctx, "%v", ki.data)
	}
}

func getCacheKeyIDAndNameProvider(ctx context.Context, objectType string, tenantID uuid.UUID, keyType string) (cache.KeyNameID, cache.KeyNameProvider) {
	objectType = fmt.Sprintf("%s%s", strings.ToLower(objectType), strings.ToLower(keyType))
	idpNP := helpers.GetCacheNameProviderForTenantID(tenantID)
	if keyID := findKeyID(idpNP.GetAllKeyIDs(), objectType); keyID != "" {
		return cache.KeyNameID(keyID), idpNP
	}
	authNP := authz.NewCacheNameProviderForTenant(tenantID)
	if keyID := findKeyID(authNP.GetAllKeyIDs(), objectType); keyID != "" {
		return cache.KeyNameID(keyID), authNP
	}
	uclog.Fatalf(ctx, "No cache key found for object type %v", objectType)
	return cache.KeyNameID(""), nil
}

func findKeyID(keys []string, objType string) string {
	for _, key := range keys {
		if strings.ToLower(key) == objType {
			return key
		}
	}
	return ""
}

func (ch *cacheHelper) getInfoForKey(ctx context.Context, key string, loadData bool) *keyInfo {
	existsVal, err := ch.redisClient.Exists(ctx, key).Result()
	if err != nil {
		uclog.Fatalf(ctx, "Failed to check if key %v exists: %v", key, err)
	}
	if existsVal != 1 {
		return nil

	}

	ttl, err := ch.redisClient.TTL(ctx, key).Result()
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get TTL for key %v: %v", key, err)
	}
	if !loadData {
		return &keyInfo{key: key, ttl: ttl, data: ""}
	}
	data, err := ch.redisClient.Get(ctx, key).Result()
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get data for key %v: %v", key, err)
	}

	var i any
	if err := json.Unmarshal([]byte(data), &i); err == nil {
		formattedJSON, err := json.MarshalIndent(i, "", "  ")
		if err == nil {
			data = string(formattedJSON)
		}
	}
	return &keyInfo{key: key, ttl: ttl, data: data}
}
