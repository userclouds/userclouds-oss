package storage

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
)

// DataTypeManager wraps the logic and data related to adding/removing data types
type DataTypeManager struct {
	idDataTypeMap   map[uuid.UUID]column.DataType
	nameDataTypeMap map[string]column.DataType
	lock            sync.RWMutex
	s               *Storage
}

// NewDataTypeManager creates a DataTypeManager initialized with current DB data type table state for given DB connection
func NewDataTypeManager(ctx context.Context, s *Storage) (*DataTypeManager, error) {
	dtm := DataTypeManager{s: s}

	if err := dtm.initDataTypeManager(ctx); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &dtm, nil
}

// initDataTypeManager initializes DataTypeManager with current DB data type table state for given DB connection.
func (dtm *DataTypeManager) initDataTypeManager(ctx context.Context) error {
	dtm.lock.Lock()
	defer dtm.lock.Unlock()

	dataTypes, err := dtm.s.ListDataTypesNonPaginated(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	dtm.idDataTypeMap = make(map[uuid.UUID]column.DataType, len(dataTypes))
	dtm.nameDataTypeMap = make(map[string]column.DataType, len(dataTypes))
	for _, dt := range dataTypes {
		dtm.idDataTypeMap[dt.ID] = dt
		dtm.nameDataTypeMap[strings.ToLower(dt.Name)] = dt
	}
	return nil
}

// GetDataTypeByID returns data type with given ID and nil if it doesn't exist
func (dtm *DataTypeManager) GetDataTypeByID(id uuid.UUID) *column.DataType {
	dt, found := dtm.idDataTypeMap[id]
	if !found {
		return nil
	}
	return &dt
}

// GetDataTypeByName returns data type with given name and nil if it doesn't exist
func (dtm *DataTypeManager) GetDataTypeByName(name string) *column.DataType {
	dt, found := dtm.nameDataTypeMap[strings.ToLower(name)]
	if !found {
		return nil
	}
	return &dt
}

// GetDataTypeByResourceID returns data type with given resource id, returning error if resource id invalid
func (dtm *DataTypeManager) GetDataTypeByResourceID(rid userstore.ResourceID) (*column.DataType, error) {
	if err := rid.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	var dataTypeForID *column.DataType
	var expectDataTypeForID bool
	if !rid.ID.IsNil() {
		dataTypeForID = dtm.GetDataTypeByID(rid.ID)
		expectDataTypeForID = true
	}

	var dataTypeForName *column.DataType
	var expectDataTypeForName bool
	if rid.Name != "" {
		dataTypeForName = dtm.GetDataTypeByName(rid.Name)
		expectDataTypeForName = true
	}

	if expectDataTypeForID {
		if dataTypeForID == nil {
			return nil, ucerr.Friendlyf(nil, "DataType ID '%v' is unrecognized", rid)
		}
		if !expectDataTypeForName {
			return dataTypeForID, nil
		}
	}

	if expectDataTypeForName {
		if dataTypeForName == nil {
			return nil, ucerr.Friendlyf(nil, "DataType Name '%v' is unrecognized", rid)
		}
		if !expectDataTypeForID {
			return dataTypeForName, nil
		}
		if dataTypeForID.ID != dataTypeForName.ID {
			return nil, ucerr.Friendlyf(nil, "DataType ID and Name '%v' do not match", rid)
		}
	}

	return dataTypeForID, nil
}

// GetDataTypes returns all data types
func (dtm *DataTypeManager) GetDataTypes() []column.DataType {
	dataTypes := make([]column.DataType, len(dtm.idDataTypeMap))
	i := 0
	for _, dt := range dtm.idDataTypeMap {
		dataTypes[i] = dt
		i++
	}
	return dataTypes
}

// SaveDataType creates data type if it doesn't exist or updates an existing data type if it does
func (dtm *DataTypeManager) SaveDataType(ctx context.Context, updated *column.DataType) (code int, err error) {
	if updated.ID.IsNil() {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "data type ID cannot be nil")
	}

	if current := dtm.GetDataTypeByID(updated.ID); current != nil {
		code, err = dtm.updateDataType(ctx, current, updated)
		return code, ucerr.Wrap(err)
	}

	code, err = dtm.createDataType(ctx, updated)
	return code, ucerr.Wrap(err)
}

// CreateDataTypeFromClient creates a data type on basis of client input
func (dtm *DataTypeManager) CreateDataTypeFromClient(ctx context.Context, cdt *userstore.ColumnDataType) (code int, err error) {
	if cdt.IsNative {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "Client cannot create a native data type")
	}

	dt := column.NewDataTypeFromClient(*cdt)

	if cdt.ID != uuid.Nil {
		if conflict := dtm.GetDataTypeByID(cdt.ID); conflict != nil {
			if dt.Equals(*conflict) {
				return http.StatusConflict,
					ucerr.Wrap(
						ucerr.WrapWithFriendlyStructure(
							jsonclient.Error{StatusCode: http.StatusConflict},
							jsonclient.SDKStructuredError{
								Error:     "This data type already exists",
								ID:        conflict.ID,
								Identical: true,
							},
						),
					)
			}
			return http.StatusConflict,
				ucerr.Wrap(
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error: fmt.Sprintf(`A data type with the ID '%s' already exists with name '%s'`, conflict.ID, conflict.Name),
							ID:    conflict.ID,
						},
					),
				)
		}
	} else {
		cdt.ID = dt.ID
	}

	code, err = dtm.createDataType(ctx, &dt)
	return code, ucerr.Wrap(err)
}

// createDataType creates data type and returns error if it exists
func (dtm *DataTypeManager) createDataType(ctx context.Context, dt *column.DataType) (code int, err error) {
	//  Check for conflicts with existing data types on name
	if conflict := dtm.GetDataTypeByName(dt.Name); conflict != nil {
		conflict.ID = dt.ID // Ignore the mismatch on ID for purposes of comparison
		if dt.Equals(*conflict) {
			return http.StatusConflict,
				ucerr.Wrap(
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error:     "This data type already exists",
							ID:        dt.ID,
							Identical: true,
						},
					),
				)
		}
		return http.StatusConflict,
			ucerr.Wrap(
				ucerr.WrapWithFriendlyStructure(
					jsonclient.Error{StatusCode: http.StatusConflict},
					jsonclient.SDKStructuredError{
						Error: fmt.Sprintf(`A data type with the name '%s' already exists with ID %v`, conflict.Name, conflict.ID),
						ID:    conflict.ID,
					},
				),
			)
	}

	if err := dt.Validate(); err != nil {
		return http.StatusBadRequest, ucerr.Wrap(err)
	}

	if err := dtm.s.SaveDataType(ctx, dt); err != nil {
		return uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
	}

	dtm.lock.Lock()
	dtm.idDataTypeMap[dt.ID] = *dt
	dtm.nameDataTypeMap[strings.ToLower(dt.Name)] = *dt
	dtm.lock.Unlock()

	return http.StatusOK, nil
}

// UpdateDataTypeFromClient updates a data type on basis of client input
func (dtm *DataTypeManager) UpdateDataTypeFromClient(ctx context.Context, cdt *userstore.ColumnDataType) (code int, err error) {
	current := dtm.GetDataTypeByID(cdt.ID)
	if current == nil {
		return http.StatusNotFound, ucerr.Errorf("Data type not found: %v", cdt.ID)
	}

	if current.IsNative() {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "Client cannot update a native data type")
	}

	if cdt.IsNative {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "Client cannot update a data type to be native")
	}

	updated := column.NewDataTypeFromClient(*cdt)

	if code, err := dtm.checkForDataTypeReference(ctx, current.ID); err != nil {
		if code != http.StatusConflict {
			return code, ucerr.Wrap(err)
		}

		currentWithAllowedChanges := *current
		currentWithAllowedChanges.Name = updated.Name
		currentWithAllowedChanges.Description = updated.Description
		if !currentWithAllowedChanges.Equals(updated) {
			return http.StatusConflict, ucerr.Friendlyf(err, "Client can only update a data type Name or Description when it is in use")
		}
	}

	code, err = dtm.updateDataType(ctx, current, &updated)
	return code, ucerr.Wrap(err)
}

// updateDataType updates a data type if it exists and returns error if it doesn't or the update is incompatible with existing data type
func (dtm *DataTypeManager) updateDataType(ctx context.Context, current *column.DataType, updated *column.DataType) (code int, err error) {
	if current.Equals(*updated) && current.Name == updated.Name { // TODO: should we do the same for column?
		return http.StatusNotModified, nil
	}

	if err := updated.Validate(); err != nil {
		return http.StatusBadRequest, ucerr.Wrap(err)
	}

	if current.Name != updated.Name {
		conflict := dtm.GetDataTypeByName(updated.Name)
		if conflict != nil && conflict.ID != current.ID {
			return http.StatusConflict,
				ucerr.Wrap(
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error: fmt.Sprintf(`A data type with the name '%s' already exists with ID %v`, updated.Name, conflict.ID),
							ID:    conflict.ID,
						},
					),
				)
		}
	}

	if err := dtm.s.SaveDataType(ctx, updated); err != nil {
		return uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
	}

	dtm.lock.Lock()
	// Remove the old name from name map in case of a rename
	delete(dtm.nameDataTypeMap, strings.ToLower(dtm.idDataTypeMap[updated.ID].Name))
	dtm.idDataTypeMap[updated.ID] = *updated
	dtm.nameDataTypeMap[strings.ToLower(updated.Name)] = *updated
	dtm.lock.Unlock()

	return http.StatusOK, nil
}

// DeleteDataType deletes given data type and returns error if it doesn't exists
func (dtm *DataTypeManager) DeleteDataType(ctx context.Context, id uuid.UUID) (code int, err error) {
	dt := dtm.GetDataTypeByID(id)
	if dt == nil {
		return http.StatusNotFound, ucerr.Friendlyf(nil, "data type %v could not be found", id)
	}

	if dt.IsNative() {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "native data types can not be deleted")
	}

	code, err = dtm.deleteDataType(ctx, dt)
	return code, ucerr.Wrap(err)
}

func (dtm *DataTypeManager) deleteDataType(ctx context.Context, dt *column.DataType) (code int, err error) {
	if code, err = dtm.checkForDataTypeReference(ctx, dt.ID); err != nil {
		return code, ucerr.Wrap(err)
	}

	if err = dtm.s.DeleteDataType(ctx, dt.ID); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	dtm.lock.Lock()
	delete(dtm.idDataTypeMap, dt.ID)
	delete(dtm.nameDataTypeMap, strings.ToLower(dt.Name))
	dtm.lock.Unlock()

	return http.StatusOK, nil
}

// checkForDataTypeReference checks if there is a reference to the data type in the DB
func (dtm *DataTypeManager) checkForDataTypeReference(ctx context.Context, id uuid.UUID) (code int, err error) {

	dt := dtm.GetDataTypeByID(id)
	if dt == nil {
		return http.StatusNotFound, ucerr.Friendlyf(nil, "Data type not found: %v", id)
	}

	// check if any columns use this data type

	cm, err := NewUserstoreColumnManager(ctx, dtm.s)
	if err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}
	columns := cm.GetColumns()

	for _, c := range columns {
		if c.DataTypeID == dt.ID {
			return http.StatusConflict, ucerr.Friendlyf(nil, "Column '%s' still references data type '%s'", c.Name, dt.Name)
		}
	}

	// check if any transformers use this data type

	transformers, err := dtm.s.ListTransformersNonPaginated(ctx)
	if err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	for _, t := range transformers {
		if t.InputDataTypeID == dt.ID {
			return http.StatusConflict,
				ucerr.Friendlyf(nil, "Transformer '%s' InputDataTypeID still references data type '%s'", t.Name, dt.Name)
		}

		if t.OutputDataTypeID == dt.ID {
			return http.StatusConflict,
				ucerr.Friendlyf(nil, "Transformer '%s' OutputDataTypeID still references data type '%s'", t.Name, dt.Name)
		}
	}

	return http.StatusOK, nil
}
