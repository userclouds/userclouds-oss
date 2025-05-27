package provisioning

import (
	"context"

	"userclouds.com/authz"
	authzprov "userclouds.com/authz/provisioning"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/provisioning/types"
)

// CleanUpUsers returns a ProvisionableMaker that can clean up users
func CleanUpUsers(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if !types.DeepProvisioning {
			return nil, nil
		}

		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot clean up users with nil tenantDB")
		}

		p, err := migrateUsersFromShadowObjects(ctx, name, pi)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		return []types.Provisionable{p}, nil
	}
}

// migrateUsersFromShadowObjects - used to create any missing AuthZ objects for the corresponding User objects
// TODO: eventually, we should have a continuous job that looks for and corrects mismatches
func migrateUsersFromShadowObjects(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) (types.Provisionable, error) {
	name = name + ":MigrateUsersFromShadowObjects"
	us := storage.NewUserStorage(ctx, pi.TenantDB, "", pi.TenantID) // TODO: this needs to be across all regions
	// read authz objects and add ids to set
	authzObjects, err := authzprov.ReadAuthZObjects(ctx, pi, authz.UserObjectTypeID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	authzUserIDs := set.NewUUIDSet()
	for _, a := range authzObjects {
		authzUserIDs.Insert(a.ID)
	}

	// TODO we could also page through authz objects and read in synch with users, since both are ordered by ID

	pager, err := storage.NewBaseUserPaginatorFromOptions(pagination.Limit(1500))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	provs := make([]types.Provisionable, 0)

	totalUsers := 0
	for {
		users, respFields, err := us.ListBaseUsersPaginated(ctx, *pager, false)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		totalUsers += len(users)

		for _, u := range users {
			if authzUserIDs.Evict(u.ID) {
				continue
			}

			// authz id does not exist for user so create

			prov := authzprov.NewEntityAuthZ(
				name,
				pi,
				nil,
				nil,
				[]authz.Object{
					{
						BaseModel:      ucdb.NewBaseWithID(u.ID),
						TypeID:         authz.UserObjectTypeID,
						OrganizationID: u.OrganizationID,
					},
				},
				nil,
				types.Provision,
				types.Validate,
			)
			provs = append(provs, prov)
		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}
	uclog.Debugf(ctx, "Read %d users", totalUsers)

	// TODO: the remaining authZUserIDs do not have a corresponding user and can be deleted

	p := types.NewParallelProvisioner(provs, name)
	return p, nil
}
