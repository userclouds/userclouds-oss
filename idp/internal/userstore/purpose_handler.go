package userstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/multitenant"
)

// OpenAPI Summary: Create Purpose
// OpenAPI Tags: Purposes
// OpenAPI Description: This endpoint creates a purpose.
func (h *handler) createPurpose(ctx context.Context, req idp.CreatePurposeRequest) (*userstore.Purpose, int, []auditlog.Entry, error) {
	if req.Purpose.IsSystem {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "the IsSystem attribute cannot be set by the client")
	}

	s := storage.MustCreateStorage(ctx)

	p, err := s.GetPurposeByName(ctx, req.Purpose.Name)
	if err == nil {
		return nil, http.StatusConflict, nil,
			ucerr.WrapWithFriendlyStructure(
				jsonclient.Error{StatusCode: http.StatusConflict},
				jsonclient.SDKStructuredError{
					Error:     "This purpose already exists",
					ID:        p.ID,
					Identical: true,
				},
			)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	purpose := storage.Purpose{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
		Name:                     req.Purpose.Name,
		Description:              req.Purpose.Description,
	}

	if req.Purpose.ID != uuid.Nil {
		purpose.ID = req.Purpose.ID
		p, err := s.GetPurpose(ctx, purpose.ID)
		if err == nil {
			return nil, http.StatusConflict, nil,
				ucerr.WrapWithFriendlyStructure(
					jsonclient.Error{StatusCode: http.StatusConflict},
					jsonclient.SDKStructuredError{
						Error: fmt.Sprintf("A purpose with ID %v already exists", p.ID),
						ID:    p.ID,
					},
				)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	} else {
		req.Purpose.ID = purpose.ID
	}

	if err := s.SavePurpose(ctx, &purpose); err != nil {
		return nil, uchttp.SQLWriteErrorMapper(err), nil, ucerr.Wrap(err)
	}

	created := purpose.ToClientModel()
	return &created, http.StatusCreated, nil, nil
}

// OpenAPI Summary: Get Purpose
// OpenAPI Tags: Purposes
// OpenAPI Description: This endpoint gets a purpose by ID.
func (h *handler) getPurpose(ctx context.Context, id uuid.UUID, _ url.Values) (*userstore.Purpose, int, []auditlog.Entry, error) {

	s := storage.MustCreateStorage(ctx)
	purpose, err := s.GetPurpose(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	p := purpose.ToClientModel()
	return &p, http.StatusOK, nil, nil
}

type listPurposesParams struct {
	pagination.QueryParams
}

// OpenAPI Summary: List Purposes
// OpenAPI Tags: Purposes
// OpenAPI Description: This endpoint returns a paginated list of all purposes in a tenant.
func (h *handler) listPurposes(ctx context.Context, req listPurposesParams) (*idp.ListPurposesResponse, int, []auditlog.Entry, error) {

	s := storage.MustCreateStorage(ctx)
	pager, err := storage.NewPurposePaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	purposes, respFields, err := s.ListPurposesPaginated(ctx, *pager)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	resp := []userstore.Purpose{}
	for _, p := range purposes {
		resp = append(resp, p.ToClientModel())
	}

	return &idp.ListPurposesResponse{
		Data:           resp,
		ResponseFields: *respFields,
	}, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update Purpose
// OpenAPI Tags: Purposes
// OpenAPI Description: This endpoint updates a specified purpose.
func (h *handler) updatePurpose(ctx context.Context, id uuid.UUID, req idp.UpdatePurposeRequest) (*userstore.Purpose, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	purpose, err := s.GetPurpose(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	if purpose.IsSystem {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "system purposes cannot be updated")
	}
	if purpose.IsSystem != req.Purpose.IsSystem {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "the IsSystem attribute cannot be changed")
	}

	if req.Purpose.Name != "" && req.Purpose.Name != purpose.Name {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "cannot change purpose name")
	}
	purpose.Description = req.Purpose.Description

	if err := s.SavePurpose(ctx, purpose); err != nil {
		return nil, uchttp.SQLWriteErrorMapper(err), nil, ucerr.Wrap(err)
	}

	p := purpose.ToClientModel()
	return &p, http.StatusOK, nil, nil
}

// OpenAPI Summary: Delete Purpose
// OpenAPI Tags: Purposes
// OpenAPI Description: This endpoint deletes a purpose by ID.
func (h *handler) deletePurpose(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	purpose, err := s.GetPurpose(ctx, id)
	if err != nil {
		return uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}
	if purpose.IsSystem {
		return http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "system purposes cannot be deleted")
	}

	// Check if any accessors reference this purpose
	pagerSource, err := storage.NewAccessorPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	for {
		accessors, respFields, err := s.ListAccessorsPaginated(ctx, *pagerSource)
		if err != nil {
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		for _, accessor := range accessors {
			if slices.Contains(accessor.PurposeIDs, id) {
				return http.StatusConflict, nil, ucerr.Friendlyf(nil, "Purpose with ID %s is still used by accessor %s", id, accessor.Name)
			}
		}

		if !pagerSource.AdvanceCursor(*respFields) {
			break
		}
	}

	// Delete any column value retention durations that reference this purpose
	if err := s.DeleteColumnValueRetentionDurationsByPurposeID(ctx, id); err != nil {
		uclog.Warningf(ctx, "Failed to delete ColumnValueRetentionDuration entries for purpose %v: %v", id, err)
	}

	if err := s.DeletePurpose(ctx, id); err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return http.StatusNoContent, nil, nil
}

// OpenAPI Summary: List Purposes for User
// OpenAPI Tags: Purposes
// OpenAPI Description: This endpoint lists all consented purposes for a specified user.
func (h *handler) getConsentedPurposesForUser(ctx context.Context, req idp.GetConsentedPurposesForUserRequest) (*idp.GetConsentedPurposesForUserResponse, int, []auditlog.Entry, error) {

	ts := multitenant.MustGetTenantState(ctx)
	s := storage.NewFromTenantState(ctx, ts)
	us := storage.NewUserMultiRegionStorage(ctx, ts.UserRegionDbMap, ts.ID)

	cm, err := storage.NewUserstoreColumnManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	user, _, _, err := us.GetUser(ctx, cm, dtm, req.UserID, false)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	columns := cm.GetColumns()

	reqColIDs := set.NewUUIDSet()
	reqColNames := set.NewStringSet()
	if req.Columns != nil {
		colIDs := set.NewUUIDSet()
		colNames := set.NewStringSet()
		for _, c := range columns {
			colIDs.Insert(c.ID)
			colNames.Insert(strings.ToLower(c.Name))
		}
		for _, c := range req.Columns {
			if !c.ID.IsNil() {
				if colIDs.Contains(c.ID) {
					reqColIDs.Insert(c.ID)
				} else {
					return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "Column with ID %v not found", c.ID)
				}
			}
			if c.Name != "" {
				if colNames.Contains(strings.ToLower(c.Name)) {
					reqColNames.Insert(strings.ToLower(c.Name))
				} else {
					return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "Column with name '%s' not found", c.Name)
				}
			}
		}
	}

	purposes, err := s.ListPurposesNonPaginated(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}
	purposeMap := map[uuid.UUID]storage.Purpose{}
	for _, p := range purposes {
		purposeMap[p.ID] = p
	}

	resp := []idp.ColumnConsentedPurposes{}
	for _, c := range columns {
		if req.Columns == nil || reqColIDs.Contains(c.ID) || reqColNames.Contains(strings.ToLower(c.Name)) {
			consentedPurposeSet := set.NewUUIDSet()
			for _, purposeIDs := range user.ProfileConsentedPurposeIDs[c.Name] {
				consentedPurposeSet.Insert(purposeIDs...)
			}

			consentedPurposes := []userstore.ResourceID{}
			for _, pID := range consentedPurposeSet.Items() {
				consentedPurposes = append(consentedPurposes, userstore.ResourceID{
					ID:   pID,
					Name: purposeMap[pID].Name,
				})
			}
			resp = append(resp, idp.ColumnConsentedPurposes{
				Column:            userstore.ResourceID{ID: c.ID, Name: c.Name},
				ConsentedPurposes: consentedPurposes,
			})
		}
	}
	return &idp.GetConsentedPurposesForUserResponse{Data: resp}, http.StatusOK, nil, nil
}
