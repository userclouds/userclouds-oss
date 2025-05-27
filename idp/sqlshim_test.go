package idp_test

import (
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp/helpers"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/internal/storage"
	userstoreInternal "userclouds.com/idp/internal/userstore"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/sqlshim"
	"userclouds.com/plex/manager"
)

func TestSqlShimObserver(t *testing.T) {
	t.Parallel()
	tf := idptesthelpers.NewTestFixture(t)

	// Use the first login app as the audit log actor
	mgr := manager.NewFromDB(tf.TenantState.TenantDB, tf.TenantState.CacheConfig)
	apps, err := mgr.GetLoginApps(tf.Ctx, tf.TenantID, tf.Company.ID)
	assert.NoErr(t, err)
	assert.True(t, len(apps) > 0)
	proxyCtx := multitenant.SetTenantState(auth.SetSubjectTypeAndUUID(tf.Ctx, apps[0].ID, authz.ObjectTypeLoginApp), tf.TenantState)
	databaseID := uuid.Must(uuid.NewV4())

	proxyHandlerFactory := userstoreInternal.ProxyHandlerFactory{}
	observer := proxyHandlerFactory.NewProxyHandler(proxyCtx, databaseID, tf.TenantState, tf.AuthzClient, nil, nil, tf.Verifier)

	// Test that IDP sqlshim observer creates the correct columns for a schema update
	columns := []sqlshim.Column{
		{
			Table: "ext_table",
			Name:  "bool_column",
			Type:  "boolean",
		},
		{
			Table: "ext_table",
			Name:  "int_column",
			Type:  "integer",
		},
		{
			Table:  "ext_table",
			Name:   "text_column",
			Type:   "varchar",
			Length: 255,
		},
	}
	s := storage.New(proxyCtx, tf.TenantState.TenantDB, tf.TenantState.ID, tf.TenantState.CacheConfig)
	err = helpers.LoadColumnsIntoUserstore(proxyCtx, s, databaseID, sqlshim.DatabaseTypePostgres, columns)
	assert.NoErr(t, err)
	resp, err := tf.IDPClient.ListColumns(tf.Ctx)
	assert.NoErr(t, err)
	columnsFound := make([]uuid.UUID, len(columns))
	for _, col := range resp.Data {
		for i, c := range columns {
			if col.Table == c.Table && col.Name == c.Name {
				if (c.Type == "boolean" && col.DataType == datatype.Boolean) ||
					(c.Type == "varchar" && col.DataType == datatype.String) ||
					(c.Type == "integer" && col.DataType == datatype.Integer) {
					columnsFound[i] = col.ID
				}
			}
		}
	}
	for _, colID := range columnsFound {
		assert.False(t, colID.IsNil())
	}

	// Update the text column to have a non-passthrough default transformer
	textCol, err := tf.IDPClient.GetColumn(tf.Ctx, columnsFound[2])
	assert.NoErr(t, err)

	// First try one that tokenizes by reference, and verify that it doesn't work
	transformer := &policy.Transformer{
		Name:           "TokenizeByReferenceTransformer",
		Description:    "a transformer that tokenizes a string by reference",
		InputDataType:  datatype.String,
		OutputDataType: datatype.String,
		TransformType:  policy.TransformTypeTokenizeByReference,
		Function:       "function transform(x, y) { return x; }",
	}
	transformer, err = tf.IDPClient.CreateTransformer(tf.Ctx, *transformer)
	assert.NoErr(t, err)

	textCol.DefaultTransformer = userstore.ResourceID{ID: transformer.ID}
	textCol.DefaultTokenAccessPolicy = userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}
	_, err = tf.IDPClient.UpdateColumn(tf.Ctx, textCol.ID, *textCol)
	assert.NotNil(t, err)

	// Change it to a transformer that tokenizes by value
	textCol.DefaultTransformer = userstore.ResourceID{ID: policy.TransformerUUID.ID}
	_, err = tf.IDPClient.UpdateColumn(tf.Ctx, textCol.ID, *textCol)
	assert.NoErr(t, err)

	// Test that IDP sqlshim observer handles queries correctly
	handleResp, _, reason, err := observer.HandleQuery(proxyCtx, sqlshim.DatabaseTypePostgres, "SELECT * FROM ext_table_2", "", uuid.Nil)
	assert.NotNil(t, err)
	assert.Equal(t, sqlshim.Passthrough, handleResp)
	assert.Equal(t, "table not found", reason)

	handleResp, tInfo, _, err := observer.HandleQuery(proxyCtx, sqlshim.DatabaseTypePostgres, "SELECT * FROM ext_table", "", uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, sqlshim.TransformResponse, handleResp)

	values := [][]byte{[]byte("false"), []byte("1"), []byte("hello world!")}
	passedAP, err := observer.TransformDataRow(proxyCtx, []string{"bool_column", "int_column", "text_column"}, values, tInfo, 0)
	assert.NoErr(t, err)
	assert.True(t, passedAP)
	assert.Equal(t, values[0], []byte("false"))
	assert.Equal(t, values[1], []byte("1"))
	assert.NotEqual(t, values[2], []byte("hello world!"))
	_, err = uuid.FromString(string(values[2]))
	assert.NoErr(t, err)

	// Test writing summary to audit log
	observer.TransformSummary(proxyCtx, tInfo, 1, 1, 0)

	// Update the text column to have a DenyAll access policy
	textCol, err = tf.IDPClient.GetColumn(tf.Ctx, columnsFound[2])
	assert.NoErr(t, err)

	textCol.AccessPolicy = userstore.ResourceID{ID: policy.AccessPolicyDenyAll.ID}
	_, err = tf.IDPClient.UpdateColumn(tf.Ctx, textCol.ID, *textCol)
	assert.NoErr(t, err)

	handleResp, tInfo, _, err = observer.HandleQuery(proxyCtx, sqlshim.DatabaseTypePostgres, "SELECT * FROM ext_table", "", uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, sqlshim.TransformResponse, handleResp)

	passedAP, err = observer.TransformDataRow(proxyCtx, []string{"bool_column", "int_column", "text_column"}, values, tInfo, 0)
	assert.NoErr(t, err)
	assert.False(t, passedAP)

	// test a query from a real example
	columns = []sqlshim.Column{
		{
			Table: "dev_shard2.workflow_workflowitem_v3",
			Name:  "id",
			Type:  "string",
		},
		{
			Table: "dev_shard2.workflow_workflowitem_v3",
			Name:  "organization_id",
			Type:  "string",
		},
		{
			Table: "dev_shard2.workflow_workflowitem_v3",
			Name:  "live",
			Type:  "string",
		},
	}
	err = helpers.LoadColumnsIntoUserstore(proxyCtx, s, databaseID, sqlshim.DatabaseTypeMySQL, columns)
	assert.NoErr(t, err)

	handleResp, _, _, err = observer.HandleQuery(proxyCtx,
		sqlshim.DatabaseTypeMySQL,
		"SELECT `workflow_workflowitem_v3`.`id`, `workflow_workflowitem_v3`.`organization_id` FROM `workflow_workflowitem_v3` WHERE (`workflow_workflowitem_v3`.`organization_id` = 1 AND `workflow_workflowitem_v3`.`live` = 1 AND `workflow_workflowitem_v3`.`object_id` = 1091527 AND `workflow_workflowitem_v3`.`object_type_id` = 450 AND `workflow_workflowitem_v3`.`organization_id` = 1) LIMIT 21;",
		"dev_shard2",
		uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, sqlshim.TransformResponse, handleResp)

	// Test that non-user columns that are in selector can be safely deleted
	cm, err := storage.NewColumnManager(proxyCtx, s, databaseID)
	assert.NoErr(t, err)
	column := cm.GetColumnByTableAndName("dev_shard2.workflow_workflowitem_v3", "live")
	assert.NotNil(t, column)
	_, err = cm.DeleteColumn(proxyCtx, column.ID)
	assert.NoErr(t, err)

	// But ensure that columns that are returned by the query cannot be deleted
	column = cm.GetColumnByTableAndName("dev_shard2.workflow_workflowitem_v3", "id")
	assert.NotNil(t, column)
	_, err = cm.DeleteColumn(proxyCtx, column.ID)
	assert.NotNil(t, err)
}
