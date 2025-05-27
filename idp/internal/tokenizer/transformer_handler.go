package tokenizer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp"
	"userclouds.com/idp/internal"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/tokenizer"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
)

type listTransformersParams struct {
	pagination.QueryParams
	Name      *string `description:"Optional - allows filtering by transformer name" query:"transformer_name"`
	Version   *string `description:"Optional - allows filtering by transformer version" query:"transformer_version"`
	Versioned *string `description:"Optional - set versioned to be true to include all prior versions of transformers in response" query:"versioned"`
}

func validateAndPopulateTransformerFields(
	dtm *storage.DataTypeManager,
	tfmr *policy.Transformer,
) error {
	inputDataType, err := dtm.GetDataTypeByResourceID(tfmr.InputDataType)
	if err != nil {
		return ucerr.Wrap(err)
	}

	tfmr.InputDataType = userstore.ResourceID{ID: inputDataType.ID, Name: inputDataType.Name}
	tfmr.InputType = inputDataType.GetClientDataType()
	tfmr.InputConstraints = inputDataType.GetTransformerConstraints()

	outputDataType, err := dtm.GetDataTypeByResourceID(tfmr.OutputDataType)
	if err != nil {
		return ucerr.Wrap(err)
	}

	tfmr.OutputDataType = userstore.ResourceID{ID: outputDataType.ID, Name: outputDataType.Name}
	tfmr.OutputType = outputDataType.GetClientDataType()
	tfmr.OutputConstraints = outputDataType.GetTransformerConstraints()

	return nil
}

// OpenAPI Summary: List Transformers
// OpenAPI Tags: Transformers
// OpenAPI Description: This endpoint returns a paginated list of all transformers in a tenant. The list can be filtered to only include transformers with a specific name.
func (h handler) listTransformers(ctx context.Context, req listTransformersParams) (*idp.ListTransformersResponse, int, []auditlog.Entry, error) {
	var clientTfs []policy.Transformer

	s := storage.MustCreateStorage(ctx)
	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	if req.Name != nil {
		tf, err := s.GetTransformerByName(ctx, *req.Name)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}

		if req.Version != nil {
			versionInt, err := strconv.Atoi(*req.Version)
			if err != nil {
				return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(err, "Invalid transformer version specified")
			}
			tf, err = s.GetTransformerByVersion(ctx, tf.ID, versionInt)
			if err != nil {
				return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
			}
		}

		clientTf := tf.ToClientModel()
		if err := validateAndPopulateTransformerFields(dtm, &clientTf); err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		clientTfs = append(clientTfs, clientTf)
		return &idp.ListTransformersResponse{Data: clientTfs}, http.StatusOK, nil, nil
	}

	pager, err := storage.NewTransformerPaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	var responseFields pagination.ResponseFields

	if req.Versioned != nil && *req.Versioned == "true" {
		tfs, respFields, err := s.ListTransformersPaginated(ctx, *pager)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		for _, tf := range tfs {
			clientTf := tf.ToClientModel()
			if err := validateAndPopulateTransformerFields(dtm, &clientTf); err != nil {
				return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
			}
			clientTfs = append(clientTfs, clientTf)
		}

		responseFields = *respFields
	} else {
		tfs, respFields, err := s.GetLatestTransformersPaginated(ctx, *pager)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		for _, tf := range tfs {
			clientTf := tf.ToClientModel()
			if err := validateAndPopulateTransformerFields(dtm, &clientTf); err != nil {
				return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
			}
			clientTfs = append(clientTfs, clientTf)
		}

		responseFields = *respFields
	}

	return &idp.ListTransformersResponse{Data: clientTfs, ResponseFields: responseFields}, http.StatusOK, nil, nil
}

// GetTransformerParams are the parameters for the Get Transformer API
type GetTransformerParams struct {
	Version *string `description:"Optional - if not specified, the latest policy will be returned" query:"transformer_version"`
}

// OpenAPI Summary: Get Transformer
// OpenAPI Tags: Transformers
// OpenAPI Description: This endpoint gets a transformer by ID.
func (h handler) getTransformer(ctx context.Context, id uuid.UUID, req GetTransformerParams) (*policy.Transformer, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	var tf *storage.Transformer
	if req.Version != nil {
		version, err := strconv.Atoi(*req.Version)
		if err != nil {
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}

		tf, err = s.GetTransformerByVersion(ctx, id, version)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
	} else {
		tf, err = s.GetLatestTransformer(ctx, id)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
	}

	clientTf := tf.ToClientModel()
	if err := validateAndPopulateTransformerFields(dtm, &clientTf); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &clientTf, http.StatusOK, nil, nil
}

// OpenAPI Summary: Create Transformer
// OpenAPI Tags: Transformers
// OpenAPI Description: This endpoint creates a new transformer.
func (h handler) createTransformer(ctx context.Context, req tokenizer.CreateTransformerRequest) (*policy.Transformer, int, []auditlog.Entry, error) {
	if req.Transformer.IsSystem {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "the IsSystem attribute cannot be set by the client")
	}

	s := storage.MustCreateStorage(ctx)
	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	clientTf := req.Transformer
	if err := validateAndPopulateTransformerFields(dtm, &clientTf); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	if err := clientTf.Validate(); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	tfIDIsNil := clientTf.ID.IsNil()
	if tfIDIsNil {
		clientTf.ID = uuid.Must(uuid.NewV4())
	}

	tf := storage.NewTransformerFromClient(clientTf)
	if err := tf.Validate(); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	// if a matching policy can be loaded, we shouldn't be using create
	// TODO: would be nice to unify this with loadTransformer but the logic is slightly different
	if !tfIDIsNil {
		if foundTf, err := s.GetLatestTransformer(ctx, clientTf.ID); !errors.Is(err, sql.ErrNoRows) {

			if err != nil {
				return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
			}

			if foundTf.EqualsIgnoringNilID(tf) {
				clientTf = foundTf.ToClientModel()
				return &clientTf, http.StatusConflict, nil,
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error:     "This transformer already exists",
							ID:        clientTf.ID,
							Identical: true,
						},
					)
			}

			return nil, http.StatusConflict, nil,
				ucerr.WrapWithFriendlyStructure(
					jsonclient.Error{StatusCode: http.StatusConflict},
					jsonclient.SDKStructuredError{
						Error: "A transformer already exists with the same ID",
						ID:    clientTf.ID,
					},
				)
		}
	}

	if foundTf, err := s.GetTransformerByName(ctx, tf.Name); !errors.Is(err, sql.ErrNoRows) {
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		if tfIDIsNil {
			tf.ID = uuid.Nil
		}
		if foundTf.EqualsIgnoringNilID(tf) {
			clientTf = foundTf.ToClientModel()
			return &clientTf, http.StatusConflict, nil,
				ucerr.WrapWithFriendlyStructure(
					jsonclient.Error{StatusCode: http.StatusConflict},
					jsonclient.SDKStructuredError{
						Error:     "This transformer already exists",
						ID:        clientTf.ID,
						Identical: true,
					},
				)
		}

		return nil, http.StatusConflict, nil,
			ucerr.WrapWithFriendlyStructure(
				jsonclient.Error{StatusCode: http.StatusConflict},
				jsonclient.SDKStructuredError{
					Error: fmt.Sprintf(`A transformer with name '%s' already exists`, tf.Name),
					ID:    foundTf.ID,
				},
			)
	}

	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}
	if err := SaveTransformerWithAuthz(ctx, s, authzClient, tf); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	h.createEventTypesForTransformer(ctx, tf.ID, tf.Version)
	return &clientTf, http.StatusCreated, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx),
		internal.AuditLogEventTypeCreateTransformer,
		auditlog.Payload{"ID": tf.ID, "Name": tf.Name},
	), nil
}

// OpenAPI Summary: Update Transformer
// OpenAPI Tags: Transformers
// OpenAPI Description: This endpoint updates a transformer.
func (h handler) updateTransformer(ctx context.Context, id uuid.UUID, req tokenizer.UpdateTransformerRequest) (*policy.Transformer, int, []auditlog.Entry, error) {

	if req.Transformer.ID != uuid.Nil && req.Transformer.ID != id {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "URL ID doesn't match request body ID")
	}

	s := storage.MustCreateStorage(ctx)
	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	clientTf := req.Transformer
	if err := validateAndPopulateTransformerFields(dtm, &clientTf); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	currentTf, err := s.GetLatestTransformer(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	if currentTf.IsSystem {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "system policies cannot be updated")
	}

	if currentTf.IsSystem != clientTf.IsSystem {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "the IsSystem attribute cannot be updated by the client")
	}

	updateNames := false
	if currentTf.Name != clientTf.Name {
		// verify that the new name isn't already in use
		tf, err := s.GetAccessPolicyByName(ctx, clientTf.Name)
		if tf != nil {
			if id != tf.ID {
				return nil, http.StatusConflict, nil,
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error: fmt.Sprintf(`A transformer with the name '%s' already exists`, clientTf.Name),
							ID:    tf.ID,
						},
					)
			}

			// if the IDs are the same, this is just a case change
			// to the name, which we should allow
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		updateNames = true
	}

	if currentTf.Version != clientTf.Version {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "Transformer version in request does not match current version")
	}

	tf := storage.NewTransformerFromClient(clientTf)
	if err := tf.Validate(); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	currentTf.Name = clientTf.Name
	currentTf.Description = clientTf.Description
	currentTf.InputDataTypeID = clientTf.InputDataType.ID
	currentTf.OutputDataTypeID = clientTf.OutputDataType.ID
	currentTf.ReuseExistingToken = clientTf.ReuseExistingToken
	currentTf.TransformType = storage.InternalTransformTypeFromClient(clientTf.TransformType)
	currentTf.TagIDs = clientTf.TagIDs
	currentTf.Function = clientTf.Function
	currentTf.Parameters = clientTf.Parameters
	currentTf.Version++

	if err := s.SaveTransformer(ctx, currentTf); err != nil {
		return nil, uchttp.SQLWriteErrorMapper(err), nil, ucerr.Wrap(err)
	}

	if updateNames {
		// change the transformer name for all versions
		tfs, err := s.GetAllTransformerVersions(ctx, id)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		for _, tf := range tfs {
			if tf.Name != clientTf.Name {
				tf.Name = clientTf.Name
				if err := s.PriorVersionSaveTransformer(ctx, &tf); err != nil {
					return nil, uchttp.SQLWriteErrorMapper(err), nil, ucerr.Wrap(err)
				}
			}
		}
	}

	entries := auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), internal.AuditLogEventTypeUpdateTransformer,
		auditlog.Payload{"ID": id, "Name": currentTf.Name, "Version": currentTf.Version})

	retTf := currentTf.ToClientModel()
	if err := validateAndPopulateTransformerFields(dtm, &retTf); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	return &retTf, http.StatusOK, entries, nil
}

// OpenAPI Summary: Delete Transformer
// OpenAPI Tags: Transformers
// OpenAPI Description: This endpoint deletes a transformer by ID.
func (h handler) deleteTransformer(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	tf, err := s.GetLatestTransformer(ctx, id)
	if err != nil {
		return uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	if tf.IsSystem {
		return http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "system transformers cannot be deleted")
	}

	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}
	if err := DeleteTransformerWithAuthz(ctx, s, authzClient, tf); err != nil {
		if errors.Is(err, storage.ErrStillInUse) {
			return http.StatusConflict, nil, ucerr.Wrap(err)
		}
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	if err := h.deleteEventsForTransformer(ctx, id, tf.Version); err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return http.StatusNoContent, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx),
		internal.AuditLogEventTypeDeleteTransformer,
		auditlog.Payload{"ID": id, "Name": tf.Name}), nil
}

// OpenAPI Summary: Test Transformer
// OpenAPI Tags: Transformers
// OpenAPI Description: This endpoint tests a specified transformer. It receives test data and returns the transformed data.
func (h handler) testTransformer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := storage.MustCreateStorage(ctx)

	var req tokenizer.TestTransformerRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.Transformer.ID.IsNil() {
		req.Transformer.ID = uuid.Must(uuid.NewV4())
	}

	if req.Transformer.Name == "" {
		req.Transformer.Name = "temp" // give the transformer a placeholder name so it doesn't fail validation
	}

	// for testing, we always want to transform
	req.Transformer.TransformType = policy.TransformTypeTransform

	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := validateAndPopulateTransformerFields(dtm, &req.Transformer); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	authZClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	t := storage.NewTransformerFromClient(req.Transformer)
	if err := t.Validate(); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	value, consoleOutput, err := ExecuteTransformer(ctx, s, authZClient, uuid.Nil, &t, req.Data, nil)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, tokenizer.TestTransformerResponse{Value: value, Debug: map[string]any{"console": consoleOutput}})
}

// ExecuteTransformer executes a transformer on the given data
func ExecuteTransformer(
	ctx context.Context,
	s *storage.Storage,
	authzClient *authz.Client,
	tokenAccessPolicyID uuid.UUID,
	transformer *storage.Transformer,
	data string,
	dataProvenance *policy.UserstoreDataProvenance,
) (result string, consoleOutput string, err error) {
	etp := ExecuteTransformerParameters{
		Transformer:         transformer,
		TokenAccessPolicyID: tokenAccessPolicyID,
		Data:                data,
		DataProvenance:      dataProvenance,
	}

	te := NewTransformerExecutor(s, authzClient)
	defer te.CleanupExecution()

	results, consoleOutput, err := te.Execute(ctx, etp)
	if err != nil {
		return "", "", ucerr.Wrap(err)
	}

	if len(results) != 1 {
		return "", "", ucerr.Errorf("ExecuteTransformers produced %d results for one request", len(results))
	}

	return results[0], consoleOutput, nil
}
