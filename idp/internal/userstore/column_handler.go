package userstore

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
)

type listColumnsParams struct {
	pagination.QueryParams
}

func validateAndPopulateColumnFields(
	ctx context.Context,
	dtm *storage.DataTypeManager,
	c *userstore.Column,
	s *storage.Storage,
) error {
	dataType, err := dtm.GetDataTypeByResourceID(c.DataType)
	if err != nil {
		return ucerr.Wrap(err)
	}

	c.DataType = userstore.ResourceID{ID: dataType.ID, Name: dataType.Name}
	c.Type = dataType.GetClientDataType()
	c.Constraints.Fields = dataType.GetColumnFields()

	var transformer *storage.Transformer

	if c.DefaultTransformer.ID.IsNil() {
		transformer, err = s.GetTransformerByName(ctx, c.DefaultTransformer.Name)
		if err != nil {
			return ucerr.Wrap(err)
		}
		c.DefaultTransformer.ID = transformer.ID
	} else {
		transformer, err = s.GetLatestTransformer(ctx, c.DefaultTransformer.ID)
		if err != nil {
			return ucerr.Wrap(err)
		}
		c.DefaultTransformer.Name = transformer.Name
	}

	transformType := transformer.TransformType.ToClient()
	if transformType == policy.TransformTypeTokenizeByReference ||
		transformType == policy.TransformTypeTokenizeByValue {
		if c.DefaultTokenAccessPolicy.Validate() != nil {
			return ucerr.Friendlyf(nil, "token resolution policy required")
		}

		aps, err := s.GetAccessPoliciesForResourceIDs(ctx, true, c.DefaultTokenAccessPolicy)
		if err != nil {
			return ucerr.Friendlyf(err, "invalid token resolution policy")
		}
		c.DefaultTokenAccessPolicy.ID = aps[0].ID
		c.DefaultTokenAccessPolicy.Name = aps[0].Name
	}
	if c.AccessPolicy.ID == uuid.Nil {
		c.AccessPolicy = userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID, Name: policy.AccessPolicyAllowAll.Name}
	} else {
		aps, err := s.GetAccessPoliciesForResourceIDs(ctx, true, c.AccessPolicy)
		if err != nil {
			return ucerr.Friendlyf(err, "invalid access policy")
		}
		c.AccessPolicy.ID = aps[0].ID
		c.AccessPolicy.Name = aps[0].Name
	}

	return nil
}

// OpenAPI Summary: List Columns
// OpenAPI Tags: Columns
// OpenAPI Description: This endpoint returns a paginated list of all columns in a tenant.
func (h *handler) listColumns(ctx context.Context, req listColumnsParams) (*idp.ListColumnsResponse, int, []auditlog.Entry, error) {

	s := storage.MustCreateStorage(ctx)
	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	pager, err := storage.NewColumnPaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	columns, respFields, err := s.ListColumnsPaginated(ctx, *pager)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	resp := make([]userstore.Column, 0, len(columns))
	for _, c := range columns {
		clientCol := c.ToClientModel()
		if err := validateAndPopulateColumnFields(ctx, dtm, &clientCol, s); err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		resp = append(resp, clientCol)
	}

	return &idp.ListColumnsResponse{Data: resp, ResponseFields: *respFields}, http.StatusOK, nil, nil
}

// OpenAPI Summary: Create Column
// OpenAPI Tags: Columns
// OpenAPI Description: This endpoint creates a new column.
func (h *handler) createColumn(ctx context.Context, req idp.CreateColumnRequest) (*userstore.Column, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	cm, err := storage.NewUserstoreColumnManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	// TODO: this is to ensure that the default transformer is set while we don't require it to be passed in the request, to make this a non-breaking change
	// If we decide to make this a breaking change, we can remove this block
	if req.Column.DefaultTransformer.ID.IsNil() && req.Column.DefaultTransformer.Name == "" {
		req.Column.DefaultTransformer = userstore.ResourceID{ID: policy.TransformerPassthrough.ID, Name: policy.TransformerPassthrough.Name}
	}

	// TODO: same as above, this is to ensure that the default access policy is set while we don't require it to be passed in the request, to make this a non-breaking change
	// If we decide to make this a breaking change, we can remove this block
	if req.Column.AccessPolicy.ID == uuid.Nil && req.Column.AccessPolicy.Name == "" {
		req.Column.AccessPolicy = userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID, Name: policy.AccessPolicyAllowAll.Name}
	}

	if err := validateAndPopulateColumnFields(ctx, dtm, &req.Column, s); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	var createdCol *userstore.Column
	code, err := cm.CreateColumnFromClient(ctx, &req.Column)
	if err == nil {
		createdCol, code, err = h.lookupColumn(ctx, req.Column.ID)
	}

	if err != nil {
		switch code {
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return createdCol, http.StatusCreated, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx),
		internal.AuditLogEventTypeCreateColumnConfig,
		auditlog.Payload{"ID": createdCol.ID, "Name": createdCol.Name, "Column": *createdCol},
	), nil
}

// OpenAPI Summary: Delete Column
// OpenAPI Tags: Columns
// OpenAPI Description: This endpoint deletes a column by ID. Note that deleting the column doesn't result in data deletion - it just results in the data being immediately unavailable. To delete the data stored in the column, you need to trigger the garbage collection process on the column which will remove the data after a configurable retention period.
func (h *handler) deleteColumn(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	// TODO: deleting a column could orphan policies and retention timeouts, or invalidate
	//       accessors that referred to this column
	// We need to decide what to do in this scenario
	// - Should we prevent the delete from happening?
	// - Should the UI show the affected accessors/policies?

	s := storage.MustCreateStorage(ctx)
	column, err := s.GetColumn(ctx, id) // check if column exists
	if err != nil {
		return uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	cm, err := storage.NewColumnManager(ctx, s, column.SQLShimDatabaseID)
	if err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	col := cm.GetColumnByID(id)
	code, err := cm.DeleteColumnFromClient(ctx, id)
	if err != nil {
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

	return http.StatusNoContent, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx),
		internal.AuditLogEventTypeDeleteColumnConfig,
		auditlog.Payload{"ID": id, "Name": col.Name},
	), nil
}

func (h *handler) lookupColumn(ctx context.Context, id uuid.UUID) (*userstore.Column, int, error) {
	s := storage.MustCreateStorage(ctx)
	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	c, err := s.GetColumn(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), ucerr.Wrap(err)
	}

	clientCol := c.ToClientModel()
	if err := validateAndPopulateColumnFields(ctx, dtm, &clientCol, s); err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	return &clientCol, http.StatusOK, nil
}

// OpenAPI Summary: Get Column
// OpenAPI Tags: Columns
// OpenAPI Description: This endpoint gets a column's configuration by ID.
func (h *handler) getColumn(ctx context.Context, id uuid.UUID, _ url.Values) (*userstore.Column, int, []auditlog.Entry, error) {
	c, code, err := h.lookupColumn(ctx, id)
	if err != nil {
		switch code {
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return c, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update Column
// OpenAPI Tags: Columns
// OpenAPI Description: This endpoint updates a specified column. Some column characteristics cannot be changed in an Update operation, once the column contains data. A column update may invalidate the accessors defined for it.
func (h *handler) updateColumn(ctx context.Context, id uuid.UUID, req idp.UpdateColumnRequest) (*userstore.Column, int, []auditlog.Entry, error) {
	if id != req.Column.ID {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "ID %v does not match updated column ID %v", id, req.Column.ID)
	}

	s := storage.MustCreateStorage(ctx)
	column, err := s.GetColumn(ctx, id) // check if column exists
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	cm, err := storage.NewColumnManager(ctx, s, column.SQLShimDatabaseID)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	// TODO: this is to ensure that the default transformer is set while we don't require it to be passed in the request, to make this a non-breaking change
	// If we decide to make this a breaking change, we can remove this block
	if req.Column.DefaultTransformer.ID.IsNil() && req.Column.DefaultTransformer.Name == "" {
		req.Column.DefaultTransformer = userstore.ResourceID{ID: policy.TransformerPassthrough.ID, Name: policy.TransformerPassthrough.Name}
	}

	// TODO: same as above, this is to ensure that the default access policy is set while we don't require it to be passed in the request, to make this a non-breaking change
	// If we decide to make this a breaking change, we can remove this block
	if req.Column.AccessPolicy.ID == uuid.Nil && req.Column.AccessPolicy.Name == "" {
		req.Column.AccessPolicy = userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID, Name: policy.AccessPolicyAllowAll.Name}
	}

	if err := validateAndPopulateColumnFields(ctx, dtm, &req.Column, s); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	var updatedCol *userstore.Column
	code, err := cm.UpdateColumnFromClient(ctx, &req.Column)
	if err == nil {
		updatedCol, code, err = h.lookupColumn(ctx, id)
	}

	if err != nil {
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

	return updatedCol, http.StatusOK, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx),
		internal.AuditLogEventTypeUpdateColumnConfig,
		auditlog.Payload{"ID": updatedCol.ID, "Name": updatedCol.Name, "Column": *updatedCol},
	), nil
}
