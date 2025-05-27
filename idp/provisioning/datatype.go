package provisioning

import (
	"context"
	"net/http"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/provisioning/defaults"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/provisioning/types"
)

// ProvisionDefaultDataTypes returns a ProvisionableMaker that can provision data types
func ProvisionDefaultDataTypes(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		return ProvisionDataTypes(ctx, name, pi, defaults.GetDefaultDataTypes()...)()
	}
}

// ProvisionDataTypes returns a ProvisionableMaker that can provision the specified data types
func ProvisionDataTypes(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	dataTypes ...column.DataType,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if len(dataTypes) == 0 {
			return nil, ucerr.New("no data types specified")
		}

		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provision data types with nil tenantDB")
		}

		s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)
		dtm, err := storage.NewDataTypeManager(ctx, s)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		var allowedDataTypes []column.DataType

		for _, dt := range dataTypes {
			if isSoftDeleted, err := s.IsDataTypeSoftDeleted(ctx, dt.ID); err != nil {
				return nil, ucerr.Wrap(err)
			} else if isSoftDeleted {
				continue
			}

			allowedDataTypes = append(allowedDataTypes, dt)
		}

		if len(allowedDataTypes) == 0 {
			return nil, nil
		}

		return []types.Provisionable{
			&dataTypeProvisioner{
				Named:          types.NewNamed(name + ":DataTypes"),
				Parallelizable: types.AllParallelizable(),
				dtm:            dtm,
				dataTypes:      allowedDataTypes,
			},
		}, nil
	}
}

// dataTypeProvisioner is a Provisionable object used to set up a set of data types
type dataTypeProvisioner struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	dtm       *storage.DataTypeManager
	dataTypes []column.DataType
}

// Provision implements Provisionable and creates a data type
func (dtp *dataTypeProvisioner) Provision(ctx context.Context) error {
	for _, dt := range dtp.dataTypes {
		if code, err := dtp.dtm.SaveDataType(ctx, &dt); err != nil {
			if code == http.StatusConflict {
				uclog.Errorf(
					ctx,
					"unable to provision data type '%s', possibly due to collision with existing data type",
					dt.Name,
				)
			} else {
				return ucerr.Wrap(err)
			}
		}
	}
	return nil
}

// Validate implements Provisionable and validates that the data type was created
func (dtp *dataTypeProvisioner) Validate(ctx context.Context) error {
	for _, dt := range dtp.dataTypes {
		currDataType := dtp.dtm.GetDataTypeByID(dt.ID)
		if currDataType == nil {
			return ucerr.Errorf("data type '%s' not found in data_types table", dt.Name)
		}

		if !dt.Equals(*currDataType) {
			return ucerr.Errorf("Found data type '%+v' does not match '%+v'", *currDataType, dt)
		}
	}

	return nil
}

// Cleanup implements Provisionable and removes the data type
func (dtp *dataTypeProvisioner) Cleanup(ctx context.Context) error {
	for _, dt := range dtp.dataTypes {
		// If the data type exists, delete it, but don't error if it doesn't
		if code, err := dtp.dtm.DeleteDataType(ctx, dt.ID); err != nil {
			if code != http.StatusNotFound {
				return ucerr.Wrap(err)
			}
		}
	}

	return nil
}
