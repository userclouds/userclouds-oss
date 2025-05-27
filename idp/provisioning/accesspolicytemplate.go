package provisioning

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"userclouds.com/authz"
	provisioningAuthZ "userclouds.com/authz/provisioning"
	idpAuthz "userclouds.com/idp/authz"
	"userclouds.com/idp/events"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/provisioning/defaults"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/provisioning/types"
)

// ProvisionDefaultAccessPolicyTemplates returns a ProvisionableMaker that can provision default access policy templates
func ProvisionDefaultAccessPolicyTemplates(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provision default access policy templates with nil tenantDB")
		}

		if pi.LogDB == nil {
			return nil, ucerr.New("cannot provision default access policy templates with nil logDB")
		}

		s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)

		var provs []types.Provisionable
		for _, dapt := range defaults.GetDefaultAccessPolicyTemplates() {
			if isSoftDeleted, err := s.IsAccessPolicyTemplateSoftDeleted(ctx, dapt.ID); err != nil {
				return nil, ucerr.Wrap(err)
			} else if isSoftDeleted {
				continue
			}

			provs = append(
				provs,
				newProvisionerAccessPolicyTemplate(
					ctx,
					name,
					pi,
					dapt,
					types.Provision, types.Validate,
				),
			)
		}

		return provs, nil
	}
}

// provisionerAccessPolicyTemplate is a Provisionable object used to set up a single access policy template
type provisionerAccessPolicyTemplate struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	s        *storage.Storage
	logDB    *ucdb.DB
	template storage.AccessPolicyTemplate
}

// newProvisionerAccessPolicyTemplate return an initialized Provisionable object for initializing template
func newProvisionerAccessPolicyTemplate(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	template storage.AccessPolicyTemplate,
	pos ...types.ProvisionOperation,
) types.Provisionable {
	var provs []types.Provisionable

	name = fmt.Sprintf("%s:AccessPolicyTemplate(%v)", name, template.ID)

	// Serially provision the access policy template
	p := newProvisionerTemplate(ctx, name, pi, template, pos...)
	wp := types.NewWrappedProvisionable(p, name)
	provs = append(provs, wp)

	// Provision the AuthZ objects
	p = provisioningAuthZ.NewEntityAuthZ(
		name,
		pi,
		nil,
		nil,
		[]authz.Object{
			{BaseModel: ucdb.NewBaseWithID(template.ID), TypeID: idpAuthz.PolicyAccessTemplateTypeID},
		},
		nil,
		pos...,
	)
	provs = append(provs, p)

	return types.NewParallelProvisioner(provs, name)
}

// newProvisionerTemplate return an initialized Provisionable object for initializing template
func newProvisionerTemplate(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	template storage.AccessPolicyTemplate,
	pos ...types.ProvisionOperation,
) types.Provisionable {
	s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)

	papt := provisionerAccessPolicyTemplate{
		Named:          types.NewNamed(name + ":TemplateObject"),
		Parallelizable: types.NewParallelizable(pos...),
		s:              s,
		logDB:          pi.LogDB,
		template:       template,
	}
	return &papt
}

// GetData implements ControlSource
func (papt *provisionerAccessPolicyTemplate) GetData(context.Context) (any, error) {
	return events.GetEventsForAccessPolicyTemplate(papt.template.ID, papt.template.Version), nil
}

// Provision will provision the transform policy
func (papt *provisionerAccessPolicyTemplate) Provision(ctx context.Context) error {
	// Check if the template is already provisioned
	lapt, err := papt.s.GetLatestAccessPolicyTemplate(ctx, papt.template.ID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	// If there is no template, provision the new one
	if lapt == nil {
		uclog.Debugf(ctx, "Writing out template %v", papt.template.ID)
		if err := papt.s.SaveAccessPolicyTemplate(ctx, &papt.template); err != nil {
			return ucerr.Wrap(err)
		}
	} else if !lapt.ToClient().EqualsIgnoringNilID(papt.template.ToClient()) {
		// TODO: logging this as an error for now but continuing since we need an escape hatch for bugs in
		// our pre-canned GPs
		uclog.Errorf(ctx, "Template changed under same ID: %v", lapt.ID)
		uclog.Debugf(ctx, "Writing out template %v", lapt.ID)
		if err := papt.s.SaveAccessPolicyTemplate(ctx, &papt.template); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

// Validate with verify that template is correctly provisioned
func (papt *provisionerAccessPolicyTemplate) Validate(ctx context.Context) error {
	// Check both accessor paths to ensure you get the right policy
	lapt, err := papt.s.GetLatestAccessPolicyTemplate(ctx, papt.template.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	if lapt == nil {
		return ucerr.Errorf("expected template ID %v doesn't exist", papt.template.ID)
	}

	if !lapt.ToClient().EqualsIgnoringNilID(papt.template.ToClient()) {
		return ucerr.Errorf("Found template %v by ID, %+v doesn't match %+v", lapt.ID, lapt, papt.template)
	}

	return nil
}

// Cleanup cleans up objects associated with this access policy
func (papt *provisionerAccessPolicyTemplate) Cleanup(ctx context.Context) error {
	lapt, err := papt.s.GetLatestAccessPolicyTemplate(ctx, papt.template.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	if !errors.Is(err, sql.ErrNoRows) {
		if err := papt.s.DeleteAccessPolicyTemplateByVersion(ctx, lapt.ID, lapt.Version); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
