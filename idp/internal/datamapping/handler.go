package datamapping

// TODO: better package name to not conflict with idp/userstore?

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/datamapping"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/datahub"
	"userclouds.com/internal/security"
	logServerClient "userclouds.com/logserver/client"
)

type handler struct {
	logServerClient *logServerClient.Client
}

// NewHandler returns a new http.Handler for the userstore service.
func NewHandler(m2mAuth jsonclient.Option, consoleTenantInfo companyconfig.TenantInfo) (http.Handler, error) {
	h := &handler{}
	// Create log server client for cross service calls
	lsc, err := logServerClient.NewClientForTenantAuth(consoleTenantInfo.TenantURL, consoleTenantInfo.TenantID, m2mAuth, security.PassXForwardedFor())
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	h.logServerClient = lsc

	hb := builder.NewHandlerBuilder()
	handlerBuilder(hb, h)
	return hb.Build(), nil
}

//go:generate genhandler /userstore/datamapping collection,DataSource,h.newRoleBasedAuthorizer(),/datasource collection,DataSourceElement,h.newRoleBasedAuthorizer(),/element

func (h *handler) newRoleBasedAuthorizer() uchttp.CollectionAuthorizer {
	return &uchttp.MethodAuthorizer{
		GetAllF: func(r *http.Request) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), false))
		},
		GetOneF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), false))
		},
		PostF: func(r *http.Request) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), true))
		},
		PutF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), true))
		},
		DeleteF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), true))
		},
		DeleteAllF: func(r *http.Request) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), true))
		},
		NestedF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), false))
		},
	}
}

func (h *handler) ensureTenantMember(ctx context.Context, adminOnly bool) error {
	return nil // TODO: figure out how to do this w/o calling authz service
}

func (h *handler) populateDataSourceElements(ctx context.Context, s *storage.Storage, ds *datamapping.DataSource) error {
	var schema datahub.Schema
	var numRows int
	var err error
	if ds.Type == datamapping.DataSourceTypePostgres {
		schema, numRows, err = datahub.ExtractSchemaFromPostgres(ctx, ds.Config.Host, ds.Config.Port, ds.Config.Database, ds.Config.Username, ds.Config.Password)
		if err != nil {
			return ucerr.Wrap(err)
		}
	} else if ds.Type == datamapping.DataSourceTypeRedshift {
		schema, numRows, err = datahub.ExtractSchemaFromRedshift(ctx, ds.Config.Host, ds.Config.Port, ds.Config.Database, ds.Config.Username, ds.Config.Password)
		if err != nil {
			return ucerr.Wrap(err)
		}
	} else {
		dsElement := datamapping.DataSourceElement{
			BaseModel:    ucdb.NewBase(),
			DataSourceID: ds.ID,
			Path:         ds.Name,
			Type:         string(ds.Type),
		}
		if err := s.SaveDataSourceElement(ctx, &dsElement); err != nil {
			return ucerr.Wrap(err)
		}

		return nil
	}

	for tableName, columns := range schema {
		for _, column := range columns {
			typeParts := strings.Split(column.GeneralType, ".")
			typeName := strings.ToLower(strings.TrimSuffix(typeParts[len(typeParts)-1], "Type"))
			dsElement := datamapping.DataSourceElement{
				BaseModel:    ucdb.NewBase(),
				DataSourceID: ds.ID,
				Path:         fmt.Sprintf("%s.%s", tableName, column.Name),
				Type:         typeName,
			}
			if err := s.SaveDataSourceElement(ctx, &dsElement); err != nil {
				return ucerr.Wrap(err)
			}
		}
	}

	ds.Metadata["info"] = map[string]any{
		"tables":  len(schema),
		"rows":    numRows,
		"updated": ds.Updated,
	}
	if err := s.SaveDataSource(ctx, ds); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// OpenAPI Summary: Create Data Source
// OpenAPI Tags: Data Sources
// OpenAPI Description: This endpoint creates a Data Source.
func (h *handler) createDataSource(ctx context.Context, req datamapping.CreateDataSourceRequest) (*datamapping.DataSource, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	if err := s.SaveDataSource(ctx, &req.DataSource); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	go func() {
		if err := h.populateDataSourceElements(context.Background(), s, &req.DataSource); err != nil {
			uclog.Errorf(ctx, "Error populating data source elements: %s", err)
		}
	}()

	return &req.DataSource, http.StatusCreated, nil, nil
}

// OpenAPI Summary: Delete Data Source
// OpenAPI Tags: Data Sources
// OpenAPI Description: This endpoint deletes a Data Source by ID.
func (h *handler) deleteDataSource(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)

	elementIDs, err := s.GetDataSourceElementIDsForSourceID(ctx, id)
	if err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	for _, elementID := range elementIDs {
		if err := s.DeleteDataSourceElement(ctx, elementID); err != nil {
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	if err := s.DeleteDataSource(ctx, id); err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return http.StatusNoContent, nil, nil
}

// OpenAPI Summary: Get Data Source
// OpenAPI Tags: Data Sources
// OpenAPI Description: This endpoint gets a Data Source by ID.
func (h *handler) getDataSource(ctx context.Context, id uuid.UUID, _ url.Values) (*datamapping.DataSource, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	ds, err := s.GetDataSource(ctx, id)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return ds, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update Data Source
// OpenAPI Tags: Data Sources
// OpenAPI Description: This endpoint updates a specified Data Source.
func (h *handler) updateDataSource(ctx context.Context, id uuid.UUID, req datamapping.UpdateDataSourceRequest) (*datamapping.DataSource, int, []auditlog.Entry, error) {
	if id != req.DataSource.ID {
		return nil, http.StatusBadRequest, nil, ucerr.Errorf("id in path does not match id in body")
	}

	s := storage.MustCreateStorage(ctx)
	if err := s.SaveDataSource(ctx, &req.DataSource); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &req.DataSource, http.StatusOK, nil, nil
}

type listDataSourcesParams struct {
	pagination.QueryParams
}

// OpenAPI Summary: List Data Sources
// OpenAPI Tags: Data Sources
// OpenAPI Description: This endpoint lists all Data Sources in a tenant.
func (h *handler) listDataSources(ctx context.Context, req listDataSourcesParams) (*datamapping.ListDataSourcesResponse, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)

	pager, err := datamapping.NewDataSourcePaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	dss, respFields, err := s.ListDataSourcesPaginated(ctx, *pager)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &datamapping.ListDataSourcesResponse{Data: dss, ResponseFields: *respFields}, http.StatusOK, nil, nil

}

// OpenAPI Summary: Create Data Source
// OpenAPI Tags: Data Sources
// OpenAPI Description: This endpoint creates a Data Source.
func (h *handler) createDataSourceElement(ctx context.Context, req datamapping.CreateDataSourceElementRequest) (*datamapping.DataSourceElement, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	if err := s.SaveDataSourceElement(ctx, &req.DataSourceElement); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &req.DataSourceElement, http.StatusCreated, nil, nil
}

// OpenAPI Summary: Delete Data Source
// OpenAPI Tags: Data Sources
// OpenAPI Description: This endpoint deletes a Data Source by ID.
func (h *handler) deleteDataSourceElement(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	if err := s.DeleteDataSourceElement(ctx, id); err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return http.StatusNoContent, nil, nil
}

// OpenAPI Summary: Get Data Source
// OpenAPI Tags: Data Sources
// OpenAPI Description: This endpoint gets a Data Source by ID.
func (h *handler) getDataSourceElement(ctx context.Context, id uuid.UUID, _ url.Values) (*datamapping.DataSourceElement, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	ds, err := s.GetDataSourceElement(ctx, id)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return ds, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update Data Source Element
// OpenAPI Tags: Data Sources
// OpenAPI Description: This endpoint updates a specified Data Source.
func (h *handler) updateDataSourceElement(ctx context.Context, id uuid.UUID, req datamapping.UpdateDataSourceElementRequest) (*datamapping.DataSourceElement, int, []auditlog.Entry, error) {
	if id != req.DataSourceElement.ID {
		return nil, http.StatusBadRequest, nil, ucerr.Errorf("id in path does not match id in body")
	}

	s := storage.MustCreateStorage(ctx)
	if err := s.SaveDataSourceElement(ctx, &req.DataSourceElement); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &req.DataSourceElement, http.StatusOK, nil, nil
}

type listDataSourceElementsParams struct {
	pagination.QueryParams
}

// OpenAPI Summary: List Data Source Elements
// OpenAPI Tags: Data Sources
// OpenAPI Description: This endpoint lists all Data Sources in a tenant.
func (h *handler) listDataSourceElements(ctx context.Context, req listDataSourceElementsParams) (*datamapping.ListDataSourceElementsResponse, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)

	pager, err := datamapping.NewDataSourceElementPaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	dss, respFields, err := s.ListDataSourceElementsPaginated(ctx, *pager)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &datamapping.ListDataSourceElementsResponse{Data: dss, ResponseFields: *respFields}, http.StatusOK, nil, nil

}
