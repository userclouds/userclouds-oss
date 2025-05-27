package tokenizer

import (
	"context"
	"errors"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	idpAuthz "userclouds.com/idp/authz"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/ucerr"
)

// SaveAccessPolicyWithAuthz saves the access policy to the DB and adds an AuthZ edge from the global Policy object to new access policy
func SaveAccessPolicyWithAuthz(ctx context.Context, s *storage.Storage, c *authz.Client, ap *storage.AccessPolicy) error {
	if err := s.SaveAccessPolicy(ctx, ap); err != nil {
		return ucerr.Wrap(err)
	}
	if err := AddAccessPolicyToAuthz(ctx, c, ap.ID, "" /* We could set alias = ap.Name if we wanted to force uniqueness */); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteAccessPolicyWithAuthz deletes the specific version of an access policy from the DB and removes an AuthZ edge from the global Policy object to the
// being deleted policy
func DeleteAccessPolicyWithAuthz(ctx context.Context, s *storage.Storage, c *authz.Client, ap *storage.AccessPolicy) error {
	aps, err := s.GetAllAccessPolicyVersions(ctx, ap.ID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if len(aps) == 1 && aps[0].Version == ap.Version {

		if err := s.CheckAccessPolicyUnused(ctx, ap.ID); err != nil {
			return ucerr.Wrap(err)
		}

		if err := RemoveAccessPolicyFromAuthz(ctx, c, ap.ID); err != nil {
			return ucerr.Wrap(err)
		}
	}

	if err := s.DeleteAccessPolicyByVersion(ctx, ap.ID, ap.Version); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// DeleteAllAccessPolicyVersionsWithAuthz deletes all versions of the access policy from the DB and removes an AuthZ edge from the global Policy object to the
// being deleted policy
func DeleteAllAccessPolicyVersionsWithAuthz(ctx context.Context, s *storage.Storage, c *authz.Client, ap *storage.AccessPolicy) error {

	if err := s.CheckAccessPolicyUnused(ctx, ap.ID); err != nil {
		return ucerr.Wrap(err)
	}

	if err := RemoveAccessPolicyFromAuthz(ctx, c, ap.ID); err != nil {
		return ucerr.Wrap(err)
	}

	if err := s.DeleteAllAccessPolicyVersions(ctx, ap.ID); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// SaveTransformerWithAuthz saves the transformer to the DB and adds an AuthZ edge from the global Policy object to new
// transformer
func SaveTransformerWithAuthz(ctx context.Context, s *storage.Storage, c *authz.Client, t storage.Transformer) error {
	if err := s.SaveTransformer(ctx, &t); err != nil {
		return ucerr.Wrap(err)
	}
	if err := AddTransformerToAuthz(ctx, c, t.ID, "" /* We could set alias = gp.Name if we wanted name uniqueness */); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteTransformerWithAuthz deletes the transformer from the DB and removes an AuthZ edge from the global Policy object to the
// being deleted policy
func DeleteTransformerWithAuthz(ctx context.Context, s *storage.Storage, c *authz.Client, transformer *storage.Transformer) error {
	if err := s.CheckTransformerUnused(ctx, transformer.ID); err != nil {
		return ucerr.Wrap(err)
	}

	if err := RemoveTransformerFromAuthz(ctx, c, transformer.ID); err != nil {
		return ucerr.Wrap(err)
	}

	if err := s.DeleteAllTransformerVersions(ctx, transformer.ID); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// AddAccessPolicyToAuthz adds an AuthZ edge from the global Policy object to the access policy
func AddAccessPolicyToAuthz(ctx context.Context, c *authz.Client, apID uuid.UUID, alias string) error {
	if c == nil {
		return nil
	}

	if _, err := c.CreateObject(ctx, apID, idpAuthz.PolicyAccessTypeID, alias, authz.OrganizationID(uuid.Nil)); err != nil {
		return ucerr.Wrap(err)
	}

	if _, err := c.CreateEdge(ctx, uuid.Must(uuid.NewV4()), idpAuthz.PoliciesObjectID, apID, idpAuthz.PolicyAccessExistsEdgeTypeID); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// RemoveAccessPolicyFromAuthz removes all AuthZ edges from the access policy
func RemoveAccessPolicyFromAuthz(ctx context.Context, c *authz.Client, apID uuid.UUID) error {
	if c == nil {
		return nil
	}

	if err := c.DeleteObject(ctx, apID); err != nil {
		if !errors.Is(err, authz.ErrObjectNotFound) {
			// If the authZ object (which implies edge as well) has already been deleted - continue
			return ucerr.Wrap(err)
		}
	}

	return nil
}

// AddTransformerToAuthz adds an AuthZ edge from the global Policy object to the transformer
func AddTransformerToAuthz(ctx context.Context, c *authz.Client, gpID uuid.UUID, alias string) error {
	if c == nil {
		return nil
	}

	if _, err := c.CreateObject(ctx, gpID, idpAuthz.PolicyTransformerTypeID, alias, authz.OrganizationID(uuid.Nil)); err != nil {
		return ucerr.Wrap(err)
	}

	if _, err := c.CreateEdge(ctx, uuid.Must(uuid.NewV4()), idpAuthz.PoliciesObjectID, gpID, idpAuthz.PolicyTransformerExistsEdgeTypeID); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// RemoveTransformerFromAuthz removes all AuthZ edges from the transformer
func RemoveTransformerFromAuthz(ctx context.Context, c *authz.Client, gpID uuid.UUID) error {
	if c == nil {
		return nil
	}

	if err := c.DeleteObject(ctx, gpID); err != nil {
		// If the authZ object (which implies edge as well) has already been deleted - continue
		if !errors.Is(err, authz.ErrObjectNotFound) {
			return ucerr.Wrap(err)
		}
	}

	return nil
}
