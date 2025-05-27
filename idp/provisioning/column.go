package provisioning

import (
	"context"
	"net/http"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/provisioning/defaults"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/provisioning/types"
)

// ProvisionDefaultColumns returns a ProvisionableMaker that can provision default columns
func ProvisionDefaultColumns(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		return ProvisionColumns(ctx, name, pi, defaults.GetDefaultColumns()...)()
	}
}

// ProvisionColumns returns a ProvisionableMaker that can provision the specified columns
func ProvisionColumns(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	columns ...storage.Column,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if len(columns) == 0 {
			return nil, ucerr.New("no columns specified")
		}

		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provision columns with nil tenantDB")
		}

		s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)

		cm, err := storage.NewUserstoreColumnManager(ctx, s)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		var allowedColumns []storage.Column

		for _, c := range columns {
			if isSoftDeleted, err := s.IsColumnSoftDeleted(ctx, c.ID); err != nil {
				return nil, ucerr.Wrap(err)
			} else if isSoftDeleted {
				continue
			}

			allowedColumns = append(allowedColumns, c)
		}

		if len(allowedColumns) == 0 {
			return nil, nil
		}

		return []types.Provisionable{
			&columnProvisioner{
				Named:          types.NewNamed(name + ":Columns"),
				Parallelizable: types.AllParallelizable(),
				cm:             cm,
				columns:        allowedColumns,
			},
		}, nil
	}
}

// columnProvisioner is a Provisionable object used to set up a set of columns
type columnProvisioner struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	cm      *storage.ColumnManager
	columns storage.Columns
}

// Provision implements Provisionable and creates a set of columns
func (cp *columnProvisioner) Provision(ctx context.Context) error {
	for _, c := range cp.columns {
		// TODO: once we enforce rule that default objects can only be
		//       updated by UserClouds, this should be unnecessary
		// if this is a change, be careful about updating it
		if existing := cp.cm.GetColumnByID(c.ID); existing != nil {
			// Don't change names of columns in case the user update
			if existing.Name != c.Name ||
				existing.DataTypeID != c.DataTypeID ||
				existing.DefaultValue != c.DefaultValue ||
				existing.IsArray != c.IsArray ||
				existing.IndexType != c.IndexType {
				uclog.Infof(ctx, "requested column %v already exists, but is different, currently %v ... not modifying", c, existing)
				continue
			}
		}

		if code, err := cp.cm.SaveColumn(ctx, &c); err != nil {
			if code == http.StatusConflict {
				uclog.Errorf(ctx, "unable to provision column %s, possibly due to name collision with existing column", c.Name)
			} else {
				return ucerr.Wrap(err)
			}
		}
	}
	return nil
}

// Validate implements Provisionable and validates that the columns were created
func (cp *columnProvisioner) Validate(ctx context.Context) error {
	for _, c := range cp.columns {
		currCol := cp.cm.GetColumnByID(c.ID)
		if currCol == nil {
			return ucerr.Errorf("column %s not found in columns table", c.Name)
		}

		if !currCol.Equals(&c) {
			return ucerr.Errorf("column %s doesn't match in columns table\nprov %v\ntable %v", c.Name, c, currCol)
		}
	}

	return nil
}

// Cleanup implements Provisionable and removes the specified set of columns
func (cp *columnProvisioner) Cleanup(ctx context.Context) error {
	for _, c := range cp.columns {
		if code, err := cp.cm.DeleteColumn(ctx, c.ID); err != nil {
			if code == http.StatusNotFound {
				continue
			}

			return ucerr.Wrap(err)
		}
	}

	return nil
}
