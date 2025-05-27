package storage

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"

	"userclouds.com/idp"
	"userclouds.com/idp/datamapping"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/infra/uctypes/uuidarray"
	"userclouds.com/internal/sqlshim"
)

// OIDCAuthn represents all 3rd party OIDC "subjects" (i.e. one row for every unique user+provider tuple,
// which OIDC calls "subject").
// OIDCAuthn - like other AuthN types - can be many-to-one with User (but usually 1:1 in practice).
type OIDCAuthn struct {
	ucdb.UserBaseModel

	Type          oidc.ProviderType `db:"type" json:"type"`
	OIDCIssuerURL string            `db:"oidc_issuer_url" json:"oidc_issuer_url"`
	OIDCSubject   string            `db:"oidc_sub" json:"oidc_sub" validate:"notempty"`
}

func (a *OIDCAuthn) extraValidate() error {
	if !a.Type.IsSupported() {
		return ucerr.Errorf("Type is unsupported: '%v'", a.Type)
	}
	if err := a.Type.ValidateIssuerURL(a.OIDCIssuerURL); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

//go:generate genpageable OIDCAuthn

//go:generate genvalidate OIDCAuthn

//go:generate genorm OIDCAuthn authns_social tenantdb

// PasswordAuthn represents a single type of credentials (u/p, social, etc) in our system
// NOTE: for u/p, using bcrypt which combines salt + hash in 1 string.
type PasswordAuthn struct {
	ucdb.UserBaseModel

	// TODO: we might want to rename these to be slightly more type-agnostic, eg identifier & token?
	Username string `db:"username" json:"username" validate:"notempty"`
	Password string `db:"password" json:"-" validate:"notempty"` // salted and hashed, but don't send it over the wire
}

//go:generate genpageable PasswordAuthn

//go:generate genvalidate PasswordAuthn

//go:generate genorm PasswordAuthn authns_password tenantdb

// UserMFAConfiguration represents the MFA configuration settings for a user
type UserMFAConfiguration struct {
	ucdb.BaseModel

	LastEvaluated time.Time        `db:"last_evaluated" json:"last_evaluated"`
	MFAChannels   oidc.MFAChannels `db:"mfa_channels" json:"mfa_channels"`
}

// MarkEvaluated will set the LastEvaluated time to now
func (umfac *UserMFAConfiguration) MarkEvaluated() {
	umfac.LastEvaluated = time.Now().UTC()
}

//go:generate genpageable UserMFAConfiguration
//go:generate genvalidate UserMFAConfiguration

//go:generate genorm UserMFAConfiguration user_mfa_configuration tenantdb

const defaultMFACode = "unissued"

// MFARequest represents a single auth request by a user
type MFARequest struct {
	ucdb.UserBaseModel

	// keep track of issued instead of expired so we can enforce different policies later
	Issued time.Time `db:"issued"`

	// use a string here in case we want more entropy later
	Code string `db:"code" validate:"notempty"`

	// the selected MFA channel id
	ChannelID uuid.UUID `db:"channel_id"`

	// contains the supported channel types for the auth request
	SupportedChannelTypes oidc.MFAChannelTypes `db:"supported_channel_types"`
}

// NewMFARequest creates a MFARequest with appropriate defaults for the user and supported channel types
func NewMFARequest(userID uuid.UUID, channelTypes oidc.MFAChannelTypeSet) *MFARequest {
	mr := &MFARequest{
		UserBaseModel:         ucdb.NewUserBase(userID),
		Code:                  defaultMFACode,
		ChannelID:             uuid.Nil,
		Issued:                time.Time{},
		SupportedChannelTypes: oidc.MFAChannelTypes{ChannelTypes: channelTypes},
	}
	return mr
}

// GetCode returns the channel id, code, and issue time if a code has been issued
func (r MFARequest) GetCode() (channelID uuid.UUID, code string, issued time.Time, err error) {
	if r.Code == defaultMFACode {
		return channelID, code, issued, ucerr.New("code has not been set")
	}

	return r.ChannelID, r.Code, r.Issued, nil
}

// SetCode sets the specified channel id and code, updating the issue time to now
func (r *MFARequest) SetCode(id uuid.UUID, code string) {
	r.Code = code
	r.ChannelID = id
	r.Issued = time.Now().UTC()
}

//go:generate genpageable MFARequest
//go:generate genvalidate MFARequest

//go:generate genorm MFARequest mfa_requests tenantdb

// Accessor represents a userstore accessor in our system
type Accessor struct {
	ucdb.SystemAttributeBaseModel

	Name                              string                       `db:"name" validate:"notempty"`
	Description                       string                       `db:"description"`
	Version                           int                          `db:"version"`
	DataLifeCycleState                column.DataLifeCycleState    `db:"data_life_cycle_state"`
	AccessPolicyID                    uuid.UUID                    `db:"access_policy_id" validate:"notnil"`
	ColumnIDs                         uuidarray.UUIDArray          `db:"column_ids" validate:"skip"`
	TransformerIDs                    uuidarray.UUIDArray          `db:"transformer_ids" validate:"skip"`
	TokenAccessPolicyIDs              uuidarray.UUIDArray          `db:"token_access_policy_ids" validate:"skip"`
	SelectorConfig                    userstore.UserSelectorConfig `db:"selector_config"`
	PurposeIDs                        uuidarray.UUIDArray          `db:"purpose_ids" validate:"skip"`
	IsAuditLogged                     bool                         `db:"is_audit_logged"`
	IsAutogenerated                   bool                         `db:"is_autogenerated"`
	AreColumnAccessPoliciesOverridden bool                         `db:"are_column_access_policies_overridden"`
	SearchColumnID                    uuid.UUID                    `db:"search_column_id" validate:"skip"`
	UseSearchIndex                    bool                         `db:"use_search_index"`
}

func (a Accessor) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "name,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"name:%v,id:%v",
				a.Name,
				a.ID,
			),
		)
	}
}

func (Accessor) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"column_ids":       pagination.UUIDArrayKeyType,
		"transformer_ids":  pagination.UUIDArrayKeyType,
		"purpose_ids":      pagination.UUIDArrayKeyType,
		"name":             pagination.StringKeyType,
		"created":          pagination.TimestampKeyType,
		"updated":          pagination.TimestampKeyType,
		"version":          pagination.IntKeyType,
		"is_autogenerated": pagination.BoolKeyType,
		"search_column_id": pagination.UUIDKeyType,
		"use_search_index": pagination.BoolKeyType,
	}
}

//go:generate genpageable Accessor

//go:generate genvalidate Accessor

//go:generate genorm --cache --followerreads --priorsave --versioned --nonpaginatedlist Accessor accessors tenantdb

// ToClientModel just translates from a storage.Accessor to a userstore.Accessor without populating missing fields
func (a Accessor) ToClientModel() userstore.Accessor {
	columns := make([]userstore.ColumnOutputConfig, len(a.ColumnIDs))

	for i, id := range a.ColumnIDs {
		columns[i].Column = userstore.ResourceID{
			ID: id,
		}
	}

	for i, id := range a.TransformerIDs {
		columns[i].Transformer = userstore.ResourceID{
			ID: id,
		}
	}

	// TODO: remove this once we are sure no client is referring to Accessor.TokenAccessPolicy anymore
	tokenAccessPolicy := userstore.ResourceID{}
	for i, id := range a.TokenAccessPolicyIDs {
		columns[i].TokenAccessPolicy = userstore.ResourceID{
			ID: id,
		}

		if id != uuid.Nil {
			// if there is any non-nil token access policy, we will use that as the stand-in token access policy for the accessor
			tokenAccessPolicy.ID = id
		}
	}

	purposes := make([]userstore.ResourceID, len(a.PurposeIDs))
	for i, id := range a.PurposeIDs {
		purposes[i] = userstore.ResourceID{
			ID: id,
		}
	}

	return userstore.Accessor{
		ID:                 a.ID,
		Name:               a.Name,
		Description:        a.Description,
		Version:            a.Version,
		DataLifeCycleState: a.DataLifeCycleState.ToClient(),
		Columns:            columns,
		AccessPolicy: userstore.ResourceID{
			ID: a.AccessPolicyID,
		},
		TokenAccessPolicy:                 tokenAccessPolicy,
		SelectorConfig:                    a.SelectorConfig,
		Purposes:                          purposes,
		IsSystem:                          a.IsSystem,
		IsAuditLogged:                     a.IsAuditLogged,
		IsAutogenerated:                   a.IsAutogenerated,
		AreColumnAccessPoliciesOverridden: a.AreColumnAccessPoliciesOverridden,
		UseSearchIndex:                    a.UseSearchIndex,
	}
}

// Equals checks if two columns have same fields outside of created/updated time
func (a Accessor) Equals(o *Accessor, versionCheck bool) bool {
	if len(a.ColumnIDs) != len(o.ColumnIDs) {
		return false
	}

	if len(a.TransformerIDs) != len(a.ColumnIDs) || len(o.TransformerIDs) != len(o.ColumnIDs) || len(a.TokenAccessPolicyIDs) != len(a.ColumnIDs) {
		// this should never happen if the db is consistent, but checking so that we don't
		// panic if the arrays are of different lengths
		return false
	}

	aTransformerMap := map[uuid.UUID]uuid.UUID{}
	aTokenAccessPolicyMap := map[uuid.UUID]uuid.UUID{}
	for i, id := range a.ColumnIDs {
		aTransformerMap[id] = a.TransformerIDs[i]
		aTokenAccessPolicyMap[id] = a.TokenAccessPolicyIDs[i]
	}

	oTransformerMap := map[uuid.UUID]uuid.UUID{}
	oTokenAccessPolicyMap := map[uuid.UUID]uuid.UUID{}
	for i, id := range o.ColumnIDs {
		oTransformerMap[id] = o.TransformerIDs[i]
		oTokenAccessPolicyMap[id] = o.TokenAccessPolicyIDs[i]
	}

	for k, v := range aTransformerMap {
		if oTransformerMap[k] != v {
			return false
		}
	}

	for k, v := range aTokenAccessPolicyMap {
		if oTokenAccessPolicyMap[k] != v {
			return false
		}
	}

	return a.ID == o.ID &&
		strings.EqualFold(a.Name, o.Name) &&
		a.Description == o.Description &&
		(a.Version == o.Version || !versionCheck) &&
		a.DataLifeCycleState == o.DataLifeCycleState &&
		a.AccessPolicyID == o.AccessPolicyID &&
		a.SelectorConfig == o.SelectorConfig &&
		set.NewUUIDSet(a.PurposeIDs...).Equal(set.NewUUIDSet(o.PurposeIDs...)) &&
		a.IsSystem == o.IsSystem &&
		a.IsAuditLogged == o.IsAuditLogged &&
		a.AreColumnAccessPoliciesOverridden == o.AreColumnAccessPoliciesOverridden &&
		a.SearchColumnID == o.SearchColumnID &&
		a.UseSearchIndex == o.UseSearchIndex
}

func (a Accessor) extraValidate() error {
	if len(a.ColumnIDs) == 0 {
		return ucerr.Friendlyf(nil, "Accessor.ColumnIDs cannot be empty")
	}

	uniqueColumnIDs := set.NewUUIDSet(a.ColumnIDs...)
	if uniqueColumnIDs.Size() != len(a.ColumnIDs) {
		return ucerr.Friendlyf(nil, "All column ids in Accessor.ColumnIDs must be unique")
	}

	if uniqueColumnIDs.Contains(uuid.Nil) {
		return ucerr.Friendlyf(nil, "All column ids in Accessor.ColumnIDs must be non-nil")
	}

	if len(a.ColumnIDs) != len(a.TransformerIDs) {
		return ucerr.Friendlyf(nil, "Accessor.ColumnIDs and Accessor.TransformerIDs must be the same length")
	}

	if len(a.ColumnIDs) != len(a.TokenAccessPolicyIDs) {
		return ucerr.Friendlyf(nil, "Accessor.ColumnIDs and Accessor.TokenAccessPolicyIDs must be the same length")
	}

	if a.UseSearchIndex {
		// TODO: currently we only index live values for a search index enabled column
		if a.DataLifeCycleState != column.DataLifeCycleStateLive {
			return ucerr.Friendlyf(nil, "UseSearchIndex can only be true for a live Accessor")
		}

		if a.SearchColumnID.IsNil() {
			return ucerr.Friendlyf(nil, "UseSearchIndex can only be true for an Accessor SelectorConfig that refers to a single SearchIndexed column with an ILIKE operator")
		}
	}

	return nil
}

// Mutator represents a userstore mutator in our system
type Mutator struct {
	ucdb.SystemAttributeBaseModel

	Name           string                       `db:"name" validate:"notempty"`
	Description    string                       `db:"description"`
	Version        int                          `db:"version"`
	ColumnIDs      uuidarray.UUIDArray          `db:"column_ids" validate:"skip"`
	NormalizerIDs  uuidarray.UUIDArray          `db:"validator_ids" validate:"skip"`
	AccessPolicyID uuid.UUID                    `db:"access_policy_id" validate:"notnil"`
	SelectorConfig userstore.UserSelectorConfig `db:"selector_config"`
}

//go:generate genvalidate Mutator

func (m Mutator) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "name,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"name:%v,id:%v",
				m.Name,
				m.ID,
			),
		)
	}
}

func (Mutator) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"id":      pagination.UUIDKeyType,
		"name":    pagination.StringKeyType,
		"created": pagination.TimestampKeyType,
		"updated": pagination.TimestampKeyType,
	}
}

//go:generate genpageable Mutator

// UsableForCreate returns true if the mutator can be used for creating a user
func (m Mutator) UsableForCreate() bool {
	return strings.ReplaceAll(m.SelectorConfig.WhereClause, " ", "") == "{id}=?"
}

//go:generate genorm --cache --followerreads --versioned --priorsave --nonpaginatedlist Mutator mutators tenantdb

// ToClientModel just translates from a storage.Mutator to a userstore.Mutator but doesn't fetch missing data
func (m Mutator) ToClientModel() userstore.Mutator {
	columns := make([]userstore.ColumnInputConfig, len(m.ColumnIDs))

	for i, id := range m.ColumnIDs {
		columns[i].Column = userstore.ResourceID{
			ID: id,
		}
	}

	for i, id := range m.NormalizerIDs {
		columns[i].Normalizer = userstore.ResourceID{
			ID: id,
		}
	}

	return userstore.Mutator{
		ID:             m.ID,
		Name:           m.Name,
		Description:    m.Description,
		Version:        m.Version,
		Columns:        columns,
		AccessPolicy:   userstore.ResourceID{ID: m.AccessPolicyID},
		SelectorConfig: m.SelectorConfig,
		IsSystem:       m.IsSystem,
	}
}

// Equals checks if two columns have same fields outside of created/updated time
func (m Mutator) Equals(o *Mutator, versionCheck bool) bool {
	if len(m.ColumnIDs) != len(o.ColumnIDs) {
		return false
	}

	if len(m.NormalizerIDs) != len(m.ColumnIDs) || len(o.NormalizerIDs) != len(o.ColumnIDs) {
		// this should never happen if the db is consistent, but checking so that we don't
		// panic if the arrays are of different lengths
		return false
	}

	mNormalizerMap := map[uuid.UUID]uuid.UUID{}
	for i, id := range m.ColumnIDs {
		mNormalizerMap[id] = m.NormalizerIDs[i]
	}

	oNormalizerMap := map[uuid.UUID]uuid.UUID{}
	for i, id := range o.ColumnIDs {
		oNormalizerMap[id] = o.NormalizerIDs[i]
	}

	for k, v := range mNormalizerMap {
		if oNormalizerMap[k] != v {
			return false
		}
	}

	return m.ID == o.ID &&
		strings.EqualFold(m.Name, o.Name) &&
		m.Description == o.Description &&
		m.AccessPolicyID == o.AccessPolicyID &&
		m.SelectorConfig == o.SelectorConfig &&
		m.IsSystem == o.IsSystem
}

func (m Mutator) extraValidate() error {
	if len(m.ColumnIDs) == 0 && !m.IsSystem {
		return ucerr.Friendlyf(nil, "Mutator.ColumnIDs cannot be empty")
	}

	uniqueColumnIDs := set.NewUUIDSet(m.ColumnIDs...)
	if uniqueColumnIDs.Size() != len(m.ColumnIDs) {
		return ucerr.Friendlyf(nil, "All column ids in Mutator.ColumnIDs must be unique")
	}

	if uniqueColumnIDs.Contains(uuid.Nil) {
		return ucerr.Friendlyf(nil, "All column ids in Mutator.ColumnIDs must be non-nil")
	}

	if len(m.ColumnIDs) != len(m.NormalizerIDs) {
		return ucerr.Friendlyf(nil, "Mutator.ColumnIDs and Mutator.NormalizerIDs must be the same length")
	}

	if set.NewUUIDSet(m.NormalizerIDs...).Contains(uuid.Nil) {
		return ucerr.Friendlyf(nil, "All normalizer ids in Mutator.NormalizerIDs must be non-nil")
	}

	return nil
}

// ColumnAttributes represents the internal attributes of a column
type ColumnAttributes struct {
	System      bool               `json:"system,omitempty"`
	SystemName  string             `json:"system_name,omitempty"`
	Immutable   bool               `json:"immutable,omitempty"`
	Constraints column.Constraints `json:"constraints"`
}

// Equals returns true if they are equal
func (ca ColumnAttributes) Equals(other ColumnAttributes) bool {
	return ca.System == other.System &&
		ca.SystemName == other.SystemName &&
		ca.Immutable == other.Immutable &&
		ca.Constraints.Equals(other.Constraints)
}

func (ca ColumnAttributes) extraValidate() error {
	if ca.System {
		if !ca.Immutable {
			return ucerr.Friendlyf(nil, "system columns must currently be immutable")
		}

		if !ca.Constraints.AreDefault() {
			return ucerr.Friendlyf(nil, "system columns cannot specify any constraints")
		}
	} else if ca.Immutable {
		return ucerr.Friendlyf(nil, "non-system columns cannot currently be immutable")
	}

	return nil
}

//go:generate genvalidate ColumnAttributes
//go:generate gendbjson ColumnAttributes

// ColumnIndexType is an enum for supported column index types
type ColumnIndexType int

const (
	columnIndexTypeNone ColumnIndexType = iota
	columnIndexTypeIndexed
	columnIndexTypeUnique
)

// ToClient converts storage.ColumnIndexType to userstore.ColumnIndexType
func (i ColumnIndexType) ToClient() userstore.ColumnIndexType {
	switch i {
	case columnIndexTypeIndexed:
		return userstore.ColumnIndexTypeIndexed
	case columnIndexTypeUnique:
		return userstore.ColumnIndexTypeUnique
	}

	return userstore.ColumnIndexTypeNone
}

// ColumnIndexTypeFromClient converts userstore.ColumnIndexType to storage.ColumnIndexType
func ColumnIndexTypeFromClient(i userstore.ColumnIndexType) ColumnIndexType {
	switch i {
	case userstore.ColumnIndexTypeIndexed:
		return columnIndexTypeIndexed
	case userstore.ColumnIndexTypeUnique:
		return columnIndexTypeUnique
	default:
		return columnIndexTypeNone
	}
}

// Column represents a single column in a userstore schema
type Column struct {
	ucdb.BaseModel

	Name                       string           `db:"name" validate:"notempty" json:"name"`
	Table                      string           `db:"tbl" json:"table" validate:"notempty"`
	SQLShimDatabaseID          uuid.UUID        `db:"sqlshim_database_id" json:"sqlshim_database_id"`
	DataTypeID                 uuid.UUID        `db:"data_type_id" json:"data_type_id" validate:"notnil"`
	IsArray                    bool             `db:"is_array" json:"is_array"`
	DefaultValue               string           `db:"default_value" json:"default_value"`
	IndexType                  ColumnIndexType  `db:"index_type" json:"index_type"`
	Attributes                 ColumnAttributes `db:"attributes" json:"attributes"`
	AccessPolicyID             uuid.UUID        `db:"access_policy_id" json:"access_policy_id" validate:"notnil"`
	DefaultTransformerID       uuid.UUID        `db:"default_transformer_id" json:"default_transformer_id" validate:"notnil"`
	DefaultTokenAccessPolicyID uuid.UUID        `db:"default_token_access_policy_id" json:"default_token_access_policy_id"`
	SearchIndexed              bool             `db:"search_indexed" json:"search_indexed"`
}

// NewColumnFromClient creates a new column from the client counterpart
func NewColumnFromClient(cc userstore.Column) Column {
	c := Column{
		Table:        cc.Table,
		Name:         cc.Name,
		DataTypeID:   cc.DataType.ID,
		IsArray:      cc.IsArray,
		DefaultValue: cc.DefaultValue,
		IndexType:    ColumnIndexTypeFromClient(cc.IndexType),
		Attributes: ColumnAttributes{
			System:      cc.IsSystem,
			Constraints: column.NewConstraintsFromClient(cc.Constraints),
		},
		AccessPolicyID:             cc.AccessPolicy.ID,
		DefaultTransformerID:       cc.DefaultTransformer.ID,
		DefaultTokenAccessPolicyID: cc.DefaultTokenAccessPolicy.ID,
		SearchIndexed:              cc.SearchIndexed,
	}
	if cc.ID != uuid.Nil {
		c.BaseModel = ucdb.NewBaseWithID(cc.ID)
	} else {
		c.BaseModel = ucdb.NewBase()
	}
	return c
}

func (c Column) updateColumnFromClient(cc userstore.Column) Column {
	c.BaseModel = ucdb.NewBaseWithID(cc.ID)
	c.Table = cc.Table
	c.Name = cc.Name
	c.DataTypeID = cc.DataType.ID
	c.IsArray = cc.IsArray
	c.DefaultValue = cc.DefaultValue
	c.IndexType = ColumnIndexTypeFromClient(cc.IndexType)
	c.Attributes.Constraints = column.NewConstraintsFromClient(cc.Constraints)
	c.AccessPolicyID = cc.AccessPolicy.ID
	c.DefaultTransformerID = cc.DefaultTransformer.ID
	c.DefaultTokenAccessPolicyID = cc.DefaultTokenAccessPolicy.ID
	c.SearchIndexed = cc.SearchIndexed
	return c
}

func (c *Column) extraValidate() error {
	if c.HasDefaultValue() {
		if c.Attributes.System {
			return ucerr.Friendlyf(nil, "system columns cannot have a default value")
		}

		if c.IsArray {
			return ucerr.Friendlyf(nil, "array columns cannot have a default value")
		}

		if c.IndexType == columnIndexTypeUnique {
			return ucerr.Friendlyf(nil, "unique columns cannot have a default value")
		}
	}

	if c.IsArray {
		if c.Attributes.System {
			return ucerr.Friendlyf(nil, "system columns must currently be single value")
		}

		if c.IndexType == columnIndexTypeUnique {
			return ucerr.Friendlyf(nil, "array columns cannot currently be unique")
		}
	} else {
		if c.Attributes.Constraints.PartialUpdates {
			return ucerr.Friendlyf(nil, "non-array column cannot have partial updates enabled")
		}

		if c.IndexType == columnIndexTypeUnique && !column.CanBeUnique(c.DataTypeID) {
			return ucerr.Friendlyf(
				nil,
				"unique columns must be of one of the following data types: [%s]",
				column.GetDisplayableUniqueDataTypes(),
			)
		}
	}

	if c.SearchIndexed {
		if err := c.confirmSearchIndexable(); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

//go:generate genvalidate Column

func (c Column) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "name,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"name:%v,id:%v",
				c.Name,
				c.ID,
			),
		)
	}
}

func (Column) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"id":      pagination.UUIDKeyType,
		"name":    pagination.StringKeyType,
		"created": pagination.TimestampKeyType,
		"updated": pagination.TimestampKeyType,
	}
}

func (c Column) confirmSearchIndexable() error {
	if !c.IsUserstoreColumn() {
		return ucerr.Friendlyf(nil, "non-userstore columns cannot be indexed in search")
	}

	if !column.IsSearchIndexable(c.DataTypeID) {
		return ucerr.Friendlyf(
			nil,
			"column data type '%v' cannot be indexed in search",
			c.DataTypeID,
		)
	}

	return nil
}

//go:generate genpageable Column

//go:generate genorm --cache --followerreads --cachepages --getbyname Column columns tenantdb

// ToClientModel just translates from a storage.Column to a userstore.Column
func (c Column) ToClientModel() userstore.Column {
	return userstore.Column{
		ID:                       c.ID,
		Table:                    c.Table,
		Name:                     c.Name,
		DataType:                 userstore.ResourceID{ID: c.DataTypeID},
		IsArray:                  c.IsArray,
		DefaultValue:             c.DefaultValue,
		IndexType:                c.IndexType.ToClient(),
		IsSystem:                 c.Attributes.System,
		Constraints:              c.Attributes.Constraints.ToClient(),
		AccessPolicy:             userstore.ResourceID{ID: c.AccessPolicyID},
		DefaultTransformer:       userstore.ResourceID{ID: c.DefaultTransformerID},
		DefaultTokenAccessPolicy: userstore.ResourceID{ID: c.DefaultTokenAccessPolicyID},
		SearchIndexed:            c.SearchIndexed,
	}
}

// CaseSensitiveEquals checks if two columns are equal, further ensuring
// that the names are an exact match
func (c Column) CaseSensitiveEquals(o *Column) bool {
	return c.Equals(o) && c.Name == o.Name
}

// Equals checks if two columns have same fields outside of created/updated time
func (c Column) Equals(o *Column) bool {
	return c.ID == o.ID &&
		strings.EqualFold(c.Table, o.Table) &&
		strings.EqualFold(c.Name, o.Name) &&
		c.DataTypeID == o.DataTypeID &&
		c.IsArray == o.IsArray &&
		c.DefaultValue == o.DefaultValue &&
		c.IndexType == o.IndexType &&
		c.Attributes.Equals(o.Attributes) &&
		c.AccessPolicyID == o.AccessPolicyID &&
		c.DefaultTransformerID == o.DefaultTransformerID &&
		c.DefaultTokenAccessPolicyID == o.DefaultTokenAccessPolicyID &&
		c.SearchIndexed == o.SearchIndexed
}

// GetConcreteDataTypeID returns the concrete data type ID for the column
func (c Column) GetConcreteDataTypeID() uuid.UUID {
	if column.IsNativeDataType(c.DataTypeID) {
		if ndt, err := column.GetNativeDataType(c.DataTypeID); err == nil {
			return ndt.ConcreteDataTypeID
		}
	}

	return datatype.Composite.ID
}

// GetUserRowColumnNames returns the user_column_values column names for the column
func (c Column) GetUserRowColumnNames() (string, string, error) {
	ci := newColumnInfo(c, column.DataLifeCycleStateLive)
	liveColName, err := ci.getUserRowColumnName()
	if err != nil {
		return "", "", ucerr.Wrap(err)
	}

	ci = newColumnInfo(c, column.DataLifeCycleStateSoftDeleted)
	softDeletedColName, err := ci.getUserRowColumnName()
	if err != nil {
		return "", "", ucerr.Wrap(err)
	}

	return liveColName, softDeletedColName, nil
}

// FullName centralizes logic for table.column name
func (c Column) FullName() string {
	return fmt.Sprintf("%s.%s", c.Table, c.Name)
}

// HasDefaultValue returns true if the column has a default value
func (c Column) HasDefaultValue() bool {
	return c.DefaultValue != ""
}

// IsUserstoreColumn returns true if the column is a userstore column
func (c Column) IsUserstoreColumn() bool {
	return c.SQLShimDatabaseID.IsNil()
}

// ToStringConcise returns the column name and ID as a formatted string
func (c Column) ToStringConcise() string {
	return fmt.Sprintf("%v.%v [%v]", c.Table, c.Name, c.ID.String())
}

// ValidateMutation makes sure values are passed appropriately mutation based on whether partial updates are enabled
func (c Column) ValidateMutation(vp idp.ValueAndPurposes) error {
	if c.Attributes.Constraints.PartialUpdates {
		if vp.Value != nil {
			return ucerr.Friendlyf(
				nil,
				"Value cannot be set in ValueAndPurposes for partial updates enabled column '%s'",
				c.Name,
			)
		}
	} else if vp.ValueAdditions != nil {
		return ucerr.Friendlyf(
			nil,
			"ValueAdditions cannot be set in ValueAndPurposes for partial updates disabled column '%s'",
			c.Name,
		)
	} else if vp.ValueDeletions != nil {
		return ucerr.Friendlyf(
			nil,
			"ValueDeletions cannot be set in ValueAndPurposes for partial updates disabled column '%s'",
			c.Name,
		)
	}

	return nil
}

// Columns is a slice of Column
type Columns []Column

// GetIDs collects the column IDs from the Column objects and returns them as a set
func (cols Columns) GetIDs() set.Set[uuid.UUID] {
	ids := set.NewUUIDSet()
	for _, col := range cols {
		ids.Insert(col.ID)
	}
	return ids
}

// GetIDsMap returns a map/hash from column ID to columns
func (cols Columns) GetIDsMap() map[uuid.UUID]Column {
	colMap := make(map[uuid.UUID]Column)
	for _, col := range cols {
		colMap[col.ID] = col
	}
	return colMap
}

// Purpose is storage model of a userstore purpose
type Purpose struct {
	ucdb.SystemAttributeBaseModel

	Name        string `db:"name" validate:"notempty"`
	Description string `db:"description"`
}

//go:generate genvalidate Purpose

//go:generate genorm --cache --followerreads --getbyname --nonpaginatedlist Purpose purposes tenantdb

func (p Purpose) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "name,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"name:%v,id:%v",
				p.Name,
				p.ID,
			),
		)
	}
}

func (Purpose) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"name":    pagination.StringKeyType,
		"created": pagination.TimestampKeyType,
		"updated": pagination.TimestampKeyType,
	}
}

//go:generate genpageable Purpose

// ToClientModel just translates from a storage.Purpose to a userstore.Purpose
func (p Purpose) ToClientModel() userstore.Purpose {
	return userstore.Purpose{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		IsSystem:    p.IsSystem,
	}
}

//go:generate genorm --cache --followerreads --nonpaginatedlist column.DataType data_types tenantdb

var _ = datamapping.DataSource{} // this var is used to ensure that this package depends on idp/datamapping so we can run genorm on it
//go:generate genorm --cache --followerreads datamapping.DataSourceElement data_source_elements tenantdb
//go:generate genorm --cache --followerreads datamapping.DataSource data_sources tenantdb

// SQLShimDatabase represents an external database that tenant customers can connect to via a SQLShim proxy
type SQLShimDatabase struct {
	ucdb.BaseModel

	Name                   string               `db:"name" validate:"notempty"`
	Type                   sqlshim.DatabaseType `db:"type"`
	Host                   string               `db:"host" validate:"notempty"`
	Port                   int                  `db:"port" validate:"notzero"`
	Username               string               `db:"username" validate:"notempty"`
	Password               secret.String        `db:"password"`
	Schemas                pq.StringArray       `db:"schemas"`
	SchemasUpdated         time.Time            `db:"schemas_updated"`
	SchemasUpdateScheduled time.Time            `db:"schemas_update_scheduled"`
}

//go:generate genpageable SQLShimDatabase
//go:generate genvalidate SQLShimDatabase
//go:generate genorm --cache --followerreads SQLShimDatabase sqlshim_databases tenantdb

// ToClientModel translates from a storage.SQLShimDatabase to a userstore.SQLShimDatabase
func (s SQLShimDatabase) ToClientModel() userstore.SQLShimDatabase {
	return userstore.SQLShimDatabase{
		ID:       s.ID,
		Name:     s.Name,
		Type:     string(s.Type),
		Host:     s.Host,
		Port:     s.Port,
		Username: s.Username,
	}
}

// ObjectStoreType is an enum for supported object store types
type ObjectStoreType string

const (
	// ObjectStoreTypeS3 represents an S3 object store
	ObjectStoreTypeS3 ObjectStoreType = "s3"
)

// ShimObjectStore represents an external object store that tenant customers can connect to via a proxy
type ShimObjectStore struct {
	ucdb.BaseModel

	Name            string          `db:"name" validate:"notempty"`
	Type            ObjectStoreType `db:"type" validate:"notempty"`
	Region          string          `db:"region" validate:"notempty"`
	AccessKeyID     string          `db:"access_key_id"`
	SecretAccessKey secret.String   `db:"secret_access_key"`
	RoleARN         string          `db:"role_arn"`
	AccessPolicyID  uuid.UUID       `db:"access_policy_id" validate:"notnil"`
}

func (s ShimObjectStore) extraValidate() error {
	if (s.AccessKeyID == "" || s.SecretAccessKey.IsEmpty()) && s.RoleARN == "" {
		return ucerr.Friendlyf(nil, "ShimObjectStore must have both AccessKeyID and SecretAccessKey; otherwise RoleARN must be provided")
	}
	if s.RoleARN != "" {
		if !ucaws.IsValidAwsARN(s.RoleARN) {
			return ucerr.Friendlyf(nil, "Invalid ARN: %s", s.RoleARN)
		}
	}
	return nil
}

//go:generate genpageable ShimObjectStore
//go:generate genvalidate ShimObjectStore
//go:generate genorm --cache --followerreads ShimObjectStore shim_object_stores tenantdb

// ToClientModel translates from a storage.ShimObjectStore to a userstore.ShimObjectStore
func (s ShimObjectStore) ToClientModel() userstore.ShimObjectStore {
	return userstore.ShimObjectStore{
		ID:           s.ID,
		Name:         s.Name,
		Type:         string(s.Type),
		Region:       s.Region,
		AccessKeyID:  s.AccessKeyID,
		RoleARN:      s.RoleARN,
		AccessPolicy: userstore.ResourceID{ID: s.AccessPolicyID},
	}
}

func (ShimObjectStore) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"id":      pagination.UUIDKeyType,
		"name":    pagination.StringKeyType,
		"created": pagination.TimestampKeyType,
		"updated": pagination.TimestampKeyType,
	}
}

func (s ShimObjectStore) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "name,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"name:%v,id:%v",
				s.Name,
				s.ID,
			),
		)
	}
}
