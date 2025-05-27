package types

type provisionMode string

const (
	// OnlineProvisionMode is for when provisioning is initiated by an online event
	OnlineProvisionMode provisionMode = "online"

	// OfflineProvisionMode is for when provisioning is initiated by cmd/provision
	OfflineProvisionMode provisionMode = "offline"
)

// ProvisionMode is the global variable that stores the mode of provisioning
var ProvisionMode = OnlineProvisionMode

// DeepProvisioning indicates that relationships between system object should be validated and corrected
var DeepProvisioning = false

// UseBaselineSchema indicates that DBs should be migrated step-by-step instead of with schema speedup
// when true, this is much slower, but enables us to test e.g., PostgreSQL compatibility in CI
var UseBaselineSchema = false
