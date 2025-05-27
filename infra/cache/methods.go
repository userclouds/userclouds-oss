package cache

import (
	"context"
	"errors"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/request"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

func logCacheError(ctx context.Context, method string, err error) {
	if errors.Is(err, context.Canceled) {
		uclog.Warningf(ctx, "%s failed: %v", method, err)
	} else {
		uclog.Errorf(ctx, "%s failed: %v", method, err)
	}
}

// CreateItem creates an item
func CreateItem[item SingleItem](ctx context.Context, cm *Manager, id uuid.UUID, i *item, keyID KeyNameID, secondaryKey Key, ifNotExists bool,
	bypassCache bool, additionalKeysToClear []Key, action func(i *item) (*item, error), equals func(input *item, current *item) bool) (*item, error) {
	var err error
	sentinel := NoLockSentinel

	if i == nil {
		return nil, ucerr.Errorf("CreateItem is called with nil input")
	}

	// Validate the item
	if err := (*i).Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	// Check if the object type already exists in the cache if we're using ifNotExists
	if ifNotExists && !bypassCache {
		keyName := secondaryKey
		if !id.IsNil() {
			keyName = cm.N.GetKeyNameWithID(keyID, id)
		}
		v, _, _, err := GetItemFromCache[item](ctx, *cm, keyName, false)
		if err != nil {
			logCacheError(ctx, "CreateItem/GetItemFromCache", err)
		} else if v != nil && equals(i, v) {
			return v, nil
		}
	}

	// On the client we always invalidate the local cache even if the cache if bypassed for read operation
	if cm != nil {
		sentinel, err = TakeItemLock(ctx, Create, *cm, *i)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		defer ReleaseItemLock(ctx, *cm, Create, *i, sentinel)
	}

	var resp *item
	if resp, err = action(i); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if !bypassCache {
		SaveItemToCache(ctx, *cm, *resp, sentinel, true, additionalKeysToClear)
	}
	return resp, nil
}

// CreateItemServer creates an item (wrapper for calls from ORM)
func CreateItemServer[item SingleItem](ctx context.Context, cm *Manager, i *item, keyID KeyNameID, additionalKeysToClear []Key, action func(i *item) error) error {
	_, err := CreateItem[item](ctx, cm, uuid.Nil, i, keyID, "", false, cm == nil, additionalKeysToClear,
		func(i *item) (*item, error) { return i, ucerr.Wrap(action(i)) }, func(input, current *item) bool { return false })
	return ucerr.Wrap(err)
}

// CreateItemClient creates an item (wrapper for calls from client)
func CreateItemClient[item SingleItem](ctx context.Context, cm *Manager, id uuid.UUID, i *item, keyID KeyNameID, secondaryKey Key, ifNotExists bool,
	bypassCache bool, additionalKeysToClear []Key, action func(i *item) (*item, error), equals func(input *item, current *item) bool) (*item, error) {
	ctx = request.NewRequestID(ctx)

	uclog.Verbosef(ctx, "CreateItemClient: %v key %v", *i, keyID)

	val, err := CreateItem[item](ctx, cm, id, i, keyID, secondaryKey, ifNotExists, bypassCache, additionalKeysToClear, action, equals)

	if err != nil {
		logCacheError(ctx, "CreateItemClient/CreateItem", err)
	}
	return val, ucerr.Wrap(err)
}

// GetItem returns the item
func GetItem[item SingleItem](ctx context.Context, cm *Manager, id uuid.UUID, keyID KeyNameID, modifiedKeyID KeyNameID, bypassCache bool, action func(id uuid.UUID, conflict Sentinel, i *item) error) (*item, error) {
	sentinel := NoLockSentinel
	conflict := GenerateTombstoneSentinel()
	if !bypassCache {
		var cachedObj *item
		var err error
		if modifiedKeyID == "" {
			cachedObj, conflict, sentinel, err = GetItemFromCache[item](ctx, *cm, cm.N.GetKeyNameWithID(keyID, id), true)
		} else {
			cachedObj, conflict, sentinel, err = GetItemFromCacheWithModifiedKey[item](ctx, *cm, cm.N.GetKeyNameWithID(keyID, id), cm.N.GetKeyNameWithID(modifiedKeyID, id), true)

		}
		if err != nil && !jsonclient.IsHTTPNotFound(err) {
			logCacheError(ctx, "GetItem/GetItemFromCache", err)
		} else if cachedObj != nil {
			return cachedObj, nil
		}
	}

	var resp item
	if err := action(id, conflict, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if !bypassCache {
		SaveItemToCache(ctx, *cm, resp, sentinel, false, nil)
	}
	return &resp, nil
}

// GetItemClient returns the item (wrapper for calls from client)
func GetItemClient[item SingleItem](ctx context.Context, cm Manager, id uuid.UUID, keyID KeyNameID, bypassCache bool, action func(id uuid.UUID, conflict Sentinel, i *item) error) (*item, error) {
	ctx = request.NewRequestID(ctx)
	uclog.Verbosef(ctx, "ClientGetItem: %v key %v", id, keyID)
	val, err := GetItem[item](ctx, &cm, id, keyID, "", bypassCache, action)
	if err != nil && !jsonclient.IsHTTPNotFound(err) {
		logCacheError(ctx, "GetItemClient/GetItem", err)
	}
	return val, ucerr.Wrap(err)
}

// ServerGetItem returns the item (wrapper for calls from ORM)
func ServerGetItem[item SingleItem](ctx context.Context, cm *Manager, id uuid.UUID, keyID KeyNameID, modifiedKeyID KeyNameID, action func(id uuid.UUID, conflict Sentinel, i *item) error) (*item, error) {
	return GetItem[item](ctx, cm, id, keyID, modifiedKeyID, cm == nil, action)
}
