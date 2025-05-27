// NOTE: automatically generated file -- DO NOT EDIT

package acmestorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

// IsCertificateSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsCertificateSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM acme_certificates WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsCertificateSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetCertificate loads a Certificate by ID
func (s *Storage) GetCertificate(ctx context.Context, id uuid.UUID) (*Certificate, error) {
	const q = "SELECT id, updated, deleted, status, order_id, private_key, certificate, certificate_chain, not_after, created FROM acme_certificates WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj Certificate
	if err := s.db.GetContext(ctx, "GetCertificate", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "Certificate %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetCertificateSoftDeleted loads a Certificate by ID iff it's soft-deleted
func (s *Storage) GetCertificateSoftDeleted(ctx context.Context, id uuid.UUID) (*Certificate, error) {
	const q = "SELECT id, updated, deleted, status, order_id, private_key, certificate, certificate_chain, not_after, created FROM acme_certificates WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj Certificate
	if err := s.db.GetContext(ctx, "GetCertificateSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted Certificate %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetCertificatesForIDs loads multiple Certificate for a given list of IDs
func (s *Storage) GetCertificatesForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]Certificate, error) {
	items := make([]Certificate, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getCertificatesHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getCertificatesHelperForIDs loads multiple Certificate for a given list of IDs from the DB
func (s *Storage) getCertificatesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]Certificate, error) {
	const q = "SELECT id, updated, deleted, status, order_id, private_key, certificate, certificate_chain, not_after, created FROM acme_certificates WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []Certificate
	if err := s.db.SelectContextWithDirty(ctx, "GetCertificatesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested Certificates  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListCertificatesPaginated loads a paginated list of Certificates for the specified paginator settings
func (s *Storage) ListCertificatesPaginated(ctx context.Context, p pagination.Paginator) ([]Certificate, *pagination.ResponseFields, error) {
	return s.listInnerCertificatesPaginated(ctx, p, false)
}

// listInnerCertificatesPaginated loads a paginated list of Certificates for the specified paginator settings
func (s *Storage) listInnerCertificatesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]Certificate, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, status, order_id, private_key, certificate, certificate_chain, not_after, created FROM (SELECT id, updated, deleted, status, order_id, private_key, certificate, certificate_chain, not_after, created FROM acme_certificates WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []Certificate
	if err := s.db.SelectContext(ctx, "ListCertificatesPaginated", &objsDB, q, queryFields...); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	objs, respFields := pagination.ProcessResults(objsDB, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())
	if respFields.HasNext {
		if err := p.ValidateCursor(respFields.Next); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	if respFields.HasPrev {
		if err := p.ValidateCursor(respFields.Prev); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	return objs, &respFields, nil
}

// SaveCertificate saves a Certificate
func (s *Storage) SaveCertificate(ctx context.Context, obj *Certificate) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerCertificate(ctx, obj))
}

// SaveCertificate saves a Certificate
func (s *Storage) saveInnerCertificate(ctx context.Context, obj *Certificate) error {
	const q = "INSERT INTO acme_certificates (id, updated, deleted, status, order_id, private_key, certificate, certificate_chain, not_after) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, status = $3, order_id = $4, private_key = $5, certificate = $6, certificate_chain = $7, not_after = $8 WHERE (acme_certificates.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveCertificate", obj, q, obj.ID, obj.Deleted, obj.Status, obj.OrderID, obj.PrivateKey, obj.Certificate, obj.CertificateChain, obj.NotAfter); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "Certificate %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteCertificate soft-deletes a Certificate which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteCertificate(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerCertificate(ctx, objID, false))
}

// deleteInnerCertificate soft-deletes a Certificate which is currently alive
func (s *Storage) deleteInnerCertificate(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE acme_certificates SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteCertificate", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting Certificate %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "Certificate %v not found", objID)
	}
	return nil
}
