package tokenizer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/tokenizer"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/multitenant"
)

func saveTenantPolicySecret(ctx context.Context, tenantID uuid.UUID, name, value string) (*secret.String, error) {
	sec, err := secret.NewString(ctx, "policySecret", fmt.Sprintf("%s/%s_%s", tenantID, name, crypto.MustRandomHex(6)), value)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return sec, nil
}

// OpenAPI Summary: Create Secret
// OpenAPI Tags: Secrets
// OpenAPI Description: This endpoint creates a new secret.
func (h handler) createSecret(ctx context.Context, req tokenizer.CreateSecretRequest) (*policy.Secret, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	ts := multitenant.MustGetTenantState(ctx)

	sc := &storage.Secret{
		BaseModel: ucdb.NewBase(),
		Name:      req.Secret.Name,
	}
	if req.Secret.ID != uuid.Nil {
		sc.ID = req.Secret.ID
	}
	sec, err := saveTenantPolicySecret(ctx, ts.ID, sc.Name, req.Secret.Value)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}
	sc.Value = *sec

	if err := s.SaveSecret(ctx, sc); err != nil {
		if errDel := sec.Delete(ctx); errDel != nil {
			uclog.Errorf(ctx, "failed to delete secret value: %v", errDel)
		}
		if ucdb.IsUniqueViolation(err) {
			return nil, http.StatusConflict, nil, ucerr.Friendlyf(nil, "secret with name '%s' already exists", sc.Name)
		}
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	ret := sc.ToClientModel()
	return &ret, http.StatusCreated, nil, nil
}

type listSecretsParams struct {
	pagination.QueryParams
}

// OpenAPI Summary: List Secrets
// OpenAPI Tags: Secrets
// OpenAPI Description: This endpoint lists all secrets.
func (h handler) listSecrets(ctx context.Context, req listSecretsParams) (*idp.ListSecretsResponse, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	pager, err := storage.NewSecretPaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	secrets, respFields, err := s.ListSecretsPaginated(ctx, *pager)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	data := make([]policy.Secret, 0, len(secrets))
	for _, secret := range secrets {
		data = append(data, secret.ToClientModel())
	}

	return &idp.ListSecretsResponse{
		Data:           data,
		ResponseFields: *respFields,
	}, http.StatusOK, nil, nil
}

// OpenAPI Summary: Delete Secret
// OpenAPI Tags: Secrets
// OpenAPI Description: This endpoint deletes a secret.
func (h handler) deleteSecret(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)

	sec, err := s.GetSecret(ctx, id)
	if err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}
	if err := sec.Value.Delete(ctx); err != nil {
		uclog.Errorf(ctx, "failed to delete secret value: %v", err)
	}

	if err := s.DeleteSecret(ctx, id); err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}
	return http.StatusNoContent, nil, nil
}
