package idptesthelpers

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	authzroutes "userclouds.com/authz/routes"
	"userclouds.com/idp"
	"userclouds.com/idp/policy"
	idproutes "userclouds.com/idp/routes"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
)

// TestFixture is a test fixture for idp client
type TestFixture struct {
	T               *testing.T
	Ctx             context.Context
	IDPClient       *idp.Client
	IDPMgmtClient   *idp.ManagementClient
	TokenizerClient *idp.TokenizerClient
	AuthzClient     *authz.Client
	TenantDB        *ucdb.DB
	TenantURL       string
	TenantID        uuid.UUID
	Company         *companyconfig.Company
	CCS             *companyconfig.Storage
	CompanyDBConfig *ucdb.Config
	LogDBConfig     *ucdb.Config
	TenantState     *tenantmap.TenantState
	Verifier        auth.Verifier
}

// NewTestFixture creates a test fixture
func NewTestFixture(t *testing.T) *TestFixture {
	ctx := context.Background()
	companyDBConfig, logDBConfig, companyConfigStorage := testhelpers.NewTestStorage(t)
	company, tenant, tenantDB := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, companyConfigStorage, companyDBConfig, logDBConfig)
	tenants := testhelpers.NewTestTenantStateMap(companyConfigStorage)
	jwtVerifier := uctest.JWTVerifier{}

	hb := builder.NewHandlerBuilder()
	idproutes.InitForTests(hb, tenants, companyConfigStorage, tenant.ID, jwtVerifier)
	authzroutes.InitForTests(hb, tenants, companyConfigStorage, jwtVerifier)
	server := httptest.NewServer(hb.Build())
	t.Cleanup(server.Close)
	testhelpers.UpdateTenantURL(ctx, t, companyConfigStorage, tenant, server)

	tsOpt := jsonclient.TokenSource(uctest.TokenSource(t, tenant.TenantURL))
	idpClient, err := idp.NewClient(tenant.TenantURL, idp.JSONClient(tsOpt))
	assert.NoErr(t, err)
	tokenizerClient := idpClient.TokenizerClient

	idpMgmtClient, err := idp.NewManagementClient(tenant.TenantURL, tsOpt)
	assert.NoErr(t, err)

	authzClient, err := authz.NewClient(tenant.TenantURL, authz.JSONClient(tsOpt))
	assert.NoErr(t, err)

	ts, err := tenants.GetTenantStateForID(ctx, tenant.ID)
	assert.NoErr(t, err)

	return &TestFixture{
		T:               t,
		Ctx:             ctx,
		IDPClient:       idpClient,
		IDPMgmtClient:   idpMgmtClient,
		TokenizerClient: tokenizerClient,
		AuthzClient:     authzClient,
		TenantDB:        tenantDB,
		TenantURL:       tenant.TenantURL,
		TenantID:        tenant.ID,
		Company:         company,
		CCS:             companyConfigStorage,
		CompanyDBConfig: companyDBConfig,
		LogDBConfig:     logDBConfig,
		TenantState:     ts,
		Verifier:        jwtVerifier,
	}
}

// CreateColumn creates a test column
func (tf *TestFixture) CreateColumn(
	name string,
	dataType userstore.ResourceID,
	isArray bool,
	defaultValue string,
	accessPolicy userstore.ResourceID,
	defaultTransformer userstore.ResourceID,
	indexType userstore.ColumnIndexType,
	fields ...userstore.ColumnField,
) (*userstore.Column, error) {
	tf.T.Helper()

	return tf.IDPClient.CreateColumn(
		tf.Ctx,
		userstore.Column{
			Name:               name,
			Table:              "users",
			DataType:           dataType,
			IsArray:            isArray,
			DefaultValue:       defaultValue,
			AccessPolicy:       accessPolicy,
			DefaultTransformer: defaultTransformer,
			IndexType:          indexType,
			Constraints:        userstore.ColumnConstraints{Fields: fields},
		},
		idp.IfNotExists(),
	)
}

// CreateTableColumn creates a test column in the specified table
func (tf *TestFixture) CreateTableColumn(
	table string,
	name string,
	dataType userstore.ResourceID,
	isArray bool,
	defaultValue string,
	defaultTransformer userstore.ResourceID,
	indexType userstore.ColumnIndexType,
	fields ...userstore.ColumnField,
) (*userstore.Column, error) {
	tf.T.Helper()

	return tf.IDPClient.CreateColumn(
		tf.Ctx,
		userstore.Column{
			Name:               name,
			Table:              table,
			DataType:           dataType,
			IsArray:            isArray,
			DefaultValue:       defaultValue,
			DefaultTransformer: defaultTransformer,
			IndexType:          indexType,
			Constraints:        userstore.ColumnConstraints{Fields: fields},
		},
		idp.IfNotExists(),
	)
}

// CreateValidColumn creates a test column
func (tf *TestFixture) CreateValidColumn(
	name string,
	dataType userstore.ResourceID,
	isArray bool,
	defaultValue string,
	accessPolicy userstore.ResourceID,
	defaultTransformer userstore.ResourceID,
	indexType userstore.ColumnIndexType,
	fields ...userstore.ColumnField,
) *userstore.Column {
	tf.T.Helper()

	col, err := tf.CreateColumn(name, dataType, isArray, defaultValue, accessPolicy, defaultTransformer, indexType, fields...)
	assert.NoErr(tf.T, err)
	assert.NotNil(tf.T, col.ID)
	assert.Equal(tf.T, col.Name, name)
	assert.Equal(tf.T, col.DataType, dataType)
	assert.Equal(tf.T, col.IsArray, isArray)
	assert.Equal(tf.T, col.DefaultValue, defaultValue)
	assert.Equal(tf.T, col.DefaultTransformer.ID, defaultTransformer.ID)
	assert.Equal(tf.T, col.IndexType, indexType)
	return col
}

// UpdateColumn updates a test column
func (tf *TestFixture) UpdateColumn(
	id uuid.UUID,
	name string,
	dataType userstore.ResourceID,
	isArray bool,
	defaultValue string,
	accessPolicy userstore.ResourceID,
	defaultTransformer userstore.ResourceID,
	indexType userstore.ColumnIndexType,
) (*userstore.Column, error) {
	tf.T.Helper()

	return tf.IDPClient.UpdateColumn(
		tf.Ctx,
		id,
		userstore.Column{
			ID:                 id,
			Table:              "users",
			Name:               name,
			DataType:           dataType,
			IsArray:            isArray,
			DefaultValue:       defaultValue,
			AccessPolicy:       accessPolicy,
			DefaultTransformer: defaultTransformer,
			IndexType:          indexType,
		},
	)
}

// CreateSoftDeletedAccessor creates a test accessor for retrieving soft-deleted data
func (tf *TestFixture) CreateSoftDeletedAccessor(
	name string,
	accessPolicyID uuid.UUID,
	columnNames []string,
	transformerIDs []uuid.UUID,
	purposeNames []string,
) (*userstore.Accessor, error) {
	tf.T.Helper()

	return tf.CreateAccessor(
		name,
		userstore.DataLifeCycleStateSoftDeleted,
		accessPolicyID,
		columnNames,
		transformerIDs,
		purposeNames,
	)
}

// CreateLiveAccessor creates a test accessor for retrieving live data
func (tf *TestFixture) CreateLiveAccessor(
	name string,
	accessPolicyID uuid.UUID,
	columnNames []string,
	transformerIDs []uuid.UUID,
	purposeNames []string,
) (*userstore.Accessor, error) {
	tf.T.Helper()

	return tf.CreateAccessor(
		name,
		userstore.DataLifeCycleStateLive,
		accessPolicyID,
		columnNames,
		transformerIDs,
		purposeNames,
	)
}

// CreateAccessor creates a test accessor
func (tf *TestFixture) CreateAccessor(
	name string,
	dlcs userstore.DataLifeCycleState,
	accessPolicyID uuid.UUID,
	columnNames []string,
	transformerIDs []uuid.UUID,
	purposeNames []string,
) (*userstore.Accessor, error) {
	tf.T.Helper()
	columnIDs := make([]uuid.UUID, len(columnNames))
	for range columnNames {
		columnIDs = append(columnIDs, uuid.Nil)
	}
	return tf.CreateAccessorWithWhereClause(
		name,
		dlcs,
		columnNames,
		columnIDs,
		transformerIDs,
		purposeNames,
		"{id} = ?",
		accessPolicyID,
	)
}

// CreateAccessorWithWhereClause creates a test accessor with a where clause
func (tf *TestFixture) CreateAccessorWithWhereClause(
	name string,
	dlcs userstore.DataLifeCycleState,
	columnNames []string,
	columnIDs []uuid.UUID,
	transformerIDs []uuid.UUID,
	purposeNames []string,
	whereClause string,
	accessPolicyIDs ...uuid.UUID,
) (*userstore.Accessor, error) {
	tf.T.Helper()

	assert.True(tf.T, len(accessPolicyIDs) > 0 && len(accessPolicyIDs) <= 2)
	accessPolicyID := accessPolicyIDs[0]
	var tokenAccessPolicyID uuid.UUID
	if len(accessPolicyIDs) == 2 {
		tokenAccessPolicyID = accessPolicyIDs[1]
	}

	columns := []userstore.ColumnOutputConfig{}
	for i, columnName := range columnNames {

		columns = append(columns, userstore.ColumnOutputConfig{
			Column: userstore.ResourceID{
				Name: columnName,
				ID:   columnIDs[i],
			},
			Transformer: userstore.ResourceID{
				ID: transformerIDs[i],
			},
			TokenAccessPolicy: userstore.ResourceID{ID: tokenAccessPolicyID},
		})
	}
	purposes := []userstore.ResourceID{}
	for _, purposeName := range purposeNames {
		purposes = append(purposes, userstore.ResourceID{
			Name: purposeName,
		})
	}
	return tf.IDPClient.CreateAccessor(tf.Ctx,
		userstore.Accessor{
			ID:                 uuid.Must(uuid.NewV4()),
			Name:               name,
			DataLifeCycleState: dlcs,
			Columns:            columns,
			AccessPolicy:       userstore.ResourceID{ID: accessPolicyID},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: whereClause,
			},
			Purposes: purposes,
		})
}

// CreateValidAccessor creates a valid test accessor
func (tf *TestFixture) CreateValidAccessor(name string) *userstore.Accessor {
	tf.T.Helper()
	tf.CreateValidColumn("test_column", datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeNone)
	accessPolicy := tf.CreateValidAccessPolicy("test_policy", "apikey1")
	accessor, err := tf.CreateLiveAccessor(name,
		accessPolicy.ID,
		[]string{"test_column"},
		[]uuid.UUID{policy.TransformerPassthrough.ID},
		[]string{"operational"},
	)
	assert.NoErr(tf.T, err)

	return accessor
}

// CreateMutatorWithWhereClause creates a mutator with the specified where clause and other parameters
func (tf *TestFixture) CreateMutatorWithWhereClause(
	name string,
	accessPolicyID uuid.UUID,
	columnNames []string,
	normalizerIDs []uuid.UUID,
	whereClause string,
) (*userstore.Mutator, error) {
	tf.T.Helper()

	columns := []userstore.ColumnInputConfig{}
	for i, columnName := range columnNames {
		columns = append(
			columns,
			userstore.ColumnInputConfig{
				Column:     userstore.ResourceID{Name: columnName},
				Normalizer: userstore.ResourceID{ID: normalizerIDs[i]},
			},
		)
	}

	return tf.IDPClient.CreateMutator(
		tf.Ctx,
		userstore.Mutator{
			ID:           uuid.Must(uuid.NewV4()),
			Columns:      columns,
			Name:         name,
			AccessPolicy: userstore.ResourceID{ID: accessPolicyID},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: whereClause,
			},
		},
	)
}

// CreateMutator creates a test mutator
func (tf *TestFixture) CreateMutator(
	name string,
	accessPolicyID uuid.UUID,
	columnNames []string,
	normalizerIDs []uuid.UUID,
) (*userstore.Mutator, error) {
	tf.T.Helper()
	return tf.CreateMutatorWithWhereClause(
		name,
		accessPolicyID,
		columnNames,
		normalizerIDs,
		"{id} = ?",
	)
}

// CreateValidMutator creates a valid test mutator
func (tf *TestFixture) CreateValidMutator(name string) *userstore.Mutator {
	tf.T.Helper()

	tf.CreateValidColumn("test_column", datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeNone)

	accessPolicy := tf.CreateValidAccessPolicy("test_policy", "apikey1")

	mutator, err := tf.CreateMutator(name,
		accessPolicy.ID,
		[]string{"test_column"},
		[]uuid.UUID{policy.TransformerPassthrough.ID},
	)
	assert.NoErr(tf.T, err)

	return mutator
}

// CreateValidAccessPolicyTemplate creates a valid test access policy template
func (tf *TestFixture) CreateValidAccessPolicyTemplate(name string) *policy.AccessPolicyTemplate {
	tf.T.Helper()

	apt, err := tf.TokenizerClient.CreateAccessPolicyTemplate(
		tf.Ctx,
		policy.AccessPolicyTemplate{
			Name: name,
			Function: fmt.Sprintf(`function policy(context, params) {
				return context.client.api_key === params.api_key; // %s
			}`, name),
		},
		idp.IfNotExists(),
	)
	assert.NoErr(tf.T, err)
	return apt
}

// CreateValidAccessPolicy creates a valid test access policy
func (tf *TestFixture) CreateValidAccessPolicy(name string, apiKey string) *policy.AccessPolicy {
	tf.T.Helper()

	apt := tf.CreateValidAccessPolicyTemplate(name + "Template")
	p, err := tf.TokenizerClient.CreateAccessPolicy(tf.Ctx,
		policy.AccessPolicy{
			Name: name,
			Components: []policy.AccessPolicyComponent{
				{Template: &userstore.ResourceID{ID: apt.ID}, TemplateParameters: fmt.Sprintf(`{"api_key": "%s"}`, apiKey)}},
			PolicyType: policy.PolicyTypeCompositeAnd,
		}, idp.IfNotExists())
	assert.NoErr(tf.T, err)
	return p
}

// CreateValidPurpose creates a valid test purpose
func (tf *TestFixture) CreateValidPurpose(name string) *userstore.Purpose {
	tf.T.Helper()

	p, err := tf.IDPClient.CreatePurpose(tf.Ctx, userstore.Purpose{
		Name:        name,
		Description: name + "_description",
	})
	assert.NoErr(tf.T, err)
	return p
}

// CreateValidTransformer creates a valid test transformer
func (tf *TestFixture) CreateValidTransformer(name string) *policy.Transformer {
	tf.T.Helper()
	transformer, err := tf.TokenizerClient.CreateTransformer(
		tf.Ctx,
		policy.Transformer{
			Name: name,
			Function: fmt.Sprintf(
				`function transform(data, params) { return ""; } // %s`, name,
			),
			Parameters:     "",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypePassThrough,
		},
	)
	assert.NoErr(tf.T, err)
	return transformer
}
