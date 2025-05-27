package tokenizer

import (
	"slices"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucerr"
)

// CreateTokenRequest is all the data needed to create a token
type CreateTokenRequest struct {
	Data string `json:"data"`

	TransformerRID  userstore.ResourceID `json:"transformer_rid"`
	AccessPolicyRID userstore.ResourceID `json:"access_policy_rid"`
}

//go:generate genvalidate CreateTokenRequest

// CreateTokenResponse is the response to a CreateToken call
type CreateTokenResponse struct {
	Token string `json:"data"`
}

// ResolveTokensRequest is the data needed to resolve a token
type ResolveTokensRequest struct {
	Tokens   []string               `json:"tokens"`
	Context  policy.ClientContext   `json:"context"`
	Purposes []userstore.ResourceID `json:"purposes"`
}

// Validate implements Validateable
func (r ResolveTokensRequest) Validate() error {

	if slices.Contains(r.Tokens, "") {
		return ucerr.New("token can't be empty")
	}
	return nil
}

// ResolveTokenResponse is the response to a ResolveToken call
type ResolveTokenResponse struct {
	Data  string `json:"data"`
	Token string `json:"token"` // include this in case it's helpful for correlating later?
}

// InspectTokenRequest contains the data required to inspect a token
type InspectTokenRequest struct {
	Token string `json:"token" validate:"notempty"`
}

//go:generate genvalidate InspectTokenRequest

// InspectTokenResponse contains the data returned by an InspectToken call
type InspectTokenResponse struct {
	Token string `json:"token"`

	ID      uuid.UUID `json:"id"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`

	AccessPolicy policy.AccessPolicy `json:"access_policy"`
	Transformer  policy.Transformer  `json:"transformer"`
}

// LookupTokensRequest contains the data required to lookup a token
type LookupTokensRequest struct {
	Data string `json:"data"`

	TransformerRID  userstore.ResourceID `json:"transformer_rid"`
	AccessPolicyRID userstore.ResourceID `json:"access_policy_rid"`
}

//go:generate genvalidate LookupTokensRequest

// LookupTokensResponse contains the data returned by a LookupToken call
type LookupTokensResponse struct {
	Tokens []string `json:"tokens"` // note that a single piece of data could tokenize many ways
}

// LookupOrCreateTokensRequest contains the data required to lookup or create tokens in bulk
type LookupOrCreateTokensRequest struct {
	Data []string `json:"data" validate:"skip"`

	TransformerRIDs  []userstore.ResourceID `json:"transformer_rids"`
	AccessPolicyRIDs []userstore.ResourceID `json:"access_policy_rids"`
}

func (l *LookupOrCreateTokensRequest) extraValidate() error {
	if len(l.Data) != len(l.TransformerRIDs) || len(l.Data) != len(l.AccessPolicyRIDs) {
		return ucerr.New("data, transformer_rid, and access_policy_rid must be the same length")
	}
	return nil
}

//go:generate genvalidate LookupOrCreateTokensRequest

// LookupOrCreateTokensResponse contains the data returned by a LookupOrCreateTokens call
type LookupOrCreateTokensResponse struct {
	Tokens []string `json:"tokens"`
}
