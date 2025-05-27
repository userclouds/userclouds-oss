package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"

	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/featureflags"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/infra/uctypes/uuidarray"
)

type userRowProcessor struct {
	ctx                           context.Context
	s                             *UserStorage
	dlcs                          column.DataLifeCycleState
	dtm                           *DataTypeManager
	expectedColumnNames           set.Set[string]
	expectedPurposeIDs            set.Set[uuid.UUID]
	nonSystemColumnsByID          map[uuid.UUID]Column
	query                         string
	retentionCutoffTime           time.Time
	systemDBNamesByColumnName     map[string]string
	values                        []any
	curUser                       User
	curColumnID                   uuid.UUID
	curColumnName                 string
	curDataType                   *column.DataType
	curConsentedValues            map[uuid.UUID]ColumnConsentedValue
	curOrderings                  set.Set[int]
	curProfileColumnValue         column.Value
	curProfileConsentedPurposeIDs []ConsentedPurposeIDs
	users                         []User
}

func newUserRowProcessor(
	ctx context.Context,
	s *UserStorage,
	minRetentionTime time.Time,
	columns Columns,
	expectedColumnIDs set.Set[uuid.UUID],
	expectedPurposeIDs set.Set[uuid.UUID],
	uqb *userQueryBuilder,
) *userRowProcessor {
	rp := userRowProcessor{
		ctx:                           ctx,
		s:                             s,
		dlcs:                          uqb.urti.dlcs,
		dtm:                           uqb.dtm,
		expectedColumnNames:           set.NewStringSet(),
		expectedPurposeIDs:            expectedPurposeIDs,
		nonSystemColumnsByID:          map[uuid.UUID]Column{},
		retentionCutoffTime:           minRetentionTime,
		systemDBNamesByColumnName:     map[string]string{},
		curConsentedValues:            map[uuid.UUID]ColumnConsentedValue{},
		curOrderings:                  set.NewIntSet(),
		curProfileConsentedPurposeIDs: []ConsentedPurposeIDs{},
		users:                         []User{},
	}

	nonSystemColumnIDs := []string{}
	for _, c := range columns {
		ci := newColumnInfo(c, rp.dlcs)
		if ci.isSystemColumn() {
			rp.systemDBNamesByColumnName[ci.getColumnName()] = ci.getDBColumnName()
		} else {
			nonSystemColumnIDs = append(nonSystemColumnIDs, c.ID.String())
			rp.nonSystemColumnsByID[c.ID] = c
		}
		if expectedColumnIDs.Contains(c.ID) {
			rp.expectedColumnNames.Insert(c.Name)
		}
	}

	rp.query, rp.values = uqb.getQuery(nonSystemColumnIDs)

	return &rp
}

func (rp *userRowProcessor) curUserValid() bool {
	if rp.curUser.ID.IsNil() {
		return false
	}

	if len(rp.curUser.ColumnValues) == 0 {
		return false
	}

	if rp.expectedColumnNames.Size() == 0 {
		return true
	}

	// if we are expecting specific columns, include the user if values for
	// any of those columns were found for the user

	for _, expectedColumnName := range rp.expectedColumnNames.Items() {
		if _, found := rp.curUser.ColumnValues[expectedColumnName]; found {
			return true
		}
	}

	return false
}

func (rp *userRowProcessor) finishColumn() {
	if len(rp.curConsentedValues) > 0 {
		rp.curUser.ColumnValues[rp.curColumnName] = rp.curConsentedValues
		rp.curUser.Profile[rp.curColumnName] = rp.curProfileColumnValue.Get(rp.ctx)
		rp.curUser.ProfileConsentedPurposeIDs[rp.curColumnName] = rp.curProfileConsentedPurposeIDs
	}

	rp.curColumnID = uuid.Nil
	rp.curColumnName = ""
	rp.curConsentedValues = map[uuid.UUID]ColumnConsentedValue{}
	rp.curOrderings = set.NewIntSet()
	rp.curProfileColumnValue = column.Value{}
	rp.curProfileConsentedPurposeIDs = []ConsentedPurposeIDs{}
}

func (rp *userRowProcessor) finishUser() error {
	rp.finishColumn()

	if rp.curUserValid() {
		if err := rp.validateOrdering(); err != nil {
			return ucerr.Wrap(err)
		}

		rp.users = append(rp.users, rp.curUser)
	}

	rp.curUser = User{}

	return nil
}

func (rp *userRowProcessor) getUsers() ([]User, error) {
	if err := rp.finishUser(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return rp.users, nil
}

func (rp *userRowProcessor) processRow(ur userRow) error {
	if err := rp.setUser(ur); err != nil {
		return ucerr.Wrap(err)
	}

	if err := rp.setColumn(ur); err != nil {
		return ucerr.Wrap(err)
	}

	if err := rp.setValue(ur); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

func (rp *userRowProcessor) setColumn(ur userRow) error {
	if ur.columnID == rp.curColumnID {
		return nil
	}

	rp.finishColumn()

	c, found := rp.nonSystemColumnsByID[ur.columnID]
	if !found {
		return ucerr.Errorf("unexpected column id '%v'", ur.columnID)
	}

	dt := rp.dtm.GetDataTypeByID(c.DataTypeID)
	if dt == nil {
		return ucerr.Errorf("unrecognized data type id '%v' for column '%s'", c.DataTypeID, c.Name)
	}
	rp.curDataType = dt

	ci := newColumnInfo(c, rp.dlcs)
	rp.curColumnID = ur.columnID
	rp.curColumnName = ci.getColumnName()
	if err := rp.curProfileColumnValue.SetType(
		*rp.curDataType,
		c.Attributes.Constraints,
		ci.useArray(),
	); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func (rp *userRowProcessor) setUser(ur userRow) error {
	if ur.userID == rp.curUser.ID {
		return nil
	}

	if err := rp.finishUser(); err != nil {
		return ucerr.Wrap(err)
	}

	rp.curUser = User{
		BaseUser: BaseUser{
			VersionBaseModel: ucdb.NewVersionBaseWithID(ur.userID),
			OrganizationID:   ur.organizationID,
		},
		Profile:                    userstore.Record{},
		ProfileConsentedPurposeIDs: map[string][]ConsentedPurposeIDs{},
		ColumnValues:               ColumnConsentedValues{},
	}
	rp.curUser.Created = ur.userCreated
	rp.curUser.Updated = ur.userUpdated
	rp.curUser.Deleted = ur.userDeleted
	rp.curUser.Version = ur.userVersion

	for columnName, dbColumnName := range rp.systemDBNamesByColumnName {
		value, err := ur.getSystemColumnValue(dbColumnName)
		if err != nil {
			return ucerr.Wrap(err)
		}

		rp.curUser.ColumnValues[columnName] =
			map[uuid.UUID]ColumnConsentedValue{
				rp.curUser.ID: {
					ID:         rp.curUser.ID,
					Version:    rp.curUser.Version,
					Ordering:   1,
					ColumnName: columnName,
					Value:      value,
					ConsentedPurposes: []ConsentedPurpose{
						{
							Purpose:          uuid.Nil,
							RetentionTimeout: userstore.GetRetentionTimeoutIndefinite(),
						},
					},
				},
			}

		rp.curUser.Profile[columnName] = value
	}

	return nil
}

func (rp *userRowProcessor) setValue(ur userRow) error {
	if ur.valueID.IsNil() {
		return nil
	}

	consentedPurposes := []ConsentedPurpose{}
	consentedPurposeIDs := set.NewUUIDSet()
	for i, retentionTimeout := range ur.retentionTimeouts {
		if retentionTimeout == userstore.GetRetentionTimeoutIndefinite() ||
			rp.retentionCutoffTime.Before(retentionTimeout) {
			purposeID := ur.consentedPurposeIDs[i]
			consentedPurposes = append(
				consentedPurposes,
				ConsentedPurpose{
					Purpose:          purposeID,
					RetentionTimeout: retentionTimeout,
				},
			)
			consentedPurposeIDs.Insert(purposeID)
		}
	}

	// do not include the value if any expected purposes are not accounted for

	if !consentedPurposeIDs.IsSupersetOf(rp.expectedPurposeIDs) {
		return nil
	}

	c, found := rp.nonSystemColumnsByID[ur.columnID]
	if !found {
		return ucerr.Errorf("unexpected column id '%v'", ur.columnID)
	}

	ci := newColumnInfo(c, rp.dlcs)
	value, err := ci.getUserRowValue(ur)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if ci.useArray() {
		if rp.dlcs == column.DataLifeCycleStateLive && rp.curOrderings.Contains(ur.ordering) {
			uclog.Warningf(
				rp.ctx,
				"have duplicate ordering %d for user id '%v' for array column '%v - %s'",
				ur.ordering,
				ur.userID,
				ur.columnID,
				ci.getColumnName(),
			)
			if err := rp.s.EnqueueUserCleanupCandidate(
				rp.ctx,
				ur.userID,
				UserCleanupReasonDuplicateValue,
			); err != nil {
				uclog.Warningf(
					rp.ctx,
					"could not enqueue user '%v' for clean up of duplicate value",
					ur.userID,
				)
			}
			return nil
		}

		if err := rp.curProfileColumnValue.Append(rp.ctx, value); err != nil {
			return ucerr.Wrap(err)
		}
	} else if rp.curOrderings.Size() > 0 {
		uclog.Warningf(
			rp.ctx,
			"have more than one value for user id '%v' for non-array column '%v - %s'",
			ur.userID,
			ur.columnID,
			ci.getColumnName(),
		)
		if err := rp.s.EnqueueUserCleanupCandidate(
			rp.ctx,
			ur.userID,
			UserCleanupReasonDuplicateValue,
		); err != nil {
			uclog.Warningf(
				rp.ctx,
				"could not enqueue user '%v' for clean up of duplicate value",
				ur.userID,
			)
		}
		return nil
	} else if err := rp.curProfileColumnValue.Set(
		*rp.curDataType,
		c.Attributes.Constraints,
		false,
		value,
	); err != nil {
		return ucerr.Wrap(err)
	}

	rp.curConsentedValues[ur.valueID] =
		ColumnConsentedValue{
			ID:                ur.valueID,
			Version:           ur.valueVersion,
			ColumnName:        ci.getColumnName(),
			ConsentedPurposes: consentedPurposes,
			Ordering:          ur.ordering,
			Value:             value,
		}

	rp.curOrderings.Insert(ur.ordering)

	rp.curProfileConsentedPurposeIDs = append(rp.curProfileConsentedPurposeIDs, consentedPurposeIDs.Items())

	return nil
}

func (rp userRowProcessor) validateOrdering() error {
	ov := NewOrderingValidator(rp.dlcs)
	for _, values := range rp.curUser.ColumnValues {
		for _, value := range values {
			ov.AddValue(value)
		}
	}

	if err := ov.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

func (s *UserStorage) getUsersForSelectorQuery(
	ctx context.Context,
	minRetentionTime time.Time,
	columns Columns,
	expectedColumnIDs set.Set[uuid.UUID],
	expectedPurposeIDs set.Set[uuid.UUID],
	uqb *userQueryBuilder,
	accessPrimaryDBOnly bool,
) ([]User, error) {
	// create user row processor and make sure it is valid

	rp := newUserRowProcessor(
		ctx,
		s,
		minRetentionTime,
		columns,
		expectedColumnIDs,
		expectedPurposeIDs,
		uqb,
	)

	// execute query and process rows

	useReplica := featureflags.IsEnabledGlobally(ctx, featureflags.ReadFromReadReplica)
	var rows []UserQueryRow
	if err := s.db.UnsafeSelectContext(ctx, "getUsersForSelectorQuery", &rows, rp.query, !useReplica || accessPrimaryDBOnly, rp.values...); err != nil {
		return nil, ucerr.Wrap(err)
	}

	for _, row := range rows {
		if err := rp.processRow(row.toUserRow()); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	users, err := rp.getUsers()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return users, nil
}

// GetUsersForSelector returns the matching user objects for a specified data lifecycle state (live or soft-deleted)
// for the user selector query and associated query parameters. The requested column values will be returned for each user
// that satisfies the selector query as long as the values have the expected purpose ids, and as long as all expected
// column IDs have values.
func (s *UserStorage) GetUsersForSelector(
	ctx context.Context,
	cm *ColumnManager,
	dtm *DataTypeManager,
	minRetentionTime time.Time,
	dlcs column.DataLifeCycleState,
	columns Columns,
	selectorConfig userstore.UserSelectorConfig,
	selectorValues userstore.UserSelectorValues,
	expectedColumnIDs set.Set[uuid.UUID],
	expectedPurposeIDs set.Set[uuid.UUID],
	p *pagination.Paginator,
	accessPrimaryDBOnly bool,
) ([]User, int, error) {
	uqb, err := newUserQueryBuilder(ctx, cm, dtm, dlcs)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if err := uqb.applySelector(
		columns,
		selectorConfig,
		selectorValues,
		expectedPurposeIDs,
		p,
	); err != nil {
		return nil, http.StatusBadRequest, ucerr.Wrap(err)
	}

	users, err := s.getUsersForSelectorQuery(
		ctx,
		minRetentionTime,
		columns,
		expectedColumnIDs,
		expectedPurposeIDs,
		uqb,
		accessPrimaryDBOnly,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, http.StatusNotFound, ucerr.Wrap(err)
		}
		if ok, msg := ucdb.IsSQLParseError(err); ok {
			return nil, http.StatusBadRequest, ucerr.Friendlyf(err, "Invalid selector arguments: %s", msg)
		}
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	return users, http.StatusOK, nil
}

func (s *UserStorage) getUser(
	ctx context.Context,
	cm *ColumnManager,
	dtm *DataTypeManager,
	userID uuid.UUID,
	minRetentionTime time.Time,
	dlcs column.DataLifeCycleState,
	accessPrimaryDBOnly bool,
) (*User, int, error) {
	columns := cm.GetColumns()

	selectorConfig := userstore.UserSelectorConfig{
		WhereClause: "{id} = ?",
	}

	selectorValues := []any{userID}

	users, code, err := s.GetUsersForSelector(
		ctx,
		cm,
		dtm,
		minRetentionTime,
		dlcs,
		columns,
		selectorConfig,
		selectorValues,
		set.NewUUIDSet(),
		set.NewUUIDSet(),
		nil,
		accessPrimaryDBOnly,
	)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	switch len(users) {
	case 1:
		return &users[0], http.StatusOK, nil
	case 0:
		return nil, http.StatusNotFound, ucerr.Friendlyf(nil, "user %v not found", userID)
	default:
		return nil, http.StatusInternalServerError, ucerr.Errorf("found %d users for user ID '%v'", userID)
	}
}

// GetUser loads a User by ID
func (s *UserStorage) GetUser(ctx context.Context, cm *ColumnManager, dtm *DataTypeManager, userID uuid.UUID, accessPrimaryDBOnly bool) (*User, int, error) {
	user, code, err := s.getUser(ctx, cm, dtm, userID, time.Now().UTC(), column.DataLifeCycleStateLive, accessPrimaryDBOnly)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}
	return user, http.StatusOK, nil
}

// GetAllUserValues loads all live and soft-deleted column values for a user ID
func (s *UserStorage) GetAllUserValues(ctx context.Context, cm *ColumnManager, dtm *DataTypeManager, userID uuid.UUID, accessPrimaryDBOnly bool) (
	user *BaseUser,
	liveValues ColumnConsentedValues,
	softDeletedValues ColumnConsentedValues,
	code int,
	err error,
) {
	minRetentionTime := userstore.GetRetentionTimeoutIndefinite()
	liveUser, code, err := s.getUser(ctx, cm, dtm, userID, minRetentionTime, column.DataLifeCycleStateLive, accessPrimaryDBOnly)
	if err != nil {
		return nil, nil, nil, code, ucerr.Wrap(err)
	}

	softDeletedUser, code, err := s.getUser(ctx, cm, dtm, userID, minRetentionTime, column.DataLifeCycleStateSoftDeleted, accessPrimaryDBOnly)
	if err != nil {
		return nil, nil, nil, code, ucerr.Wrap(err)
	}

	return &liveUser.BaseUser, liveUser.ColumnValues, softDeletedUser.ColumnValues, http.StatusOK, nil
}

type userQueryBuilder struct {
	ctx                       context.Context
	cm                        *ColumnManager
	dtm                       *DataTypeManager
	urti                      *userRowsTableInfo
	innerOrderByClause        string
	innerSelectorClause       string
	nonSystemSortColumnClause string
	outerOrderByClause        string
	outerSelectorClause       string
	requiresInnerQuery        bool
	selectorJoinClause        string
	sortJoinClause            string
	systemSortColumnClause    string
	values                    []any
}

func newUserQueryBuilder(
	ctx context.Context,
	cm *ColumnManager,
	dtm *DataTypeManager,
	dlcs column.DataLifeCycleState,
) (*userQueryBuilder, error) {
	urti, err := newUserRowsTableInfo(dlcs)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	uqb := userQueryBuilder{
		ctx:  ctx,
		cm:   cm,
		dtm:  dtm,
		urti: urti,
	}

	return &uqb, nil
}

func (uqb userQueryBuilder) appendClause(clause string, addition string) string {
	if clause == "" {
		clause = "\nWHERE\n"
	} else {
		clause += "\nAND "
	}
	return clause + addition
}

func (uqb *userQueryBuilder) applyPagination(p *pagination.Paginator) (err error) {
	if p == nil {
		uqb.outerOrderByClause = "\nu.id ASC"
		return nil
	}

	uqb.requiresInnerQuery = true

	totalSortKeyJoins := 0
	var sortFields []string
	var sortFieldsNullable []bool
	for _, sortKey := range p.GetSortKey().Split() {
		columnID, columnName, isSystem, err := uqb.getColumnInfoForSortKey(sortKey)
		if err != nil {
			return ucerr.Wrap(err)
		}

		if isSystem {
			if columnName != "u.id" {
				uqb.systemSortColumnClause += fmt.Sprintf("\n%s,", columnName)
			}
			sortFields = append(sortFields, columnName)
			sortFieldsNullable = append(sortFieldsNullable, false)
		} else {
			totalSortKeyJoins++
			sortFieldTable := fmt.Sprintf("sort_key_table_%d", totalSortKeyJoins)
			sortField := fmt.Sprintf("%s.%s", sortFieldTable, columnName)
			sortFields = append(sortFields, sortField)
			sortFieldsNullable = append(sortFieldsNullable, true)

			uqb.nonSystemSortColumnClause += fmt.Sprintf(
				"\n%s,",
				sortField,
			)

			uqb.values = append(uqb.values, columnID)
			uqb.sortJoinClause += fmt.Sprintf(
				"\nLEFT JOIN user_column_pre_delete_values %s\nON %s.user_id = u.id\nAND %s.column_id = $%d",
				sortFieldTable,
				sortFieldTable,
				sortFieldTable,
				len(uqb.values),
			)
		}
	}

	orderByJoiner := ""
	uqb.innerOrderByClause = "\nORDER BY"
	orderDirection := p.GetInnerOrderByDirection()
	for _, sortField := range sortFields {
		// ugh ... we accidentally baked an ordering dependency (that is different between cockroach and postgres)
		// into the query builder (and tests), so we need to override the direction for postgres
		dir := orderDirection
		if dir == "ASC" {
			dir = "ASC NULLS FIRST"
		} else {
			dir = "DESC NULLS LAST"
		}
		uqb.outerOrderByClause += fmt.Sprintf("%s\n%s %s", orderByJoiner, sortField, dir)
		uqb.innerOrderByClause += fmt.Sprintf("%s\n%s %s", orderByJoiner, sortField, dir)
		orderByJoiner = ","
	}
	if limit := p.GetLimit() * p.GetLimitMultiplier(); limit > 0 {
		// always overfetch by 1
		uqb.innerOrderByClause += fmt.Sprintf("\nLIMIT %d", limit+1)
	}

	if cursorClause, _ := p.GetCursorClause(len(uqb.values)+1, sortFields, sortFieldsNullable); cursorClause != "" {
		uqb.innerSelectorClause = uqb.appendClause(uqb.innerSelectorClause, cursorClause)
		uqb.values, err = p.GetCursorFields(uqb.values)
		if err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

func (uqb *userQueryBuilder) applySelector(
	columns Columns,
	selectorConfig userstore.UserSelectorConfig,
	selectorValues userstore.UserSelectorValues,
	purposeIDs set.Set[uuid.UUID],
	p *pagination.Paginator,
) error {
	uqb.reset(selectorValues)

	ucvTableName := uqb.urti.getUserColumnValuesTableName()

	if len(columns) == 0 {
		return ucerr.Friendlyf(nil, "no columns specified")
	}

	anyNonSystemColumns := false
	for _, column := range columns {
		ci := newColumnInfo(column, uqb.urti.dlcs)
		if !ci.isSystemColumn() {
			anyNonSystemColumns = true
			break
		}
	}

	if !anyNonSystemColumns {
		// ignore purposes if only selecting system columns
		purposeIDs = set.NewUUIDSet()
	}
	anyPurposes := purposeIDs.Size() > 0

	if !anyNonSystemColumns || uqb.urti.dlcs == column.DataLifeCycleStateLive {
		uqb.innerSelectorClause = uqb.appendClause(uqb.innerSelectorClause, "u.deleted = '0001-01-01 00:00:00'::TIMESTAMP")
		uqb.outerSelectorClause = uqb.appendClause(uqb.outerSelectorClause, "u.deleted = '0001-01-01 00:00:00'::TIMESTAMP")
	}

	if selectorConfig.MatchesAll() {
		if len(uqb.values) > 0 {
			return ucerr.Friendlyf(nil, "cannot specify UserSelectorValues without a WhereClause")
		}
	} else {
		uqb.requiresInnerQuery = true
		selectorClause, _, err :=
			newExecutionSelectorConfigFormatter(
				uqb.cm,
				uqb.dtm,
				selectorConfig,
				len(uqb.values),
			).format()
		if err != nil {
			return ucerr.Wrap(err)
		}
		uqb.innerSelectorClause = uqb.appendClause(uqb.innerSelectorClause, selectorClause)
	}

	if err := uqb.applyPagination(p); err != nil {
		return ucerr.Wrap(err)
	}

	if !uqb.requiresInnerQuery {
		return nil
	}

	// iterate over columns and build inner query clauses

	numNonSystemColumnsWithSelectors := 0
	// TODO (sgarrity 1/25): see TODO below
	// var nonSystemColumnIDsWithoutSelectors uuidarray.UUIDArray
	purposeParameter := fmt.Sprintf("$%d", len(uqb.values)+1)

	// TODO (sgarrity 1/25): see TODO below
	// nonSystemColumnIDsWithoutSelectorsParameter := fmt.Sprintf("$%d", len(uqb.values)+2)

	for _, column := range columns {
		ci := newColumnInfo(column, uqb.urti.dlcs)
		bracketedColumnName := ci.getBracketedColumnName()
		hasColumnSelector := strings.Contains(uqb.innerSelectorClause, bracketedColumnName)

		if !ci.isSystemColumn() {
			// TODO (sgarrity 1/25): see TODO below
			if !hasColumnSelector {
				// if anyPurposes {
				// 	nonSystemColumnIDsWithoutSelectors = append(nonSystemColumnIDsWithoutSelectors, column.ID)
				// }
				continue
			}

			numNonSystemColumnsWithSelectors++
			columnTableName := fmt.Sprintf("column_%d", numNonSystemColumnsWithSelectors)

			// update selector join clause

			uqb.selectorJoinClause += fmt.Sprintf(
				"\nLEFT JOIN %s %s\nON %s.user_id = u.id\nAND %s.column_id = '%v'::UUID",
				ucvTableName,
				columnTableName,
				columnTableName,
				columnTableName,
				column.ID,
			)

			if anyPurposes {
				uqb.selectorJoinClause += fmt.Sprintf(
					"\nAND %s.consented_purpose_ids @> %s",
					columnTableName,
					purposeParameter,
				)
			}

			// update selector clause

			if hasColumnSelector {
				columnName, err := ci.getUserRowColumnName()
				if err != nil {
					return ucerr.Wrap(err)
				}

				uqb.innerSelectorClause = strings.ReplaceAll(
					uqb.innerSelectorClause,
					bracketedColumnName,
					fmt.Sprintf("%s.%s", columnTableName, columnName),
				)
			}
		} else if hasColumnSelector {
			uqb.innerSelectorClause = strings.ReplaceAll(
				uqb.innerSelectorClause,
				bracketedColumnName,
				"u."+ci.getDBColumnName(),
			)
		}
	}

	// TODO (sgarrity 1/25): I believe this was intended to enforce purpose use for non-system, non-selector columns,
	// but as it was written, it just slows down queries without doing any work (LEFT JOIN is an OUTER JOIN).
	// I'm removing it for now to improve performance, but we should revisit if we want to enforce purpose use later
	// if len(nonSystemColumnIDsWithoutSelectors) > 0 {
	// 	uqb.selectorJoinClause += fmt.Sprintf(
	// 		"\nLEFT JOIN %s columns\nON columns.user_id = u.id\nAND columns.column_id = ANY(%s)\nAND columns.consented_purpose_ids @> %s",
	// 		ucvTableName,
	// 		nonSystemColumnIDsWithoutSelectorsParameter,
	// 		purposeParameter,
	// 	)
	// }

	for i, value := range uqb.values {
		valueType := reflect.TypeOf(value)
		if valueType.Kind() == reflect.Slice {
			uqb.values[i] = pq.Array(value)
		}
	}

	if anyPurposes && numNonSystemColumnsWithSelectors > 0 {
		uqb.values = append(uqb.values, (uuidarray.UUIDArray)(purposeIDs.Items()))
		// TODO (sgarrity 1/25): see TODO above
		// if len(nonSystemColumnIDsWithoutSelectors) > 0 {
		// 	uqb.values = append(uqb.values, nonSystemColumnIDsWithoutSelectors)
		// }
	}

	return nil
}

func (uqb userQueryBuilder) getColumnInfoForSortKey(
	sortKey string,
) (columnID uuid.UUID, columnName string, isSystem bool, err error) {
	c := uqb.cm.GetUserColumnByName(sortKey)
	if c == nil {
		return uuid.Nil,
			"",
			false,
			ucerr.Errorf("sort key '%s' does not match a known column", sortKey)
	}

	ci := newColumnInfo(*c, uqb.urti.dlcs)
	if ci.isSystemColumn() {
		switch c.ID {
		case column.IDColumnID:
			return c.ID, "u.id", true, nil
		case column.CreatedColumnID:
			return c.ID, "u.created", true, nil
		case column.UpdatedColumnID:
			return c.ID, "u.updated", true, nil
		case column.OrganizationColumnID:
			return c.ID, "u.organization_id", true, nil
		case column.VersionColumnID:
			return c.ID, "u._version", true, nil
		}
	} else if columnName, err := ci.getUserRowColumnName(); err == nil {
		return c.ID, columnName, false, nil
	}

	return uuid.Nil,
		"",
		false,
		ucerr.Errorf("sort key '%s' has unsupported data type '%v'", sortKey, c.DataTypeID)
}

const getUsersInnerQueryTemplate = `u.id IN (
SELECT
du.id
FROM
(
SELECT DISTINCT%s%s
u.id
FROM
users u%s%s%s%s
) du
)`

func (uqb userQueryBuilder) getQuery(
	nonSystemColumnIDs []string,
) (query string, values []any) {
	if uqb.requiresInnerQuery {
		innerQuery := fmt.Sprintf(
			getUsersInnerQueryTemplate,
			uqb.nonSystemSortColumnClause,
			uqb.systemSortColumnClause,
			uqb.selectorJoinClause,
			uqb.sortJoinClause,
			uqb.innerSelectorClause,
			uqb.innerOrderByClause,
		)
		uqb.outerSelectorClause = uqb.appendClause(uqb.outerSelectorClause, innerQuery)
	}

	values = append(values, uqb.values...)

	// NOTE: in practice, we always retrieve all non-system columns, since access policies may
	//       depend on any column value, so we always fall through to the default case below
	switch len(nonSystemColumnIDs) {
	case 0:
		query = uqb.urti.getSystemColumnQuery(uqb.outerSelectorClause, uqb.outerOrderByClause)
	case 1:
		values = append(values, nonSystemColumnIDs[0])
		columnJoinClause := fmt.Sprintf("\nAND ucv.column_id = $%d", len(values))
		query = uqb.urti.getNonSystemColumnQuery(
			uqb.nonSystemSortColumnClause,
			columnJoinClause,
			uqb.sortJoinClause,
			uqb.outerSelectorClause,
			uqb.outerOrderByClause,
		)
	default:
		values = append(values, pq.Array(nonSystemColumnIDs))
		columnJoinClause := fmt.Sprintf("\nAND ucv.column_id = ANY($%d)", len(values))
		query = uqb.urti.getNonSystemColumnQuery(
			uqb.nonSystemSortColumnClause,
			columnJoinClause,
			uqb.sortJoinClause,
			uqb.outerSelectorClause,
			uqb.outerOrderByClause,
		)
	}

	return query, values
}

func (uqb *userQueryBuilder) reset(values []any) {
	uqb.innerOrderByClause = ""
	uqb.innerSelectorClause = ""
	uqb.nonSystemSortColumnClause = ""
	uqb.outerOrderByClause = ""
	uqb.outerSelectorClause = ""
	uqb.requiresInnerQuery = false
	uqb.selectorJoinClause = ""
	uqb.sortJoinClause = ""
	uqb.systemSortColumnClause = ""
	uqb.values = values
}
