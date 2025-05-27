package tokenizer

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp/internal"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/internal/userstore/purposehelpers"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/tokenizer"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/multitenant"
)

// how many times do we let transformer try to produce a unique token?
const (
	maxTokenUniquenessTries = 5
	maxTokenBatch           = 1000
)

func getAccessPolicyForResourceID(ctx context.Context, s *storage.Storage, accessPolicyRID userstore.ResourceID) (*storage.AccessPolicy, error) {
	if accessPolicyRID.ID != uuid.Nil {
		ap, err := s.GetLatestAccessPolicy(ctx, accessPolicyRID.ID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if accessPolicyRID.Name != "" && ap.Name != accessPolicyRID.Name {
			return nil, ucerr.Errorf("access policy name %s does not match ID %s", accessPolicyRID.Name, accessPolicyRID.ID)
		}
		return ap, nil
	}

	ap, err := s.GetAccessPolicyByName(ctx, accessPolicyRID.Name)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return ap, nil
}

func getTransformerForResourceID(ctx context.Context, s *storage.Storage, transformer userstore.ResourceID) (*storage.Transformer, error) {
	transformers, err := storage.GetTransformerMapForResourceIDs(ctx, s, true, transformer)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if transformer.ID != uuid.Nil {
		tf, err := transformers.ForID(transformer.ID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		return tf, nil
	}
	tf, err := transformers.ForName(transformer.Name)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return tf, nil
}

// OpenAPI Summary: Create Token
// OpenAPI Tags: Tokens
// OpenAPI Description: This endpoint creates a token for a piece of data. CreateToken will always generate a unique token. If you want to reuse a token that was already generated, use LookupToken.
func (h handler) createToken(ctx context.Context, req tokenizer.CreateTokenRequest) (*tokenizer.CreateTokenResponse, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	ap, err := getAccessPolicyForResourceID(ctx, s, req.AccessPolicyRID)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	transformer, err := getTransformerForResourceID(ctx, s, req.TransformerRID)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	if transformer.TransformType.ToClient() != policy.TransformTypeTokenizeByValue {
		return nil, http.StatusBadRequest, nil, ucerr.Errorf("invalid transformer type %s", transformer.TransformType)
	}

	token, _, err := ExecuteTransformer(ctx, s, authzClient, ap.ID, transformer, req.Data, nil)
	if err != nil {
		logTransformerError(ctx, req.TransformerRID.ID, transformer.Version)
		if ucdb.IsUniqueViolation(err) {
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		}

		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &tokenizer.CreateTokenResponse{
		Token: token,
	}, http.StatusCreated, nil, nil
}

type accessPolicyStatus int

const (
	accessPolicyStatusSucceeded     accessPolicyStatus = 0
	accessPolicyStatusRateLimited   accessPolicyStatus = 1
	accessPolicyStatusFailed        accessPolicyStatus = 2
	accessPolicyStatusResultLimited accessPolicyStatus = 3
)

type accessPolicyInfo struct {
	thresholdAP    *storage.AccessPolicy
	status         accessPolicyStatus
	totalEvaluated int
}

func newAccessPolicyInfo(
	ctx context.Context,
	s *storage.Storage,
	authzClient *authz.Client,
	apc policy.AccessPolicyContext,
	accessPolicyID uuid.UUID,
) (*accessPolicyInfo, error) {
	_, tokenAP, thresholdAP, err :=
		s.GetAccessPolicies(
			ctx,
			multitenant.MustGetTenantState(ctx).ID,
			policy.AccessPolicyAllowAll.ID,
			accessPolicyID,
		)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	status := accessPolicyStatusSucceeded

	// token resolution is identified for rate limiting via a well-known sentinel UUID
	sentinelTokenResolutionID := uuid.Must(uuid.FromString("416ad6d5-1062-4043-b0be-e583d0c843fb"))

	allowed, err := thresholdAP.CheckRateThreshold(ctx, s, apc, sentinelTokenResolutionID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if allowed {
		allowed, _, err = ExecuteAccessPolicy(ctx, tokenAP, apc, authzClient, s)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if !allowed {
			status = accessPolicyStatusFailed
		}
	} else {
		status = accessPolicyStatusRateLimited
	}

	return &accessPolicyInfo{
			thresholdAP: thresholdAP,
			status:      status,
		},
		nil
}

func (api *accessPolicyInfo) incrementAccessCount() {
	if api.status == accessPolicyStatusSucceeded {
		api.totalEvaluated++
		if !api.thresholdAP.CheckResultThreshold(api.totalEvaluated) {
			api.status = accessPolicyStatusResultLimited
		}
	}
}

// OpenAPI Summary: Resolve Token
// OpenAPI Tags: Tokens
// OpenAPI Description: This endpoint receives a list of tokens, applies the associated access policy for each token, and returns the associated token data if the conditions of the access policy are met.
func (h handler) resolveToken(
	ctx context.Context,
	req tokenizer.ResolveTokensRequest,
) (resp []tokenizer.ResolveTokenResponse, code int, auditLogEntries []auditlog.Entry, err error) {
	if len(req.Tokens) > maxTokenBatch {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "Too many tokens provided")
	}

	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	apc := BuildBaseAPContext(ctx, req.Context, policy.ActionResolve)

	s := storage.MustCreateStorage(ctx)
	tokenRecords, err := s.ListTokenRecordsByTokens(ctx, req.Tokens)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	apis := map[uuid.UUID]*accessPolicyInfo{}
	tokenRecordsByToken := make(map[string]storage.TokenRecord, len(tokenRecords))

	for _, tr := range tokenRecords {
		if _, found := tokenRecordsByToken[tr.Token]; found {
			continue
		}
		tokenRecordsByToken[tr.Token] = tr

		api, found := apis[tr.AccessPolicyID]
		if !found {
			api, err = newAccessPolicyInfo(ctx, s, authzClient, apc, tr.AccessPolicyID)
			if err != nil {
				return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
			}
			apis[tr.AccessPolicyID] = api
		}
		api.incrementAccessCount()
	}

	resp = make([]tokenizer.ResolveTokenResponse, 0, len(req.Tokens))
	failedTokens := []string{}
	rateLimitedTokens := []string{}
	resolvedTokens := []string{}
	resultLimitedTokens := []string{}

	for _, token := range req.Tokens {
		tr, found := tokenRecordsByToken[token]
		if !found {
			failedTokens = append(failedTokens, token)
			resp = append(resp, tokenizer.ResolveTokenResponse{Token: token})
			continue
		}

		data := ""

		api := apis[tr.AccessPolicyID]
		if api.status == accessPolicyStatusSucceeded {
			if tr.UserID.IsNil() {
				data = tr.Data
			} else if data, code, err = getUserColumnValue(ctx, tr.UserID, tr.ColumnID, req.Purposes); err != nil {
				switch code {
				case http.StatusNotFound:
					return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
				default:
					return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
				}
			}
		}

		if data != "" {
			resolvedTokens = append(resolvedTokens, token)
		} else {
			switch api.status {
			case accessPolicyStatusRateLimited:
				rateLimitedTokens = append(rateLimitedTokens, token)
			case accessPolicyStatusResultLimited:
				resultLimitedTokens = append(resultLimitedTokens, token)
			default:
				failedTokens = append(failedTokens, token)
			}
		}

		resp = append(resp, tokenizer.ResolveTokenResponse{Data: data, Token: token})
	}

	return resp,
		http.StatusOK,
		auditlog.NewEntryArray(
			auth.GetAuditLogActor(ctx),
			internal.AuditLogEventTypeResolveToken,
			auditlog.Payload{
				"AccessPolicyContext": apc,
				"Name":                "Tokenizer",
				"TokensFail":          failedTokens,
				"TokensRateLimited":   rateLimitedTokens,
				"TokensResultLimited": resultLimitedTokens,
				"TokensSuccess":       resolvedTokens,
			},
		),
		nil
}

func getUserColumnValue(ctx context.Context, userID uuid.UUID, columnID uuid.UUID, purposes []userstore.ResourceID) (string, int, error) {

	ts := multitenant.MustGetTenantState(ctx)
	s := storage.NewFromTenantState(ctx, ts)

	cm, err := storage.NewUserstoreColumnManager(ctx, s)
	if err != nil {
		return "", http.StatusInternalServerError, ucerr.Wrap(err)
	}

	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return "", http.StatusInternalServerError, ucerr.Wrap(err)
	}

	umrs := storage.NewUserMultiRegionStorage(ctx, ts.UserRegionDbMap, ts.ID)
	user, _, code, err := umrs.GetUser(ctx, cm, dtm, userID, false)
	if err != nil {
		return "", code, ucerr.Wrap(err)
	}

	c, err := s.GetColumn(ctx, columnID)
	if err != nil {
		return "", uchttp.SQLReadErrorMapper(err), ucerr.Wrap(err)
	}

	dt, err := s.GetDataType(ctx, c.DataTypeID)
	if err != nil {
		return "", http.StatusInternalServerError, ucerr.Wrap(err)
	}

	var value any

	if c.Attributes.System {
		value = user.Profile[c.Name]
	} else {
		allPurposes, err := s.ListPurposesNonPaginated(ctx)
		if err != nil {
			return "", http.StatusInternalServerError, ucerr.Wrap(err)
		}

		purposeIDs := set.NewUUIDSet()
		for _, purpose := range purposes {
			found := false
			for _, p := range allPurposes {
				if p.Name == purpose.Name || p.ID == purpose.ID {
					purposeIDs.Insert(p.ID)
					found = true
					break
				}
			}
			if !found {
				return "", http.StatusNotFound, ucerr.Friendlyf(nil, "purpose not found %+v", purpose)
			}
		}

		columnConsented, consentedValue, err := purposehelpers.CheckPurposeIDs(ctx, c, dt, purposeIDs, user.Profile[c.Name], user.ProfileConsentedPurposeIDs)
		if err != nil {
			return "", http.StatusInternalServerError, ucerr.Wrap(err)
		}

		if columnConsented {
			value = consentedValue
		} else {
			value = nil
		}
	}

	var cv column.Value
	if err := cv.Set(*dt, c.Attributes.Constraints, c.IsArray, value); err != nil {
		return "", http.StatusInternalServerError, ucerr.Wrap(err)
	}

	ret, err := cv.GetString(ctx)
	if err != nil {
		return "", http.StatusInternalServerError, ucerr.Wrap(err)
	}

	return ret, http.StatusOK, nil
}

type deleteTokenParams struct {
	Token *string `description:"ID of token to be deleted" query:"token"`
}

// OpenAPI Summary: Delete Token
// OpenAPI Tags: Tokens
// OpenAPI Description: This endpoint deletes a token by ID.
func (h handler) deleteToken(ctx context.Context, req deleteTokenParams) (int, []auditlog.Entry, error) {

	if req.Token == nil || *req.Token == "" {
		return http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "token is required")
	}

	s := storage.MustCreateStorage(ctx)

	tr, err := s.GetTokenRecordByToken(ctx, *req.Token)
	if err != nil {
		return http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	// TODO: auth in some way? should this be described in access policy?

	if err := s.DeleteTokenRecord(ctx, tr.ID); err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return http.StatusNoContent,
		auditlog.NewEntryArray(
			auth.GetAuditLogActor(ctx),
			internal.AuditLogEventTypeDeleteToken,
			auditlog.Payload{
				"Name":          "Tokenizer",
				"TokensSuccess": []string{*req.Token},
				"TokensFail":    []string{},
			},
		),
		nil
}

// OpenAPI Summary: Inspect Token
// OpenAPI Tags: Tokens
// OpenAPI Description: This endpoint gets a token. It is a primarily a debugging API that allows you to query a token without resolving it.
func (h handler) inspectToken(ctx context.Context, req tokenizer.InspectTokenRequest) (*tokenizer.InspectTokenResponse, int, []auditlog.Entry, error) {

	// TODO: auth in some way? in access policies?
	s := storage.MustCreateStorage(ctx)

	tr, err := s.GetTokenRecordByToken(ctx, req.Token)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	clientTransformer := policy.Transformer{}
	if transformer, err := s.GetTransformerByVersion(ctx, tr.TransformerID, tr.TransformerVersion); err == nil {
		dtm, err := storage.NewDataTypeManager(ctx, s)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		clientTransformer = transformer.ToClientModel()
		if err := validateAndPopulateTransformerFields(dtm, &clientTransformer); err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	} else {
		uclog.Errorf(ctx, "Failed to get transformer %s version %d for token, possibly deleted: %v", tr.TransformerID, tr.TransformerVersion, err)
	}

	ap, err := s.GetLatestAccessPolicy(ctx, tr.AccessPolicyID)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	resp := tokenizer.InspectTokenResponse{
		Token:        req.Token,
		ID:           tr.ID,
		Created:      tr.Created,
		Updated:      tr.Updated,
		Transformer:  clientTransformer,
		AccessPolicy: *ap.ToClientModel(),
	}

	return &resp,
		http.StatusOK,
		auditlog.NewEntryArray(
			auth.GetAuditLogActor(ctx),
			internal.AuditLogEventTypeInspectToken,
			auditlog.Payload{
				"Name":          "Tokenizer",
				"TokensSuccess": []string{req.Token},
				"TokensFail":    []string{},
			},
		),
		nil
}

// OpenAPI Summary: Lookup Token
// OpenAPI Tags: Tokens
// OpenAPI Description: This endpoint helps you re-use existing tokens. It receives a piece of data and an access policy. It returns existing tokens that match across the full set of parameters. If no token matches, an error is returned.
func (h handler) lookupTokens(ctx context.Context, req tokenizer.LookupTokensRequest) (*tokenizer.LookupTokensResponse, int, []auditlog.Entry, error) {

	s := storage.MustCreateStorage(ctx)

	// we still need to allow passing in by ID or function/params
	ap, err := getAccessPolicyForResourceID(ctx, s, req.AccessPolicyRID)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	gp, err := getTransformerForResourceID(ctx, s, req.TransformerRID)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	trs, err := s.ListTokenRecordsByDataAndPolicy(ctx, req.Data, gp.ID, ap.ID)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	tokens := make([]string, 0, len(trs))
	for _, tr := range trs {
		tokens = append(tokens, tr.Token)
	}

	return &tokenizer.LookupTokensResponse{Tokens: tokens},
		http.StatusOK,
		auditlog.NewEntryArray(
			auth.GetAuditLogActor(ctx),
			internal.AuditLogEventTypeLookupToken,
			auditlog.Payload{
				"Name":          "Tokenizer",
				"TokensSuccess": tokens,
				"TokensFail":    []string{},
			},
		),
		nil
}

// OpenAPI Summary: Lookup Or Create Tokens
// OpenAPI Tags: Tokens
// OpenAPI Description: This endpoint helps you re-use existing tokens by only creating new tokens when they don't exist already.
func (h handler) lookupOrCreateTokens(ctx context.Context, req tokenizer.LookupOrCreateTokensRequest) (*tokenizer.LookupOrCreateTokensResponse, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)

	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	// Validate the transformers and create a list of just the UUIDs
	transformerIDMap := map[uuid.UUID]*storage.Transformer{}
	transformerNameMap := map[string]*storage.Transformer{}
	transformerIDs := []uuid.UUID{}

	for _, tRID := range req.TransformerRIDs {
		if tRID.ID != uuid.Nil {
			if foundTransformer, ok := transformerIDMap[tRID.ID]; ok {
				if tRID.Name == "" || tRID.Name == foundTransformer.Name {
					transformerIDs = append(transformerIDs, foundTransformer.ID)
					continue
				} else {
					return nil, http.StatusBadRequest, nil, ucerr.Errorf("access policy name %s does not match ID %s", tRID.Name, tRID.ID)
				}
			}
		} else {
			if foundTransformer, ok := transformerNameMap[tRID.Name]; ok {
				transformerIDs = append(transformerIDs, foundTransformer.ID)
				continue
			}
		}

		dbTransformer, err := getTransformerForResourceID(ctx, s, tRID)
		if err != nil {
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}
		transformerIDMap[dbTransformer.ID] = dbTransformer
		transformerNameMap[dbTransformer.Name] = dbTransformer
		transformerIDs = append(transformerIDs, dbTransformer.ID)
	}

	// Validate the access policies and create a list of just the UUIDs
	accessPolicyIDMap := map[uuid.UUID]*storage.AccessPolicy{}
	accessPolicyNameMap := map[string]*storage.AccessPolicy{}
	accessPolicyIDs := []uuid.UUID{}

	for _, apRID := range req.AccessPolicyRIDs {
		if apRID.ID != uuid.Nil {
			if foundAP, ok := accessPolicyIDMap[apRID.ID]; ok {
				if apRID.Name == "" || apRID.Name == foundAP.Name {
					accessPolicyIDs = append(accessPolicyIDs, foundAP.ID)
					continue
				} else {
					return nil, http.StatusBadRequest, nil, ucerr.Errorf("access policy name %s does not match ID %s", apRID.Name, apRID.ID)
				}
			}
		} else {
			if foundAP, ok := accessPolicyNameMap[apRID.Name]; ok {
				accessPolicyIDs = append(accessPolicyIDs, foundAP.ID)
				continue
			}
		}

		dbAP, err := getAccessPolicyForResourceID(ctx, s, apRID)
		if err != nil {
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}
		accessPolicyIDMap[dbAP.ID] = dbAP
		accessPolicyNameMap[dbAP.Name] = dbAP
		accessPolicyIDs = append(accessPolicyIDs, dbAP.ID)
	}

	// Lookup tokens for each piece of data passed in
	tokens, err := s.BatchListTokensByDataAndPolicy(ctx, req.Data, transformerIDs, accessPolicyIDs)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	successTokens := []string{}

	// Create tokens for any data that didn't have a token
	for i, token := range tokens {
		if token == "" {
			transformer := transformerIDMap[transformerIDs[i]]
			if transformer.TransformType.ToClient() != policy.TransformTypeTokenizeByValue {
				return nil, http.StatusBadRequest, nil, ucerr.Errorf("invalid transformer type %s", transformer.TransformType)
			}

			ap := accessPolicyIDMap[accessPolicyIDs[i]]

			token, _, err := ExecuteTransformer(ctx, s, authzClient, ap.ID, transformer, req.Data[i], nil)
			if err != nil {
				logTransformerError(ctx, transformerIDs[i], transformer.Version)
				if ucdb.IsUniqueViolation(err) {
					return nil, http.StatusConflict, nil, ucerr.Wrap(err)
				}

				return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
			}

			tokens[i] = token

		} else {
			successTokens = append(successTokens, token)
		}
	}

	return &tokenizer.LookupOrCreateTokensResponse{Tokens: tokens},
		http.StatusOK,
		auditlog.NewEntryArray(
			auth.GetAuditLogActor(ctx),
			internal.AuditLogEventTypeLookupToken,
			auditlog.Payload{
				"Name":          "Tokenizer",
				"TokensSuccess": successTokens,
				"TokensFail":    []string{},
			},
		),
		nil
}
