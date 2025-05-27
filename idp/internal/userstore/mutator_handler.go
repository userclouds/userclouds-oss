package userstore

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
)

func getNormalizerRIDs(columns []userstore.ColumnInputConfig) ([]userstore.ResourceID, error) {
	var normalizerRIDs []userstore.ResourceID
	var validatorRIDs []userstore.ResourceID

	for _, c := range columns {
		if err := c.Normalizer.Validate(); err == nil {
			normalizerRIDs = append(normalizerRIDs, c.Normalizer)
		}
		if err := c.Validator.Validate(); err == nil {
			validatorRIDs = append(validatorRIDs, c.Validator)
		}
	}

	if len(validatorRIDs) == len(columns) {
		return validatorRIDs, nil
	}

	if len(normalizerRIDs) == len(columns) {
		return normalizerRIDs, nil
	}

	return nil, ucerr.Friendlyf(nil, "mutator must specify a valid normalizer or validator for each column")
}

func validateAndPopulateMutatorFields(
	ctx context.Context,
	s *storage.Storage,
	mutator *userstore.Mutator,
	validateSelectorConfig bool,
) error {
	ap, err := s.GetAccessPoliciesForResourceIDs(ctx, true, mutator.AccessPolicy)
	if err != nil {
		return ucerr.Friendlyf(err, "invalid access policy: %v", mutator.AccessPolicy)
	}
	mutator.AccessPolicy.ID = ap[0].ID
	mutator.AccessPolicy.Name = ap[0].Name

	cm, err := storage.NewUserstoreColumnManager(ctx, s)
	if err != nil {
		return ucerr.Wrap(err)
	}

	normalizerRIDs, err := getNormalizerRIDs(mutator.Columns)
	if err != nil {
		return ucerr.Wrap(err)
	}

	normalizerMap, err := storage.GetTransformerMapForResourceIDs(ctx, s, true, normalizerRIDs...)
	if err != nil {
		return ucerr.Wrap(err)
	}

	for i, mc := range mutator.Columns {
		var col *storage.Column
		if !mc.Column.ID.IsNil() {
			col = cm.GetColumnByID(mc.Column.ID)
			if col == nil {
				return ucerr.Friendlyf(nil, "invalid column ID: %s", mc.Column.ID)
			}

			if mc.Column.Name == "" {
				mutator.Columns[i].Column.Name = col.Name
			} else if mc.Column.Name != col.Name {
				return ucerr.Errorf("column name %s does not match ID %s", mc.Column.Name, mc.Column.ID)
			}
		} else {
			col = cm.GetUserColumnByName(mc.Column.Name)
			if col == nil {
				return ucerr.Friendlyf(nil, "invalid column name: %s", mc.Column.Name)
			}
			mutator.Columns[i].Column.ID = col.ID
		}

		if col.Attributes.System {
			return ucerr.Friendlyf(nil, "mutators cannot modify system column %s", col.Name)
		}

		if !mc.Normalizer.ID.IsNil() {
			n, err := normalizerMap.ForID(mc.Normalizer.ID)
			if err != nil {
				return ucerr.Wrap(err)
			}
			mutator.Columns[i].Normalizer.Name = n.Name
		} else if mc.Normalizer.Name != "" {
			n, err := normalizerMap.ForName(mc.Normalizer.Name)
			if err != nil {
				return ucerr.Wrap(err)
			}
			mutator.Columns[i].Normalizer.ID = n.ID
		} else if !mc.Validator.ID.IsNil() {
			n, err := normalizerMap.ForID(mc.Validator.ID)
			if err != nil {
				return ucerr.Wrap(err)
			}
			mutator.Columns[i].Normalizer.ID = n.ID
			mutator.Columns[i].Normalizer.Name = n.Name
		} else {
			n, err := normalizerMap.ForName(mc.Validator.Name)
			if err != nil {
				return ucerr.Wrap(err)
			}
			mutator.Columns[i].Normalizer.ID = n.ID
			mutator.Columns[i].Normalizer.Name = n.Name
		}
		mutator.Columns[i].Validator.ID = mutator.Columns[i].Normalizer.ID
		mutator.Columns[i].Validator.Name = mutator.Columns[i].Normalizer.Name
	}

	if validateSelectorConfig {
		cm, err := storage.NewUserstoreColumnManager(ctx, s)
		if err != nil {
			return ucerr.Wrap(err)
		}

		dtm, err := storage.NewDataTypeManager(ctx, s)
		if err != nil {
			return ucerr.Wrap(err)
		}

		uscv := storage.NewUserSelectorConfigValidator(
			ctx,
			s,
			cm,
			dtm,
			column.DataLifeCycleStateLive,
		)
		if err := uscv.Validate(mutator.SelectorConfig); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

// OpenAPI Summary: Create Mutator
// OpenAPI Tags: Mutators
// OpenAPI Description: This endpoint creates a mutator.
func (h *handler) createMutator(ctx context.Context, req idp.CreateMutatorRequest) (*userstore.Mutator, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)

	// validate the mutator
	if err := validateAndPopulateMutatorFields(ctx, s, &req.Mutator, true); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	mm := storage.NewMethodManager(ctx, s)
	code, err := mm.CreateMutatorFromClient(ctx, &req.Mutator)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	h.createEventTypesForMutator(ctx, req.Mutator.ID, req.Mutator.Version)
	return &req.Mutator, http.StatusCreated, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), internal.AuditLogEventTypeCreateMutatorConfig,
		auditlog.Payload{"ID": req.Mutator.ID, "Name": req.Mutator.Name, "Mutator": req.Mutator}), nil
}

// OpenAPI Summary: Delete Mutator
// OpenAPI Tags: Mutators
// OpenAPI Description: This endpoint deletes a mutator by ID.
func (h *handler) deleteMutator(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {

	s := storage.MustCreateStorage(ctx)
	mm := storage.NewMethodManager(ctx, s)

	mutator, err := s.GetLatestMutator(ctx, id)
	if err != nil {
		return uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	version, code, err := mm.DeleteMutatorFromClient(ctx, id)
	if err != nil {
		switch code {
		case http.StatusNotFound:
			return http.StatusNotFound, nil, ucerr.Wrap(err)
		case http.StatusBadRequest:
			return http.StatusBadRequest, nil, ucerr.Wrap(err)
		default:
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	if err := h.deleteEventTypesForMutator(ctx, id, version); err != nil {
		uclog.Errorf(ctx, "failed to delete event types for mutator: %v", err)
	}

	return http.StatusNoContent, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), internal.AuditLogEventTypeDeleteMutatorConfig,
		auditlog.Payload{"ID": id, "Name": mutator.Name}), nil
}

// GetMutatorParams is the request params for the Get Mutator API
type GetMutatorParams struct {
	Version *string `description:"Optional - if not specified, the latest mutator will be returned" query:"mutator_version"`
}

// OpenAPI Summary: Get Mutator
// OpenAPI Tags: Mutators
// OpenAPI Description: This endpoint gets a mutator by ID.
func (h *handler) getMutator(ctx context.Context, id uuid.UUID, req GetMutatorParams) (*userstore.Mutator, int, []auditlog.Entry, error) {

	s := storage.MustCreateStorage(ctx)
	var m *storage.Mutator
	var err error

	if req.Version != nil && *req.Version != "" {
		version, err := strconv.Atoi(*req.Version)
		if err != nil {
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}

		m, err = s.GetMutatorByVersion(ctx, id, version)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
	} else {
		m, err = s.GetLatestMutator(ctx, id)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
	}

	cMutator := m.ToClientModel()
	if err := validateAndPopulateMutatorFields(ctx, s, &cMutator, false); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &cMutator, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update Mutator
// OpenAPI Tags: Mutators
// OpenAPI Description: This endpoint updates a specified mutator.
func (h *handler) updateMutator(ctx context.Context, id uuid.UUID, req idp.UpdateMutatorRequest) (*userstore.Mutator, int, []auditlog.Entry, error) {
	if id != req.Mutator.ID {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "ID %v does not match updated mutator ID %v", id, req.Mutator.ID)
	}

	s := storage.MustCreateStorage(ctx)

	if err := validateAndPopulateMutatorFields(ctx, s, &req.Mutator, true); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	mm := storage.NewMethodManager(ctx, s)
	code, err := mm.UpdateMutatorFromClient(ctx, &req.Mutator)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return &req.Mutator, http.StatusOK, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), internal.AuditLogEventTypeUpdateMutatorConfig,
		auditlog.Payload{"ID": req.Mutator.ID, "Name": req.Mutator.Name, "Mutator": req.Mutator}), nil
}

type listMutatorsParams struct {
	pagination.QueryParams
	Versioned *string `description:"Optional - if versioned is true, endpoint will return all versions of each mutator in the response. Otherwise, only the latest version of each mutator will be returned." query:"versioned"`
}

// OpenAPI Summary: List Mutators
// OpenAPI Tags: Mutators
// OpenAPI Description: This endpoint lists all mutators in a tenant.
func (h *handler) listMutators(ctx context.Context, req listMutatorsParams) (*idp.ListMutatorsResponse, int, []auditlog.Entry, error) {

	s := storage.MustCreateStorage(ctx)
	var resp []userstore.Mutator

	pager, err := storage.NewMutatorPaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)

	}

	var mutators []storage.Mutator
	var respFields *pagination.ResponseFields

	if req.Versioned != nil && *req.Versioned == "true" {

		mutators, respFields, err = s.ListMutatorsPaginated(ctx, *pager)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	} else {
		mutators, respFields, err = s.GetLatestMutators(ctx, *pager)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	for _, mut := range mutators {
		cMutator := mut.ToClientModel()
		if err := validateAndPopulateMutatorFields(ctx, s, &cMutator, false); err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		resp = append(resp, cMutator)
	}

	return &idp.ListMutatorsResponse{
		Data:           resp,
		ResponseFields: *respFields,
	}, http.StatusOK, nil, nil
}
