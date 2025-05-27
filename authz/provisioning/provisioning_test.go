package provisioning_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/provisioning"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/internal/testhelpers"
)

func TestEntityProvisioning(t *testing.T) {
	ctx := context.Background()
	companyDB, logDB, companyStorage := testhelpers.NewTestStorage(t)
	_, tenant, _, tenantDB, _, _ := testhelpers.CreateTestServer(ctx, t)
	cacheCfg := cachetesthelpers.NewCacheConfig()
	pi := types.ProvisionInfo{CompanyStorage: companyStorage, TenantDB: tenantDB, TenantID: tenant.ID, CacheCfg: cacheCfg}
	t.Run("ProvisionEmpty", func(t *testing.T) {
		eP := provisioning.NewEntityAuthZ("", pi, nil, nil, nil, nil)
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))

		eP = provisioning.NewEntityAuthZ("", pi, []authz.ObjectType{}, []authz.EdgeType{}, []authz.Object{}, []authz.Edge{})
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))
		assert.NoErr(t, eP.Cleanup(ctx))
		assert.NoErr(t, eP.Close(ctx))
	})

	t.Run("ProvisionSingle", func(t *testing.T) {
		objectTypes := []authz.ObjectType{
			{BaseModel: ucdb.NewBase(), TypeName: uuid.Must(uuid.NewV4()).String()},
			{BaseModel: ucdb.NewBase(), TypeName: uuid.Must(uuid.NewV4()).String()},
			{BaseModel: ucdb.NewBase(), TypeName: uuid.Must(uuid.NewV4()).String()},
		}

		edgeTypes := []authz.EdgeType{
			{BaseModel: ucdb.NewBase(), TypeName: objectTypes[0].TypeName + "_Role1", SourceObjectTypeID: objectTypes[0].ID,
				TargetObjectTypeID: objectTypes[1].ID},
			{BaseModel: ucdb.NewBase(), TypeName: objectTypes[0].TypeName + "_Role2", SourceObjectTypeID: objectTypes[0].ID,
				TargetObjectTypeID: objectTypes[1].ID},
		}

		alias := uuid.Must(uuid.NewV4()).String()
		objects := []authz.Object{
			{BaseModel: ucdb.NewBase(), TypeID: objectTypes[0].ID},
			{BaseModel: ucdb.NewBase(), Alias: &alias, TypeID: objectTypes[1].ID},
			{BaseModel: ucdb.NewBase(), TypeID: objectTypes[2].ID},
		}

		edges := []authz.Edge{
			{BaseModel: ucdb.NewBase(), SourceObjectID: objects[0].ID, TargetObjectID: objects[1].ID, EdgeTypeID: edgeTypes[0].ID},
			{BaseModel: ucdb.NewBase(), SourceObjectID: objects[0].ID, TargetObjectID: objects[1].ID, EdgeTypeID: edgeTypes[1].ID},
		}

		eP := provisioning.NewEntityAuthZ("TestName", pi, []authz.ObjectType{objectTypes[0], objectTypes[1]}, []authz.EdgeType{edgeTypes[0]},
			[]authz.Object{objects[0], objects[1]}, []authz.Edge{edges[0]})
		err := eP.Provision(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, eP.Name(), "TestName:EntityAuthZ")

		err = eP.Validate(ctx)
		assert.NoErr(t, err)

		// Should fail (object type)
		eP = provisioning.NewEntityAuthZ("", pi, []authz.ObjectType{objectTypes[2]}, []authz.EdgeType{edgeTypes[0]},
			[]authz.Object{objects[0], objects[1]}, []authz.Edge{edges[0]})
		err = eP.Validate(ctx)
		assert.NotNil(t, err)

		// Should fail (edge type)
		eP = provisioning.NewEntityAuthZ("", pi, []authz.ObjectType{objectTypes[0]}, []authz.EdgeType{edgeTypes[1]},
			[]authz.Object{objects[0], objects[1]}, []authz.Edge{edges[0]})
		err = eP.Validate(ctx)
		assert.NotNil(t, err)

		// Should fail (objects)
		eP = provisioning.NewEntityAuthZ("", pi, []authz.ObjectType{objectTypes[0], objectTypes[1]}, []authz.EdgeType{edgeTypes[0]},
			[]authz.Object{objects[2]}, []authz.Edge{edges[0]})
		err = eP.Validate(ctx)
		assert.NotNil(t, err)

		// Should fail (edges)
		eP = provisioning.NewEntityAuthZ("", pi, []authz.ObjectType{objectTypes[0], objectTypes[1]}, []authz.EdgeType{edgeTypes[0]},
			[]authz.Object{objects[0], objects[1]}, []authz.Edge{edges[1]})
		err = eP.Validate(ctx)
		assert.NotNil(t, err)
	})

	t.Run("ProvisionArray", func(t *testing.T) {
		objects := []authz.Object{
			{BaseModel: ucdb.NewBase(), TypeID: authz.GroupObjectTypeID},
			{BaseModel: ucdb.NewBase(), TypeID: authz.GroupObjectTypeID},
		}

		eP := provisioning.NewEntityAuthZ("", pi, authz.DefaultAuthZObjectTypes, append(authz.DefaultAuthZEdgeTypes, provisioning.ConsoleAuthZEdgeTypes...), objects, nil)
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))
	})

	t.Run("ProvisionDuplicate", func(t *testing.T) {
		eP := provisioning.NewEntityAuthZ(
			"",
			pi,
			[]authz.ObjectType{authz.DefaultAuthZObjectTypes[1]},
			[]authz.EdgeType{provisioning.ConsoleAuthZEdgeTypes[1]},
			nil,
			nil,
		)

		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))
		assert.NoErr(t, eP.Provision(ctx))
		assert.NoErr(t, eP.Validate(ctx))
	})

	t.Run("ProvisionMultiThreaded", func(t *testing.T) {
		wg := sync.WaitGroup{}
		for range 10 {
			wg.Add(1)
			go func(gpID uuid.UUID, apID uuid.UUID) {
				objectTypes := []authz.ObjectType{
					{BaseModel: ucdb.NewBase(), TypeName: uuid.Must(uuid.NewV4()).String()},
					{BaseModel: ucdb.NewBase(), TypeName: uuid.Must(uuid.NewV4()).String()},
				}

				alias := uuid.Must(uuid.NewV4()).String()
				objects := []authz.Object{
					{BaseModel: ucdb.NewBase(), TypeID: objectTypes[0].ID},
					{BaseModel: ucdb.NewBase(), Alias: &alias, TypeID: objectTypes[1].ID},
				}

				edgeTypes := []authz.EdgeType{
					{BaseModel: ucdb.NewBase(), TypeName: objectTypes[0].TypeName + "_Role1", SourceObjectTypeID: objectTypes[0].ID,
						TargetObjectTypeID: objectTypes[1].ID},
					{BaseModel: ucdb.NewBase(), TypeName: objectTypes[0].TypeName + "_Role2", SourceObjectTypeID: objectTypes[0].ID,
						TargetObjectTypeID: objectTypes[1].ID},
				}

				edges := []authz.Edge{
					{BaseModel: ucdb.NewBase(), SourceObjectID: objects[0].ID, TargetObjectID: objects[1].ID, EdgeTypeID: edgeTypes[0].ID},
					{BaseModel: ucdb.NewBase(), SourceObjectID: objects[0].ID, TargetObjectID: objects[1].ID, EdgeTypeID: edgeTypes[1].ID},
				}

				eP := provisioning.NewEntityAuthZ("", pi, objectTypes, edgeTypes, objects, edges)
				assert.NoErr(t, eP.Provision(ctx))
				assert.NoErr(t, eP.Validate(ctx))

				wg.Done()
			}(uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()))
		}

		wg.Wait()
	})

	t.Run("ProvisionOrganization", func(t *testing.T) {
		pi := types.ProvisionInfo{CompanyStorage: companyStorage, TenantDB: tenantDB, LogDB: nil, CacheCfg: cacheCfg, TenantID: tenant.ID}
		// Try org with nil org id
		eP, err := provisioning.NewOrganizationProvisioner("TestName", pi, uuid.Nil, "TestOrg", "")
		assert.NotNil(t, err)
		assert.IsNil(t, eP)
		// Try org with empty name
		eP, err = provisioning.NewOrganizationProvisioner("TestName", pi, uuid.Must(uuid.NewV4()), "", "")
		assert.NotNil(t, err)
		assert.IsNil(t, eP)
		// Try valid org
		orgID := uuid.Must(uuid.NewV4())
		eP, err = provisioning.NewOrganizationProvisioner("TestName", pi, orgID, "TestOrg", "")
		assert.Equal(t, eP.Name(), fmt.Sprintf("TestName:Org[%v, TestOrg]", orgID)) // TestName:Org[476e019a-a71d-4527-b94b-7cc6f9f43844, TestOrg]
		assert.NoErr(t, err)

		// Provision the org
		assert.NoErr(t, eP.Provision(ctx))

		// Validate the org
		assert.NoErr(t, eP.Validate(ctx))

		assert.NoErr(t, eP.Close(ctx))

		// Provision a user object in the new org
		aP := provisioning.NewEntityAuthZ("TestName", pi, nil, nil,
			[]authz.Object{{BaseModel: ucdb.NewBase(), TypeID: authz.UserObjectTypeID, OrganizationID: orgID}}, nil)
		assert.NoErr(t, aP.Provision(ctx))
	})

	t.Run("ProvisionAuthTenant", func(t *testing.T) {
		t.Skip()
		_, ct, ctDB := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, companyStorage, companyDB, logDB)
		company := testhelpers.NewCompanyForTest(ctx, t, companyStorage, ctDB, ct.ID, ct.CompanyID)
		testhelpers.ProvisionTestTenant(ctx, t, companyStorage, companyDB, logDB, company.ID)
		tenantDB, _, _, err := tenantdb.Connect(ctx, companyStorage, tenant.ID)
		assert.NoErr(t, err)
		// Try nil tenant id
		eT, err := provisioning.NewTenantAuthZ("TestName", types.ProvisionInfo{
			CompanyStorage: companyStorage,
			TenantDB:       tenantDB,
			LogDB:          nil,
			CacheCfg:       cacheCfg,
			TenantID:       uuid.Nil,
		}, company)
		assert.NotNil(t, err)
		assert.IsNil(t, eT)
		// Try nil tenant db
		eT, err = provisioning.NewTenantAuthZ("TestName", types.ProvisionInfo{
			CompanyStorage: companyStorage,
			TenantDB:       nil,
			LogDB:          nil,
			CacheCfg:       cacheCfg,
			TenantID:       tenant.ID,
		}, company)
		assert.NotNil(t, err)
		assert.IsNil(t, eT)
		// Try nil company
		eT, err = provisioning.NewTenantAuthZ("TestName", pi, nil)
		assert.NotNil(t, err)
		assert.IsNil(t, eT)
		// Try empty company name
		company.Name = ""
		eT, err = provisioning.NewTenantAuthZ("TestName", pi, company)
		assert.NotNil(t, err)
		assert.IsNil(t, eT)
		company.Name = "TestCo"

		// Try correct one
		eT, err = provisioning.NewTenantAuthZ("TestName", pi, company)
		assert.NoErr(t, err)
		assert.Equal(t, eT.Name(), "TestName:AuthZFoundationTypes")
		// Provision the authz for tenant
		assert.NoErr(t, eT.Provision(ctx))

		// Validate the authz for tenant
		assert.NoErr(t, eT.Validate(ctx))

		assert.NoErr(t, eT.Cleanup(ctx))
		assert.NoErr(t, eT.Provision(ctx))
		assert.NoErr(t, eT.Close(ctx))
	})

	t.Run("ProvisionAuthService", func(t *testing.T) {
		t.Skip()
		_, _, companyDB := testhelpers.NewTestStorage(t)
		company := companyconfig.NewCompany("TestCo", companyconfig.CompanyTypeCustomer)
		tenantID := uuid.Must(uuid.NewV4())
		pi := types.ProvisionInfo{
			CompanyStorage: companyDB,
			TenantDB:       tenantDB,
			LogDB:          nil,
			CacheCfg:       cacheCfg,
			TenantID:       tenantID,
		}
		name := "test"
		op := types.NewOrderedProvisioner(name)
		op.AddProvisionables(
			types.NewOrderedProvisionable(provisioning.ProvisionDefaultOrg(name, pi, &company)).
				Named("default_org"),
			types.NewOrderedProvisionable(provisioning.ProvisionDefaultTypes(name, pi)).
				Named("authz_types"),
			types.NewOrderedProvisionable(provisioning.ProvisionEmployeeEntities(name, pi, &company)).
				Named("employees_authz").
				After("authz_types"),
			types.NewOrderedProvisionable(provisioning.ProvisionLoginApp(ctx, name, pi)).
				Named("login_app").
				After("authz_types"),
			types.NewOrderedProvisionable(provisioning.ProvisionLoginAppObjects(ctx, name, pi)).
				Named("login_app_objects").
				After("login_app"),
			types.NewOrderedProvisionable(provisioning.ProvisionCompanyOrgs(ctx, name, pi, &company)).
				Named("company_orgs").
				After("default_org"),
		)
		_, err := op.CreateProvisionable()
		assert.NoErr(t, err)
	})
}
