package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"

	"userclouds.com/idp"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/featureflags"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
)

const (
	lookAsideCacheTTL        = 24 * time.Hour
	defaultInvalidationDelay = 50 * time.Millisecond // Every idp config write incurs this delay. Starting very conservatively but we can lower this
	idpCacheName             = "idpCache"

	userTableName = "users"
)

// Since we recreate idp.Storage on every request we need to share the cache provider across all instances to have cross request cache
var sharedCache cache.Provider
var sharedCacheOnce sync.Once

// Storage handles IDP database access
type Storage struct {
	db       *ucdb.DB
	tenantID uuid.UUID
	cm       *cache.Manager
}

// MustCreateStorage returns a Storage object by getting the tenant state from context
func MustCreateStorage(ctx context.Context) *Storage {
	ts := multitenant.MustGetTenantState(ctx)
	return NewFromTenantState(ctx, ts)
}

// NewFromTenantState returns a Storage object
func NewFromTenantState(ctx context.Context, ts *tenantmap.TenantState) *Storage {
	return New(ctx, ts.TenantDB, ts.ID, ts.CacheConfig)
}

// New returns a Storage object
func New(ctx context.Context, db *ucdb.DB, tenantID uuid.UUID, cc *cache.Config) *Storage {
	return NewWithCacheInvalidationWrapper(ctx, db, tenantID, getSharedCache(ctx, cc, defaultInvalidationDelay))
}

func getSharedCache(ctx context.Context, cfg *cache.Config, invalidationDelay time.Duration) cache.Provider {
	if cfg == nil || cfg.RedisCacheConfig == nil {
		return nil
	}

	if universe.Current().IsTestOrCI() {
		invalidationDelay = 1
	}

	sharedCacheOnce.Do(func() {
		var err error
		sharedCache, err = cache.InitializeInvalidatingCacheFromConfig(
			ctx,
			cfg,
			idpCacheName,
			CachePrefix,
			cache.Layered(),
			cache.InvalidationDelay(invalidationDelay),
		)
		if err != nil {
			uclog.Errorf(ctx, "failed to create cache invalidation wrapper: %v", err)
		}
	})
	return sharedCache
}

// GetCacheManager returns a cache manager configured same way as the cache manager used by storage layer
func GetCacheManager(ctx context.Context, cfg *cache.Config, tenantID uuid.UUID) (*cache.Manager, error) {
	return getCacheManagerInternal(tenantID, getSharedCache(ctx, cfg, defaultInvalidationDelay))
}

// getCacheManagerInternal returns a cache manager
func getCacheManagerInternal(tenantID uuid.UUID, sc cache.Provider) (*cache.Manager, error) {
	if sc == nil {
		return nil, nil
	}

	ttlP := newIDPCacheTTLProvider(lookAsideCacheTTL)
	np := NewCacheNameProviderForTenant(tenantID)
	cm := cache.NewManager(sc, np, ttlP)
	return &cm, nil
}

// NewWithCacheInvalidationWrapper returns a Storage object with a specified cache invalidation wrapper (exposed publicly only for testing)
func NewWithCacheInvalidationWrapper(ctx context.Context, ucdb *ucdb.DB, tenantID uuid.UUID, sc cache.Provider) *Storage {
	storage := &Storage{db: ucdb, tenantID: tenantID}
	cm, err := getCacheManagerInternal(tenantID, sc)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to create idp cache manager: %v", err)
	}
	storage.cm = cm

	return storage
}

// CacheManager returns the cache manager for this storage (exposed publicly only for tests)
func (s *Storage) CacheManager() *cache.Manager {
	return s.cm
}

// GetTenantID returns the id of the storage tenant
func (s *Storage) GetTenantID() uuid.UUID {
	return s.tenantID
}

// GetObjectOrganizationID returns the organization ID for the given object ID
func (s *Storage) GetObjectOrganizationID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}

	obj, err := authzClient.GetObject(ctx, id)

	if err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}

	return obj.OrganizationID, nil
}

// GetPasswordAuthnForUsername returns the username/password authn associated with a username
func (s *Storage) GetPasswordAuthnForUsername(ctx context.Context, username string) (*PasswordAuthn, error) {
	const q = `SELECT id, created, updated, deleted, user_id, username, password FROM authns_password WHERE username=$1 AND deleted='0001-01-01 00:00:00';`

	var authn PasswordAuthn
	if err := s.db.GetContext(ctx, "GetPasswordAuthnForUsername", &authn, q, username); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &authn, nil
}

// UserHasAuthnType checks if a user has a given authn type
func (s *Storage) UserHasAuthnType(ctx context.Context, userID uuid.UUID, authnType idp.AuthnType) (bool, error) {
	if authnType == idp.AuthnTypePassword || authnType == idp.AuthnTypeAll {
		const passwordQ = `SELECT COUNT(user_id) FROM authns_password WHERE user_id=$1 AND deleted='0001-01-01 00:00:00';`
		var count int
		err := s.db.GetContext(ctx, "UserHasAuthnType", &count, passwordQ, userID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, ucerr.Wrap(err)
		} else if err == nil && count > 0 {
			return true, nil
		}
	}

	if authnType == idp.AuthnTypeOIDC || authnType == idp.AuthnTypeAll {
		const socialQ = `SELECT COUNT(user_id) FROM authns_social WHERE user_id=$1 AND deleted='0001-01-01 00:00:00';`
		var count int
		err := s.db.GetContext(ctx, "UserHasAuthnType", &count, socialQ, userID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, ucerr.Wrap(err)
		} else if err == nil && count > 0 {
			return true, nil
		}
	}

	return false, nil
}

// GetOIDCAuthnForSubject returns the OIDC authn associated with an oidc provider type, issuer URL, and OIDC subject.
func (s *Storage) GetOIDCAuthnForSubject(ctx context.Context, provider oidc.ProviderType, issuerURL string, subject string) (*OIDCAuthn, error) {
	const q = `SELECT id, created, updated, deleted, user_id, type, oidc_issuer_url, oidc_sub FROM authns_social WHERE type=$1 AND oidc_issuer_url=$2 AND oidc_sub=$3 AND deleted='0001-01-01 00:00:00';`

	var oidcAuthn OIDCAuthn
	if err := s.db.GetContext(ctx, "GetOIDCAuthnForSubject", &oidcAuthn, q, provider, issuerURL, subject); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &oidcAuthn, nil
}

// GetAccessorByVersion looks up a specific version of an access policy
func (s Storage) GetAccessorByVersion(ctx context.Context, id uuid.UUID, version int) (*Accessor, error) {
	const q = `SELECT id, created, updated, deleted, name, description, version, data_life_cycle_state, column_ids, transformer_ids, token_access_policy_ids, access_policy_id, are_column_access_policies_overridden, selector_config, purpose_ids, is_system, is_audit_logged, is_autogenerated, search_column_id, use_search_index FROM accessors WHERE id=$1 AND version=$2 AND deleted='0001-01-01 00:00:00';`

	var ac Accessor
	if err := s.db.GetContext(ctx, "GetAccessorByVersion", &ac, q, id, version); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &ac, nil
}

// GetAccessorByName is used for de-duping to ensure names are not reused, it returns the latest version of the accessor with the given name
func (s Storage) GetAccessorByName(ctx context.Context, name string) (*Accessor, error) {
	const q = `
/* lint-sql-allow-multi-column-aggregation */
SELECT a.id, a.created, a.updated, a.deleted, a.name, a.description, a.version, a.data_life_cycle_state, a.column_ids, a.transformer_ids, a.token_access_policy_ids, a.access_policy_id, a.are_column_access_policies_overridden, a.selector_config, a.purpose_ids, a.is_system, a.is_audit_logged, a.is_autogenerated, a.search_column_id, a.use_search_index
FROM accessors AS a
JOIN (SELECT id, name, MAX(version) version FROM accessors WHERE deleted='0001-01-01 00:00:00' GROUP BY id, name) AS b
ON a.id=b.id
AND a.name=b.name
AND a.version=b.version
WHERE a.deleted='0001-01-01 00:00:00'
AND LOWER(a.name)=LOWER($1);`

	var a Accessor
	if err := s.db.GetContext(ctx, "GetAccessorByName", &a, q, name); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &a, nil
}

// GetLatestAccessors returns the latest version of all accessors
func (s Storage) GetLatestAccessors(ctx context.Context, p pagination.Paginator) ([]Accessor, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf(`SELECT id, created, updated, deleted, name, description, version, data_life_cycle_state, column_ids, transformer_ids, token_access_policy_ids, access_policy_id,
	 are_column_access_policies_overridden, selector_config, purpose_ids, is_system, is_audit_logged, is_autogenerated, search_column_id, use_search_index FROM(
		SELECT id, created, updated, deleted, name, description, version, data_life_cycle_state, column_ids, transformer_ids, token_access_policy_ids, access_policy_id,
		are_column_access_policies_overridden, selector_config, purpose_ids, is_system, is_audit_logged, is_autogenerated, search_column_id, use_search_index FROM(
			SELECT a.id, a.created, a.updated, a.deleted, a.name, a.description, a.version, a.data_life_cycle_state, a.column_ids, a.transformer_ids, a.token_access_policy_ids, a.access_policy_id,
				a.are_column_access_policies_overridden, a.selector_config, a.purpose_ids, a.is_system, a.is_audit_logged, a.is_autogenerated, a.search_column_id, a.use_search_index FROM accessors AS a
			JOIN (select id, max(version) version
			FROM accessors
	 		WHERE deleted='0001-01-01 00:00:00' GROUP BY id) AS b
	 		ON a.id = b.id AND a.version=b.version) tmp
	 	WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp2
	 ORDER BY %s; /* lint-sql-unsafe-columns */`, p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var acs []Accessor
	if err := s.db.SelectContext(ctx, "GetLatestAccessors", &acs, q, queryFields...); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	objs, respFields := pagination.ProcessResults(acs, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())

	if respFields.HasNext {
		if err := p.ValidateCursor(respFields.Next); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	if respFields.HasPrev {
		if err := p.ValidateCursor(respFields.Prev); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	return objs, &respFields, nil
}

// GetAllAccessorVersions returns all versions of a given accessor
func (s Storage) GetAllAccessorVersions(ctx context.Context, id uuid.UUID) ([]Accessor, error) {
	const q = `SELECT id, created, updated, deleted, name, description, version, data_life_cycle_state, column_ids, transformer_ids, token_access_policy_ids, access_policy_id, are_column_access_policies_overridden, selector_config, purpose_ids, is_system, is_audit_logged, is_autogenerated, search_column_id, use_search_index FROM accessors WHERE id=$1 AND deleted='0001-01-01 00:00:00';`

	var acs []Accessor
	if err := s.db.SelectContext(ctx, "GetAllAccessorVersions", &acs, q, id); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return acs, nil
}

// GetMutatorByVersion looks up a specific version of an access policy
func (s Storage) GetMutatorByVersion(ctx context.Context, id uuid.UUID, version int) (*Mutator, error) {
	const q = `SELECT id, created, updated, deleted, name, description, version, column_ids, access_policy_id, validator_ids, selector_config, is_system FROM mutators WHERE id=$1 AND version=$2 AND deleted='0001-01-01 00:00:00';`

	var m Mutator
	if err := s.db.GetContext(ctx, "GetMutatorByVersion", &m, q, id, version); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &m, nil
}

// GetMutatorByName is used for de-duping to ensure names are not reused, it returns the latest version of the mutator with the given name
func (s Storage) GetMutatorByName(ctx context.Context, name string) (*Mutator, error) {
	const q = `
/* lint-sql-allow-multi-column-aggregation */
SELECT a.id, a.created, a.updated, a.deleted, a.name, a.description, a.version, a.column_ids, a.access_policy_id, a.validator_ids, a.selector_config, a.is_system
FROM mutators AS a
JOIN (SELECT id, name, MAX(version) version FROM mutators WHERE deleted='0001-01-01 00:00:00' GROUP BY id, name) AS b
ON a.id=b.id
AND a.name=b.name
AND a.version=b.version
WHERE a.deleted='0001-01-01 00:00:00'
AND LOWER(a.name)=LOWER($1);`

	var m Mutator
	if err := s.db.GetContext(ctx, "GetMutatorByName", &m, q, name); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &m, nil
}

// GetLatestMutators returns the latest version of all mutators
func (s Storage) GetLatestMutators(ctx context.Context, p pagination.Paginator) ([]Mutator, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf(`
				SELECT id, created, updated, deleted, name, description, version, column_ids, access_policy_id, validator_ids, selector_config, is_system FROM(
					SELECT id, created, updated, deleted, name, description, version, column_ids, access_policy_id, validator_ids, selector_config, is_system FROM(
						SELECT a.id, a.created, a.updated, a.deleted, a.name, a.description, a.version, a.column_ids, a.access_policy_id, a.validator_ids, selector_config, a.is_system FROM mutators AS a
						JOIN (select id, max(version) version FROM mutators WHERE deleted='0001-01-01 00:00:00' GROUP BY id) AS b
						ON a.id = b.id AND a.version=b.version) tmp
					WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp2
				ORDER BY %s; /* lint-sql-unsafe-columns */`, p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var ms []Mutator
	if err := s.db.SelectContext(ctx, "GetLatestMutators", &ms, q, queryFields...); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	objs, respFields := pagination.ProcessResults(ms, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())

	if respFields.HasNext {
		if err := p.ValidateCursor(respFields.Next); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	if respFields.HasPrev {
		if err := p.ValidateCursor(respFields.Prev); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	return objs, &respFields, nil
}

// GetAllMutatorVersions returns all versions of a given mutator
func (s Storage) GetAllMutatorVersions(ctx context.Context, id uuid.UUID) ([]Mutator, error) {
	const q = `SELECT id, created, updated, deleted, name, description, version, column_ids, access_policy_id, validator_ids, selector_config, is_system FROM mutators WHERE id=$1 AND deleted='0001-01-01 00:00:00';`

	var m []Mutator
	if err := s.db.SelectContext(ctx, "GetAllMutatorVersions", &m, q, id); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return m, nil
}

// GetUserColumnByName is used for de-duping to ensure names are not reused, it returns the column with the given name
func (s Storage) GetUserColumnByName(ctx context.Context, name string) (*Column, error) {
	return s.GetColumnByDatabaseTableAndName(ctx, uuid.Nil, userTableName, name)
}

// GetColumnByDatabaseTableAndName is used for de-duping to ensure names are not reused, it returns the column with the given database, table, and name
func (s Storage) GetColumnByDatabaseTableAndName(ctx context.Context, databaseID uuid.UUID, table, name string) (*Column, error) {
	var secondaryKey cache.Key
	if s.cm != nil {
		secondaryKey = s.cm.N.GetKeyName(ColumnNameKeyID, []string{databaseID.String(), strings.ToLower(table), strings.ToLower(name)})
	}
	return s.getColumnByColumns(ctx, secondaryKey, []string{"sqlshim_database_id", "LOWER(tbl)", "LOWER(name)"}, []any{databaseID, strings.ToLower(table), strings.ToLower(name)})
}

// GetPurposeByName is used for de-duping to ensure names are not reused, it returns the purpose with the given name
func (s Storage) GetPurposeByName(ctx context.Context, name string) (*Purpose, error) {
	var secondaryKey cache.Key
	if s.cm != nil {
		secondaryKey = s.cm.N.GetKeyName(PurposeNameKeyID, []string{name})
	}
	return s.getPurposeByColumns(ctx, secondaryKey, []string{"LOWER(name)"}, []any{strings.ToLower(name)})
}

// GetPurposesForResourceIDs returns a list of purposes for the given resource ids
func (s Storage) GetPurposesForResourceIDs(ctx context.Context, errorOnMissing bool, purposeRIDs ...userstore.ResourceID) ([]Purpose, error) {
	var ids []uuid.UUID
	var names []string

	for _, rid := range purposeRIDs {
		if rid.ID != uuid.Nil {
			ids = append(ids, rid.ID)
		} else if rid.Name != "" {
			names = append(names, strings.ToLower(rid.Name))
		} else {
			return nil, ucerr.Friendlyf(nil, "invalid purpose id: %+v", rid)
		}
	}

	const q = `
SELECT id, updated, deleted, name, description, created, is_system
FROM purposes
WHERE (id=ANY($1) OR LOWER(name)=ANY($2))
AND deleted='0001-01-01 00:00:00';`

	var objects []Purpose
	if err := s.db.SelectContext(ctx, "GetPurposesForResourceIDs", &objects, q, pq.Array(ids), pq.Array(names)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	found := make([]bool, len(purposeRIDs))
	for _, obj := range objects {
		for i, rid := range purposeRIDs {
			if obj.ID == rid.ID || strings.EqualFold(obj.Name, rid.Name) {
				found[i] = true
				if rid.Name != "" && !strings.EqualFold(obj.Name, rid.Name) {
					return nil, ucerr.Errorf("purpose name mismatch for resource ID: %v, got %s", rid, obj.Name)
				}
				if rid.ID != uuid.Nil && obj.ID != rid.ID {
					return nil, ucerr.Errorf("purpose  ID mismatch for resource ID: %v, got %s", rid, obj.ID)
				}
			}
		}
	}

	if errorOnMissing {
		missingRIDs := []string{}
		for i := range found {
			if !found[i] {
				missingRIDs = append(missingRIDs, fmt.Sprintf("%+v", purposeRIDs[i]))
			}
		}

		if len(missingRIDs) > 0 {
			return nil, ucerr.Errorf("Not all requested IDs where loaded. Missing: [%s]", strings.Join(missingRIDs, ", "))
		}
	}
	return objects, nil
}

// GetTransformersMap returns a map of the requested transformer IDs to transformer objects.
func (s *Storage) GetTransformersMap(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*Transformer, error) {
	transformerMap := make(map[uuid.UUID]*Transformer)

	for _, id := range ids {
		if _, ok := transformerMap[id]; !ok {
			itemFromDB, err := s.GetLatestTransformer(ctx, id)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			transformerMap[id] = itemFromDB
		}
	}

	return transformerMap, nil
}

// GetDataSourceElementIDsForSourceID returns the IDs of the data source elements for a given data source
func (s *Storage) GetDataSourceElementIDsForSourceID(ctx context.Context, sourceID uuid.UUID) ([]uuid.UUID, error) {
	const q = `SELECT id FROM data_source_elements WHERE data_source_id=$1 AND deleted='0001-01-01 00:00:00'; /* lint-sql-unsafe-columns bypass-known-table-check */`

	var ids []uuid.UUID
	if err := s.db.SelectContext(ctx, "GetDataSourceElementIDsForSource", &ids, q, sourceID); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return ids, nil
}

func (s *Storage) getGlobalAccessPolicy(
	ctx context.Context,
	tenantID uuid.UUID,
	globalAccessPolicyID uuid.UUID,
) (*AccessPolicy, error) {
	if featureflags.IsEnabledForTenant(ctx, featureflags.GlobalAccessPolicies, tenantID) {
		if accessPolicy, err := s.GetLatestAccessPolicy(ctx, globalAccessPolicyID); err == nil {
			return accessPolicy, nil
		}
	}

	accessPolicy, err := s.GetLatestAccessPolicy(ctx, policy.AccessPolicyAllowAll.ID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return accessPolicy, nil
}

// GetAccessPolicies looks up the global and action-specific access policies, and determines
// which should be used for threshold limiting
func (s *Storage) GetAccessPolicies(
	ctx context.Context,
	tenantID uuid.UUID,
	globalAccessPolicyID uuid.UUID,
	actionAccessPolicyID uuid.UUID,
) (
	globalAP *policy.AccessPolicy,
	actionAP *policy.AccessPolicy,
	thresholdAP *AccessPolicy,
	err error,
) {
	gap, err := s.getGlobalAccessPolicy(ctx, tenantID, globalAccessPolicyID)
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}

	aap, err := s.GetLatestAccessPolicy(ctx, actionAccessPolicyID)
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}

	if !aap.hasThreshold() && gap.hasThreshold() {
		thresholdAP = gap
	} else {
		thresholdAP = aap
	}

	return gap.ToClientModel(), aap.ToClientModel(), thresholdAP, nil
}
