package userstore

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/search"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
)

type listUserSearchIndicesParams struct {
	pagination.QueryParams
}

type userSearchIndexHandler struct {
	ctx context.Context
	s   *storage.Storage
	cm  *storage.ColumnManager
	sim *storage.SearchIndexManager
}

func newUserSearchIndexHandler(ctx context.Context) (*userSearchIndexHandler, error) {
	s := storage.MustCreateStorage(ctx)

	cm, err := storage.NewUserstoreColumnManager(ctx, s)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	sim, err := storage.NewSearchIndexManager(ctx, s)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &userSearchIndexHandler{
		ctx: ctx,
		s:   s,
		cm:  cm,
		sim: sim,
	}, nil
}

func (usih userSearchIndexHandler) populateUserSearchIndexFields(indices []*search.UserSearchIndex) error {
	for i, index := range indices {
		var accessors []search.UserSearchIndexAccessor
		for _, asi := range usih.sim.GetAccessorsByID(index.ID) {
			a, err := usih.s.GetLatestAccessor(usih.ctx, asi.ID)
			if err != nil {
				return ucerr.Wrap(err)
			}

			accessors = append(
				accessors,
				search.UserSearchIndexAccessor{
					Accessor:  userstore.ResourceID{ID: a.ID, Name: a.Name},
					QueryType: asi.QueryType,
				},
			)
		}
		indices[i].Accessors = accessors

		for j, cid := range index.Columns {
			if cid.ID.IsNil() {
				c := usih.cm.GetUserColumnByName(cid.Name)
				if c == nil {
					return ucerr.Friendlyf(nil, "index column '%v' is unrecognized", cid)
				}

				cid.ID = c.ID
			} else {
				c := usih.cm.GetColumnByID(cid.ID)
				if c == nil {
					return ucerr.Friendlyf(nil, "index column '%v' is unrecognized", cid)
				}

				if cid.Name != "" && !strings.EqualFold(cid.Name, c.Name) {
					return ucerr.Friendlyf(nil, "index column '%v' ID and Name do not match", cid)
				}

				cid.Name = c.Name
			}

			indices[i].Columns[j] = cid
		}
	}

	return nil
}

// OpenAPI Summary: List User Search Indexes
// OpenAPI Tags: User Search Indices
// OpenAPI Description: This endpoint returns a paginated list of all user search indices in a tenant.
func (h *handler) listUserSearchIndexes(
	ctx context.Context,
	req listUserSearchIndicesParams,
) (*idp.ListUserSearchIndicesResponse, int, []auditlog.Entry, error) {
	usih, err := newUserSearchIndexHandler(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	pager, err := storage.NewUserSearchIndexPaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	indices, respFields, err := usih.s.ListUserSearchIndexesPaginated(ctx, *pager)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	cusis := make([]*search.UserSearchIndex, 0, len(indices))
	for _, index := range indices {
		cusi := index.ToClientModel()
		cusis = append(cusis, &cusi)
	}

	if err := usih.populateUserSearchIndexFields(cusis); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	respData := make([]search.UserSearchIndex, 0, len(indices))
	for _, cusi := range cusis {
		respData = append(respData, *cusi)
	}

	return &idp.ListUserSearchIndicesResponse{
			Data:           respData,
			ResponseFields: *respFields,
		},
		http.StatusOK,
		nil,
		nil
}

// OpenAPI Summary: Create User Search Index
// OpenAPI Tags: User Search Indices
// OpenAPI Description: This endpoint creates a new user search index.
func (h *handler) createUserSearchIndex(
	ctx context.Context,
	req idp.CreateUserSearchIndexRequest,
) (*search.UserSearchIndex, int, []auditlog.Entry, error) {
	usih, err := newUserSearchIndexHandler(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	if err := usih.populateUserSearchIndexFields([]*search.UserSearchIndex{&req.Index}); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	createdIndex := storage.NewUserSearchIndexFromClient(req.Index)
	if code, err := usih.sim.CreateIndex(ctx, &createdIndex); err != nil {
		switch code {
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	clientIndex := createdIndex.ToClientModel()
	if err := usih.populateUserSearchIndexFields([]*search.UserSearchIndex{&clientIndex}); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &clientIndex,
		http.StatusCreated,
		auditlog.NewEntryArray(
			auth.GetAuditLogActor(ctx),
			internal.AuditLogEventTypeCreateUserSearchIndex,
			auditlog.Payload{"ID": clientIndex.ID, "Name": clientIndex.Name, "Index": clientIndex},
		),
		nil
}

// OpenAPI Summary: Delete User Search Index
// OpenAPI Tags: User Search Indices
// OpenAPI Description: This endpoint deletes a user search index by ID.
func (h *handler) deleteUserSearchIndex(
	ctx context.Context,
	id uuid.UUID,
	_ url.Values,
) (int, []auditlog.Entry, error) {
	usih, err := newUserSearchIndexHandler(ctx)
	if err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	usi := usih.sim.GetIndexByID(id)
	if code, err := usih.sim.DeleteIndex(ctx, id); err != nil {
		switch code {
		case http.StatusNotFound:
			return http.StatusNotFound, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusBadRequest:
			return http.StatusBadRequest, nil, ucerr.Wrap(err)
		default:
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return http.StatusNoContent,
		auditlog.NewEntryArray(
			auth.GetAuditLogActor(ctx),
			internal.AuditLogEventTypeDeleteUserSearchIndex,
			auditlog.Payload{"ID": id, "Name": usi.Name},
		),
		nil
}

// OpenAPI Summary: Get User Search Index
// OpenAPI Tags: User Search Indices
// OpenAPI Description: This endpoint gets a user search index's configuration by ID.
func (h *handler) getUserSearchIndex(
	ctx context.Context,
	id uuid.UUID,
	_ url.Values,
) (*search.UserSearchIndex, int, []auditlog.Entry, error) {
	usih, err := newUserSearchIndexHandler(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	usi := usih.sim.GetIndexByID(id)
	if usi == nil {
		return nil,
			http.StatusNotFound,
			nil,
			ucerr.Friendlyf(nil, "UserSearchIndex '%v' is unrecognized", id)
	}

	cusi := usi.ToClientModel()
	if err := usih.populateUserSearchIndexFields([]*search.UserSearchIndex{&cusi}); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &cusi, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update User Search Index
// OpenAPI Tags: User Search Indices
// OpenAPI Description: This endpoint updates a specified user search index.
func (h *handler) updateUserSearchIndex(
	ctx context.Context,
	id uuid.UUID,
	req idp.UpdateUserSearchIndexRequest,
) (*search.UserSearchIndex, int, []auditlog.Entry, error) {
	if id != req.Index.ID {
		return nil,
			http.StatusBadRequest,
			nil,
			ucerr.Friendlyf(
				nil,
				"ID %v does not match updated user search index ID %v",
				id,
				req.Index.ID,
			)
	}

	usih, err := newUserSearchIndexHandler(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	if err := usih.populateUserSearchIndexFields([]*search.UserSearchIndex{&req.Index}); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	updatedIndex := storage.NewUserSearchIndexFromClient(req.Index)
	if code, err := usih.sim.UpdateIndex(ctx, &updatedIndex); err != nil {
		switch code {
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	clientIndex := updatedIndex.ToClientModel()
	if err := usih.populateUserSearchIndexFields([]*search.UserSearchIndex{&clientIndex}); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &clientIndex,
		http.StatusOK,
		auditlog.NewEntryArray(
			auth.GetAuditLogActor(ctx),
			internal.AuditLogEventTypeUpdateUserSearchIndex,
			auditlog.Payload{"ID": clientIndex.ID, "Name": clientIndex.Name, "Index": clientIndex},
		),
		nil
}

func (h *handler) removeAccessorUserSearchIndex(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()

	var req idp.RemoveAccessorUserSearchIndexRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	usih, err := newUserSearchIndexHandler(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	code, err := usih.sim.RemoveAccessor(ctx, req.AccessorID)
	if err != nil {
		switch code {
		case http.StatusNotFound:
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		default:
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		}
		return
	}

	if code == http.StatusOK {
		auditlog.PostMultipleAsync(
			ctx,
			auditlog.NewEntryArray(
				auth.GetAuditLogActor(ctx),
				internal.AuditLogEventTypeRemoveAccessorUserSearchIndex,
				auditlog.Payload{"AccessorID": req.AccessorID},
			),
		)
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(code))
}

func (h *handler) setAccessorUserSearchIndex(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()

	var req idp.SetAccessorUserSearchIndexRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	usih, err := newUserSearchIndexHandler(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	code, err := usih.sim.SetAccessor(ctx, req.AccessorID, req.IndexID, req.QueryType)
	if err != nil {
		switch code {
		case http.StatusNotFound:
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		default:
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		}
		return
	}

	if code == http.StatusOK {
		auditlog.PostMultipleAsync(
			ctx,
			auditlog.NewEntryArray(
				auth.GetAuditLogActor(ctx),
				internal.AuditLogEventTypeSetAccessorUserSearchIndex,
				auditlog.Payload{
					"AccessorID": req.AccessorID,
					"IndexID":    req.IndexID,
					"QueryType":  req.QueryType,
				},
			),
		)
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(code))
}
