package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"
	"github.com/lib/pq"

	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/infra/uctypes/uuidarray"
	"userclouds.com/internal/auditlog"
)

// TokenRecord stores the actual tokenized data, the resulting token, and the associated policies
type TokenRecord struct {
	ucdb.BaseModel

	Data  string `db:"data"`
	Token string `db:"token" validate:"notempty"`

	// fields for tokenize-by-reference
	UserID   uuid.UUID `db:"user_id"`
	ColumnID uuid.UUID `db:"column_id"`

	TransformerID      uuid.UUID `db:"transformer_id" validate:"notnil"`
	TransformerVersion int       `db:"transformer_version"`
	AccessPolicyID     uuid.UUID `db:"access_policy_id" validate:"notnil"`
}

func (tr *TokenRecord) extraValidate() error {
	if len(tr.Data) > 0 {
		if !tr.UserID.IsNil() || !tr.ColumnID.IsNil() {
			return ucerr.New("Data must be empty if UserID and ColumnID are specified")
		}
	} else if tr.UserID.IsNil() {
		if !tr.ColumnID.IsNil() {
			return ucerr.New("UserID must be specified if ColumnID is specified")
		}
	} else if tr.ColumnID.IsNil() {
		return ucerr.New("ColumnID must be specified if UserID is specified")
	}

	return nil
}

//go:generate genpageable TokenRecord

//go:generate genvalidate TokenRecord

// NB: we use --avoid-upsert here since the PK for the token_records table
// is not (id, deleted) (as is convention) but (token, deleted) in order to give
// us single-disk-seek access to the token record. This means that UPSERT's built-in
// conflict detection will fail to correctly update records by ID on SaveTokenRecord,
// so we use a generated `INSERT...ON CONFLICT (id) DO UPDATE` instead.
//go:generate genorm TokenRecord token_records tenantdb

// AccessPolicyComponentType describes the type of component in an access policy
type AccessPolicyComponentType int

const (
	// AccessPolicyComponentTypeInvalid is an invalid component type
	AccessPolicyComponentTypeInvalid AccessPolicyComponentType = 0

	// AccessPolicyComponentTypePolicy is a policy component
	AccessPolicyComponentTypePolicy AccessPolicyComponentType = 1

	// AccessPolicyComponentTypeTemplate is a template component
	AccessPolicyComponentTypeTemplate AccessPolicyComponentType = 2
)

// InternalPolicyType is the storage representation of a policy type
type InternalPolicyType int

const (
	policyTypeCompositeAnd InternalPolicyType = 1
	policyTypeCompositeOr  InternalPolicyType = 2
)

// Validate implements Validateable
func (pt InternalPolicyType) Validate() error {
	switch pt {
	case policyTypeCompositeAnd, policyTypeCompositeOr:
		return nil
	}
	return ucerr.Friendlyf(nil, "Invalid policy type %d", pt)
}

// ToClient converts an InternalPolicyType to a policy.PolicyType
func (pt InternalPolicyType) ToClient() policy.PolicyType {
	switch pt {
	case policyTypeCompositeAnd:
		return policy.PolicyTypeCompositeAnd
	case policyTypeCompositeOr:
		return policy.PolicyTypeCompositeOr
	}
	return policy.PolicyTypeCompositeAnd
}

// InternalPolicyTypeFromClient converts a policy.PolicyType to an InternalPolicyType
func InternalPolicyTypeFromClient(pt policy.PolicyType) InternalPolicyType {
	switch pt {
	case policy.PolicyTypeCompositeAnd:
		return policyTypeCompositeAnd
	case policy.PolicyTypeCompositeOr:
		return policyTypeCompositeOr
	}
	return policyTypeCompositeAnd
}

// AccessPolicyMetadata holds metadata for an access policy
type AccessPolicyMetadata struct {
	RequiredContext map[string]string `json:"required_context"`
}

//go:generate gendbjson AccessPolicyMetadata

// AccessPolicyTemplate describes an access policy template
type AccessPolicyTemplate struct {
	ucdb.SystemAttributeBaseModel

	Name        string `db:"name" validate:"notempty"`
	Description string `db:"description"`
	Function    string `db:"function" validate:"notempty"`
	Version     int    `db:"version"`
}

func (apt AccessPolicyTemplate) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "name,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"name:%v,id:%v",
				apt.Name,
				apt.ID,
			),
		)
	}
}

func (AccessPolicyTemplate) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"name":        pagination.StringKeyType,
		"description": pagination.StringKeyType,
		"created":     pagination.TimestampKeyType,
		"updated":     pagination.TimestampKeyType,
	}
}

//go:generate genpageable AccessPolicyTemplate

func (apt *AccessPolicyTemplate) extraValidate() error {
	if err := validateJSScript("AccessPolicyTemplate", apt.Function, auditlog.AccessPolicyCustom); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

//go:generate genvalidate AccessPolicyTemplate

//go:generate genorm --cache --followerreads --versioned --priorsave --nonpaginatedlist AccessPolicyTemplate access_policy_templates tenantdb

// NewAccessPolicyTemplateFromClient creates an access policy template from the client version
func NewAccessPolicyTemplateFromClient(apt policy.AccessPolicyTemplate) AccessPolicyTemplate {
	return AccessPolicyTemplate{
		SystemAttributeBaseModel: apt.SystemAttributeBaseModel,
		Name:                     apt.Name,
		Description:              apt.Description,
		Function:                 apt.Function,
		Version:                  apt.Version,
	}
}

// ToClient returns the client version of the access policy template
func (apt AccessPolicyTemplate) ToClient() policy.AccessPolicyTemplate {
	return policy.AccessPolicyTemplate{
		SystemAttributeBaseModel: apt.SystemAttributeBaseModel,
		Name:                     apt.Name,
		Description:              apt.Description,
		Function:                 apt.Function,
		Version:                  apt.Version,
	}
}

// AccessPolicyThresholds describes the thresholds associated with an access policy
type AccessPolicyThresholds struct {
	AnnounceMaxExecutionFailure bool `json:"announce_max_execution_failure"`
	AnnounceMaxResultFailure    bool `json:"announce_max_result_failure"`
	MaxExecutions               int  `json:"max_executions"`
	MaxExecutionDurationSeconds int  `json:"max_execution_duration_seconds"`
	MaxResultsPerExecution      int  `json:"max_results_per_execution"`
}

func (AccessPolicyThresholds) secondsPerBucket() int {
	return 1
}

func (AccessPolicyThresholds) minThresholdWindowSeconds() int {
	return 5
}

func (AccessPolicyThresholds) maxThresholdWindowSeconds() int {
	return 60
}

func (apt *AccessPolicyThresholds) extraValidate() error {
	if apt.MaxExecutions < 0 {
		return ucerr.Friendlyf(nil, "MaxExecutions must be non-negative: '%+v'", *apt)
	}

	if apt.MaxExecutionDurationSeconds < 0 {
		return ucerr.Friendlyf(nil, "MaxExecutionDurationSeconds must be non-negative: '%+v'", *apt)
	}

	if apt.MaxResultsPerExecution < 0 {
		return ucerr.Friendlyf(nil, "MaxResultsPerExecution must be non-negative: '%+v'", *apt)
	}

	if apt.MaxExecutions > 0 {
		if apt.MaxExecutionDurationSeconds < apt.minThresholdWindowSeconds() ||
			apt.MaxExecutionDurationSeconds > apt.maxThresholdWindowSeconds() {
			return ucerr.Friendlyf(
				nil,
				"MaxExecutionDurationSeconds must be less than or equal to %d and greater than or equal to %d if MaxExecutions is set: '%+v'",
				apt.minThresholdWindowSeconds(),
				apt.maxThresholdWindowSeconds(),
				*apt,
			)
		}

		if apt.MaxExecutionDurationSeconds%apt.secondsPerBucket() != 0 {
			return ucerr.Friendlyf(
				nil,
				"MaxExecutionDurationSeconds must a multiple of %d if MaxeExecutions is set: '%+v'",
				apt.secondsPerBucket(),
				*apt,
			)
		}
	}

	return nil
}

//go:generate gendbjson AccessPolicyThresholds
//go:generate genvalidate AccessPolicyThresholds

// AccessPolicyThresholdsFromClient creates an access policy thresholds instance from the client version
func AccessPolicyThresholdsFromClient(apt policy.AccessPolicyThresholds) AccessPolicyThresholds {
	return AccessPolicyThresholds{
		AnnounceMaxExecutionFailure: apt.AnnounceMaxExecutionFailure,
		AnnounceMaxResultFailure:    apt.AnnounceMaxResultFailure,
		MaxExecutions:               apt.MaxExecutions,
		MaxExecutionDurationSeconds: apt.MaxExecutionDurationSeconds,
		MaxResultsPerExecution:      apt.MaxResultsPerExecution,
	}
}

func (apt AccessPolicyThresholds) hasRateLimit() bool {
	return apt.MaxExecutions > 0 && apt.MaxExecutionDurationSeconds > 0
}

// ToClient converts the access policy thresholds to the client version
func (apt AccessPolicyThresholds) ToClient() policy.AccessPolicyThresholds {
	return policy.AccessPolicyThresholds{
		AnnounceMaxExecutionFailure: apt.AnnounceMaxExecutionFailure,
		AnnounceMaxResultFailure:    apt.AnnounceMaxResultFailure,
		MaxExecutions:               apt.MaxExecutions,
		MaxExecutionDurationSeconds: apt.MaxExecutionDurationSeconds,
		MaxResultsPerExecution:      apt.MaxResultsPerExecution,
	}
}

// AccessPolicyRateLimit represents a set of rate limit thresholds, an entity id, and a subject
type AccessPolicyRateLimit struct {
	thresholds AccessPolicyThresholds
	entityID   string
	subject    string
}

const globalSubject = "GLOBAL-ACCESS-POLICY-RATE-LIMIT-SUBJECT"

func newAccessPolicyRateLimit(
	apt AccessPolicyThresholds,
	apc policy.AccessPolicyContext,
	entityID uuid.UUID,
) (*AccessPolicyRateLimit, error) {
	subject, found := apc.Server.Claims["sub"]
	if !found {
		if apc.ConnectionID.IsNil() {
			subject = globalSubject
		} else {
			subject = apc.ConnectionID.String()
		}
	}

	subjectString, ok := subject.(string)
	if !ok {
		return nil, ucerr.Friendlyf(nil, "server context subject is not a string '%v'", subject)
	}

	aprl := AccessPolicyRateLimit{
		thresholds: apt,
		entityID:   entityID.String(),
		subject:    subjectString,
	}
	if err := aprl.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &aprl, nil
}

// GetRateLimitKeys is part of the RateLimitableItem interface
func (aprl AccessPolicyRateLimit) GetRateLimitKeys(knp cache.KeyNameProvider) []cache.RateLimitKey {
	var keys []cache.RateLimitKey

	bucketSize := int64(aprl.thresholds.secondsPerBucket())
	maxBucket := (time.Now().UTC().Unix() / bucketSize) * bucketSize
	for bucket := maxBucket - int64(aprl.thresholds.MaxExecutionDurationSeconds) + bucketSize; bucket <= maxBucket; bucket += bucketSize {
		keys = append(
			keys,
			knp.GetRateLimitKeyName(AccessPolicyRateLimitKeyID, fmt.Sprintf("%s_%s_%v", aprl.entityID, aprl.subject, bucket)),
		)
	}

	return keys
}

// GetRateLimit is part of the RateLimitableItem interface
func (aprl AccessPolicyRateLimit) GetRateLimit() int64 {
	return int64(aprl.thresholds.MaxExecutions)
}

// TTL is part of the RateLimitableItem interface
func (aprl AccessPolicyRateLimit) TTL(ttlp cache.TTLProvider) time.Duration {
	return ttlp.TTL(AccessPolicyRateLimitTTL)
}

// Validate is part of the Validateable interface
func (aprl AccessPolicyRateLimit) Validate() error {
	if !aprl.thresholds.hasRateLimit() {
		return ucerr.Errorf("thresholds does not have rate limit: '%v'", aprl)
	}

	if aprl.entityID == "" {
		return ucerr.Errorf("entityID cannot be empty: '%v'", aprl)
	}

	if aprl.subject == "" {
		return ucerr.Errorf("subject cannot be empty: '%v'", aprl)
	}

	return nil
}

// AccessPolicy describes an access policy
type AccessPolicy struct {
	ucdb.SystemAttributeBaseModel

	Name            string                 `db:"name" validate:"notempty"`
	Description     string                 `db:"description"`
	PolicyType      InternalPolicyType     `db:"policy_type"`
	TagIDs          uuidarray.UUIDArray    `db:"tag_ids" validate:"skip"`
	Version         int                    `db:"version"`
	IsAutogenerated bool                   `db:"is_autogenerated"`
	Thresholds      AccessPolicyThresholds `db:"thresholds"`

	ComponentIDs        uuidarray.UUIDArray `db:"component_ids" validate:"skip"`
	ComponentParameters pq.StringArray      `db:"component_parameters" validate:"skip"`
	ComponentTypes      pq.Int32Array       `db:"component_types" validate:"skip"`

	Metadata AccessPolicyMetadata `db:"metadata" validate:"skip"`
}

//go:generate genvalidate AccessPolicy

// CheckRateThreshold will return false if the configured max execution rate for the access policy
// and specified context and entity id is exceeded
func (ap AccessPolicy) CheckRateThreshold(
	ctx context.Context,
	s *Storage,
	apc policy.AccessPolicyContext,
	entityID uuid.UUID,
) (bool, error) {
	if !ap.Thresholds.hasRateLimit() {
		return true, nil
	}

	if !s.cm.Provider.SupportsRateLimits(ctx) {
		uclog.Warningf(ctx, "access policy '%v' has a rate limit but cache provider does not support rate limits", ap.ID)
		return true, nil
	}

	aprl, err := newAccessPolicyRateLimit(ap.Thresholds, apc, entityID)
	if err != nil {
		return false, ucerr.Wrap(err)
	}

	reserved, _, err := cache.ReserveRateLimitSlot(ctx, *s.cm, *aprl, true)
	if err != nil {
		return false, ucerr.Wrap(err)
	}

	return reserved, nil
}

// CheckResultThreshold will return false if the configured max result threshold for the access policy
// is exceeded
func (ap AccessPolicy) CheckResultThreshold(totalResults int) bool {
	if ap.HasResultThreshold() {
		return totalResults <= ap.GetResultThreshold()
	}
	return true
}

// GetResultThreshold returns the max result threshold for the access policy
func (ap AccessPolicy) GetResultThreshold() int {
	return ap.Thresholds.MaxResultsPerExecution
}

func (ap AccessPolicy) hasThreshold() bool {
	return ap.HasResultThreshold() || ap.Thresholds.hasRateLimit()
}

// HasResultThreshold returns true if the access policy has a max result threshold
func (ap AccessPolicy) HasResultThreshold() bool {
	return ap.Thresholds.MaxResultsPerExecution > 0
}

func (ap AccessPolicy) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "name,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"name:%v,id:%v",
				ap.Name,
				ap.ID,
			),
		)
	}
}

func (AccessPolicy) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"name":             pagination.StringKeyType,
		"description":      pagination.StringKeyType,
		"created":          pagination.TimestampKeyType,
		"updated":          pagination.TimestampKeyType,
		"is_autogenerated": pagination.BoolKeyType,
	}
}

//go:generate genpageable AccessPolicy

func (ap *AccessPolicy) extraValidate() error {
	for _, id := range ap.ComponentIDs {
		if id.IsNil() {
			return ucerr.New("component_ids must not contain nil UUID")
		}
	}

	if len(ap.ComponentParameters) != len(ap.ComponentIDs) {
		return ucerr.New("component_template_parameters must be the same length as component_template_ids")
	}

	if len(ap.ComponentTypes) != len(ap.ComponentIDs) {
		return ucerr.New("component_types must be the same length as component_template_ids")
	}

	return nil
}

// EqualsIgnoringNilID returns true if the two policies are equal, ignoring the ID field if one is nil
func (ap *AccessPolicy) EqualsIgnoringNilID(other *AccessPolicy) bool {
	return (ap.ID == other.ID || ap.ID.IsNil() || other.ID.IsNil()) &&
		strings.EqualFold(ap.Name, other.Name) &&
		ap.PolicyType == other.PolicyType &&
		cmp.Equal(ap.ComponentParameters, other.ComponentParameters) &&
		cmp.Equal(ap.ComponentIDs, other.ComponentIDs) &&
		ap.Thresholds == other.Thresholds &&
		ap.IsSystem == other.IsSystem
}

// ToClientModel converts an AccessPolicy to a policy.AccessPolicy
func (ap *AccessPolicy) ToClientModel() *policy.AccessPolicy {
	components := make([]policy.AccessPolicyComponent, len(ap.ComponentIDs))
	for i, id := range ap.ComponentIDs {
		if ap.ComponentTypes[i] == int32(AccessPolicyComponentTypeTemplate) {
			components[i] = policy.AccessPolicyComponent{
				Template:           &userstore.ResourceID{ID: id},
				TemplateParameters: ap.ComponentParameters[i],
			}
		} else {
			components[i] = policy.AccessPolicyComponent{
				Policy: &userstore.ResourceID{ID: id},
			}
		}
	}

	return &policy.AccessPolicy{
		ID:              ap.ID,
		Name:            ap.Name,
		Description:     ap.Description,
		PolicyType:      ap.PolicyType.ToClient(),
		TagIDs:          ap.TagIDs,
		Version:         ap.Version,
		Components:      components,
		IsSystem:        ap.IsSystem,
		IsAutogenerated: ap.IsAutogenerated,
		Thresholds:      ap.Thresholds.ToClient(),
		RequiredContext: ap.Metadata.RequiredContext,
	}

}

// AccessPolicy doesn't have a getter because they're versioned and we always want a specific one
// and no delete because we never want to delete all versions "accidentally"
//go:generate genorm --cache --followerreads --nonpaginatedlist --versioned --priorsave AccessPolicy access_policies tenantdb

// InternalTransformType is a storage-level enum for supported transform types
type InternalTransformType int

const (
	transformTypePassThrough         InternalTransformType = 1
	transformTypeTransform           InternalTransformType = 2
	transformTypeTokenizeByValue     InternalTransformType = 3
	transformTypeTokenizeByReference InternalTransformType = 4
)

// Validate implements Validateable
func (tt InternalTransformType) Validate() error {
	switch tt {
	case transformTypePassThrough, transformTypeTransform, transformTypeTokenizeByValue, transformTypeTokenizeByReference:
		return nil
	}
	return ucerr.Friendlyf(nil, "Invalid transform type %d", tt)
}

// ToClient converts storage.InternalTransformType to policy.TransformType
func (tt InternalTransformType) ToClient() policy.TransformType {
	switch tt {
	case transformTypePassThrough:
		return policy.TransformTypePassThrough
	case transformTypeTransform:
		return policy.TransformTypeTransform
	case transformTypeTokenizeByValue:
		return policy.TransformTypeTokenizeByValue
	case transformTypeTokenizeByReference:
		return policy.TransformTypeTokenizeByReference
	}

	return policy.TransformTypePassThrough
}

// InternalTransformTypeFromClient converts policy.TransformType to storage.InternalTransformType
func InternalTransformTypeFromClient(tt policy.TransformType) InternalTransformType {
	switch tt {
	case policy.TransformTypePassThrough:
		return transformTypePassThrough
	case policy.TransformTypeTransform:
		return transformTypeTransform
	case policy.TransformTypeTokenizeByValue:
		return transformTypeTokenizeByValue
	case policy.TransformTypeTokenizeByReference:
		return transformTypeTokenizeByReference
	default:
		return transformTypePassThrough
	}
}

// Transformer describes a token transformer
type Transformer struct {
	ucdb.SystemAttributeBaseModel

	Name               string                `db:"name" json:"name" validate:"length:1,128"`
	Description        string                `db:"description" json:"description"`
	InputDataTypeID    uuid.UUID             `db:"input_data_type_id" json:"input_data_type_id" validate:"notnil"`
	OutputDataTypeID   uuid.UUID             `db:"output_data_type_id" json:"output_data_type_id" validate:"notnil"`
	ReuseExistingToken bool                  `db:"reuse_existing_token" json:"reuse_existing_token"`
	TransformType      InternalTransformType `db:"transform_type" json:"transform_type"`
	TagIDs             uuidarray.UUIDArray   `db:"tag_ids" json:"tag_ids" validate:"skip"`

	Function   string `db:"function" json:"function"`
	Parameters string `db:"parameters" json:"parameters"`

	Version int `db:"version" json:"version"`
}

// NewTransformerFromClient creates a transformer from a client counterpart
func NewTransformerFromClient(pt policy.Transformer) Transformer {
	t := Transformer{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(pt.ID),
		Name:                     pt.Name,
		Description:              pt.Description,
		InputDataTypeID:          pt.InputDataType.ID,
		OutputDataTypeID:         pt.OutputDataType.ID,
		ReuseExistingToken:       pt.ReuseExistingToken,
		TransformType:            InternalTransformTypeFromClient(pt.TransformType),
		TagIDs:                   pt.TagIDs,
		Function:                 pt.Function,
		Parameters:               pt.Parameters,
		Version:                  0,
	}

	return t
}

// CanInput returns true if the transformer has an input data type
// that is compatible with the data type
func (t Transformer) CanInput(dt column.DataType) bool {
	if t.InputDataTypeID == dt.ID {
		return true
	}

	switch t.InputDataTypeID {
	case datatype.String.ID:
		return true
	case datatype.Date.ID:
		return dt.ConcreteDataTypeID == datatype.Date.ID
	case datatype.Timestamp.ID:
		return dt.ConcreteDataTypeID == datatype.Date.ID ||
			dt.ConcreteDataTypeID == datatype.Timestamp.ID
	default:
		return false
	}
}

// CanOutput returns true if the transformer has an output data type
// that exactly matches the data type
func (t Transformer) CanOutput(dt column.DataType) bool {
	return t.OutputDataTypeID == dt.ID
}

// Equals returns true if the two transformers are exactly equal
func (t Transformer) Equals(other Transformer) bool {
	return t.EqualsIgnoringNilID(other) &&
		t.ID == other.ID &&
		t.Name == other.Name &&
		t.Description == other.Description &&
		set.NewUUIDSet(t.TagIDs...).Equal(set.NewUUIDSet(other.TagIDs...))
}

// EqualsIgnoringNilID returns true if the two transformers are equal,
// ignoring the case where the name, the description, or the tags are
// different, or the IDs if either are nil
func (t Transformer) EqualsIgnoringNilID(other Transformer) bool {
	return (t.ID == other.ID || t.ID.IsNil() || other.ID.IsNil()) &&
		strings.EqualFold(t.Name, other.Name) &&
		t.InputDataTypeID == other.InputDataTypeID &&
		t.OutputDataTypeID == other.OutputDataTypeID &&
		t.TransformType == other.TransformType &&
		t.ReuseExistingToken == other.ReuseExistingToken &&
		t.Function == other.Function &&
		t.Parameters == other.Parameters
}

func (t Transformer) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "name,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"name:%v,id:%v",
				t.Name,
				t.ID,
			),
		)
	}
}

func (Transformer) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"name":                pagination.StringKeyType,
		"description":         pagination.StringKeyType,
		"input_data_type_id":  pagination.UUIDKeyType,
		"output_data_type_id": pagination.UUIDKeyType,
		"transform_type":      pagination.StringKeyType,
		"created":             pagination.TimestampKeyType,
		"updated":             pagination.TimestampKeyType,
	}
}

//go:generate genpageable Transformer

// RequiresDataProvenance returns whether the transformer requires data provenance for execution
func (t Transformer) RequiresDataProvenance() bool {
	return t.TransformType == transformTypeTokenizeByReference
}

// RequiresTokenAccessPolicy returns whether an access policy is required for execution
func (t Transformer) RequiresTokenAccessPolicy() bool {
	return t.TransformType == transformTypeTokenizeByValue ||
		t.TransformType == transformTypeTokenizeByReference
}

// ValidateProvisioningUpdate ensures that no disallowed properties of a transformer have changed
func (t Transformer) ValidateProvisioningUpdate(updated Transformer) error {
	// TODO: this is necessary until we unify transformer update logic between
	//       our IDP endpoints and provisioning
	if !strings.EqualFold(t.Name, updated.Name) {
		return ucerr.Errorf(
			"transformer '%v' name cannot change from '%s' to '%s' in provisioning",
			t.ID,
			t.Name,
			updated.Name,
		)
	}

	if t.InputDataTypeID != updated.InputDataTypeID ||
		t.OutputDataTypeID != updated.OutputDataTypeID ||
		t.ReuseExistingToken != updated.ReuseExistingToken ||
		t.TransformType != updated.TransformType ||
		t.Function != updated.Function ||
		t.Parameters != updated.Parameters {
		return ucerr.Errorf(
			"transformer signature cannot change from '%v' to '%v' in provisioning",
			t,
			updated,
		)
	}

	return nil
}

// ToClientModel just translates from a storage.Transformer to a policy.Transformer
func (t Transformer) ToClientModel() policy.Transformer {
	return policy.Transformer{
		ID:                 t.ID,
		Name:               t.Name,
		Description:        t.Description,
		InputDataType:      userstore.ResourceID{ID: t.InputDataTypeID},
		OutputDataType:     userstore.ResourceID{ID: t.OutputDataTypeID},
		ReuseExistingToken: t.ReuseExistingToken,
		TransformType:      t.TransformType.ToClient(),
		TagIDs:             t.TagIDs,
		Function:           t.Function,
		Parameters:         t.Parameters,
		IsSystem:           t.IsSystem,
		Version:            t.Version,
	}
}

func (t Transformer) extraValidate() error {
	if err := validateJSScript("Transformer", t.Function, auditlog.TransformerCustom); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

//go:generate genvalidate Transformer
//go:generate genorm --cache --followerreads --nonpaginatedlist --versioned --priorsave Transformer transformers tenantdb

// Secret is the model for a customer secret
type Secret struct {
	ucdb.BaseModel
	Name  string        `db:"name" validate:"notempty"`
	Value secret.String `db:"value" validate:"notempty"`
}

//go:generate genpageable Secret
//go:generate genvalidate Secret
//go:generate genorm Secret policy_secrets tenantdb

// ToClientModel converts a storage.Secret to a policy.Secret
func (s *Secret) ToClientModel() policy.Secret {
	return policy.Secret{
		ID:      s.ID,
		Name:    s.Name,
		Created: s.Created.Unix(),
	}
}

func (Secret) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"id":      pagination.UUIDKeyType,
		"name":    pagination.StringKeyType,
		"created": pagination.TimestampKeyType,
		"updated": pagination.TimestampKeyType,
	}
}

func (s Secret) getCursor(key pagination.Key, cursor *pagination.Cursor) {
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
