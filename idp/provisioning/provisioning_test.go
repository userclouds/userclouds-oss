package provisioning_test

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/provisioning"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/logdb"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/internal/testhelpers"
)

var testCol1ID = uuid.Must(uuid.FromString("056d6a20-dfce-49ab-8fc1-e0268fc39fbd"))
var testCol2ID = uuid.Must(uuid.FromString("a518ab84-3b51-464a-80db-69ac14f1aa75"))

var defaultTestColumns = []storage.Column{
	{
		BaseModel:            ucdb.NewBaseWithID(testCol1ID),
		Table:                "users",
		Name:                 "testCol1",
		DataTypeID:           datatype.String.ID,
		IsArray:              false,
		IndexType:            storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeNone),
		DefaultValue:         "",
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	},
	{
		BaseModel:            ucdb.NewBaseWithID(testCol2ID),
		Table:                "users",
		Name:                 "testCol2",
		DataTypeID:           datatype.String.ID,
		IsArray:              false,
		IndexType:            storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeUnique),
		DefaultValue:         "",
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	},
}

func provisionUserstoreObjects(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) (types.Provisionable, error) {
	op := types.NewOrderedProvisioner(name)

	op.AddProvisionables(
		types.NewOrderedProvisionable(provisioning.ProvisionDefaultAccessPolicyTemplates(ctx, name, pi)).
			Named("access_policy_templates"),
		types.NewOrderedProvisionable(provisioning.ProvisionDefaultColumns(ctx, name, pi)).
			Named("columns"),
		types.NewOrderedProvisionable(provisioning.ProvisionDefaultPurposes(ctx, name, pi)).
			Named("purposes"),
		types.NewOrderedProvisionable(provisioning.ProvisionDefaultTransformers(ctx, name, pi)).
			Named("transformers"),
		types.NewOrderedProvisionable(provisioning.ProvisionDefaultAccessPolicies(ctx, name, pi)).
			Named("access_policies").
			After("access_policy_templates"),
		types.NewOrderedProvisionable(provisioning.ProvisionDefaultAccessors(ctx, name, pi)).
			Named("accessors").
			After("columns", "purposes", "access_policies", "transformers"),
		types.NewOrderedProvisionable(provisioning.ProvisionDefaultMutators(ctx, name, pi)).
			Named("mutators").
			After("columns", "purposes", "access_policies", "transformers"),
	)

	p, err := op.CreateProvisionable()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return p, nil
}

func uniqueName(name string) string {
	return name + "_" + uuid.Must(uuid.NewV4()).String()
}

func uniqueColumn(c storage.Column) storage.Column {
	c.ID = uuid.Must(uuid.NewV4())
	c.Name = uniqueName(c.Name)
	return c
}

func TestIDPProvisioning(t *testing.T) {
	ctx := context.Background()
	cacheCfg := cachetesthelpers.NewCacheConfig()
	// TODO: company/tenant/middleware makes test setup hard...hmw simplify?
	companyDB, logDBCfg, companyStorage := testhelpers.NewTestStorage(t)
	company := testhelpers.ProvisionTestCompanyWithoutACL(ctx, t, companyStorage)
	logDB, err := ucdb.NewWithLimits(ctx, logDBCfg, migrate.SchemaValidator(logdb.Schema), 10, 10)
	assert.NoErr(t, err)

	tenant, tenantDB := testhelpers.ProvisionTestTenant(ctx, t, companyStorage, companyDB, logDBCfg, company.ID)
	assert.NoErr(t, err)

	pi := types.ProvisionInfo{
		CompanyStorage: companyStorage,
		TenantDB:       tenantDB,
		LogDB:          logDB,
		CacheCfg:       cacheCfg,
		TenantID:       tenant.ID,
	}

	t.Run("ProvisionAccessorObject", func(t *testing.T) {
		aID := uuid.Must(uuid.NewV4())
		a := storage.Accessor{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(aID),
			Name:                     uniqueName("TestUserAccessor"),
			Description:              "Accessor used to retrieve a user’s profile data via the GetUser API",
			Version:                  0,
			DataLifeCycleState:       column.DataLifeCycleStateLive,
			ColumnIDs:                []uuid.UUID{column.IDColumnID},
			TransformerIDs:           []uuid.UUID{policy.TransformerPassthrough.ID},
			TokenAccessPolicyIDs:     []uuid.UUID{uuid.Nil},
			AccessPolicyID:           policy.AccessPolicyAllowAll.ID,
			SelectorConfig:           userstore.UserSelectorConfig{WhereClause: "{id} = ?"},
			PurposeIDs:               []uuid.UUID{constants.OperationalPurposeID},
		}

		eP, err := provisioning.ProvisionAccessors(ctx, "ProvisionAccessorObject_1", pi, &a)()
		assert.NoErr(t, err)
		assert.Equal(t, len(eP), 1)
		assert.NoErr(t, eP[0].Provision(ctx))
		assert.NoErr(t, eP[0].Validate(ctx))

		eP, err = provisioning.ProvisionAccessors(ctx, "ProvisionAccessorObject_2", pi, &a)()
		assert.NoErr(t, err)
		assert.Equal(t, len(eP), 1)

		assert.NoErr(t, eP[0].Provision(ctx))
		assert.NoErr(t, eP[0].Validate(ctx))
		assert.NoErr(t, eP[0].Cleanup(ctx))
		assert.NoErr(t, eP[0].Close(ctx))
	})

	t.Run("ProvisionAccessor", func(t *testing.T) {
		aID := uuid.Must(uuid.NewV4())
		a := storage.Accessor{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(aID),
			Name:                     uniqueName("TestUserAccessor"),
			Description:              "Accessor used to retrieve a user’s profile data via the GetUser API",
			Version:                  0,
			DataLifeCycleState:       column.DataLifeCycleStateLive,
			ColumnIDs:                []uuid.UUID{column.IDColumnID},
			TransformerIDs:           []uuid.UUID{policy.TransformerPassthrough.ID},
			TokenAccessPolicyIDs:     []uuid.UUID{uuid.Nil},
			AccessPolicyID:           policy.AccessPolicyAllowAll.ID,
			SelectorConfig:           userstore.UserSelectorConfig{WhereClause: "{id} = ?"},
			PurposeIDs:               []uuid.UUID{constants.OperationalPurposeID},
		}

		eP, err := provisioning.ProvisionAccessors(ctx, "ProvisionAccessor_1", types.ProvisionInfo{LogDB: logDB, TenantDB: nil, CacheCfg: cacheCfg, TenantID: tenant.ID}, &a)()
		assert.NotNil(t, err)
		assert.Equal(t, len(eP), 0)

		eP, err = provisioning.ProvisionAccessors(ctx, "ProvisionAccessor_2", types.ProvisionInfo{LogDB: nil, TenantDB: tenantDB, CacheCfg: cacheCfg, TenantID: tenant.ID}, &a)()
		assert.NotNil(t, err)
		assert.Equal(t, len(eP), 0)
		pi := types.ProvisionInfo{LogDB: logDB, TenantDB: tenantDB, CacheCfg: cacheCfg, TenantID: tenant.ID}
		eP, err = provisioning.ProvisionAccessors(ctx, "ProvisionAccessor", pi, &a)()
		assert.NoErr(t, err)
		assert.Equal(t, len(eP), 1)

		assert.NotNil(t, eP[0].Validate(ctx))
		assert.NoErr(t, eP[0].Provision(ctx))
		assert.NoErr(t, eP[0].Validate(ctx))

		eP, err = provisioning.ProvisionAccessors(ctx, "ProvisionAccessor_3", pi, &a)()
		assert.NoErr(t, err)
		assert.Equal(t, len(eP), 1)
		assert.NoErr(t, eP[0].Provision(ctx))
		assert.NoErr(t, eP[0].Validate(ctx))

		a.Description = "Updated Accessor"

		assert.Equal(t, a.Version, 0)
		assert.NoErr(t, eP[0].Provision(ctx))
		assert.Equal(t, a.Version, 1)

		assert.NoErr(t, eP[0].Cleanup(ctx))
		assert.NoErr(t, eP[0].Close(ctx))
	})

	t.Run("ProvisionMutatorObject", func(t *testing.T) {
		mID := uuid.Must(uuid.NewV4())

		m := storage.Mutator{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(mID),
			Name:                     uniqueName("TestCreateAndUpdateUserMutator"),
			Description:              "Mutator used to modify a user’s data by CreateUser and UpdateUser APIs",
			Version:                  0,
			ColumnIDs:                []uuid.UUID{column.IDColumnID},
			NormalizerIDs:            []uuid.UUID{policy.TransformerPassthrough.ID},
			AccessPolicyID:           policy.AccessPolicyAllowAll.ID,
			SelectorConfig:           userstore.UserSelectorConfig{WhereClause: "{id} = ?"},
		}

		mP, err := provisioning.ProvisionMutators(ctx, "ProvisionMutatorObject_1", pi, &m)()
		assert.NoErr(t, err)
		assert.Equal(t, len(mP), 1)
		assert.NoErr(t, mP[0].Provision(ctx))
		assert.NoErr(t, mP[0].Validate(ctx))

		mP, err = provisioning.ProvisionMutators(ctx, "ProvisionMutatorObject_2", pi, &m)()
		assert.NoErr(t, err)
		assert.Equal(t, len(mP), 1)
		assert.NoErr(t, mP[0].Provision(ctx))
		assert.NoErr(t, mP[0].Validate(ctx))

		assert.NoErr(t, mP[0].Cleanup(ctx))
		assert.NoErr(t, mP[0].Close(ctx))
	})

	t.Run("ProvisionMutator", func(t *testing.T) {
		mID := uuid.Must(uuid.NewV4())
		m := storage.Mutator{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(mID),
			Name:                     uniqueName("TestCreateAndUpdateUserMutator"),
			Description:              "Mutator used to modify a user’s data by CreateUser and UpdateUser APIs",
			Version:                  0,
			ColumnIDs:                []uuid.UUID{column.IDColumnID},
			NormalizerIDs:            []uuid.UUID{policy.TransformerPassthrough.ID},
			AccessPolicyID:           policy.AccessPolicyAllowAll.ID,
			SelectorConfig:           userstore.UserSelectorConfig{WhereClause: "{id} = ?"},
		}

		mP, err := provisioning.ProvisionMutators(ctx, "ProvisionMutator_1", types.ProvisionInfo{CompanyStorage: companyStorage, TenantDB: nil, LogDB: logDB, CacheCfg: cacheCfg, TenantID: tenant.ID}, &m)()
		assert.NotNil(t, err)
		assert.Equal(t, len(mP), 0)

		mP, err = provisioning.ProvisionMutators(ctx, "ProvisionMutator_2", types.ProvisionInfo{CompanyStorage: companyStorage, TenantDB: tenantDB, LogDB: nil, CacheCfg: cacheCfg, TenantID: tenant.ID}, &m)()
		assert.NotNil(t, err)
		assert.Equal(t, len(mP), 0)

		mP, err = provisioning.ProvisionMutators(ctx, "ProvisionMutator_3", pi, &m)()
		assert.NoErr(t, err)
		assert.Equal(t, len(mP), 1)
		assert.NotNil(t, mP[0].Validate(ctx))

		assert.NoErr(t, mP[0].Provision(ctx))
		assert.NoErr(t, mP[0].Validate(ctx))

		mP, err = provisioning.ProvisionMutators(ctx, "ProvisionMutator_4", pi, &m)()
		assert.NoErr(t, err)
		assert.Equal(t, len(mP), 1)
		assert.NoErr(t, mP[0].Provision(ctx))
		assert.NoErr(t, mP[0].Validate(ctx))

		m.Description = "Updated Mutator"

		assert.Equal(t, m.Version, 0)
		assert.NoErr(t, mP[0].Provision(ctx))
		assert.Equal(t, m.Version, 1)

		assert.NoErr(t, mP[0].Cleanup(ctx))
		assert.NoErr(t, mP[0].Close(ctx))
	})
	t.Run("ProvisionColumns", func(t *testing.T) {
		// Add default columns
		var colIDs []uuid.UUID
		var transformerIDs []uuid.UUID
		var tokenAccessPolicyIDs []uuid.UUID
		var storageColumns storage.Columns
		for _, dc := range defaultTestColumns {
			uC := uniqueColumn(dc)
			storageColumns = append(storageColumns, uC)
			colIDs = append(colIDs, uC.ID)
			transformerIDs = append(transformerIDs, policy.TransformerPassthrough.ID)
			tokenAccessPolicyIDs = append(tokenAccessPolicyIDs, uuid.Nil)
		}

		cP, err := provisioning.ProvisionColumns(ctx, "ProvisionColumns_1", pi, storageColumns...)()
		assert.NoErr(t, err)
		assert.Equal(t, len(cP), 1)

		err = cP[0].Validate(ctx)
		assert.NotNil(t, err)

		err = cP[0].Provision(ctx)
		assert.NoErr(t, err)

		err = cP[0].Validate(ctx)
		assert.NoErr(t, err)

		// Try to provision again with changed names

		var storageColumnsChanged storage.Columns
		for _, dc := range storageColumns {
			if dc.Attributes.Immutable || dc.Attributes.System {
				continue
			}
			nc := dc
			nc.Name = "changed" + dc.Name
			storageColumnsChanged = append(storageColumnsChanged, nc)
		}

		cPChanged, err := provisioning.ProvisionColumns(ctx, "ProvisionColumns_2", pi, storageColumnsChanged...)()
		assert.NoErr(t, err)
		assert.Equal(t, len(cPChanged), 1)

		// provisioning will not fail, but validation will because nothing changed

		err = cPChanged[0].Provision(ctx)
		assert.NoErr(t, err)

		err = cPChanged[0].Validate(ctx)
		assert.NotNil(t, err)

		// validation against original mutation will still succeed

		err = cP[0].Validate(ctx)
		assert.NoErr(t, err)

		aID := uuid.Must(uuid.NewV4())
		a := storage.Accessor{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(aID),
			Name:                     uniqueName("TestUserAccessor"),
			Description:              "Accessor used to retrieve a user’s profile data via the GetUser API",
			Version:                  0,
			DataLifeCycleState:       column.DataLifeCycleStateLive,
			AccessPolicyID:           policy.AccessPolicyAllowAll.ID,
			ColumnIDs:                colIDs,
			TransformerIDs:           transformerIDs,
			TokenAccessPolicyIDs:     tokenAccessPolicyIDs,
			SelectorConfig:           userstore.UserSelectorConfig{WhereClause: "{id} = ?"},
			PurposeIDs:               []uuid.UUID{constants.OperationalPurposeID},
		}

		aP, err := provisioning.ProvisionAccessors(ctx, "ProvisionColumns_3", pi, &a)()
		assert.NoErr(t, err)
		assert.Equal(t, len(aP), 1)

		err = aP[0].Provision(ctx)
		assert.NoErr(t, err)

		err = aP[0].Validate(ctx)
		assert.NoErr(t, err)

		mID := uuid.Must(uuid.NewV4())
		m := storage.Mutator{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(mID),
			Name:                     uniqueName("TestCreateAndUpdateUserMutator"),
			Description:              "Mutator used to modify a user’s data by CreateUser and UpdateUser APIs",
			Version:                  0,
			ColumnIDs:                colIDs,
			NormalizerIDs:            transformerIDs,
			AccessPolicyID:           policy.AccessPolicyAllowAll.ID,
			SelectorConfig:           userstore.UserSelectorConfig{WhereClause: "{id} = ?"},
		}

		mP, err := provisioning.ProvisionMutators(ctx, "ProvisionColumns_4", pi, &m)()
		assert.NoErr(t, err)
		assert.Equal(t, len(mP), 1)

		assert.NoErr(t, mP[0].Provision(ctx))
		assert.NoErr(t, mP[0].Validate(ctx))
		assert.NoErr(t, aP[0].Cleanup(ctx))
		assert.NoErr(t, mP[0].Cleanup(ctx))

		// Clean up the columns
		assert.NoErr(t, cP[0].Cleanup(ctx))
		assert.NoErr(t, cP[0].Close(ctx))

	})

	t.Run("ProvisionUserStore", func(t *testing.T) {
		uP, err := provisionUserstoreObjects(ctx, "ProvisionUserStore_1", pi)
		assert.NoErr(t, err)

		assert.NoErr(t, uP.Provision(ctx))
		assert.NoErr(t, uP.Validate(ctx))

		// Add a few more columns before we clean up to make sure we don't end up with no fields accessor/mutator once we delete the default columns
		// this will also test triggering state refresh in uP.ColumnManager since new columns will not be in its state
		var storageColumns storage.Columns
		for _, dc := range defaultTestColumns {
			storageColumns = append(storageColumns, uniqueColumn(dc))
		}
		cP, err := provisioning.ProvisionColumns(ctx, "ProvisionUserStore_2", pi, storageColumns...)()
		assert.NoErr(t, err)
		assert.Equal(t, len(cP), 1)

		assert.NoErr(t, cP[0].Provision(ctx))
		assert.NoErr(t, cP[0].Validate(ctx))

		s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)

		// soft-delete a default column
		assert.NoErr(t, s.DeleteColumn(ctx, column.NameColumnID))
		got, err := s.GetColumnSoftDeleted(ctx, column.NameColumnID)
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, column.NameColumnID)

		// reprovision
		uP, err = provisionUserstoreObjects(ctx, "ProvisionUserStore_3", pi)
		assert.NoErr(t, err)
		assert.NoErr(t, uP.Provision(ctx))

		// ensure we didn't overwrite it
		got, err = s.GetColumn(ctx, column.NameColumnID)
		assert.NotNil(t, err)
		assert.IsNil(t, got)

		// now rename a different column
		col, err := s.GetColumn(ctx, column.ExternalAliasColumnID)
		assert.NoErr(t, err)
		cc := col.ToClientModel()
		cm, err := storage.NewUserstoreColumnManager(ctx, s)
		assert.NoErr(t, err)
		cc.Name = "external_id"
		code, err := cm.UpdateColumnFromClient(ctx, &cc)
		assert.NoErr(t, err)
		assert.Equal(t, code, http.StatusOK)

		// add a custom accessor
		mm := storage.NewMethodManager(ctx, s)
		a := userstore.Accessor{
			ID:   uuid.Must(uuid.NewV4()),
			Name: uniqueName("TestUserAccessor"),
			Columns: []userstore.ColumnOutputConfig{{
				Column:      userstore.ResourceID{ID: column.ExternalAliasColumnID},
				Transformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
			}},
			SelectorConfig: userstore.UserSelectorConfig{WhereClause: "{external_id} = ?"},
			AccessPolicy:   userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
		}
		code, err = mm.CreateAccessorFromClient(ctx, &a)
		assert.NoErr(t, err)
		assert.Equal(t, code, http.StatusCreated)

		// and a custom mutator
		m := userstore.Mutator{
			ID:   uuid.Must(uuid.NewV4()),
			Name: uniqueName("TestCreateAndUpdateUserMutator"),
			Columns: []userstore.ColumnInputConfig{{
				Column:     userstore.ResourceID{ID: column.ExternalAliasColumnID},
				Normalizer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
			}},
			SelectorConfig: userstore.UserSelectorConfig{WhereClause: "{external_id} = ?"},
			AccessPolicy:   userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
		}
		code, err = mm.CreateMutatorFromClient(ctx, &m)
		assert.NoErr(t, err)
		assert.Equal(t, code, http.StatusCreated)

		// reprovision again
		uP, err = provisionUserstoreObjects(ctx, "ProvisionUserStore_4", pi)
		assert.NoErr(t, err)
		assert.NoErr(t, uP.Provision(ctx))

		// ensure the column name didn't change back
		got, err = s.GetColumn(ctx, column.ExternalAliasColumnID)
		assert.NoErr(t, err)
		assert.Equal(t, got.Name, "external_id")

		// ensure the accessor selector naming is still correct
		da, err := s.GetLatestAccessor(ctx, constants.GetUserAccessorID)
		assert.NoErr(t, err)
		assert.Equal(t, da.SelectorConfig.WhereClause, "{id} = ANY (?)")
		da, err = s.GetLatestAccessor(ctx, a.ID)
		assert.NoErr(t, err)
		assert.Equal(t, da.SelectorConfig.WhereClause, "{external_id} = ?")

		// ensure the mutator selector naming is still correct
		dm, err := s.GetLatestMutator(ctx, constants.UpdateUserMutatorID)
		assert.NoErr(t, err)
		assert.Equal(t, dm.SelectorConfig.WhereClause, "{id} = ?")
		dm, err = s.GetLatestMutator(ctx, m.ID)
		assert.NoErr(t, err)
		assert.Equal(t, dm.SelectorConfig.WhereClause, "{external_id} = ?")
	})

	t.Run("ProvisionPurposes", func(t *testing.T) {
		testPurposeName := uniqueName("TestPurpose")
		p := storage.Purpose{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(uuid.Must(uuid.NewV4())),
			Name:                     testPurposeName,
			Description:              "Test purpose",
		}

		pP, err := provisioning.ProvisionPurposes(ctx, "ProvisionPurposes_1", pi, p)()
		assert.NoErr(t, err)
		assert.Equal(t, len(pP), 1)

		assert.NoErr(t, pP[0].Provision(ctx))
		assert.NoErr(t, pP[0].Validate(ctx))

		// try changing the description
		p.Description = "updated"
		pP, err = provisioning.ProvisionPurposes(ctx, "ProvisionPurposes_2", pi, p)()
		assert.NoErr(t, err)
		assert.Equal(t, len(pP), 1)

		assert.NoErr(t, pP[0].Provision(ctx))
		assert.NoErr(t, pP[0].Validate(ctx))

		s := storage.New(ctx, tenantDB, tenant.ID, cacheCfg)
		got, err := s.GetPurpose(ctx, p.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got.Description, "Test purpose") // the old one

		// try to add a new TestPurpose with a different ID
		newP := storage.Purpose{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(uuid.Must(uuid.NewV4())),
			Name:                     testPurposeName,
			Description:              "Test purpose",
		}
		pP, err = provisioning.ProvisionPurposes(ctx, "ProvisionPurposes_3", pi, newP)()
		assert.NoErr(t, err)
		assert.Equal(t, len(pP), 1)
		assert.NoErr(t, pP[0].Provision(ctx))
		assert.NotNil(t, pP[0].Validate(ctx)) // validate should fail since we didn't provision this purpose

		_, err = s.GetPurpose(ctx, newP.ID)
		assert.ErrorIs(t, err, sql.ErrNoRows)

		// now delete & re-provision to ensure we don't recreate again
		assert.NoErr(t, s.DeletePurpose(ctx, p.ID))
		got, err = s.GetPurposeSoftDeleted(ctx, p.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, p.ID)
		assert.Equal(t, got.Name, p.Name)
		assert.Equal(t, got.Description, "Test purpose")

		pP, err = provisioning.ProvisionPurposes(ctx, "ProvisionPurposes_4", pi, p)()
		assert.NoErr(t, err)
		assert.Equal(t, len(pP), 0)

		got, err = s.GetPurpose(ctx, p.ID)
		assert.NotNil(t, err)
		assert.IsNil(t, got)
	})
}
