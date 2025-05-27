package tenant

import (
	"context"

	"github.com/gofrs/uuid"

	authz "userclouds.com/authz/provisioning"
	idp "userclouds.com/idp/provisioning"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctrace"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/logdb"
	"userclouds.com/internal/provisioning/types"
)

var tracer = uctrace.NewTracer("provisioning")

// ServicesProvisioner wraps all the services we need to provision and keeps it responsible for the lifecycle of the DB connections
// to tenant and log dbs
type ServicesProvisioner struct {
	types.Named
	types.Parallelizable
	tenantDB *ucdb.DB
	logDB    *ucdb.DB
	p        types.Provisionable
}

// NewServicesProvisioner returns an initialized Provisionable for initializing default service objects
func NewServicesProvisioner(
	ctx context.Context,
	name string,
	companyDB *companyconfig.Storage,
	tenantDB *ucdb.DB,
	logDBConfig *ucdb.Config,
	cacheCfg *cache.Config,
	company *companyconfig.Company,
	tenantID uuid.UUID,
	employeeIDs ...uuid.UUID,
) (types.Provisionable, error) {
	logDB, err := logdb.ConnectWithConfig(ctx, logDBConfig)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	name += ":ServicesProvisioner"

	op := types.NewOrderedProvisioner(name)
	pi := types.ProvisionInfo{
		CompanyStorage: companyDB,
		TenantDB:       tenantDB,
		LogDB:          logDB,
		CacheCfg:       cacheCfg,
		TenantID:       tenantID,
	}

	op.AddProvisionables(
		types.NewOrderedProvisionable(authz.ProvisionDefaultTypes(name, pi)).
			Named("authz_types"),
		types.NewOrderedProvisionable(authz.ProvisionDefaultOrg(name, pi, company)).
			Named("default_org").
			After("authz_types"),
		types.NewOrderedProvisionable(authz.ProvisionEmployeeEntities(name, pi, company, employeeIDs...)).
			Named("employees_authz").
			After("default_org"),
		types.NewOrderedProvisionable(idp.ProvisionTokenizerAuthzEntities(name, pi, company)).
			Named("tokenizer_authz_entities").
			After("default_org"),
		types.NewOrderedProvisionable(authz.ProvisionLoginApp(ctx, name, pi)).
			Named("login_app").
			After("tokenizer_authz_entities"),
		types.NewOrderedProvisionable(authz.ProvisionLoginAppObjects(ctx, name, pi)).
			Named("login_app_objects").
			After("login_app"),
		types.NewOrderedProvisionable(authz.ProvisionCompanyOrgs(ctx, name, pi, company)).
			Named("company_orgs").
			After("default_org"),
		types.NewOrderedProvisionable(idp.ProvisionDefaultAccessPolicyTemplates(ctx, name, pi)).
			Named("access_policy_templates").
			After("tokenizer_authz_entities"),
		types.NewOrderedProvisionable(idp.ProvisionDefaultDataTypes(ctx, name, pi)).
			Named("data_types").
			After("tokenizer_authz_entities"),
		types.NewOrderedProvisionable(idp.ProvisionDefaultPurposes(ctx, name, pi)).
			Named("purposes").
			After("tokenizer_authz_entities"),
		types.NewOrderedProvisionable(idp.ProvisionDefaultTransformers(ctx, name, pi)).
			Named("transformers").
			After("data_types"),
		types.NewOrderedProvisionable(idp.CleanUpTransformers(ctx, name, pi)).
			Named("cleanup_transformers").
			After("transformers"),
		types.NewOrderedProvisionable(idp.ProvisionDefaultAccessPolicies(ctx, name, pi)).
			Named("access_policies").
			After("access_policy_templates"),
		types.NewOrderedProvisionable(idp.CleanUpAccessPolicies(ctx, name, pi)).
			Named("cleanup_access_policies").
			After("access_policies"),
		types.NewOrderedProvisionable(idp.ProvisionDefaultColumns(ctx, name, pi)).
			Named("columns").
			After("transformers", "cleanup_access_policies"),
		types.NewOrderedProvisionable(idp.ProvisionDefaultAccessors(ctx, name, pi)).
			Named("accessors").
			After("columns", "purposes", "cleanup_access_policies", "cleanup_transformers"),
		types.NewOrderedProvisionable(idp.ProvisionDefaultMutators(ctx, name, pi)).
			Named("mutators").
			After("columns", "purposes", "cleanup_access_policies", "cleanup_transformers"),
		types.NewOrderedProvisionable(idp.CleanUpUsers(ctx, name, pi)).
			Named("cleanup_users").
			After("accessors", "mutators", "company_orgs"),
	)

	p, err := op.CreateProvisionable()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	sp := ServicesProvisioner{
		Named:          types.NewNamed(name),
		Parallelizable: types.NewParallelizable(),
		tenantDB:       tenantDB,
		logDB:          logDB,
		p:              p,
	}
	return &sp, nil
}

// Provision implements Provisionable
func (s *ServicesProvisioner) Provision(ctx context.Context) error {
	return uctrace.Wrap0(ctx, tracer, "ProvisionServices", true, s.p.Provision)
}

// Validate implements Provisionable.
func (s *ServicesProvisioner) Validate(ctx context.Context) error {
	return ucerr.Wrap(s.p.Validate(ctx))
}

// Cleanup implements Provisionable.
func (s *ServicesProvisioner) Cleanup(ctx context.Context) error {
	return ucerr.Wrap(s.p.Cleanup(ctx))
}

// Close cleans up resources that maybe used by a ServicesProvisioner
func (s *ServicesProvisioner) Close(ctx context.Context) error {
	if s.logDB != nil {
		if err := s.logDB.Close(ctx); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
