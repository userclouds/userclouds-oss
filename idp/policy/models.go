package policy

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/uuidarray"
)

var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

// TransformType describes the type of transform to be performed
type TransformType string

const (
	// TransformTypePassThrough is a no-op transformation
	TransformTypePassThrough TransformType = "passthrough"

	// TransformTypeTransform is a transformation that doesn't tokenize
	TransformTypeTransform TransformType = "transform"

	// TransformTypeTokenizeByValue is a transformation that tokenizes the value passed in
	TransformTypeTokenizeByValue TransformType = "tokenizebyvalue"

	// TransformTypeTokenizeByReference is a transformation that tokenizes the userstore reference to the value passed in
	TransformTypeTokenizeByReference TransformType = "tokenizebyreference"
)

//go:generate genconstant TransformType

// Transformer describes a token transformer
type Transformer struct {
	ID                 uuid.UUID                   `json:"id"`
	Name               string                      `json:"name" validate:"length:1,128" required:"true"`
	Description        string                      `json:"description"`
	InputDataType      userstore.ResourceID        `json:"input_data_type" required:"true"`
	InputType          string                      `json:"input_type" validate:"skip"`
	InputConstraints   userstore.ColumnConstraints `json:"input_type_constraints" validate:"skip"`
	OutputDataType     userstore.ResourceID        `json:"output_data_type" required:"true"`
	OutputType         string                      `json:"output_type" validate:"skip"`
	OutputConstraints  userstore.ColumnConstraints `json:"output_type_constraints" validate:"skip"`
	ReuseExistingToken bool                        `json:"reuse_existing_token" validate:"skip" description:"Specifies if the tokenizing transformer should return existing token instead of creating a new one."`
	TransformType      TransformType               `json:"transform_type" required:"true"`
	TagIDs             uuidarray.UUIDArray         `json:"tag_ids" validate:"skip"`
	Function           string                      `json:"function" required:"true"`
	Parameters         string                      `json:"parameters"`
	Version            int                         `json:"version"`
	IsSystem           bool                        `json:"is_system" description:"Whether this transformer is a system transformer. System transformers cannot be deleted or modified. This property cannot be changed."`
}

//go:generate genvalidate Transformer

// IsPolicyRequiredForExecution checks the transformation type and returns if an access policy is required to execute the transformer
func (g Transformer) IsPolicyRequiredForExecution() bool {
	return g.TransformType == TransformTypeTokenizeByValue || g.TransformType == TransformTypeTokenizeByReference
}

func (g Transformer) extraValidate() error {

	if !validIdentifier.MatchString(string(g.Name)) {
		return ucerr.Friendlyf(nil, `Transformer name "%s" has invalid characters`, g.Name)
	}

	params := map[string]any{}
	if err := json.Unmarshal([]byte(g.Parameters), &params); g.Parameters != "" && err != nil {
		paramsArr := []any{}
		if err := json.Unmarshal([]byte(g.Parameters), &paramsArr); err != nil {
			return ucerr.New("Transformer.Parameters must be either empty, or a JSON dictionary or JSON array")
		}
	}

	if g.ReuseExistingToken && g.TransformType != TransformTypeTokenizeByValue && g.TransformType != TransformTypeTokenizeByReference {
		return ucerr.Friendlyf(nil, "ReuseExistingToken can only be true for tokenization transformers")
	}

	return nil
}

// UserstoreDataProvenance is used by TransformTypeTokenizeByReference to describe the provenance of the data
type UserstoreDataProvenance struct {
	UserID   uuid.UUID `json:"user_id" validate:"notnil"`
	ColumnID uuid.UUID `json:"column_id" validate:"notnil"`
}

//go:generate genvalidate UserstoreDataProvenance

// PolicyType describes the type of an access policy
type PolicyType string //revive:disable-line:exported

const (
	// PolicyTypeInvalid is an invalid policy type
	PolicyTypeInvalid PolicyType = "invalid"

	// PolicyTypeCompositeAnd is the type for composite policies in which all components must be satisfied to grant access
	PolicyTypeCompositeAnd = "composite_and"

	// PolicyTypeCompositeOr is the type for composite policies in which any component must be satisfied to grant access
	PolicyTypeCompositeOr = "composite_or"
)

//go:generate genconstant PolicyType

// AccessPolicyTemplate describes a template for an access policy
type AccessPolicyTemplate struct {
	ucdb.SystemAttributeBaseModel `validate:"skip"`

	Name        string `json:"name" validate:"length:1,128" required:"true"`
	Description string `json:"description"`
	Function    string `json:"function" required:"true"`
	Version     int    `json:"version"`
}

func (a AccessPolicyTemplate) extraValidate() error {
	if !validIdentifier.MatchString(string(a.Name)) {
		return ucerr.Friendlyf(nil, `Access policy template name "%s" has invalid characters`, a.Name)
	}
	return nil
}

//go:generate genvalidate AccessPolicyTemplate

// EqualsIgnoringNilID returns true if the two templates are equal, ignoring the description, version, and ID if one is nil
func (a AccessPolicyTemplate) EqualsIgnoringNilID(other AccessPolicyTemplate) bool {
	return (a.ID == other.ID || a.ID.IsNil() || other.ID.IsNil()) &&
		strings.EqualFold(a.Name, other.Name) &&
		a.Function == other.Function &&
		a.IsSystem == other.IsSystem
}

// AccessPolicyComponent is either an access policy a template paired with parameters to fill it with
type AccessPolicyComponent struct {
	Policy             *userstore.ResourceID `json:"policy,omitempty"`
	Template           *userstore.ResourceID `json:"template,omitempty"`
	TemplateParameters string                `json:"template_parameters,omitempty"`
}

// Validate implements Validateable
func (a AccessPolicyComponent) Validate() error {
	policyValidErr := a.Policy.Validate()
	templateValidErr := a.Template.Validate()
	if (policyValidErr != nil && templateValidErr != nil) || (policyValidErr == nil && templateValidErr == nil) {
		return ucerr.New("AccessPolicyComponent must have either a Policy or a Template specified, but not both")
	}

	if templateValidErr == nil {
		params := map[string]any{}
		if err := json.Unmarshal([]byte(a.TemplateParameters), &params); a.TemplateParameters != "" && err != nil {
			return ucerr.New("AccessPolicyComponent.Parameters must be either empty, or a JSON dictionary")
		}
	} else if a.TemplateParameters != "" {
		return ucerr.New("AccessPolicyComponent.Parameters must be empty when a Policy is specified")
	}

	return nil
}

// AccessPolicyThresholds describes the thresholds for an access policy
type AccessPolicyThresholds struct {
	AnnounceMaxExecutionFailure bool `json:"announce_max_execution_failure" description:"If true, we return '429 Too Many Requests' if max_executions is exceeded for the past max_execution_duration_seconds seconds."`
	AnnounceMaxResultFailure    bool `json:"announce_max_result_failure" description:"If true, we return '400 Bad Request' if the action would involve more than max_results_per_execution results."`
	MaxExecutions               int  `json:"max_executions" description:"If non-zero, specifies the maximum number of executions for the past max_execution_duration_seconds seconds."`
	MaxExecutionDurationSeconds int  `json:"max_execution_duration_seconds" description:"Specifies the duration in seconds over which we limit the max executions. If max_executions is non-zero, this value must be between 5 and 60, inclusive."`
	MaxResultsPerExecution      int  `json:"max_results_per_execution" description:"If non-zero, specifies the max number of results that an action can involve."`
}

// AccessPolicy describes an access policy
type AccessPolicy struct {
	ID              uuid.UUID           `json:"id" validate:"skip"`
	Name            string              `json:"name" validate:"length:1,128" required:"true"`
	Description     string              `json:"description"`
	PolicyType      PolicyType          `json:"policy_type" required:"true"`
	TagIDs          uuidarray.UUIDArray `json:"tag_ids" validate:"skip"`
	Version         int                 `json:"version"`
	IsSystem        bool                `json:"is_system" description:"Whether this policy is a system policy. System policies cannot be deleted or modified. This property cannot be changed."`
	IsAutogenerated bool                `json:"is_autogenerated" description:"Whether this policy is autogenerated from an accessor or mutator."`

	Components []AccessPolicyComponent `json:"components" validate:"skip"`

	RequiredContext map[string]string      `json:"required_context" validate:"skip" description:"What context is required for this policy to be executed"`
	Thresholds      AccessPolicyThresholds `json:"thresholds" validate:"skip" description:"Execution thresholds for users of this access policy"`
}

// EqualsIgnoringNilID returns true if the two policies are equal, ignoring the description, version, and ID if one is nil
func (a AccessPolicy) EqualsIgnoringNilID(other AccessPolicy) bool {
	if (a.ID == other.ID || a.ID.IsNil() || other.ID.IsNil()) &&
		strings.EqualFold(a.Name, other.Name) &&
		strings.EqualFold(a.Description, other.Description) &&
		a.PolicyType == other.PolicyType &&
		a.Thresholds == other.Thresholds &&
		a.IsSystem == other.IsSystem {
		if len(a.Components) == len(other.Components) {
			// for components, we assume the order is the same, and that all UUIDs are populated in the resource IDs
			for i, c := range a.Components {
				if (c.Policy == nil && other.Components[i].Policy != nil) || (c.Policy != nil && other.Components[i].Policy == nil) {
					return false
				}
				if c.Policy != nil && other.Components[i].Policy != nil {
					if c.Policy.ID != other.Components[i].Policy.ID {
						return false
					}
				}
				if (c.Template == nil && other.Components[i].Template != nil) || (c.Template != nil && other.Components[i].Template == nil) {
					return false
				}
				if c.Template != nil && other.Components[i].Template != nil {
					if c.Template.ID != other.Components[i].Template.ID {
						return false
					}
				}
				if c.TemplateParameters != other.Components[i].TemplateParameters {
					return false
				}
			}
			return true
		}
	}
	return false
}

//go:generate genvalidate AccessPolicy

func (a AccessPolicy) extraValidate() error {

	if !validIdentifier.MatchString(string(a.Name)) {
		return ucerr.Friendlyf(nil, `Access policy name "%s" has invalid characters`, a.Name)
	}

	if len(a.Components) == 0 {
		return ucerr.New("AccessPolicy must have at least one component")
	}

	return nil
}

// ClientContext is passed by the client at resolution time
type ClientContext map[string]any

// AccessPolicyContext gets passed to the access policy's function(context, params) at resolution time
type AccessPolicyContext struct {
	Server       ServerContext     `json:"server"`
	Client       ClientContext     `json:"client"`
	User         userstore.Record  `json:"user,omitempty"`
	Query        map[string]string `json:"query,omitempty"`
	RowData      map[string]string `json:"row_data,omitempty"`
	ConnectionID uuid.UUID         `json:"connection_id,omitempty"`
}

// ServerContext is automatically injected by the server at resolution time
type ServerContext struct {
	// TODO: add token creation time
	IPAddress    string         `json:"ip_address"`
	Action       Action         `json:"action"`
	PurposeNames []string       `json:"purpose_names"`
	Claims       map[string]any `json:"claims"`
}

// Action identifies the reason access policy is being invoked
type Action string

// Different reasons for running access policy
const (
	ActionResolve Action = "Resolve"
	ActionInspect Action = "Inspect"
	ActionLookup  Action = "Lookup"
	ActionDelete  Action = "Delete"
	ActionExecute Action = "Execute" // TODO: should this be a unique action?
)

// Secret describes a secret that can be used in access policy templates and transformers
type Secret struct {
	ID      uuid.UUID `json:"id" validate:"notnil"`
	Name    string    `json:"name" validate:"length:1,128" required:"true"`
	Value   string    `json:"value" validate:"skip" required:"true"`
	Created int64     `json:"created" validate:"skip"`
}

//go:generate genvalidate Secret
