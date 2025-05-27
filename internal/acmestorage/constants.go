package acmestorage

// OrderStatus is a basic state machine for an ACME order
type OrderStatus string

// OrderStatus constants
const (
	OrderStatusNew       OrderStatus = "new"
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusValidated OrderStatus = "validated"
	OrderStatusReady     OrderStatus = "ready"
)

// CertificateStatus is a basic state machine for an ACME certificate
type CertificateStatus string

// CertificateStatus constants
const (
	CertificateStatusNew       CertificateStatus = "new"
	CertificateStatusRequested CertificateStatus = "requested"
	CertificateStatusReceived  CertificateStatus = "received"
	CertificateStatusUploaded  CertificateStatus = "uploaded"
)
