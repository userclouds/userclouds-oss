// TODO: not sure this is the right place for this package, but it's a start
// putting it in worker seems shortsighted since we'll clearly want UI to expose it,
// and plex .well-known handler should really reference it for token validation
// but it doesn't really live in plex either? it is using tenantdb

package acmestorage

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// Storage handles objects for ACME lifecycle
type Storage struct {
	db *ucdb.DB
}

// New returns a new acme storage
func New(db *ucdb.DB) *Storage {
	return &Storage{
		db: db,
	}
}

// GetOrderByToken looks up an ACME order by the HTTP-01 challenge token
func (s *Storage) GetOrderByToken(ctx context.Context, token string) (*Order, error) {
	const q = `SELECT id, created, updated, deleted, url, host, token, status, challenge_url, tenant_url_id FROM acme_orders WHERE token=$1 AND deleted='0001-01-01 00:00:00';`

	var order Order
	if err := s.db.GetContext(ctx, "GetOrderByToken", &order, q, token); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &order, nil
}

// ListOrdersByTenantURLID returns all Orders for a given tenant URL
func (s *Storage) ListOrdersByTenantURLID(ctx context.Context, id uuid.UUID) ([]Order, error) {
	const q = `SELECT id, created, updated, deleted, url, host, token, status, challenge_url, tenant_url_id FROM acme_orders WHERE tenant_url_id=$1 AND deleted='0001-01-01 00:00:00';`

	var orders []Order
	if err := s.db.SelectContext(ctx, "ListOrdersByTenantURLID", &orders, q, id); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return orders, nil
}

// GetCertificateByOrderID looks up a Certificate by OrderID
func (s *Storage) GetCertificateByOrderID(ctx context.Context, id uuid.UUID) (*Certificate, error) {
	const q = `SELECT id, created, updated, deleted, status, private_key, certificate, certificate_chain, order_id, not_after FROM acme_certificates WHERE order_id=$1 AND deleted='0001-01-01 00:00:00';`

	var cert Certificate
	if err := s.db.GetContext(ctx, "GetCertificateByOrderID", &cert, q, id); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &cert, nil
}
