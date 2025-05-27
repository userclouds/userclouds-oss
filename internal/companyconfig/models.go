package companyconfig

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz/ucauthz"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// CompanyType represents the different types of companies/customers in UserClouds
type CompanyType string

// CompanyType values
const (
	CompanyTypeProspect CompanyType = "prospect"
	CompanyTypeCustomer CompanyType = "customer"
	CompanyTypeInternal CompanyType = "internal"
	CompanyTypeDemo     CompanyType = "demo"
)

//go:generate genconstant CompanyType

// Company describes a single UserClouds company, with
// multiple tenants but a single IAM and billing structure.
type Company struct {
	ucdb.BaseModel

	Name string      `db:"name" json:"name" validate:"notempty"`
	Type CompanyType `db:"type" json:"type"`
}

//go:generate genvalidate Company

//go:generate genorm --cache --followerreads Company companies companyconfig

// NewCompany creates a new Company with the given name and type.
func NewCompany(name string, companyType CompanyType) Company {
	return Company{BaseModel: ucdb.NewBase(), Name: name, Type: companyType}
}

func (Company) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"name": pagination.StringKeyType,
	}
}

//go:generate genpageable Company

// TenantState defines the possible states of a tenant.
type TenantState string

// TenantState values
const (
	TenantStateUnknown           TenantState = ""
	TenantStateCreating          TenantState = "creating"
	TenantStateActive            TenantState = "active"
	TenantStateFailedToProvision TenantState = "failed_to_provision"
)

// IsFailed returns true if the tenant is in a failed state
func (ts TenantState) IsFailed() bool {
	return ts == TenantStateFailedToProvision
}

// IsActive returns true if the tenant is in an active state
func (ts TenantState) IsActive() bool {
	return ts == TenantStateActive
}

// Tenant describes a single, isolated UserClouds tenant.
// A single auth tenant may have multiple applications but is isolated
// from other tenants even within the same company.
type Tenant struct {
	ucdb.BaseModel

	Name      string    `db:"name" json:"name" yaml:"name" validate:"length:2,30"`
	CompanyID uuid.UUID `db:"company_id" json:"company_id" yaml:"company_id" validate:"notnil"`
	// Tenant URL is of the form:
	// http[s]://<tenantname>.<tenantsubdomain>.com[:port]
	TenantURL string `db:"tenant_url" json:"tenant_url" yaml:"tenant_url" validate:"notempty"`

	// UseOrganizations indicates whether this tenant should use orgs (normally yes for B2B, no for B2C)
	UseOrganizations bool `db:"use_organizations" json:"use_organizations" yaml:"use_organizations"`

	State TenantState `db:"state" json:"state" yaml:"state"`

	// this controls whether or not we try to sync users between IDPs, and exists primarily
	// on companyconfig.Tenant to prevent worker from connection storms to tenant DBs to check
	// this can be eliminated when worker is doing enough stuff to have stable connections to
	// enough tenant DBs that we don't care.
	SyncUsers bool `db:"sync_users" json:"sync_users" yaml:"sync_users"`
}

//go:generate genvalidate Tenant

// Validate implements Validateable
func (t Tenant) extraValidate() error {
	// NOTE: code to sanitize domains should already have made this lowercase.
	if t.TenantURL != strings.ToLower(t.TenantURL) {
		return ucerr.New("Tenant URL should be all lowercase")
	}

	// TODO: more specific validation of tenant URL?
	if _, err := url.Parse(t.TenantURL); err != nil {
		return ucerr.Errorf("failed to parse tenant URL '%s': %w", err)
	}

	return nil
}

//go:generate genorm --cache --followerreads Tenant tenants companyconfig

func (Tenant) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"company_id": pagination.UUIDKeyType,
		"name":       pagination.StringKeyType,
	}
}

//go:generate genpageable Tenant

// GetHostName returns the hostname of the tenant URL
func (t Tenant) GetHostName() string {
	if strings.HasPrefix(t.TenantURL, "https://") {
		return strings.TrimPrefix(t.TenantURL, "https://")
	}
	return strings.TrimPrefix(t.TenantURL, "http://")
}

// GetRegionalURL returns the URL for the tenant in the given region
func (t Tenant) GetRegionalURL(region region.MachineRegion, isEKS bool) string {
	return GetTenantRegionalURL(t.TenantURL, region, isEKS)
}

// GetTenantRegionalURL returns the a regional URL for the given tenant URL
func GetTenantRegionalURL(tenantURL string, region region.MachineRegion, isEKS bool) string {
	fmtstr := ".tenant-%s."
	if isEKS {
		fmtstr = ".tenant-%s-eks."
	}
	return strings.Replace(tenantURL, ".tenant.", fmt.Sprintf(fmtstr, region), 1)
}

// CanUseRegionWithTenantURL returns true if the tenant URL can be converted to a regional URL
func CanUseRegionWithTenantURL(tenantURL string) bool {
	return strings.Contains(tenantURL, ".tenant.")
}

// UserRegionDBConfigMap is a map of region to DB config, used for remote user regions
type UserRegionDBConfigMap map[region.DataRegion]ucdb.Config

//go:generate gendbjson UserRegionDBConfigMap

// TenantInternal is the DB model for the internal config for a tenant.
type TenantInternal struct {
	ucdb.BaseModel

	TenantDBConfig ucdb.Config     `db:"tenant_db_config" json:"tenant_db_config" yaml:"tenant_db_config"`
	LogConfig      TenantLogConfig `db:"log_config" json:"log_config" yaml:"log_config"`

	PrimaryUserRegion         region.DataRegion     `db:"primary_user_region" json:"primary_user_region" yaml:"primary_user_region"`
	RemoteUserRegionDBConfigs UserRegionDBConfigMap `db:"remote_user_region_db_configs" json:"remote_user_region_db_configs" yaml:"remote_user_region_db_configs"`

	ConnectOnStartup bool `db:"connect_on_startup" json:"connect_on_startup" yaml:"connect_on_startup"`
}

//go:generate genpageable TenantInternal

//go:generate genvalidate TenantInternal

//go:generate genorm --cache --followerreads TenantInternal tenants_internal companyconfig

// TenantURL describes an alternate URL for accessing the tenant's services
// This could be a region-specific URL like `https://contoso.tenant-aws-us-east-1.userclouds.com`
// or in the future (need to sort out SSL certs), a custom CNAME like `https://auth.contoso.com`
// NB: unlike TenantInternal and TenantPlex, TenantURL does not share an ID with Tenant
// because Tenant-to-TenantURL is one-to-many
type TenantURL struct {
	ucdb.BaseModel

	TenantID  uuid.UUID `db:"tenant_id" json:"tenant_id" validate:"notnil"`
	TenantURL string    `db:"tenant_url" json:"tenant_url" validate:"notempty"`

	Validated bool `db:"validated" json:"validated"` // have we checked you control this URL?
	System    bool `db:"system" json:"system"`       // is this a system-generated (eg regional) URL?
	Active    bool `db:"active" json:"active"`       // is this URL currently CNAMEd correctly to the tenant?

	DNSVerifier string `db:"dns_verifier" json:"dns_verifier"` // a random string to verify you control the DNS

	CertificateValidUntil time.Time `db:"certificate_valid_until" json:"certificate_valid_until"`
}

//go:generate genpageable TenantURL

//go:generate genvalidate TenantURL

//go:generate genorm TenantURL --cache --followerreads tenants_urls companyconfig

// InviteKeyType represents the different types of user invites in Console
type InviteKeyType int

//go:generate genconstant InviteKeyType

// InviteKeyType constants
const (
	InviteKeyTypeUnknown InviteKeyType = 0

	// InviteKeyTypeNewCompany represents an invite to a new/existing user to create an company, now deprecated
	// InviteKeyTypeNewCompany InviteKeyType = 1

	// InviteKeyTypeExistingCompany represents an invite to a new/existing user to join an existing company.
	InviteKeyTypeExistingCompany InviteKeyType = 2
)

// TenantRoles is a map of TenantID -> Role for a user invited to join a company
type TenantRoles map[uuid.UUID]string

//go:generate gendbjson TenantRoles

// Validate implements Validateable
func (tr *TenantRoles) Validate() error {
	for _, role := range *tr {
		if role != ucauthz.AdminRole && role != ucauthz.MemberRole {
			return ucerr.Friendlyf(nil, "invalid role: %s", role)
		}
	}
	return nil
}

// InviteKey stores state & privileges associated with invites sent from the Console.
type InviteKey struct {
	ucdb.BaseModel

	Type    InviteKeyType `db:"type"`
	Key     string        `db:"key" validate:"notempty"`
	Expires time.Time     `db:"expires"`
	Used    bool          `db:"used"`

	// If Type == InviteKeyTypeExistingCompany, this is the Company the user us being invited to join
	// and their role; unused otherwise.
	CompanyID   uuid.UUID   `db:"company_id"`
	Role        string      `db:"role"`
	TenantRoles TenantRoles `db:"tenant_roles"`

	InviteeEmail string `db:"invitee_email"`
	// When a user accepts the invite and either signs-up for a new account or logs in with an existing one,
	// the invite is "bound" to this ID. The Key is not actually "used" until the action is finalized, but the invite can't be re-bound.
	InviteeUserID uuid.UUID `db:"invitee_user_id"`
}

//go:generate genvalidate InviteKey

// Don't generate a getter because we want to get by Key, not ID
//go:generate genorm --noget InviteKey invite_keys companyconfig

func (ik InviteKey) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "expires,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"expires:%v,id:%v",
				ik.Expires.UnixMicro(),
				ik.ID,
			),
		)
	}
}

func (InviteKey) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"company_id": pagination.UUIDKeyType,
		"expires":    pagination.TimestampKeyType,
		"used":       pagination.BoolKeyType,
	}
}

//go:generate genpageable InviteKey

// Session represents a user's session in Console.
// TODO: this REALLY shouldn't be here but I don't have time to plumb in a console DB tonight
type Session struct {
	ucdb.BaseModel

	IDToken      string `db:"id_token"`
	AccessToken  string `db:"access_token"`
	RefreshToken string `db:"refresh_token"`
	State        string `db:"state"`

	ImpersonatorIDToken      string `db:"impersonator_id_token"`
	ImpersonatorAccessToken  string `db:"impersonator_access_token"`
	ImpersonatorRefreshToken string `db:"impersonator_refresh_token"`
}

//go:generate genpageable Session

//go:generate genvalidate Session

//go:generate genorm Session --cache --followerreads sessions companyconfig

// Certificate represents a cert used to support TLS for the SQL proxy
type Certificate struct {
	Certificate string        `json:"certificate" yaml:"certificate" validate:"notempty"`
	PrivateKey  secret.String `json:"private_key" yaml:"private_key"`
}

//go:generate genvalidate Certificate

// Certificates is a slice of Certificate
type Certificates []Certificate

//go:generate gendbjson Certificates

// SQLShimProxy represents a SQLShimProxy configuration
type SQLShimProxy struct {
	ucdb.BaseModel

	Host string `db:"host" json:"host" yaml:"host" validate:"notempty"`
	Port int    `db:"port" json:"port" yaml:"port" validate:"notzero"`

	Certificates Certificates `db:"certificates" json:"certificates" yaml:"certificates"`
	// PublicKey is derived from Certificates but expensive to generate on every connection
	PublicKey string `db:"public_key" json:"public_key" yaml:"public_key"` // TODO (sgarrity 7/24): remove this after migration is complete (and remove the default from the DB column

	TenantID   uuid.UUID `db:"tenant_id" json:"tenant_id" yaml:"tenant_id" validate:"notnil"`
	DatabaseID uuid.UUID `db:"database_id" json:"database_id" yaml:"database_id" validate:"notnil"`
}

//go:generate genpageable SQLShimProxy

//go:generate genvalidate SQLShimProxy

//go:generate genorm --cache --followerreads SQLShimProxy sqlshim_proxies companyconfig
