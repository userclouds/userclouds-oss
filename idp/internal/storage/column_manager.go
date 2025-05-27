package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
)

// ColumnManager wraps the logic and data related to adding/removing columns
type ColumnManager struct {
	idColumnMap   map[uuid.UUID]Column
	nameColumnMap map[string]Column
	colLock       sync.RWMutex
	s             *Storage
	databaseID    uuid.UUID
}

// NewUserstoreColumnManager creates a ColumnManager initialized with UserClouds database ID (nil)
func NewUserstoreColumnManager(ctx context.Context, s *Storage) (*ColumnManager, error) {
	return NewColumnManager(ctx, s, uuid.Nil)
}

// NewColumnManager creates a ColumnManager initialized with current DB column table state for given DB connection
func NewColumnManager(ctx context.Context, s *Storage, databaseID uuid.UUID) (*ColumnManager, error) {
	uclog.Verbosef(ctx, "NewColumnManager for %v", databaseID)
	cm := ColumnManager{s: s, databaseID: databaseID}

	if err := cm.initColumnManager(ctx); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &cm, nil
}

func ncmKey(table, name string) string {
	return strings.ToLower(fmt.Sprintf("%s.%s", table, name))
}

// initColumnManager initializes ColumnManager with current DB column table state for given DB connection.
func (cm *ColumnManager) initColumnManager(ctx context.Context) error {
	cm.colLock.Lock()
	defer cm.colLock.Unlock()

	pager, err := NewColumnPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return ucerr.Wrap(err)
	}
	dbColumns := make([]Column, 0)
	for {
		objRead, respFields, err := cm.s.ListColumnsPaginated(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}
		dbColumns = append(dbColumns, objRead...)
		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	// don't allocate these since we have no idea how many cols are in the database ID we're looking for
	cm.idColumnMap = make(map[uuid.UUID]Column)
	cm.nameColumnMap = make(map[string]Column)
	for _, col := range dbColumns {
		if col.SQLShimDatabaseID == cm.databaseID {
			cm.idColumnMap[col.ID] = col
			cm.nameColumnMap[ncmKey(col.Table, col.Name)] = col
		}
	}
	return nil
}

// validateColumnManagerAgainstColumnTable verifies that ColumnManager matches the current DB column table state and return true if state matches
func (cm *ColumnManager) validateColumnManagerAgainstColumnTable(ctx context.Context) (bool, error) {
	pager, err := NewColumnPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return false, ucerr.Wrap(err)
	}
	dbcolumns := make([]Column, 0)
	for {
		objRead, respFields, err := cm.s.ListColumnsPaginated(ctx, *pager)
		if err != nil {
			return false, ucerr.Wrap(err)
		}
		dbcolumns = append(dbcolumns, objRead...)
		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}
	foundColumns := 0
	for _, dbCol := range dbcolumns {
		if dbCol.SQLShimDatabaseID == cm.databaseID {
			col, ok := cm.idColumnMap[dbCol.ID]
			if !ok {
				return false, nil
			}
			if !col.Equals(&dbCol) {
				uclog.Infof(ctx, "Cols db %v state %v", dbCol, col)
				return false, nil
			}
			foundColumns++
		}
	}
	if foundColumns != len(cm.idColumnMap) {
		return false, nil
	}

	return true, nil
}

// GetColumnByID returns columns with given ID and nil if it doesn't exist
func (cm *ColumnManager) GetColumnByID(id uuid.UUID) *Column {
	col, ok := cm.idColumnMap[id]
	if !ok {
		return nil
	}
	return &col
}

// GetColumnByTableAndName returns columns with given table/name and nil if it doesn't exist
func (cm *ColumnManager) GetColumnByTableAndName(table, name string) *Column {
	col, ok := cm.nameColumnMap[ncmKey(table, name)]
	if !ok {
		return nil
	}
	return &col
}

// GetUserColumnByName returns columns with given name specifically from the users table, and nil if it doesn't exist
func (cm *ColumnManager) GetUserColumnByName(name string) *Column {
	return cm.GetColumnByTableAndName(userTableName, name)
}

// GetColumns returns all columns
func (cm *ColumnManager) GetColumns() []Column {
	columns := make([]Column, len(cm.idColumnMap))
	i := 0
	for _, c := range cm.idColumnMap {
		columns[i] = c
		i++
	}
	return columns
}

// GetColumnsByTable returns all columns for a given table
func (cm *ColumnManager) GetColumnsByTable(table string) []Column {
	columns := []Column{}
	for _, c := range cm.idColumnMap {
		if c.Table == table {
			columns = append(columns, c)
		}
	}
	return columns
}

// SaveColumn creates column if it doesn't exist or updates an existing column if it does
func (cm *ColumnManager) SaveColumn(ctx context.Context, updated *Column) (code int, err error) {
	if updated.SQLShimDatabaseID != cm.databaseID {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "column SQLShimDatabaseID must match the column manager's database ID")
	}

	if updated.ID.IsNil() {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "column ID cannot be nil")
	}

	if current := cm.GetColumnByID(updated.ID); current != nil {
		code, err := cm.updateColumn(ctx, current, updated)
		return code, ucerr.Wrap(err)
	}

	code, err = cm.createColumn(ctx, updated)
	return code, ucerr.Wrap(err)
}

// CreateColumnFromClient creates a column on basis of client input
func (cm *ColumnManager) CreateColumnFromClient(ctx context.Context, clientCol *userstore.Column) (code int, err error) {
	if clientCol.IsSystem {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "system columns cannot be created by the client")
	}

	col := NewColumnFromClient(*clientCol)
	col.SQLShimDatabaseID = cm.databaseID

	if clientCol.ID != uuid.Nil {
		if c := cm.GetColumnByID(clientCol.ID); c != nil {
			if col.Equals(c) {
				return http.StatusConflict,
					ucerr.Wrap(
						ucerr.WrapWithFriendlyStructure(
							jsonclient.Error{StatusCode: http.StatusConflict},
							jsonclient.SDKStructuredError{
								Error:     "This column already exists",
								ID:        c.ID,
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
							Error: fmt.Sprintf(`A column with the ID '%s' already exists with name '%s'`, col.ID, c.Name),
							ID:    col.ID,
						},
					),
				)
		}
	} else {
		clientCol.ID = col.ID
	}

	code, err = cm.createColumn(ctx, &col)
	return code, ucerr.Wrap(err)
}

var validTableNameRegex = regexp.MustCompile(`^[a-zA-Z_]((\.?)[a-zA-Z0-9_-]*)*$`)

// createColumn creates column and returns error if it exists
func (cm *ColumnManager) createColumn(ctx context.Context, col *Column) (code int, err error) {
	// TODO (sgarrity 6/24): keeping this for back compat, should be able to remove soon
	if col.Table == "" {
		uclog.Warningf(ctx, "Column table is empty, defaulting to %s", userTableName)
		col.Table = userTableName
	}

	if !validTableNameRegex.MatchString(col.Table) {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "column table name '%s' is invalid", col.Table)
	}

	if err := cm.validateColumnName(col.Name); err != nil {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "column name '%s' is invalid", col.Name)
	}

	//  Check for conflicts with existing columns on name
	if c := cm.GetColumnByTableAndName(col.Table, col.Name); c != nil {
		col.ID = c.ID // Ignore the mismatch on ID for purposes of comparison
		if col.Equals(c) {
			return http.StatusConflict,
				ucerr.Wrap(
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error:     "This column already exists",
							ID:        c.ID,
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
						Error: fmt.Sprintf(`A column with the name '%s' already exists with ID %v`, col.Name, c.ID),
						ID:    c.ID,
					},
				),
			)
	}

	if err := cm.validateColumnAndDataType(ctx, col); err != nil {
		return http.StatusBadRequest, ucerr.Wrap(err)
	}

	if err := cm.validateDefaultValue(ctx, col); err != nil {
		return http.StatusBadRequest, ucerr.Wrap(err)
	}

	if err := cm.validateDefaultTransformer(ctx, col); err != nil {
		return http.StatusBadRequest, ucerr.Wrap(err)
	}

	if err := cm.s.SaveColumn(ctx, col); err != nil {
		return uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
	}

	cm.colLock.Lock()
	cm.idColumnMap[col.ID] = *col
	cm.nameColumnMap[ncmKey(col.Table, col.Name)] = *col
	cm.colLock.Unlock()

	if err := cm.updateDefaultUserstoreAccessorAndMutator(ctx); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	return http.StatusOK, nil
}

// UpdateColumnFromClient updates a column on basis of client input and doesn't allow override/setting of attributes
func (cm *ColumnManager) UpdateColumnFromClient(ctx context.Context, clientCol *userstore.Column) (code int, err error) {
	current := cm.GetColumnByID(clientCol.ID)
	if current == nil {
		return http.StatusNotFound, ucerr.Errorf("Column not found: %v", clientCol.ID)
	}

	if current.Attributes.System {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "System columns cannot be updated by the client")
	}

	if clientCol.IsSystem {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "Column cannot be made a system column by the client")
	}

	updated := current.updateColumnFromClient(*clientCol)
	code, err = cm.updateColumn(ctx, current, &updated)
	return code, ucerr.Wrap(err)
}

// updateColumn updates a column if it exists and returns error if it doesn't or the update is incompatible with existing column
func (cm *ColumnManager) updateColumn(ctx context.Context, current *Column, updated *Column) (code int, err error) {
	if current.CaseSensitiveEquals(updated) {
		return http.StatusNotModified, nil
	}

	// TODO (sgarrity 6/24): keeping this for back compat, should be able to remove soon
	if updated.Table == "" {
		uclog.Warningf(ctx, "Column table is empty, defaulting to %s", current.Table)
		updated.Table = current.Table
	} else if updated.Table != current.Table {
		return http.StatusBadRequest,
			ucerr.Friendlyf(
				nil,
				"Cannot change column table: %v->%v %v",
				current.Table,
				updated.Table,
				current.ToStringConcise(),
			)
	}

	if err := updated.Validate(); err != nil {
		return http.StatusBadRequest, ucerr.Wrap(err)
	}

	if current.DataTypeID != updated.DataTypeID {
		return http.StatusBadRequest,
			ucerr.Friendlyf(
				nil,
				"Cannot change column data type: %v->%v %v",
				current.DataTypeID,
				updated.DataTypeID,
				current.ToStringConcise(),
			)
	}

	// partial updates can be enabled via an update, but not disabled
	if !current.Attributes.Constraints.PartialUpdates && updated.Attributes.Constraints.PartialUpdates {
		current.Attributes.Constraints.PartialUpdates = true
	}

	if !current.Attributes.Constraints.Equals(updated.Attributes.Constraints) {
		return http.StatusBadRequest,
			ucerr.Friendlyf(
				nil,
				"Cannot change column constraints: %v->%v %v",
				current.Attributes.Constraints,
				updated.Attributes.Constraints,
				current.ToStringConcise(),
			)
	}

	if current.IsArray != updated.IsArray {
		return http.StatusBadRequest,
			ucerr.Friendlyf(
				nil,
				"Cannot change column is_array: %v->%v %v",
				current.IsArray,
				updated.IsArray,
				current.ToStringConcise(),
			)
	}

	if current.IndexType != updated.IndexType {
		if current.IndexType == columnIndexTypeUnique || updated.IndexType == columnIndexTypeUnique {
			return http.StatusBadRequest,
				ucerr.Friendlyf(
					nil,
					"Cannot currently add/remove index_type unique constraint for column: %v",
					current.ToStringConcise(),
				)
		}
		current.IndexType = updated.IndexType
	}

	// data type id can be set via an update if it was not set before
	if current.DataTypeID != updated.DataTypeID {
		if !current.DataTypeID.IsNil() {
			return http.StatusBadRequest,
				ucerr.Friendlyf(
					nil,
					"Cannot change column data_type_id: %v->%v %v",
					current.DataTypeID,
					updated.DataTypeID,
					current.ToStringConcise(),
				)
		}

		current.DataTypeID = updated.DataTypeID
	}

	if current.DefaultValue != updated.DefaultValue {
		if err := cm.validateDefaultValue(ctx, updated); err != nil {
			return http.StatusBadRequest, ucerr.Wrap(err)
		}
		current.DefaultValue = updated.DefaultValue
	}

	if current.AccessPolicyID != updated.AccessPolicyID {
		if err := cm.validateAccessPolicy(ctx, updated); err != nil {
			return http.StatusBadRequest, ucerr.Wrap(err)
		}
		current.AccessPolicyID = updated.AccessPolicyID
	}

	if current.DefaultTransformerID != updated.DefaultTransformerID || current.DefaultTokenAccessPolicyID != updated.DefaultTokenAccessPolicyID {
		if err := cm.validateDefaultTransformer(ctx, updated); err != nil {
			return http.StatusBadRequest, ucerr.Wrap(err)
		}
		current.DefaultTransformerID = updated.DefaultTransformerID
		current.DefaultTokenAccessPolicyID = updated.DefaultTokenAccessPolicyID
	}

	if !current.Attributes.Equals(updated.Attributes) {
		if current.Attributes.System != updated.Attributes.System {
			// TODO these can potentially change on provisioning and we need to write more code to handle that
			return http.StatusBadRequest, ucerr.New("System attribute update is not implemented")
		}
		current.Attributes = updated.Attributes
	}

	if current.SearchIndexed != updated.SearchIndexed {
		if !updated.SearchIndexed {
			sim, err := NewSearchIndexManager(ctx, cm.s)
			if err != nil {
				return http.StatusInternalServerError, ucerr.Wrap(err)
			}

			if err := sim.CheckColumnUnindexed(*updated); err != nil {
				return http.StatusBadRequest, ucerr.Wrap(err)
			}
		}
		current.SearchIndexed = updated.SearchIndexed
	}

	oldName := current.Name
	current.Name = updated.Name
	if oldName != current.Name {
		if err := cm.validateColumnName(current.Name); err != nil {
			return http.StatusBadRequest, ucerr.Friendlyf(nil, "column name '%s' is invalid", current.Name)
		}

		c := cm.GetColumnByTableAndName(current.Table, current.Name)
		if c != nil && c.ID != current.ID {
			return http.StatusConflict,
				ucerr.Wrap(
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error: fmt.Sprintf(`A column with the name '%s' already exists with ID %v`, current.Name, c.ID),
							ID:    c.ID,
						},
					),
				)
		}
	}

	if err := cm.s.SaveColumn(ctx, current); err != nil {
		return uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
	}

	cm.colLock.Lock()
	// Remove the old name from name map in case of a rename
	delete(cm.nameColumnMap, ncmKey(cm.idColumnMap[updated.ID].Table, cm.idColumnMap[updated.ID].Name))
	cm.idColumnMap[updated.ID] = *current
	cm.nameColumnMap[ncmKey(updated.Table, updated.Name)] = *current
	cm.colLock.Unlock()

	if err := cm.updateDefaultUserstoreAccessorAndMutator(ctx); err != nil {
		// this is a bad state to leave the system in, so retry
		time.Sleep(100 * time.Millisecond)
		if err := cm.updateDefaultUserstoreAccessorAndMutator(ctx); err != nil {
			return http.StatusInternalServerError, ucerr.Friendlyf(err, "Failed to update default accessor or mutator on column %v update", current.ToStringConcise())
		}
		uclog.Debugf(ctx, "Needed to retry updating default accessor or mutator on column %v update", current.ToStringConcise())
	}

	if oldName != current.Name {
		// this is a bad state to leave the system in, so retry
		if err := cm.updateColumnNameInSelectorConfigs(ctx, oldName, current.Name); err != nil {
			time.Sleep(100 * time.Millisecond)
			if err := cm.updateColumnNameInSelectorConfigs(ctx, oldName, current.Name); err != nil {
				return http.StatusInternalServerError, ucerr.Friendlyf(err, "Failed to update selector configs on column %v update", current.ToStringConcise())
			}
			uclog.Debugf(ctx, "Needed to retry updating selectors for accessors and mutators on column %v update", current.ToStringConcise())
		}
	}

	return http.StatusOK, nil
}

// DeleteColumnFromClient deletes given column for a client, disallowing deletion of system columns.
func (cm *ColumnManager) DeleteColumnFromClient(ctx context.Context, id uuid.UUID) (code int, err error) {
	col := cm.GetColumnByID(id)
	if col == nil {
		return http.StatusNotFound, ucerr.Friendlyf(nil, "column %v could not be found", id)
	}

	if col.Attributes.System {
		return http.StatusBadRequest, ucerr.Friendlyf(nil, "System columns can not be deleted")
	}
	code, err = cm.deleteColumn(ctx, col)
	return code, ucerr.Wrap(err)
}

// DeleteColumn deletes given column and returns error if it doesn't exists
func (cm *ColumnManager) DeleteColumn(ctx context.Context, id uuid.UUID) (code int, err error) {
	col := cm.GetColumnByID(id)
	if col == nil {
		return http.StatusNotFound, ucerr.Friendlyf(nil, "column %v could not be found", id)
	}

	code, err = cm.deleteColumn(ctx, col)
	return code, ucerr.Wrap(err)
}

func (cm *ColumnManager) deleteColumn(ctx context.Context, col *Column) (code int, err error) {
	if code, err := cm.checkForColumnReferenceForDeletion(ctx, col.ID); err != nil {
		return code, ucerr.Wrap(err)
	}

	if err := cm.s.DeleteColumn(ctx, col.ID); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	cm.colLock.Lock()
	delete(cm.idColumnMap, col.ID)
	delete(cm.nameColumnMap, ncmKey(col.Table, col.Name))
	cm.colLock.Unlock()

	// TODO this will fail on deletion of the last column in the userstore because accessor needs at least one column to fetch
	if err := cm.updateDefaultUserstoreAccessorAndMutator(ctx); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	return http.StatusOK, nil
}

// updateDefaultUserstoreAccessorAndMutator updates the default functions on basis of new column state with retry
func (cm *ColumnManager) updateDefaultUserstoreAccessorAndMutator(ctx context.Context) error {
	if !cm.databaseID.IsNil() {
		return nil
	}

	// TODO because we are not doing this in a transaction this will result in corrupt state if another caller adds/removes/renames a column during
	// execution of this command and neither this thread nor the other caller has complete state. To minimize the duration of corrupt state, we re-read
	// the state post update and repeat the operation if state changed
	numRetries := 0
	for {
		err := cm.doUpdateDefaultUserstoreAccessorAndMutator(ctx)
		if err != nil {
			return ucerr.Wrap(err)
		}

		noChanges, err := cm.validateColumnManagerAgainstColumnTable(ctx)
		if err != nil {
			return ucerr.Wrap(err)
		}
		if noChanges {
			break
		}

		// Reload the new state from DB
		if numRetries > 5 {
			return ucerr.Friendlyf(nil, "Too many retries on updating default accessor/mutator")
		}
		if err := cm.initColumnManager(ctx); err != nil {
			return ucerr.Wrap(err)
		}
		numRetries++
	}

	return nil
}

// doUpdateDefaultUserstoreAccessorAndMutator updates the default functions on basis of new column state
func (cm *ColumnManager) doUpdateDefaultUserstoreAccessorAndMutator(ctx context.Context) error {
	if !cm.databaseID.IsNil() {
		return nil
	}

	cm.colLock.RLock()
	defer cm.colLock.RUnlock()

	accessorColumnIDs := []uuid.UUID{}
	accessorTransformerIDs := []uuid.UUID{}
	accessorTokenAccessPolicyIDs := []uuid.UUID{}
	mutatorColumnIDs := []uuid.UUID{}
	mutatorNormalizerIDs := []uuid.UUID{}
	for _, column := range cm.idColumnMap {

		accessorColumnIDs = append(accessorColumnIDs, column.ID)
		accessorTransformerIDs = append(accessorTransformerIDs, policy.TransformerPassthrough.ID)
		accessorTokenAccessPolicyIDs = append(accessorTokenAccessPolicyIDs, uuid.Nil)
		if !column.Attributes.Immutable {
			mutatorColumnIDs = append(mutatorColumnIDs, column.ID)
			mutatorNormalizerIDs = append(mutatorNormalizerIDs, policy.TransformerPassthrough.ID)
		}
	}

	// this may occur during provisioning, which is parallelized, so we want to properly handle
	// the case where the default accessor and mutator may not yet be created

	accessor, err := cm.s.GetLatestAccessor(ctx, constants.GetUserAccessorID)
	if accessor != nil {
		accessor.ColumnIDs = accessorColumnIDs
		accessor.TransformerIDs = accessorTransformerIDs
		accessor.TokenAccessPolicyIDs = accessorTokenAccessPolicyIDs
		if err := cm.s.SaveAccessor(ctx, accessor); err != nil {
			return ucerr.Wrap(err)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	mutator, err := cm.s.GetLatestMutator(ctx, constants.UpdateUserMutatorID)
	if mutator != nil {
		mutator.ColumnIDs = mutatorColumnIDs
		mutator.NormalizerIDs = mutatorNormalizerIDs
		if err := cm.s.SaveMutator(ctx, mutator); err != nil {
			return ucerr.Wrap(err)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	return nil
}

// checkForColumnReferenceForDeletion checks if there is a reference to the column in the DB
func (cm *ColumnManager) checkForColumnReferenceForDeletion(ctx context.Context, id uuid.UUID) (code int, err error) {

	col := cm.GetColumnByID(id)
	if col == nil {
		return http.StatusNotFound, ucerr.Friendlyf(nil, "Column not found: %v", id)
	}

	// Need to check if any accessors or mutators use this column
	pager, err := NewAccessorPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}
	for {
		accessors, respFields, err := cm.s.GetLatestAccessors(ctx, *pager)
		if err != nil {
			return http.StatusInternalServerError, ucerr.Wrap(err)
		}

		for _, accessor := range accessors {
			if accessor.ID == constants.GetUserAccessorID {
				continue
			}

			isUsersTableAccessor := true
			for _, columnID := range accessor.ColumnIDs {
				if columnID == id {
					return http.StatusConflict, ucerr.Friendlyf(nil, "Column with ID %s is still used by accessor '%s'", id, accessor.Name)
				}
				refCol := cm.GetColumnByID(columnID)
				if refCol == nil {
					return http.StatusNotFound, ucerr.Friendlyf(nil, "Referenced column not found: %v", columnID)
				}
				if refCol.Table != userTableName {
					isUsersTableAccessor = false
				}
			}

			if isUsersTableAccessor {
				// we only need to check for references in the selector config if the accessor is for the users table; otherwise, the column references in the selector don't matter
				refColumns := getReferencedColumnNamesInUserSelectorConfig(accessor.SelectorConfig)
				if slices.Contains(refColumns, col.Name) {
					return http.StatusConflict, ucerr.Friendlyf(nil, "Column with name '%s' is still referenced in the SelectorConfig for accessor '%s'", col.Name, accessor.Name)
				}
			}
		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	pager, err = NewMutatorPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}
	for {
		mutators, respFields, err := cm.s.GetLatestMutators(ctx, *pager)
		if err != nil {
			return http.StatusInternalServerError, ucerr.Wrap(err)
		}

		for _, mutator := range mutators {
			if mutator.ID == constants.UpdateUserMutatorID {
				continue
			}
			if slices.Contains(mutator.ColumnIDs, id) {
				return http.StatusConflict, ucerr.Friendlyf(nil, "Column with ID %s is still used by mutator '%s'", id, mutator.Name)
			}

			refColumns := getReferencedColumnNamesInUserSelectorConfig(mutator.SelectorConfig)
			if slices.Contains(refColumns, col.Name) {
				return http.StatusConflict, ucerr.Friendlyf(nil, "Column with name '%s' is still referenced in the SelectorConfig for mutator '%s'", col.Name, mutator.Name)
			}

		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	return http.StatusOK, nil
}

// updateColumnNameInSelectorConfig updates all selector configs that reference the column with the new name
func (cm *ColumnManager) updateColumnNameInSelectorConfigs(ctx context.Context, oldName string, newName string) error {

	// Need to update any accessors or mutators that use this column

	accessorsToUpdate := []Accessor{}

	pager, err := NewAccessorPaginatorFromOptions(
		pagination.ResultType(Accessor{}),
		pagination.Limit(pagination.MaxLimit),
	)
	if err != nil {
		return ucerr.Wrap(err)
	}
	for {
		accessors, respFields, err := cm.s.GetLatestAccessors(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}

		for _, accessor := range accessors {
			if accessor.ID == constants.GetUserAccessorID {
				continue
			}
			refColumns := getReferencedColumnNamesInUserSelectorConfig(accessor.SelectorConfig)
			for _, refColumn := range refColumns {
				if refColumn == oldName {
					accessorsToUpdate = append(accessorsToUpdate, accessor)
				}
			}
		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	mm := NewMethodManager(ctx, cm.s)
	for _, accessor := range accessorsToUpdate {
		accessor.SelectorConfig.WhereClause = strings.ReplaceAll(accessor.SelectorConfig.WhereClause, "{"+oldName+"}", "{"+newName+"}")
		// use MethodManager to manage the version update
		if _, err := mm.SaveAccessor(ctx, &accessor); err != nil {
			return ucerr.Wrap(err)
		}
	}

	// Do the same for mutators
	mutatorsToUpdate := []Mutator{}

	pager, err = NewMutatorPaginatorFromOptions(
		pagination.ResultType(Mutator{}),
		pagination.Limit(pagination.MaxLimit),
	)
	if err != nil {
		return ucerr.Wrap(err)
	}
	for {
		mutators, respFields, err := cm.s.GetLatestMutators(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}

		for _, mutator := range mutators {
			if mutator.ID == constants.UpdateUserMutatorID {
				continue
			}
			refColumns := getReferencedColumnNamesInUserSelectorConfig(mutator.SelectorConfig)
			for _, refColumn := range refColumns {
				if refColumn == oldName {
					mutatorsToUpdate = append(mutatorsToUpdate, mutator)
				}
			}

		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	for _, mutator := range mutatorsToUpdate {
		mutator.SelectorConfig.WhereClause = strings.ReplaceAll(mutator.SelectorConfig.WhereClause, "{"+oldName+"}", "{"+newName+"}")
		// use MethodManager to manage the version updates
		if _, err := mm.SaveMutator(ctx, &mutator); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

func (cm *ColumnManager) validateColumnAndDataType(ctx context.Context, c *Column) error {
	if err := c.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	dt, err := cm.s.GetDataType(ctx, c.DataTypeID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := c.Attributes.Constraints.ValidateForDataType(*dt); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

func (cm *ColumnManager) validateColumnName(name string) error {
	usc := userstore.UserSelectorConfig{WhereClause: fmt.Sprintf("{%s} = ?", name)}
	return ucerr.Wrap(usc.Validate())
}

func (cm *ColumnManager) validateDefaultValue(ctx context.Context, c *Column) error {
	if !c.HasDefaultValue() {
		return nil
	}

	dt, err := cm.s.GetDataType(ctx, c.DataTypeID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	var cv column.Value
	if err := cv.Set(
		*dt,
		c.Attributes.Constraints,
		c.IsArray,
		c.DefaultValue,
	); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

func (cm *ColumnManager) validateAccessPolicy(ctx context.Context, c *Column) error {
	if c.AccessPolicyID.IsNil() {
		return ucerr.Friendlyf(nil, "column %v has no access policy", c.ID)
	}

	if _, err := cm.s.GetLatestAccessPolicy(ctx, c.AccessPolicyID); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

func (cm *ColumnManager) validateDefaultTransformer(ctx context.Context, c *Column) error {
	if c.DefaultTransformerID.IsNil() {
		return ucerr.Friendlyf(nil, "column %v has no default transformer", c.ID)
	}

	t, err := cm.s.GetLatestTransformer(ctx, c.DefaultTransformerID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if t.TransformType == transformTypeTokenizeByReference ||
		t.TransformType == transformTypeTokenizeByValue {
		if c.DefaultTokenAccessPolicyID.IsNil() {
			return ucerr.Friendlyf(nil, "column %v has no default token access policy", c.ID)
		}

		if _, err := cm.s.GetLatestAccessPolicy(ctx, c.DefaultTokenAccessPolicyID); err != nil {
			return ucerr.Wrap(err)
		}
	}

	if t.TransformType == transformTypeTokenizeByReference && c.Table != userTableName {
		return ucerr.Friendlyf(nil, "tokenization by reference is only supported for the users table")
	}

	return nil
}

func getReferencedColumnNamesInUserSelectorConfig(config userstore.UserSelectorConfig) []string {
	matches := constants.ReferencedColumnRE.FindAllStringSubmatch(config.WhereClause, -1)
	names := make([]string, len(matches))
	for i, match := range matches {
		names[i] = match[1]
	}
	return names
}
