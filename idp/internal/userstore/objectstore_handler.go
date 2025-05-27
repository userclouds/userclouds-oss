package userstore

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/multitenant"
)

func saveObjectStoreSecret(ctx context.Context, secretAccessKey string, tenantID, dbID uuid.UUID) (*secret.String, error) {
	sec, err := secret.NewString(ctx, "s3proxy", fmt.Sprintf("obj_store_secret_%s_%s_%s", tenantID, dbID, crypto.MustRandomHex(6)), secretAccessKey)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return sec, nil
}

func validateAndPopulateObjectStoreFields(ctx context.Context,
	s *storage.Storage,
	objectStores []*userstore.ShimObjectStore,
) error {
	for i, objectStore := range objectStores {
		if objectStore.AccessPolicy.ID != uuid.Nil {
			ap, err := s.GetLatestAccessPolicy(ctx, objectStore.AccessPolicy.ID)
			if err != nil {
				return ucerr.Wrap(err)
			}
			if objectStore.AccessPolicy.Name != "" && objectStore.AccessPolicy.Name != ap.Name {
				return ucerr.Friendlyf(nil, "access policy name does not match")
			}
			objectStores[i].AccessPolicy.Name = ap.Name
		} else if objectStore.AccessPolicy.Name != "" {
			ap, err := s.GetAccessPolicyByName(ctx, objectStore.AccessPolicy.Name)
			if err != nil {
				return ucerr.Wrap(err)
			}
			objectStores[i].AccessPolicy.ID = ap.ID
		} else {
			return ucerr.Friendlyf(nil, "access policy ID or name is required")
		}
	}

	return nil
}

func (h *handler) listObjectStores(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := storage.MustCreateStorage(ctx)

	pager, err := storage.NewShimObjectStorePaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	objectStores, respFields, err := s.ListShimObjectStoresPaginated(ctx, *pager)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	clientObjectStores := make([]*userstore.ShimObjectStore, 0, len(objectStores))
	for _, objStore := range objectStores {
		clientObjectStore := objStore.ToClientModel()
		clientObjectStores = append(clientObjectStores, &clientObjectStore)
	}
	if err := validateAndPopulateObjectStoreFields(ctx, s, clientObjectStores); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp := make([]userstore.ShimObjectStore, 0, len(clientObjectStores))
	for _, objStore := range clientObjectStores {
		resp = append(resp, *objStore)
	}
	jsonapi.Marshal(w, &idp.ListObjectStoresResponse{Data: resp, ResponseFields: *respFields})
}

func (h *handler) getObjectStore(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	s := storage.MustCreateStorage(ctx)

	objectStore, err := s.GetShimObjectStore(ctx, id)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	clientObjectStore := objectStore.ToClientModel()
	if err := validateAndPopulateObjectStoreFields(ctx, s, []*userstore.ShimObjectStore{&clientObjectStore}); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, clientObjectStore)
}

func (h *handler) createObjectStore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := storage.MustCreateStorage(ctx)

	var req idp.CreateObjectStoreRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := validateAndPopulateObjectStoreFields(ctx, s, []*userstore.ShimObjectStore{&req.ObjectStore}); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	objStore := &storage.ShimObjectStore{
		BaseModel:      ucdb.NewBase(),
		Name:           req.ObjectStore.Name,
		Type:           storage.ObjectStoreType(req.ObjectStore.Type),
		Region:         req.ObjectStore.Region,
		AccessKeyID:    req.ObjectStore.AccessKeyID,
		RoleARN:        req.ObjectStore.RoleARN,
		AccessPolicyID: req.ObjectStore.AccessPolicy.ID,
	}
	if req.ObjectStore.ID != uuid.Nil {
		objStore.ID = req.ObjectStore.ID
	}

	if req.ObjectStore.SecretAccessKey != "" {
		ts := multitenant.MustGetTenantState(ctx)
		secretKey, err := saveObjectStoreSecret(ctx, req.ObjectStore.SecretAccessKey, ts.ID, objStore.ID)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		objStore.SecretAccessKey = *secretKey
	}

	if err := s.SaveShimObjectStore(ctx, objStore); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	clientObjectStore := objStore.ToClientModel()
	if err := validateAndPopulateObjectStoreFields(ctx, s, []*userstore.ShimObjectStore{&clientObjectStore}); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, clientObjectStore, jsonapi.Code(http.StatusCreated))
}

func (h *handler) updateObjectStore(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	s := storage.MustCreateStorage(ctx)

	var req idp.UpdateObjectStoreRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := validateAndPopulateObjectStoreFields(ctx, s, []*userstore.ShimObjectStore{&req.ObjectStore}); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	objStore, err := s.GetShimObjectStore(ctx, id)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	objStore.Name = req.ObjectStore.Name
	objStore.Type = storage.ObjectStoreType(req.ObjectStore.Type)
	objStore.Region = req.ObjectStore.Region
	objStore.AccessKeyID = req.ObjectStore.AccessKeyID
	objStore.RoleARN = req.ObjectStore.RoleARN
	objStore.AccessPolicyID = req.ObjectStore.AccessPolicy.ID

	oldSecretKey, err := objStore.SecretAccessKey.Resolve(ctx)
	if err != nil {
		uclog.Warningf(ctx, "failed to resolve secret access key for object store %s: %v", objStore.ID, err)
		oldSecretKey = ""
	}
	if req.ObjectStore.SecretAccessKey != "" && req.ObjectStore.SecretAccessKey != oldSecretKey {
		if err := objStore.SecretAccessKey.Delete(ctx); err != nil {
			uclog.Warningf(ctx, "failed to delete secret access key for object store %s: %v", objStore.ID, err)
		}
		ts := multitenant.MustGetTenantState(ctx)
		secretKey, err := saveObjectStoreSecret(ctx, req.ObjectStore.SecretAccessKey, ts.ID, objStore.ID)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		objStore.SecretAccessKey = *secretKey
	}

	if err := s.SaveShimObjectStore(ctx, objStore); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	clientObjectStore := objStore.ToClientModel()
	if err := validateAndPopulateObjectStoreFields(ctx, s, []*userstore.ShimObjectStore{&clientObjectStore}); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, clientObjectStore)
}

func (h *handler) deleteObjectStore(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	s := storage.MustCreateStorage(ctx)

	if err := s.DeleteShimObjectStore(ctx, id); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(http.StatusNoContent))
}
