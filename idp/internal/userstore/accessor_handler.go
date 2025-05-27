package userstore

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
)

func ridSorter(items []userstore.ResourceID) {
	sort.Slice(items, func(i, j int) bool {
		left, right := items[i], items[j]
		if !left.ID.IsNil() || !right.ID.IsNil() {
			for char := range left.ID {
				if left.ID[char] != right.ID[char] {
					return left.ID[char] < right.ID[char]
				}
			}
			return false
		}
		return strings.Compare(left.Name, right.Name) < 0
	})
}

func validateAndPopulateAccessorFields(
	ctx context.Context,
	s *storage.Storage,
	accessors []*userstore.Accessor,
	validateSelectorConfig bool,
	errorOnMissing bool,
) error {
	apRIDs := set.New(ridSorter)
	for _, a := range accessors {
		apRIDs.Insert(a.AccessPolicy)
		for _, c := range a.Columns {
			if c.TokenAccessPolicy.Validate() == nil {
				apRIDs.Insert(c.TokenAccessPolicy)
			}
		}
	}

	aps, err := s.GetAccessPoliciesForResourceIDs(ctx, errorOnMissing, apRIDs.Items()...)
	if err != nil {
		return ucerr.Wrap(err)
	}

	apIDMap := map[uuid.UUID]storage.AccessPolicy{}
	apNameMap := map[string]storage.AccessPolicy{}
	for _, a := range aps {
		apIDMap[a.ID] = a
		apNameMap[a.Name] = a
	}

	transformerRIDs := []userstore.ResourceID{}
	for _, a := range accessors {
		for _, c := range a.Columns {
			if c.Transformer.ID != uuid.Nil || c.Transformer.Name != "" {
				transformerRIDs = append(transformerRIDs, c.Transformer)
			}
		}
	}

	transformerMap, err := storage.GetTransformerMapForResourceIDs(ctx, s, errorOnMissing, transformerRIDs...)
	if err != nil {
		return ucerr.Wrap(err)
	}

	purposeIDs := set.New(ridSorter)
	for _, a := range accessors {
		for _, p := range a.Purposes {
			purposeIDs.Insert(p)
		}
	}

	purposes, err := s.GetPurposesForResourceIDs(ctx, errorOnMissing, purposeIDs.Items()...)
	if err != nil {
		return ucerr.Wrap(err)
	}

	purposeIDMap := map[uuid.UUID]storage.Purpose{}
	purposeNameMap := map[string]storage.Purpose{}
	for _, p := range purposes {
		purposeIDMap[p.ID] = p
		purposeNameMap[p.Name] = p
	}

	nonUserstoreAccessorIDs := set.NewUUIDSet()

	for _, a := range accessors {
		if ap, ok := apIDMap[a.AccessPolicy.ID]; ok {
			if a.AccessPolicy.Name == "" {
				a.AccessPolicy.Name = ap.Name
			} else if a.AccessPolicy.Name != ap.Name {
				return ucerr.Errorf("access policy name %s does not match ID %s", a.AccessPolicy.Name, a.AccessPolicy.ID)
			}
		} else if ap, ok := apNameMap[a.AccessPolicy.Name]; ok {
			if a.AccessPolicy.ID.IsNil() {
				a.AccessPolicy.ID = ap.ID
			} else if a.AccessPolicy.ID != ap.ID {
				return ucerr.Errorf("access policy ID %s does not match name %s", a.AccessPolicy.ID, a.AccessPolicy.Name)
			}
		}

		for i := range a.Columns {
			if a.Columns[i].Column.ID != uuid.Nil {
				if col, err := s.GetColumn(ctx, a.Columns[i].Column.ID); err != nil {
					if errorOnMissing {
						return ucerr.Friendlyf(nil, "invalid column ID: %s", a.Columns[i].Column.ID)
					}
				} else {
					if a.Columns[i].Column.Name == "" {
						a.Columns[i].Column.Name = col.Name
					} else if !strings.EqualFold(a.Columns[i].Column.Name, col.Name) {
						return ucerr.Errorf("column name %s does not match ID %s", a.Columns[i].Column.Name, a.Columns[i].Column.ID)
					}

					if !col.IsUserstoreColumn() {
						nonUserstoreAccessorIDs.Insert(a.ID)
					}
				}
			} else {
				if col, err := s.GetUserColumnByName(ctx, a.Columns[i].Column.Name); err != nil {
					if errorOnMissing {
						return ucerr.Friendlyf(nil, "invalid column name: %s", a.Columns[i].Column.Name)
					}
				} else {
					if a.Columns[i].Column.ID.IsNil() {
						a.Columns[i].Column.ID = col.ID
					} else if a.Columns[i].Column.ID != col.ID {
						return ucerr.Errorf("column ID %s does not match name %s", a.Columns[i].Column.ID, a.Columns[i].Column.Name)
					}

					if !col.IsUserstoreColumn() {
						nonUserstoreAccessorIDs.Insert(a.ID)
					}
				}
			}

			var tokenizingTransformer bool
			if a.Columns[i].Transformer.ID != uuid.Nil {
				if tf, err := transformerMap.ForID(a.Columns[i].Transformer.ID); err != nil {
					if errorOnMissing {
						return ucerr.Wrap(err)
					}
				} else {
					if a.Columns[i].Transformer.Name == "" {
						a.Columns[i].Transformer.Name = tf.Name
					} else if a.Columns[i].Transformer.Name != tf.Name {
						return ucerr.Errorf("transformer name %s does not match ID %s", a.Columns[i].Transformer.Name, a.Columns[i].Transformer.ID)
					}

					if storage.InternalTransformTypeFromClient(policy.TransformTypeTokenizeByValue) == tf.TransformType ||
						storage.InternalTransformTypeFromClient(policy.TransformTypeTokenizeByReference) == tf.TransformType {
						tokenizingTransformer = true
					}
				}
			} else if a.Columns[i].Transformer.Name != "" {
				if tf, err := transformerMap.ForName(a.Columns[i].Transformer.Name); err != nil {
					if errorOnMissing {
						return ucerr.Wrap(err)
					}
				} else {
					if a.Columns[i].Transformer.ID.IsNil() {
						a.Columns[i].Transformer.ID = tf.ID
					} else if a.Columns[i].Transformer.ID != tf.ID {
						return ucerr.Errorf("transformer ID %s does not match name %s", a.Columns[i].Transformer.ID, a.Columns[i].Transformer.Name)
					}

					if storage.InternalTransformTypeFromClient(policy.TransformTypeTokenizeByValue) == tf.TransformType ||
						storage.InternalTransformTypeFromClient(policy.TransformTypeTokenizeByReference) == tf.TransformType {
						tokenizingTransformer = true
					}
				}
			}

			if tokenizingTransformer {
				if a.Columns[i].TokenAccessPolicy.Validate() != nil {
					// TODO: remove this once SDKs are all passing token access policy as part of columns
					if a.TokenAccessPolicy.Validate() == nil {
						a.Columns[i].TokenAccessPolicy = a.TokenAccessPolicy
					} else {
						return ucerr.Errorf("token access policy is required for tokenizing transformer")
					}
				}

				if ap, ok := apIDMap[a.Columns[i].TokenAccessPolicy.ID]; ok {
					if a.Columns[i].TokenAccessPolicy.Name == "" {
						a.Columns[i].TokenAccessPolicy.Name = ap.Name
					} else if a.Columns[i].TokenAccessPolicy.Name != ap.Name {
						return ucerr.Errorf("token access policy name %s does not match ID %s", a.Columns[i].TokenAccessPolicy.Name, a.Columns[i].TokenAccessPolicy.ID)
					}
				} else if ap, ok := apNameMap[a.Columns[i].TokenAccessPolicy.Name]; ok {
					if a.Columns[i].TokenAccessPolicy.ID.IsNil() {
						a.Columns[i].TokenAccessPolicy.ID = ap.ID
					} else if a.Columns[i].TokenAccessPolicy.ID != ap.ID {
						return ucerr.Errorf("token access policy ID %s does not match name %s", a.Columns[i].TokenAccessPolicy.ID, a.Columns[i].TokenAccessPolicy.Name)
					}
				}
			}
		}

		for i := range a.Purposes {
			if a.Purposes[i].ID != uuid.Nil {
				if purpose, ok := purposeIDMap[a.Purposes[i].ID]; ok {
					if a.Purposes[i].Name == "" {
						a.Purposes[i].Name = purpose.Name
					} else if a.Purposes[i].Name != purpose.Name {
						return ucerr.Errorf("purpose name %s does not match ID %s", a.Purposes[i].Name, a.Purposes[i].ID)
					}
				}
			} else {
				if purpose, ok := purposeNameMap[a.Purposes[i].Name]; ok {
					if a.Purposes[i].ID.IsNil() {
						a.Purposes[i].ID = purpose.ID
					} else if a.Purposes[i].ID != purpose.ID {
						return ucerr.Errorf("purpose ID %s does not match name %s", a.Purposes[i].ID, a.Purposes[i].Name)
					}
				}
			}
		}
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

		for _, a := range accessors {
			if nonUserstoreAccessorIDs.Contains(a.ID) {
				continue
			}

			uscv := storage.NewUserSelectorConfigValidator(
				ctx,
				s,
				cm,
				dtm,
				column.DataLifeCycleStateFromClient(a.DataLifeCycleState),
			)
			if err := uscv.Validate(a.SelectorConfig); err != nil {
				return ucerr.Wrap(err)
			}
		}
	}

	return nil
}

// OpenAPI Summary: Create Accessor
// OpenAPI Tags: Accessors
// OpenAPI Description: This endpoint creates an accessor - a custom read API.
func (h *handler) createAccessor(ctx context.Context, req idp.CreateAccessorRequest) (*userstore.Accessor, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	mm := storage.NewMethodManager(ctx, s)

	// validate and populate missing fields in the accessor
	if err := validateAndPopulateAccessorFields(ctx, s, []*userstore.Accessor{&req.Accessor}, true, true); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	code, err := mm.CreateAccessorFromClient(ctx, &req.Accessor)
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

	h.createEventTypesForAccessor(ctx, req.Accessor.ID, req.Accessor.Version)

	return &req.Accessor, http.StatusCreated, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), internal.AuditLogEventTypeCreateAccessorConfig,
		auditlog.Payload{"ID": req.Accessor.ID, "Name": req.Accessor.Name, "Accessor": req.Accessor}), nil
}

// OpenAPI Summary: Delete Accessor
// OpenAPI Tags: Accessors
// OpenAPI Description: This endpoint deletes an accessor by ID.
func (h *handler) deleteAccessor(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	mm := storage.NewMethodManager(ctx, s)

	accessor, err := s.GetLatestAccessor(ctx, id)
	if err != nil {
		return uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	version, code, err := mm.DeleteAccessorFromClient(ctx, id)
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

	if err := h.deleteEventTypesForAccessor(ctx, id, version); err != nil {
		uclog.Errorf(ctx, "failed to delete event types for accessor: %v", err)
	}

	return http.StatusNoContent, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), internal.AuditLogEventTypeDeleteAccessorConfig,
		auditlog.Payload{"ID": id, "Name": accessor.Name}), nil
}

// GetAccessorParams is the request params for the Get Accessor API
type GetAccessorParams struct {
	Version *string `description:"Optional - if not specified, the latest accessor will be returned" query:"accessor_version"`
}

// OpenAPI Summary: Get Accessor
// OpenAPI Tags: Accessors
// OpenAPI Description: This endpoint gets an existing accessor's configuration by ID.
func (h *handler) getAccessor(ctx context.Context, id uuid.UUID, req GetAccessorParams) (*userstore.Accessor, int, []auditlog.Entry, error) {

	s := storage.MustCreateStorage(ctx)
	var ac *storage.Accessor
	var err error

	versioned := req.Version != nil && *req.Version != ""
	if versioned {
		version, err := strconv.Atoi(*req.Version)
		if err != nil {
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}

		ac, err = s.GetAccessorByVersion(ctx, id, version)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
	} else {
		ac, err = s.GetLatestAccessor(ctx, id)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
	}

	cAccessor := ac.ToClientModel()
	if err := validateAndPopulateAccessorFields(ctx, s, []*userstore.Accessor{&cAccessor}, false, !versioned); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &cAccessor, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update Accessor
// OpenAPI Tags: Accessors
// OpenAPI Description: This endpoint updates a specified accessor.
func (h *handler) updateAccessor(ctx context.Context, id uuid.UUID, req idp.UpdateAccessorRequest) (*userstore.Accessor, int, []auditlog.Entry, error) {

	if id != req.Accessor.ID {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "ID %v does not match updated accessor ID %v", id, req.Accessor.ID)
	}

	s := storage.MustCreateStorage(ctx)

	if err := validateAndPopulateAccessorFields(ctx, s, []*userstore.Accessor{&req.Accessor}, true, true); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	mm := storage.NewMethodManager(ctx, s)
	code, err := mm.UpdateAccessorFromClient(ctx, &req.Accessor)
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

	return &req.Accessor, http.StatusOK, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), internal.AuditLogEventTypeUpdateAccessorConfig,
		auditlog.Payload{"ID": req.Accessor.ID, "Name": req.Accessor.Name, "Accessor": req.Accessor}), nil
}

type listAccessorsParams struct {
	pagination.QueryParams
	Versioned *string `description:"Optional - if versioned is true, endpoint will return all versions of each accessor in the response. Otherwise, only the latest version of each accessor will be returned." query:"versioned"`
}

// OpenAPI Summary: List Accessors
// OpenAPI Tags: Accessors
// OpenAPI Description: This endpoint lists all accessors in a tenant.
func (h *handler) listAccessors(ctx context.Context, req listAccessorsParams) (*idp.ListAccessorsResponse, int, []auditlog.Entry, error) {

	s := storage.MustCreateStorage(ctx)

	var pagerOptions []pagination.Option

	pager, err := storage.NewAccessorPaginatorFromQuery(req, pagerOptions...)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	var accessors []storage.Accessor
	var respFields *pagination.ResponseFields

	versioned := req.Versioned != nil && *req.Versioned == "true"
	if versioned {
		accessors, respFields, err = s.ListAccessorsPaginated(ctx, *pager)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)

		}
	} else {
		accessors, respFields, err = s.GetLatestAccessors(ctx, *pager)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	var cAccessors []*userstore.Accessor
	for _, a := range accessors {
		cAccessor := a.ToClientModel()
		cAccessors = append(cAccessors, &cAccessor)
	}

	if err := validateAndPopulateAccessorFields(ctx, s, cAccessors, false, !versioned); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	var resp []userstore.Accessor
	for _, a := range cAccessors {
		resp = append(resp, *a)
	}

	return &idp.ListAccessorsResponse{
		Data:           resp,
		ResponseFields: *respFields,
	}, http.StatusOK, nil, nil
}
