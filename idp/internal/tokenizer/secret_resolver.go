package tokenizer

import (
	"context"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/ucerr"
)

type policySecretResolver struct {
	s *storage.Storage
}

func newPolicySecretResolver(s *storage.Storage) *policySecretResolver {
	return &policySecretResolver{s: s}
}

func (r *policySecretResolver) ResolveSecret(ctx context.Context, name string) (string, error) {
	secret, err := r.s.GetSecretByName(ctx, name)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	val, err := secret.Value.Resolve(ctx)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	return val, nil
}
