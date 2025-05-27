package tokenizer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/idp/events"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/internal"
	"userclouds.com/idp/internal/storage"
	internalTokenizer "userclouds.com/idp/internal/tokenizer"
	"userclouds.com/idp/paths"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/tokenizer"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/middleware"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/request"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
	"userclouds.com/test/testlogtransport"
)

func doRequest(t *testing.T, h *uchttp.ServeMux, method string, path string, body any, authToken, host string) *httptest.ResponseRecorder {
	reader := uctest.IOReaderFromJSONStruct(t, body)
	r := httptest.NewRequest(method, path, reader)
	r.Host = host
	r.Header.Set("Authorization", authToken)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	return rr
}
func createTransformerHelper(t *testing.T, tr policy.Transformer, h *uchttp.ServeMux, hostname string, authToken string) policy.Transformer {
	rr := doRequest(t, h, http.MethodPost, paths.CreateTransformer, tokenizer.CreateTransformerRequest{Transformer: tr}, authToken, hostname)
	assert.Equal(t, rr.Code, http.StatusCreated)
	var transformer policy.Transformer
	assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &transformer))
	return transformer
}

func createAccessPolicyHelper(t *testing.T, ap policy.AccessPolicy, h *uchttp.ServeMux, hostname string, authToken string) policy.AccessPolicy {
	rr := doRequest(t, h, http.MethodPost, paths.CreateAccessPolicy, tokenizer.CreateAccessPolicyRequest{AccessPolicy: ap}, authToken, hostname)
	assert.Equal(t, rr.Code, http.StatusCreated)
	var accessPolicy policy.AccessPolicy
	assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &accessPolicy))
	return accessPolicy
}

func createAccessPolicyTemplateHelper(t *testing.T, apt policy.AccessPolicyTemplate, h *uchttp.ServeMux, hostname string, authToken string) policy.AccessPolicyTemplate {
	rr := doRequest(t, h, http.MethodPost, paths.CreateAccessPolicyTemplate, tokenizer.CreateAccessPolicyTemplateRequest{AccessPolicyTemplate: apt}, authToken, hostname)
	assert.Equal(t, rr.Code, http.StatusCreated)
	var accessPolicyTemplate policy.AccessPolicyTemplate
	assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &accessPolicyTemplate))
	return accessPolicyTemplate
}

func TestTokenHandler(t *testing.T) {
	ctx := context.Background()
	_, tenant, _, tenantDB, _, ts := testhelpers.CreateTestServer(ctx, t)
	jwtVerifier := uctest.JWTVerifier{}
	mw := middleware.Chain(
		multitenant.Middleware(ts),
		auth.Middleware(jwtVerifier, uuid.Nil),
		request.Middleware(),
	)
	m2mAuth, err := m2m.GetM2MTokenSource(ctx, tenant.ID)
	assert.NoErr(t, err)
	consoleTenantInfo := companyconfig.TenantInfo{CompanyID: tenant.CompanyID, TenantID: tenant.ID, TenantURL: tenant.TenantURL}
	assert.NoErr(t, err)
	// wrap everything in "/tokenizer" so we can use the same paths
	apiHandler, err := internalTokenizer.NewHandler(m2mAuth, consoleTenantInfo, false)
	assert.NoErr(t, err)
	h := builder.NewHandlerBuilder().
		Handle(fmt.Sprintf("%s/", paths.TokenizerBasePath), mw.Apply(apiHandler)).
		Build()

	hostname := tenant.GetHostName()
	s := idptesthelpers.NewStorage(ctx, t, tenantDB, tenant.ID)

	// parameterized to take username per test so we can check audit log entries well
	createAuthToken := func(username string) string {
		return fmt.Sprintf("Bearer %s", uctest.CreateJWT(t,
			oidc.UCTokenClaims{
				StandardClaims: oidc.StandardClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: username}}},
			tenant.TenantURL))
	}
	checkAuditLog := func(t *testing.T, et auditlog.EventType, startTime time.Time, username string, expected int, success int, fail int) {
		var successCount int
		var failCount int
		for _, e := range testhelpers.CheckAuditLog(ctx, t, tenantDB, et, username, startTime, expected) {
			uclog.Debugf(ctx, "%v", e.Payload)
			ts, ok := e.Payload["TokensSuccess"]
			assert.True(t, ok)
			successCount += len(ts.([]any))
			fs, ok := e.Payload["TokensFail"]
			assert.True(t, ok)
			failCount += len(fs.([]any))
		}
		assert.Equal(t, successCount, success)
		assert.Equal(t, failCount, fail)
	}

	t.Run("TestCreate", func(t *testing.T) {
		tt := testlogtransport.InitLoggerAndTransportsForTests(t)

		authToken := createAuthToken("create")

		transformer := createTransformerHelper(t, policy.Transformer{
			Name:           "Transformer_2",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTokenizeByValue,
			Function: `function sleep(ms) {
						return new Promise(resolve => setTimeout(resolve, ms));
					}
					async function uuidv4() {
					await sleep(1);
					return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
						var r = Math.random() * 16 | 0, v = c == 'x' ? r : (r & 0x3 | 0x8);
						return v.toString(16);
					});
				};
				function transform(data, params) {
					return JSON.stringify(uuidv4());
				};`,
		}, h, hostname, authToken)

		apt := createAccessPolicyTemplateHelper(t, policy.AccessPolicyTemplate{
			Name:     "Template_2",
			Function: "function policy(x, y) { return true; }",
		}, h, hostname, authToken)

		ap := createAccessPolicyHelper(t, policy.AccessPolicy{
			Name: "Policy_2",
			Components: []policy.AccessPolicyComponent{
				{Template: &userstore.ResourceID{ID: apt.ID}},
			},
			PolicyType: policy.PolicyTypeCompositeAnd,
		}, h, hostname, authToken)

		crt := tokenizer.CreateTokenRequest{
			Data:            `"create"`,
			TransformerRID:  userstore.ResourceID{ID: transformer.ID},
			AccessPolicyRID: userstore.ResourceID{ID: ap.ID},
		}

		rr := doRequest(t, h, http.MethodPost, paths.CreateToken, crt, authToken, hostname)
		assert.Equal(t, rr.Code, http.StatusCreated)
		var resp tokenizer.CreateTokenResponse
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))

		tr, err := s.GetTokenRecordByToken(ctx, resp.Token)
		assert.NoErr(t, err)

		assert.Equal(t, tr.Data, `"create"`)

		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryCall, events.TransformerPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryDuration, events.TransformerPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryInputError, events.TransformerPrefix, "", 0))), 0)

		// Put back in if we decide to add audit log event for token create
		// checkAuditLog(t, internal.AuditLogEventTypeCreateToken, "create", 1)
	})

	t.Run("TestCreateDuplicate", func(t *testing.T) {
		tt := testlogtransport.InitLoggerAndTransportsForTests(t)

		authToken := createAuthToken("dupe")

		transformer := createTransformerHelper(t, policy.Transformer{
			Name:           "Transformer_3",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTokenizeByValue,
			Function:       `function transform(data, params) { return 'foo'; }`,
		}, h, hostname, authToken)

		apt := createAccessPolicyTemplateHelper(t, policy.AccessPolicyTemplate{
			Name:     "Template_3",
			Function: "function policy(x, y) { return true;  }",
		}, h, hostname, authToken)

		ap := createAccessPolicyHelper(t, policy.AccessPolicy{
			Name: "Policy_3",
			Components: []policy.AccessPolicyComponent{
				{Template: &userstore.ResourceID{ID: apt.ID}},
			},
			PolicyType: policy.PolicyTypeCompositeAnd,
		}, h, hostname, authToken)

		ctr := tokenizer.CreateTokenRequest{
			Data:            `"create"`,
			TransformerRID:  userstore.ResourceID{ID: transformer.ID},
			AccessPolicyRID: userstore.ResourceID{ID: ap.ID}}

		rr := doRequest(t, h, http.MethodPost, paths.CreateToken, ctr, authToken, hostname)
		assert.Equal(t, rr.Code, http.StatusCreated)

		var resp tokenizer.CreateTokenResponse
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))

		tr, err := s.GetTokenRecordByToken(ctx, resp.Token)
		assert.NoErr(t, err)

		// create another one, which given our GP will be the same token
		ctr.Data = "createnew"
		rr = doRequest(t, h, http.MethodPost, paths.CreateToken, ctr, createAuthToken("dupe"), hostname)
		assert.Equal(t, rr.Code, http.StatusConflict)

		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryCall, events.TransformerPrefix, "", 0))), 2)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryDuration, events.TransformerPrefix, "", 0))), 2)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryInputError, events.TransformerPrefix, events.SubCategoryConflict, 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryInputError, events.TransformerPrefix, "", 0))), 1)

		// Put back in if we decide to add audit log event for token create
		// checkAuditLog(t, internal.AuditLogEventTypeCreateToken, "dupe", 1)
	})

	t.Run("TestResolve", func(t *testing.T) {

		tt := testlogtransport.InitLoggerAndTransportsForTests(t)
		startTime := time.Now().UTC()

		token := uuid.Must(uuid.NewV4())
		tokenBytes, err := json.Marshal(token)
		assert.NoErr(t, err)

		tr := &storage.TokenRecord{
			BaseModel:          ucdb.NewBase(),
			Data:               `"resolve"`,
			Token:              string(tokenBytes),
			TransformerID:      uuid.Must(uuid.NewV4()),
			TransformerVersion: 0,
			AccessPolicyID:     uuid.Must(uuid.NewV4()),
		}
		assert.IsNil(t, s.SaveTokenRecord(ctx, tr), assert.Must())

		apt := storage.AccessPolicyTemplate{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "Template_4",
			Function:                 "function policy(context, params) { return context.client.value }",
		}
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, &apt), assert.Must())

		ap := storage.AccessPolicy{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(tr.AccessPolicyID),
			Name:                     "Policy_4",
			ComponentIDs:             []uuid.UUID{apt.ID},
			ComponentParameters:      []string{""},
			ComponentTypes:           []int32{int32(storage.AccessPolicyComponentTypeTemplate)},
			PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
		}
		assert.IsNil(t, s.SaveAccessPolicy(ctx, &ap), assert.Must())

		req := tokenizer.ResolveTokensRequest{
			Tokens:  []string{string(tokenBytes)},
			Context: policy.ClientContext{"value": true},
		}
		rr := doRequest(t, h, http.MethodPost, paths.ResolveToken, req, createAuthToken("resolve"), hostname)
		assert.Equal(t, rr.Code, http.StatusOK)
		var resp []tokenizer.ResolveTokenResponse
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))

		assert.Equal(t, resp[0].Token, fmt.Sprintf(`"%s"`, token))
		assert.Equal(t, resp[0].Data, `"resolve"`)

		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryCall, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryDuration, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryResultSuccess, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryResultFailure, events.APPrefix, "", 0))), 0)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryInputError, events.APPrefix, "", 0))), 0)
		tt.ClearEvents()

		req = tokenizer.ResolveTokensRequest{
			Tokens:  []string{string(tokenBytes)},
			Context: policy.ClientContext{"value": false},
		}
		rr = doRequest(t, h, http.MethodPost, paths.ResolveToken, req, createAuthToken("resolve"), hostname)
		assert.Equal(t, rr.Code, http.StatusOK)
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))

		assert.Equal(t, resp[0].Token, fmt.Sprintf(`"%s"`, token))
		assert.Equal(t, resp[0].Data, "")

		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryCall, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryDuration, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryResultSuccess, events.APPrefix, "", 0))), 0)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryResultFailure, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryInputError, events.APPrefix, "", 0))), 0)

		// Delay for the async audit write to complete (fragile but yet we want to test the actual code path)
		time.Sleep(time.Millisecond * 100)
		checkAuditLog(t, internal.AuditLogEventTypeResolveToken, startTime, "resolve", 2, 1, 1)
	})

	t.Run("TestDelete", func(t *testing.T) {
		startTime := time.Now().UTC()

		token := uuid.Must(uuid.NewV4())
		tokenBytes, err := json.Marshal(token)
		assert.NoErr(t, err)

		tr := &storage.TokenRecord{
			BaseModel:          ucdb.NewBase(),
			Data:               `"delete"`,
			Token:              string(tokenBytes),
			TransformerID:      uuid.Must(uuid.NewV4()),
			TransformerVersion: 0,
			AccessPolicyID:     uuid.Must(uuid.NewV4()),
		}
		assert.IsNil(t, s.SaveTokenRecord(ctx, tr), assert.Must())

		// delete it
		rr := doRequest(t, h, http.MethodDelete, fmt.Sprintf("%s?token=%s", paths.DeleteToken, string(tokenBytes)), nil, createAuthToken("delete"), hostname)
		assert.Equal(t, rr.Code, http.StatusNoContent)

		resreq := tokenizer.ResolveTokensRequest{
			Tokens: []string{string(tokenBytes)},
		}
		rr = doRequest(t, h, http.MethodPost, paths.ResolveToken, resreq, createAuthToken("delete"), hostname)
		assert.Equal(t, rr.Code, http.StatusOK)

		// Delay for the async audit write to complete (fragile but yet we want to test the actual code path)
		time.Sleep(time.Millisecond * 100)
		checkAuditLog(t, internal.AuditLogEventTypeDeleteToken, startTime, "delete", 1, 1, 0)
	})

	t.Run("TestInspect", func(t *testing.T) {
		startTime := time.Now().UTC()

		tr := &storage.TokenRecord{
			BaseModel:          ucdb.NewBase(),
			Data:               `"resolve"`,
			Token:              `"placeholder"`,
			TransformerID:      uuid.Must(uuid.NewV4()),
			TransformerVersion: 0,
			AccessPolicyID:     uuid.Must(uuid.NewV4()),
		}
		assert.IsNil(t, s.SaveTokenRecord(ctx, tr), assert.Must())

		apt := storage.AccessPolicyTemplate{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "Template_1",
			Function:                 "function policy(x, y) { return true; /* TODO this defeats our uniqueness check */ }",
		}
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, &apt), assert.Must())

		ap := storage.AccessPolicy{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(tr.AccessPolicyID),
			Name:                     "Policy_1",
			ComponentIDs:             []uuid.UUID{apt.ID},
			ComponentParameters:      []string{""},
			ComponentTypes:           []int32{int32(storage.AccessPolicyComponentTypeTemplate)},
			PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
		}

		assert.IsNil(t, s.SaveAccessPolicy(ctx, &ap), assert.Must())

		// note that gp(data) != token, but that shouldn't matter for this test
		gp := &storage.Transformer{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(tr.TransformerID),
			Name:                     "Transformer_1",
			InputDataTypeID:          datatype.String.ID,
			OutputDataTypeID:         datatype.String.ID,
			TransformType:            storage.InternalTransformTypeFromClient(policy.TransformTypeTokenizeByValue),
			Function:                 "function transform(x, y) { return x; }",
		}
		assert.IsNil(t, s.SaveTransformer(ctx, gp), assert.Must())

		req := tokenizer.InspectTokenRequest{
			Token: tr.Token,
		}
		rr := doRequest(t, h, http.MethodPost, paths.InspectToken, req, createAuthToken("inspect"), hostname)
		assert.Equal(t, rr.Code, http.StatusOK, assert.Must())
		var resp tokenizer.InspectTokenResponse
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))

		assert.Equal(t, resp.Token, tr.Token)
		assert.Equal(t, resp.ID, tr.ID)
		assert.Equal(t, resp.Created, tr.Created)
		assert.Equal(t, resp.Updated, tr.Updated)

		assert.Equal(t, resp.AccessPolicy.ID, ap.ID)
		assert.Equal(t, resp.AccessPolicy.Components[0].Template.ID, ap.ComponentIDs[0])

		assert.Equal(t, resp.Transformer.ID, gp.ID)
		assert.Equal(t, resp.Transformer.Function, gp.Function)
		assert.Equal(t, resp.Transformer.Parameters, gp.Parameters)

		// Delay for the async audit write to complete (fragile but yet we want to test the actual code path)
		time.Sleep(time.Millisecond * 100)
		checkAuditLog(t, internal.AuditLogEventTypeInspectToken, startTime, "inspect", 1, 1, 0)
	})

	t.Run("TestLookup", func(t *testing.T) {
		startTime := time.Now().UTC()

		tt := testlogtransport.InitLoggerAndTransportsForTests(t)

		authToken := createAuthToken("lookup")

		transformer := createTransformerHelper(t, policy.Transformer{
			Name:           "Transformer_5",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTokenizeByValue,
			Function: `function uuidv4() {
				return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
					var r = Math.random() * 16 | 0, v = c == 'x' ? r : (r & 0x3 | 0x8);
					return v.toString(16);
				});
			};
			/* TODO this defeats our uniqueness check */
			function transform(data, params) {
				return JSON.stringify(uuidv4());
			};`,
		}, h, hostname, authToken)

		apt := createAccessPolicyTemplateHelper(t, policy.AccessPolicyTemplate{
			Name:     "Template_5",
			Function: "function policy(x, y) { return false; }",
		}, h, hostname, authToken)

		ap := createAccessPolicyHelper(t, policy.AccessPolicy{
			Name: "Policy_5",
			Components: []policy.AccessPolicyComponent{
				{Template: &userstore.ResourceID{ID: apt.ID}},
			},
			PolicyType: policy.PolicyTypeCompositeAnd,
		}, h, hostname, authToken)

		ctr := tokenizer.CreateTokenRequest{
			Data:            `"lookup"`,
			TransformerRID:  userstore.ResourceID{ID: transformer.ID},
			AccessPolicyRID: userstore.ResourceID{ID: ap.ID},
		}
		rr := doRequest(t, h, http.MethodPost, paths.CreateToken, ctr, authToken, hostname)
		assert.Equal(t, rr.Code, http.StatusCreated)
		var resp tokenizer.CreateTokenResponse
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))

		tr, err := s.GetTokenRecordByToken(ctx, resp.Token)
		assert.NoErr(t, err)
		assert.Equal(t, tr.Data, `"lookup"`)

		// create a second one to test array response to lookup
		rr = doRequest(t, h, http.MethodPost, paths.CreateToken, ctr, createAuthToken("lookup"), hostname)
		assert.Equal(t, rr.Code, http.StatusCreated)
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))

		tr2, err := s.GetTokenRecordByToken(ctx, resp.Token)
		assert.NoErr(t, err)
		assert.Equal(t, tr2.Data, `"lookup"`)

		// now lookup the two tokens
		ltr := tokenizer.LookupTokensRequest{
			Data:            `"lookup"`,
			TransformerRID:  userstore.ResourceID{ID: transformer.ID},
			AccessPolicyRID: userstore.ResourceID{ID: ap.ID},
		}
		rr = doRequest(t, h, http.MethodPost, paths.LookupToken, ltr, createAuthToken("lookup"), hostname)
		assert.Equal(t, rr.Code, http.StatusOK)
		var lookupResp tokenizer.LookupTokensResponse
		assert.IsNil(t, json.Unmarshal(rr.Body.Bytes(), &lookupResp), assert.Must())
		assert.Equal(t, len(lookupResp.Tokens), 2, assert.Must())

		tokens := []string{tr.Token, tr2.Token}
		// sort so we can compare easily
		slices.Sort(lookupResp.Tokens)
		slices.Sort(tokens)
		assert.Equal(t, lookupResp.Tokens, tokens)

		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryCall, events.TransformerPrefix, "", 0))), 2)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryDuration, events.TransformerPrefix, "", 0))), 2)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryInputError, events.TransformerPrefix, "", 0))), 0)

		// Delay for the async audit write to complete (fragile but yet we want to test the actual code path)
		time.Sleep(time.Millisecond * 100)
		checkAuditLog(t, internal.AuditLogEventTypeLookupToken, startTime, "lookup", 1, 2, 0)
	})

	t.Run("TestResolveBatch", func(t *testing.T) {
		startTime := time.Now().UTC()

		tt := testlogtransport.InitLoggerAndTransportsForTests(t)

		authToken := createAuthToken("resolve")
		transformer := createTransformerHelper(t, policy.Transformer{
			Name:           "Transformer_Batch",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTokenizeByValue,
			Function: `function uuidv4() {
			return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
				var r = Math.random() * 16 | 0, v = c == 'x' ? r : (r & 0x3 | 0x8);
				return v.toString(16);
			});
		};
		/* TODO this defeats our uniqueness check */
		function transform(data, params) {
			return JSON.stringify(uuidv4());
		};`,
		}, h, hostname, authToken)

		apt := createAccessPolicyTemplateHelper(t, policy.AccessPolicyTemplate{
			Name:     "Template_Batch",
			Function: "function policy(context, params) { return context.client.value; }",
		}, h, hostname, authToken)

		ap := createAccessPolicyHelper(t, policy.AccessPolicy{
			Name: "Policy_Batch",
			Components: []policy.AccessPolicyComponent{
				{Template: &userstore.ResourceID{ID: apt.ID}},
			},
			PolicyType: policy.PolicyTypeCompositeAnd,
		}, h, hostname, authToken)
		ctr := tokenizer.CreateTokenRequest{
			Data:            `"token1"`,
			TransformerRID:  userstore.ResourceID{ID: transformer.ID},
			AccessPolicyRID: userstore.ResourceID{ID: ap.ID},
		}
		rr := doRequest(t, h, http.MethodPost, paths.CreateToken, ctr, createAuthToken("resolve"), hostname)
		assert.Equal(t, rr.Code, http.StatusCreated)
		var resp tokenizer.CreateTokenResponse
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))

		tr, err := s.GetTokenRecordByToken(ctx, resp.Token)
		assert.NoErr(t, err)
		assert.Equal(t, tr.Data, `"token1"`)

		// create a second one to test array response
		ctr2 := tokenizer.CreateTokenRequest{
			Data:            `"token2"`,
			TransformerRID:  userstore.ResourceID{ID: transformer.ID},
			AccessPolicyRID: userstore.ResourceID{ID: ap.ID},
		}
		rr = doRequest(t, h, http.MethodPost, paths.CreateToken, ctr2, createAuthToken("lookup"), hostname)
		assert.Equal(t, rr.Code, http.StatusCreated)
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))

		tr2, err := s.GetTokenRecordByToken(ctx, resp.Token)
		assert.NoErr(t, err)
		assert.Equal(t, tr2.Data, `"token2"`)

		// resolve both tokens at once
		rTokens, err := s.ListTokenRecordsByTokens(ctx, []string{tr.Token, tr2.Token})
		assert.NoErr(t, err)
		assert.Equal(t, len(rTokens), 2)

		tokens := []string{tr.Token, tr2.Token}
		rtokens := []string{rTokens[0].Token, rTokens[1].Token}
		// sort so we can compare easily
		slices.Sort(rtokens)
		slices.Sort(tokens)
		assert.Equal(t, rtokens, tokens)

		// now lookup the two tokens
		rtr := tokenizer.ResolveTokensRequest{
			Tokens:  []string{tr.Token, tr2.Token},
			Context: policy.ClientContext{"value": true},
		}
		rr = doRequest(t, h, http.MethodPost, paths.ResolveToken, rtr, createAuthToken("resolve"), hostname)
		assert.Equal(t, rr.Code, http.StatusOK)
		var resolveResp []tokenizer.ResolveTokenResponse
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resolveResp))
		assert.Equal(t, len(resolveResp), 2, assert.Must())

		uclog.Debugf(ctx, "%v", resolveResp)
		for _, resp := range resolveResp {
			if resp.Token == tr.Token {
				assert.Equal(t, tr.Data, resp.Data)
			} else if resp.Token == tr2.Token {
				assert.Equal(t, tr2.Data, resp.Data)
			}
		}

		// We generated two tokens so ensure enough events were fired
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryCall, events.TransformerPrefix, "", 0))), 2)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryDuration, events.TransformerPrefix, "", 0))), 2)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryInputError, events.TransformerPrefix, "", 0))), 0)

		// We should have only done a sngle evaluation of the access policy
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryCall, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryDuration, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryResultSuccess, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryResultFailure, events.APPrefix, "", 0))), 0)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryInputError, events.APPrefix, "", 0))), 0)

		tt.ClearEvents()

		apt = createAccessPolicyTemplateHelper(t, policy.AccessPolicyTemplate{
			Name:     "Template_Batch_2",
			Function: "function policy(context, params) { return !context.client.value }",
		}, h, hostname, authToken)

		apF := createAccessPolicyHelper(t, policy.AccessPolicy{
			Name: "Policy_Batch_2",
			Components: []policy.AccessPolicyComponent{
				{Template: &userstore.ResourceID{ID: apt.ID}},
			},
			PolicyType: policy.PolicyTypeCompositeAnd,
		}, h, hostname, authToken)

		ctr3 := tokenizer.CreateTokenRequest{
			Data:            `"token3"`,
			TransformerRID:  userstore.ResourceID{ID: transformer.ID},
			AccessPolicyRID: userstore.ResourceID{ID: apF.ID},
		}
		rr = doRequest(t, h, http.MethodPost, paths.CreateToken, ctr3, createAuthToken("resolve"), hostname)

		assert.Equal(t, rr.Code, http.StatusCreated)
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))

		tr3, err := s.GetTokenRecordByToken(ctx, resp.Token)
		assert.NoErr(t, err)
		assert.Equal(t, tr3.Data, `"token3"`)

		// now lookup the three tokens - two should resolve and one should fail
		rtr2 := tokenizer.ResolveTokensRequest{
			Tokens:  []string{tr.Token, tr2.Token, tr3.Token},
			Context: policy.ClientContext{"value": true},
		}
		rr = doRequest(t, h, http.MethodPost, paths.ResolveToken, rtr2, createAuthToken("resolve"), hostname)
		assert.Equal(t, rr.Code, http.StatusOK)
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resolveResp))
		assert.Equal(t, len(resolveResp), 3, assert.Must())

		uclog.Debugf(ctx, "%v", resolveResp)
		for _, resp := range resolveResp {
			if resp.Token == tr.Token {
				assert.Equal(t, tr.Data, resp.Data)
			} else if resp.Token == tr2.Token {
				assert.Equal(t, tr2.Data, resp.Data)
			} else if resp.Token == tr3.Token {
				assert.Equal(t, "", resp.Data)
			}
		}

		// We generated one token so ensure enough events were fired
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryCall, events.TransformerPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryDuration, events.TransformerPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.TransformerID, uclog.EventCategoryInputError, events.TransformerPrefix, "", 0))), 0)

		// We should have on a single evaluation of each of the access policies
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryCall, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryDuration, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryResultSuccess, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryResultFailure, events.APPrefix, "", 0))), 0)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr.AccessPolicyID, uclog.EventCategoryInputError, events.APPrefix, "", 0))), 0)

		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr3.AccessPolicyID, uclog.EventCategoryCall, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr3.AccessPolicyID, uclog.EventCategoryDuration, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr3.AccessPolicyID, uclog.EventCategoryResultSuccess, events.APPrefix, "", 0))), 0)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr3.AccessPolicyID, uclog.EventCategoryResultFailure, events.APPrefix, "", 0))), 1)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(tr3.AccessPolicyID, uclog.EventCategoryInputError, events.APPrefix, "", 0))), 0)

		// Delay for the async audit write to complete (fragile but yet we want to test the actual code path)
		time.Sleep(time.Millisecond * 100)
		checkAuditLog(t, internal.AuditLogEventTypeResolveToken, startTime, "resolve", 2, 4, 1)
	})

	t.Run("TestLookupOrCreate", func(t *testing.T) {
		authToken := createAuthToken("lookuporcreate")

		transformer := createTransformerHelper(t, policy.Transformer{
			Name:           "Transformer_6",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTokenizeByValue,
			Function: `function uuidv4() {
				return 'xxxxxxxx-xxxx-5xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
					var r = Math.random() * 16 | 0, v = c == 'x' ? r : (r & 0x3 | 0x8);
					return v.toString(16);
				});
			};
			/* TODO this defeats our uniqueness check */
			function transform(data, params) {
				return JSON.stringify(uuidv4());
			};`,
		}, h, hostname, authToken)

		apt := createAccessPolicyTemplateHelper(t, policy.AccessPolicyTemplate{
			Name:     "Template_6",
			Function: "function policy(x, y) { return false; } /* template 6 */",
		}, h, hostname, authToken)

		ap := createAccessPolicyHelper(t, policy.AccessPolicy{
			Name: "Policy_6",
			Components: []policy.AccessPolicyComponent{
				{Template: &userstore.ResourceID{ID: apt.ID}},
			},
			PolicyType: policy.PolicyTypeCompositeAnd,
		}, h, hostname, authToken)

		crt := tokenizer.CreateTokenRequest{
			Data:            `"lookuporcreate"`,
			TransformerRID:  userstore.ResourceID{ID: transformer.ID},
			AccessPolicyRID: userstore.ResourceID{ID: ap.ID},
		}
		rr := doRequest(t, h, http.MethodPost, paths.CreateToken, crt, authToken, hostname)
		assert.Equal(t, rr.Code, http.StatusCreated)
		var resp tokenizer.CreateTokenResponse
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))

		tr, err := s.GetTokenRecordByToken(ctx, resp.Token)
		assert.NoErr(t, err)
		assert.Equal(t, tr.Data, `"lookuporcreate"`)

		// now lookup the token
		locrt := tokenizer.LookupOrCreateTokensRequest{
			Data:             []string{`"lookuporcreate"`},
			TransformerRIDs:  []userstore.ResourceID{{ID: transformer.ID}},
			AccessPolicyRIDs: []userstore.ResourceID{{ID: ap.ID}},
		}
		rr = doRequest(t, h, http.MethodPost, paths.LookupOrCreateTokens, locrt, createAuthToken("lookuporcreate"), hostname)
		assert.Equal(t, rr.Code, http.StatusOK)
		var lookupResp tokenizer.LookupTokensResponse
		assert.IsNil(t, json.Unmarshal(rr.Body.Bytes(), &lookupResp), assert.Must())
		assert.Equal(t, len(lookupResp.Tokens), 1, assert.Must())

		// lookup the token again, but including a new piece of data that hasn't had a token created for it yet
		locrt = tokenizer.LookupOrCreateTokensRequest{
			Data:             []string{`"lookuporcreate"`, `"lookuporcreate2"`},
			TransformerRIDs:  []userstore.ResourceID{{ID: transformer.ID}, {ID: transformer.ID}},
			AccessPolicyRIDs: []userstore.ResourceID{{ID: ap.ID}, {ID: ap.ID}},
		}
		rr = doRequest(t, h, http.MethodPost, paths.LookupOrCreateTokens, locrt, createAuthToken("lookuporcreate"), hostname)
		assert.Equal(t, rr.Code, http.StatusOK)
		assert.IsNil(t, json.Unmarshal(rr.Body.Bytes(), &lookupResp), assert.Must())
		assert.Equal(t, len(lookupResp.Tokens), 2, assert.Must())
	})
}
