package acmestorage

import (
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
)

// Order represents an ACME order lifecycle
type Order struct {
	ucdb.BaseModel

	TenantURLID uuid.UUID `db:"tenant_url_id"` // which TenantURL object this is related to

	URL   string `db:"url"` // ACME order URL
	Host  string `db:"host"`
	Token string `db:"token"`

	Status OrderStatus `db:"status"`

	ChallengeURL string `db:"challenge_url"` // store the URL to the challenge to simplify dns-01 RespondToChallenge
}

//go:generate genpageable Order

//go:generate genvalidate Order

//go:generate genorm Order acme_orders tenantdb

// Certificate represents an ACME cert lifecycle
// JSON tags are here before we ever serialize to ensure PK isn't :)
type Certificate struct {
	ucdb.BaseModel

	Status CertificateStatus `db:"status" json:"status"`

	OrderID          uuid.UUID     `db:"order_id" json:"order_id"`
	PrivateKey       secret.String `db:"private_key" json:"-"`
	Certificate      string        `db:"certificate" json:"certificate"`
	CertificateChain string        `db:"certificate_chain" json:"certificate_chain"`

	NotAfter time.Time `db:"not_after" json:"not_after"`
}

//go:generate genpageable Certificate

//go:generate genvalidate Certificate

//go:generate genorm Certificate acme_certificates tenantdb
