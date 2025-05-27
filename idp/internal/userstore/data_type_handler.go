package userstore

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
)

type listDataTypesParams struct {
	pagination.QueryParams
}

func validateAndPopulateDataTypeFields(
	dtm *storage.DataTypeManager,
	dt *userstore.ColumnDataType,
) error {
	if len(dt.CompositeAttributes.Fields) == 0 {
		return nil
	}

	for i, field := range dt.CompositeAttributes.Fields {
		if err := field.DataType.Validate(); err != nil {
			return ucerr.Wrap(err)
		}

		fieldDataType, err := dtm.GetDataTypeByResourceID(field.DataType)
		if err != nil {
			return ucerr.Wrap(err)
		}
		field.DataType.ID = fieldDataType.ID
		field.DataType.Name = fieldDataType.Name
		dt.CompositeAttributes.Fields[i] = field
	}

	return nil
}

// OpenAPI Summary: List Data Types
// OpenAPI Tags: Data Types
// OpenAPI Description: This endpoint returns a paginated list of all data types in a tenant.
func (h *handler) listDataTypes(ctx context.Context, req listDataTypesParams) (*idp.ListDataTypesResponse, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	pager, err := column.NewDataTypePaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	dataTypes, respFields, err := s.ListDataTypesPaginated(ctx, *pager)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	resp := make([]userstore.ColumnDataType, 0, len(dataTypes))
	for _, dt := range dataTypes {
		cdt := dt.ToClient()
		if err := validateAndPopulateDataTypeFields(dtm, &cdt); err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		resp = append(resp, cdt)
	}

	return &idp.ListDataTypesResponse{Data: resp, ResponseFields: *respFields}, http.StatusOK, nil, nil
}

// OpenAPI Summary: Create Data Type
// OpenAPI Tags: Data Types
// OpenAPI Description: This endpoint creates a new data type.
func (h *handler) createDataType(ctx context.Context, req idp.CreateDataTypeRequest) (*userstore.ColumnDataType, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	if err := validateAndPopulateDataTypeFields(dtm, &req.DataType); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	code, err := dtm.CreateDataTypeFromClient(ctx, &req.DataType)
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

	return &req.DataType, http.StatusCreated, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx),
		internal.AuditLogEventTypeCreateDataType,
		auditlog.Payload{"ID": req.DataType.ID, "Name": req.DataType.Name, "DataType": req.DataType},
	), nil
}

// OpenAPI Summary: Delete Data Type
// OpenAPI Tags: Data Types
// OpenAPI Description: This endpoint deletes a data type by ID.
func (h *handler) deleteDataType(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	dt := dtm.GetDataTypeByID(id)
	code, err := dtm.DeleteDataType(ctx, id)
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
		internal.AuditLogEventTypeDeleteDataType,
		auditlog.Payload{"ID": id, "Name": dt.Name},
	), nil
}

// OpenAPI Summary: Get Data Type
// OpenAPI Tags: Data Types
// OpenAPI Description: This endpoint gets a data type's configuration by ID.
func (h *handler) getDataType(ctx context.Context, id uuid.UUID, _ url.Values) (*userstore.ColumnDataType, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	dt := dtm.GetDataTypeByID(id)
	if dt == nil {
		return nil, http.StatusNotFound, nil, ucerr.Friendlyf(nil, "DataType '%v' is unrecognized", id)
	}

	cdt := dt.ToClient()
	if err := validateAndPopulateDataTypeFields(dtm, &cdt); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &cdt, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update Data Type
// OpenAPI Tags: Data Types
// OpenAPI Description: This endpoint updates a specified data type.
func (h *handler) updateDataType(ctx context.Context, id uuid.UUID, req idp.UpdateDataTypeRequest) (*userstore.ColumnDataType, int, []auditlog.Entry, error) {
	if id != req.DataType.ID {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "ID %v does not match updated data type ID %v", id, req.DataType.ID)
	}

	s := storage.MustCreateStorage(ctx)
	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	if err := validateAndPopulateDataTypeFields(dtm, &req.DataType); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	code, err := dtm.UpdateDataTypeFromClient(ctx, &req.DataType)
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

	return &req.DataType, http.StatusOK, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx),
		internal.AuditLogEventTypeUpdateDataType,
		auditlog.Payload{"ID": req.DataType.ID, "Name": req.DataType.Name, "DataType": req.DataType},
	), nil
}
