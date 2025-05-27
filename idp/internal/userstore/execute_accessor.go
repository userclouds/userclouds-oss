package userstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp"
	"userclouds.com/idp/config"
	"userclouds.com/idp/internal"
	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/internal/tokenizer"
	"userclouds.com/idp/policy"
	userstorePublic "userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/infra/uctypes/timer"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/ucopensearch"
)

// PageableUser makes a User a PageableType by adding necessary pagination methods
type PageableUser struct {
	storage.User
	keyTypes pagination.KeyTypes
}

func newPageableUser() PageableUser {
	return PageableUser{
		keyTypes: pagination.KeyTypes{
			"id":      pagination.UUIDKeyType,
			"created": pagination.TimestampKeyType,
			"updated": pagination.TimestampKeyType,
		},
	}
}

func (pu *PageableUser) addColumn(c storage.Column) {
	if _, found := pu.keyTypes[c.Name]; found {
		return
	}

	if c.Attributes.System {
		switch c.GetConcreteDataTypeID() {
		case datatype.Boolean.ID:
			pu.keyTypes[c.Name] = pagination.BoolKeyType
		case datatype.Integer.ID:
			pu.keyTypes[c.Name] = pagination.IntKeyType
		case datatype.String.ID:
			pu.keyTypes[c.Name] = pagination.StringKeyType
		case datatype.Timestamp.ID:
			pu.keyTypes[c.Name] = pagination.TimestampKeyType
		case datatype.UUID.ID:
			pu.keyTypes[c.Name] = pagination.UUIDKeyType
		}
	} else {
		switch c.GetConcreteDataTypeID() {
		case datatype.Boolean.ID:
			pu.keyTypes[c.Name] = pagination.NullableBoolKeyType
		case datatype.Integer.ID:
			pu.keyTypes[c.Name] = pagination.NullableIntKeyType
		case datatype.String.ID:
			pu.keyTypes[c.Name] = pagination.NullableStringKeyType
		case datatype.Timestamp.ID:
			pu.keyTypes[c.Name] = pagination.NullableTimestampKeyType
		case datatype.UUID.ID:
			pu.keyTypes[c.Name] = pagination.NullableUUIDKeyType
		}
	}
}

// GetCursor is part of the pagination.PageableType interface
func (pu PageableUser) GetCursor(sortKeys pagination.Key) pagination.Cursor {
	cursor, err := pu.getCursor(sortKeys)
	if err != nil {
		return pagination.CursorBegin
	}

	return cursor
}

func (pu PageableUser) getCursor(sortKeys pagination.Key) (cursor pagination.Cursor, err error) {
	var builder strings.Builder
	joiner := ""
	for _, sortKey := range sortKeys.Split() {
		switch sortKey {
		case "id":
			fmt.Fprintf(&builder, "%sid:%v", joiner, pu.ID)
		case "created":
			fmt.Fprintf(&builder, "%screated:%v", joiner, pu.Created.UnixMicro())
		case "updated":
			fmt.Fprintf(&builder, "%supdated:%v", joiner, pu.Updated.UnixMicro())
		default:
			v, valueFound := pu.Profile[sortKey]
			if !valueFound {
				v = ""
			}

			sortKeyType := pu.keyTypes[sortKey]
			switch sortKeyType {
			case pagination.BoolKeyType, pagination.IntKeyType, pagination.StringKeyType, pagination.UUIDKeyType:
				if !valueFound {
					return pagination.CursorBegin,
						ucerr.Errorf(
							"user '%v' profile missing expected '%s' column: '%v'",
							pu.ID,
							sortKey,
							pu.Profile,
						)
				}
			case pagination.NullableBoolKeyType, pagination.NullableIntKeyType, pagination.NullableStringKeyType, pagination.NullableUUIDKeyType:
			case pagination.NullableTimestampKeyType:
				v, err = pu.getTimestampCursorValue(sortKey, v, true)
				if err != nil {
					return pagination.CursorBegin, ucerr.Wrap(err)
				}
			case pagination.TimestampKeyType:
				v, err = pu.getTimestampCursorValue(sortKey, v, false)
				if err != nil {
					return pagination.CursorBegin, ucerr.Wrap(err)
				}
			default:
				return pagination.CursorBegin,
					ucerr.Errorf(
						"sort key '%s' has unexpected pagination type '%v'",
						sortKey,
						sortKeyType,
					)
			}

			fmt.Fprintf(&builder, "%s%s:%v", joiner, sortKey, v)
		}
		joiner = ","
	}
	return pagination.Cursor(builder.String()), nil
}

// GetPaginationKeys is part of the pagination.PageableType interface
func (pu PageableUser) GetPaginationKeys() pagination.KeyTypes {
	return pu.keyTypes
}

func (pu PageableUser) getTimestampCursorValue(
	sortKey string,
	value any,
	allowEmpty bool,
) (any, error) {
	if value == "" {
		if allowEmpty {
			return value, nil
		}

		return nil,
			ucerr.Errorf(
				"user '%v' profile missing expected '%s' column: '%v'",
				pu.ID,
				sortKey,
				pu.Profile,
			)
	}

	dt, err := column.GetNativeDataType(datatype.Timestamp.ID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	dc, err := column.NewDataCoercer(*dt, column.Constraints{})
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	ts, err := dc.ToTimestamp(value)
	if err != nil {
		return nil,
			ucerr.Errorf(
				"user '%v' column '%s' value '%v' is of type '%T', not '%T'",
				pu.ID,
				sortKey,
				value,
				ts,
			)
	}

	return ts.UnixMicro(), nil
}

type accessorLimits struct {
	pager      *pagination.Paginator
	pu         PageableUser
	firstUser  storage.User
	lastUser   storage.User
	hasMore    bool
	limit      int
	maxLimit   int
	numAllowed int
}

func (al accessorLimits) getCursor(u storage.User) (c pagination.Cursor, err error) {
	if al.pager == nil {
		return pagination.CursorBegin, nil
	}

	al.pu.User = u
	cursor, err := al.pu.getCursor(al.pager.GetSortKey())
	if err != nil {
		return pagination.CursorBegin, ucerr.Wrap(err)
	}

	return cursor, nil
}

func (accessorLimits) getDefaultResponseFields() *pagination.ResponseFields {
	return &pagination.ResponseFields{}
}

func (al accessorLimits) getResponseFields() (respFields *pagination.ResponseFields, err error) {
	if al.pager == nil || al.maxExceeded() {
		return al.getDefaultResponseFields(), nil
	}

	var next, prev pagination.Cursor
	if al.pager.IsForward() {
		next = pagination.CursorEnd
		if al.hasMore {
			next, err = al.getCursor(al.lastUser)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
		}

		prev = pagination.CursorBegin
		if al.pager.GetCursor() != pagination.CursorBegin {
			if !al.firstUser.ID.IsNil() {
				prev, err = al.getCursor(al.firstUser)
				if err != nil {
					return nil, ucerr.Wrap(err)
				}
			}
		}
	} else {
		prev = pagination.CursorBegin
		if al.hasMore {
			prev, err = al.getCursor(al.firstUser)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
		}

		next = pagination.CursorEnd
		if al.pager.GetCursor() != pagination.CursorEnd {
			if !al.lastUser.ID.IsNil() {
				next, err = al.getCursor(al.lastUser)
				if err != nil {
					return nil, ucerr.Wrap(err)
				}
			}
		}
	}

	return &pagination.ResponseFields{
			HasNext: next != pagination.CursorEnd,
			Next:    next,
			HasPrev: prev != pagination.CursorBegin,
			Prev:    prev,
		},
		nil
}

func (al accessorLimits) maxExceeded() bool {
	return al.maxLimit > 0 && al.numAllowed > al.maxLimit
}

func (al *accessorLimits) orderUsers(users []storage.User) []storage.User {
	if al.pager == nil || al.pager.IsForward() {
		return users
	}

	slices.Reverse(users)
	u := al.firstUser
	al.firstUser = al.lastUser
	al.lastUser = u
	return users
}

type accessorExecutor struct {
	ctx                 context.Context
	s                   *storage.Storage
	searchUpdateConfig  *config.SearchUpdateConfig
	authzClient         *authz.Client
	req                 idp.ExecuteAccessorRequest
	startTime           time.Time
	includeDebug        bool
	accessor            *storage.Accessor
	pu                  PageableUser
	pager               *pagination.Paginator
	apContext           policy.AccessPolicyContext
	accessorColumns     storage.Columns
	clientAP            *policy.AccessPolicy
	thresholdAP         *storage.AccessPolicy
	columns             storage.Columns
	cm                  *storage.ColumnManager
	dtm                 *storage.DataTypeManager
	expectedColumnIDs   set.Set[uuid.UUID]
	expectedPurposeIDs  set.Set[uuid.UUID]
	transformerMap      map[uuid.UUID]*storage.Transformer
	accessPolicyConsole string
	transformerConsole  string
	numSearchRows       int
	numSelectorRows     int
	numReturned         int
	numDenied           int
	succeeded           bool
	truncated           bool
}

func newAccessorExecutor(
	ctx context.Context,
	s *storage.Storage,
	searchUpdateConfig *config.SearchUpdateConfig,
	req idp.ExecuteAccessorRequest,
	startTime time.Time,
	includeDebug bool,
	accessor *storage.Accessor,
) accessorExecutor {
	return accessorExecutor{
		ctx:                ctx,
		s:                  s,
		searchUpdateConfig: searchUpdateConfig,
		req:                req,
		startTime:          startTime,
		includeDebug:       includeDebug,
		accessor:           accessor,
		expectedPurposeIDs: set.NewUUIDSet(accessor.PurposeIDs...),
	}
}

func (ae accessorExecutor) auditLogInfo() []auditlog.Entry {
	if !ae.accessor.IsAuditLogged {
		return nil
	}

	return auditlog.NewEntryArray(
		auth.GetAuditLogActor(ae.ctx),
		internal.AuditLogEventTypeExecuteAccessor,
		auditlog.Payload{
			"ID":                      ae.accessor.ID,
			"Name":                    ae.accessor.Name,
			"Version":                 ae.accessor.Version,
			"Succeeded":               ae.succeeded,
			"SelectorValues":          ae.req.SelectorValues,
			"SearchRowCount":          ae.numSearchRows,
			"Truncated":               ae.truncated,
			"SelectorRowCount":        ae.numSelectorRows,
			"RowsReturned":            ae.numReturned,
			"AccessPolicyContext":     ae.apContext,
			"AccessPolicyDeniedCount": ae.numDenied,
		},
	)
}

func (ae *accessorExecutor) checkRateThreshold() (bool, error) {
	if !ae.isInitialQuery() {
		return true, nil
	}

	allowed, err := ae.thresholdAP.CheckRateThreshold(ae.ctx, ae.s, ae.apContext, ae.accessor.ID)
	if err != nil {
		return false, ucerr.Wrap(err)
	}

	return allowed, nil
}

func (ae accessorExecutor) debugInfo() map[string]any {
	if !ae.includeDebug {
		return nil
	}

	di := map[string]any{
		"request":                               ae.req,
		"access_policy_context":                 ae.apContext,
		"users_from_search_count":               ae.numSearchRows,
		"search_users_truncated":                ae.truncated,
		"users_for_selector_and_purposes_count": ae.numReturned,
		"access_policy_fail_count":              ae.numDenied,
		"access_policy_console":                 ae.accessPolicyConsole,
		"transformer_console":                   ae.transformerConsole,
	}
	if ae.pager != nil {
		di["limit"] = ae.pager.GetLimit()
	}

	return di
}

func (ae *accessorExecutor) filterPaginatedSearchUserIDs(
	sentinelCursor pagination.Cursor,
	userIDs []uuid.UUID,
	idSorter func([]uuid.UUID) []uuid.UUID,
) ([]uuid.UUID, error) {
	firstIndex := 0

	if cursor := ae.pager.GetCursor(); cursor != sentinelCursor {
		// extract the initial id from the cursor
		cursorParts := strings.Split(string(cursor), ":")
		if len(cursorParts) != 2 {
			return nil, ucerr.Errorf("cursor is malformed: '%v'", cursor)
		}
		initialID, err := uuid.FromString(cursorParts[1])
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		// make sure the initial ID is in the list of IDs and sort
		uniqueUserIDs := set.NewUUIDSet(userIDs...)
		uniqueUserIDs.Insert(initialID)
		userIDs = idSorter(uniqueUserIDs.Items())

		// determine where the initial ID is in the list
		for firstIndex < len(userIDs) {
			firstIndex++
			if userIDs[firstIndex-1] == initialID {
				break
			}
		}
	} else {
		userIDs = idSorter(userIDs)
	}

	lastIndex := min(firstIndex+ae.pager.GetLimit()*ae.pager.GetLimitMultiplier(), len(userIDs))

	return userIDs[firstIndex:lastIndex], nil
}

func (ae *accessorExecutor) filterSearchUserIDs(userIDs []uuid.UUID) ([]uuid.UUID, error) {
	if ae.pager == nil || ae.pager.GetSortKey() != "id" {
		return userIDs, nil
	}

	if ae.pager.IsForward() {
		filteredIDs, err :=
			ae.filterPaginatedSearchUserIDs(
				pagination.CursorBegin,
				userIDs,
				func(ids []uuid.UUID) []uuid.UUID {
					sort.Slice(
						ids,
						func(i int, j int) bool {
							return ids[i].String() < ids[j].String()
						},
					)
					return ids
				},
			)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		return filteredIDs, nil
	}

	filteredIDs, err :=
		ae.filterPaginatedSearchUserIDs(
			pagination.CursorEnd,
			userIDs,
			func(ids []uuid.UUID) []uuid.UUID {
				sort.Slice(
					ids,
					func(i int, j int) bool {
						return ids[i].String() > ids[j].String()
					},
				)
				return ids
			},
		)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return filteredIDs, nil
}

func (accessorExecutor) getAccessorColumns(cols storage.Columns, accessor *storage.Accessor) (storage.Columns, error) {
	missingColIDs := set.NewUUIDSet(accessor.ColumnIDs...).Difference(cols.GetIDs())
	if missingColIDs.Size() > 0 {
		return nil, ucerr.Friendlyf(nil, "Missing columns specified in accessor %v: [%v]", accessor.ID, missingColIDs.String())
	}
	colMap := cols.GetIDsMap()
	accessorColumns := make(storage.Columns, len(accessor.ColumnIDs))
	for i, columnID := range accessor.ColumnIDs {
		accessorColumns[i] = colMap[columnID]
	}
	return accessorColumns, nil
}

func (ae *accessorExecutor) sortMergedCandidateUsers(candidateUsers []storage.User) ([]storage.User, int, error) {
	limits, err := ae.getLimits()
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if limits.pager == nil || limits.pager.GetSortKey() == "id" {
		// Sort candidateUsers by id
		sort.Slice(candidateUsers, func(i, j int) bool {
			return candidateUsers[i].ID.String() < candidateUsers[j].ID.String()
		})
	} else {
		sortKeys := strings.Split(string(limits.pager.GetSortKey()), ",")
		sortColumns := make([]*storage.Column, len(sortKeys))
		sortDataTypes := make([]*column.DataType, len(sortKeys))
		for k, sortK := range sortKeys {
			c := ae.cm.GetUserColumnByName(sortK)
			if c == nil {
				return nil, http.StatusBadRequest, ucerr.Friendlyf(nil, "Invalid sort key: %v", sortK)
			}
			sortColumns[k] = c
			dt := ae.dtm.GetDataTypeByID(c.DataTypeID)
			if dt == nil {
				return nil, http.StatusInternalServerError, ucerr.Friendlyf(nil, "Invalid data type: %v", c.DataTypeID)
			}
			sortDataTypes[k] = dt
		}

		// Sort candidateUsers by sort keys
		sort.Slice(candidateUsers, func(i, j int) bool {
			for k, sortKey := range sortKeys {
				if candidateUsers[i].Profile[sortKey] == nil && candidateUsers[j].Profile[sortKey] != nil {
					return true
				} else if candidateUsers[i].Profile[sortKey] != nil && candidateUsers[j].Profile[sortKey] == nil {
					return false
				}
				if sortColumns[k].IsArray {
					switch sortDataTypes[k].ConcreteDataTypeID {
					case datatype.String.ID:
						iArray, ok := candidateUsers[i].Profile[sortKey].([]string)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a string array: %v", sortKey, candidateUsers[i].Profile[sortKey])
							return false
						}
						jArray, ok := candidateUsers[j].Profile[sortKey].([]string)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a string array: %v", sortKey, candidateUsers[j].Profile[sortKey])
							return false
						}
						if len(iArray) < len(jArray) {
							return true
						} else if len(iArray) > len(jArray) {
							return false
						}
						for l := range iArray {
							if iArray[l] != jArray[l] {
								return iArray[l] < jArray[l]
							}
						}
					case datatype.Boolean.ID:
						iArray, ok := candidateUsers[i].Profile[sortKey].([]bool)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a boolean array: %v", sortKey, candidateUsers[i].Profile[sortKey])
							return false
						}
						jArray, ok := candidateUsers[j].Profile[sortKey].([]bool)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a boolean array: %v", sortKey, candidateUsers[j].Profile[sortKey])
							return false
						}
						if len(iArray) < len(jArray) {
							return true
						} else if len(iArray) > len(jArray) {
							return false
						}
						for l := range iArray {
							if iArray[l] != jArray[l] {
								return !iArray[l]
							}
						}
					case datatype.Timestamp.ID, datatype.Date.ID:
						iArray, ok := candidateUsers[i].Profile[sortKey].([]time.Time)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a timestamp array: %v", sortKey, candidateUsers[i].Profile[sortKey])
							return false
						}
						jArray, ok := candidateUsers[j].Profile[sortKey].([]time.Time)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a timestamp array: %v", sortKey, candidateUsers[j].Profile[sortKey])
							return false
						}
						if len(iArray) < len(jArray) {
							return true
						} else if len(iArray) > len(jArray) {
							return false
						}
						for l := range iArray {
							if iArray[l] != jArray[l] {
								return iArray[l].Before(jArray[l])
							}
						}
					case datatype.Integer.ID:
						iArray, ok := candidateUsers[i].Profile[sortKey].([]int)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not an integer array: %v", sortKey, candidateUsers[i].Profile[sortKey])
							return false
						}
						jArray, ok := candidateUsers[j].Profile[sortKey].([]int)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not an integer array: %v", sortKey, candidateUsers[j].Profile[sortKey])
							return false
						}
						if len(iArray) < len(jArray) {
							return true
						} else if len(iArray) > len(jArray) {
							return false
						}
						for l := range iArray {
							if iArray[l] != jArray[l] {
								return iArray[l] < jArray[l]
							}
						}
					case datatype.UUID.ID:
						iArray, ok := candidateUsers[i].Profile[sortKey].([]uuid.UUID)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a UUID array: %v", sortKey, candidateUsers[i].Profile[sortKey])
							return false
						}
						jArray, ok := candidateUsers[j].Profile[sortKey].([]uuid.UUID)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a UUID array: %v", sortKey, candidateUsers[j].Profile[sortKey])
							return false
						}
						if len(iArray) < len(jArray) {
							return true
						} else if len(iArray) > len(jArray) {
							return false
						}
						for l := range iArray {
							if iArray[l] != jArray[l] {
								return iArray[l].String() < jArray[l].String()
							}
						}
					case datatype.Composite.ID:
						return fmt.Sprintf("%v", candidateUsers[i].Profile[sortKey]) < fmt.Sprintf("%v", candidateUsers[j].Profile[sortKey])
					default:
						uclog.Warningf(ae.ctx, "unexpected data type: %v", sortDataTypes[k].ConcreteDataTypeID)
						return false
					}

				} else if candidateUsers[i].Profile[sortKey] != candidateUsers[j].Profile[sortKey] {
					switch sortDataTypes[k].ConcreteDataTypeID {
					case datatype.String.ID:
						iValue, ok := candidateUsers[i].Profile[sortKey].(string)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a string: %v", sortKey, candidateUsers[i].Profile[sortKey])
							return false
						}
						jValue, ok := candidateUsers[j].Profile[sortKey].(string)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a string: %v", sortKey, candidateUsers[j].Profile[sortKey])
							return false
						}
						return iValue < jValue
					case datatype.Boolean.ID:
						iValue, ok := candidateUsers[i].Profile[sortKey].(bool)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a boolean: %v", sortKey, candidateUsers[i].Profile[sortKey])
							return false
						}
						return !iValue
					case datatype.Timestamp.ID, datatype.Date.ID:
						iValue, ok := candidateUsers[i].Profile[sortKey].(time.Time)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a timestamp: %v", sortKey, candidateUsers[i].Profile[sortKey])
							return false
						}
						jValue, ok := candidateUsers[j].Profile[sortKey].(time.Time)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a timestamp: %v", sortKey, candidateUsers[j].Profile[sortKey])
							return false
						}
						return iValue.Before(jValue)
					case datatype.Integer.ID:
						iValue, ok := candidateUsers[i].Profile[sortKey].(int)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not an integer: %v", sortKey, candidateUsers[i].Profile[sortKey])
							return false
						}
						jValue, ok := candidateUsers[j].Profile[sortKey].(int)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not an integer: %v", sortKey, candidateUsers[j].Profile[sortKey])
							return false
						}
						return iValue < jValue
					case datatype.UUID.ID:
						iValue, ok := candidateUsers[i].Profile[sortKey].(uuid.UUID)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a UUID: %v", sortKey, candidateUsers[i].Profile[sortKey])
							return false
						}
						jValue, ok := candidateUsers[j].Profile[sortKey].(uuid.UUID)
						if !ok {
							uclog.Warningf(ae.ctx, "profile value '%v' is not a UUID: %v", sortKey, candidateUsers[j].Profile[sortKey])
							return false
						}
						return iValue.String() < jValue.String()
					case datatype.Composite.ID:
						return fmt.Sprintf("%v", candidateUsers[i].Profile[sortKey]) < fmt.Sprintf("%v", candidateUsers[j].Profile[sortKey])
					default:
						uclog.Warningf(ae.ctx, "unexpected data type: %v", sortDataTypes[k].ConcreteDataTypeID)
						return false
					}
				}
			}

			return candidateUsers[i].ID.String() < candidateUsers[j].ID.String()
		})
	}

	if limits.pager != nil &&
		((!limits.pager.IsForward() && limits.pager.GetSortOrder() == pagination.OrderAscending) ||
			(limits.pager.IsForward() && limits.pager.GetSortOrder() == pagination.OrderDescending)) {
		slices.Reverse(candidateUsers)
	}

	return candidateUsers, http.StatusOK, nil
}

func (ae *accessorExecutor) getAllowedUsers() ([]storage.User, *pagination.ResponseFields, int, error) {
	limits, err := ae.getLimits()
	if err != nil {
		return nil, nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if allowed, err := ae.checkRateThreshold(); err != nil {
		return nil, nil, http.StatusInternalServerError, ucerr.Wrap(err)
	} else if !allowed {
		uclog.Infof(ae.ctx, "accessor '%v' execution failed due to rate limit", ae.accessor.ID)
		if ae.thresholdAP.Thresholds.AnnounceMaxExecutionFailure {
			return nil, nil, http.StatusTooManyRequests, ucerr.Friendlyf(nil, "access policy rate threshold exceeded")
		}
		return nil, limits.getDefaultResponseFields(), http.StatusOK, nil
	}

	shouldGetCandidates, code, err := ae.preProcessSelector()
	if err != nil {
		return nil, nil, code, ucerr.Wrap(err)
	}

	candidateUsers := []storage.User{}
	needSort := false

	if shouldGetCandidates {
		ts := multitenant.MustGetTenantState(ae.ctx)

		if ae.req.Region != "" {
			regDB, ok := ts.UserRegionDbMap[ae.req.Region]
			if !ok {
				return nil, nil, http.StatusBadRequest, ucerr.Friendlyf(nil, "data region '%s' is not available for tenant", ae.req.Region)
			}
			us := storage.NewUserStorage(ae.ctx, regDB, ae.req.Region, ts.ID)
			candidateUsers, code, err = us.GetUsersForSelector(
				ae.ctx,
				ae.cm,
				ae.dtm,
				ae.startTime,
				ae.accessor.DataLifeCycleState,
				ae.columns,
				ae.accessor.SelectorConfig,
				ae.req.SelectorValues,
				ae.expectedColumnIDs,
				ae.expectedPurposeIDs,
				limits.pager,
				ae.req.AccessPrimaryDBOnly,
			)
			if err != nil {
				return nil, nil, code, ucerr.Wrap(err)
			}
		} else {
			umrs := storage.NewUserMultiRegionStorage(ae.ctx, ts.UserRegionDbMap, ts.ID)
			candidateUsersByRegion, code, err := umrs.GetUsersForSelector(
				ae.ctx,
				ae.cm,
				ae.dtm,
				ae.startTime,
				ae.accessor.DataLifeCycleState,
				ae.columns,
				ae.accessor.SelectorConfig,
				ae.req.SelectorValues,
				ae.expectedColumnIDs,
				ae.expectedPurposeIDs,
				limits.pager,
				ae.req.AccessPrimaryDBOnly,
			)
			if err != nil {
				return nil, nil, code, ucerr.Wrap(err)
			}
			for _, users := range candidateUsersByRegion {
				candidateUsers = append(candidateUsers, users...)
			}
			if len(candidateUsersByRegion) > 1 {
				needSort = true
			}
		}
	}

	if needSort {
		// we need to re-sort because we are combining results from multiple regions
		candidateUsers, code, err = ae.sortMergedCandidateUsers(candidateUsers)
		if err != nil {
			return nil, nil, code, ucerr.Wrap(err)
		}
	}

	if limits.pager != nil && limits.limit > 0 {
		// because we overfetch by one, we know whether there
		// are more results beyond the requested limit
		if limit := limits.limit * limits.pager.GetLimitMultiplier(); len(candidateUsers) > limit {
			limits.hasMore = true
			candidateUsers = candidateUsers[:limit]
		}
	}

	if len(candidateUsers) > 0 {
		limits.firstUser = candidateUsers[0]
		limits.lastUser = candidateUsers[len(candidateUsers)-1]
	}

	var allowedUsers []storage.User
	for i, u := range candidateUsers {
		apContext := ae.apContext
		apContext.User = u.Profile

		allowed, console, err := tokenizer.ExecuteAccessPolicy(ae.ctx, ae.clientAP, apContext, ae.authzClient, ae.s)
		if err != nil {
			return nil, nil, http.StatusInternalServerError, ucerr.Wrap(err)
		}
		ae.accessPolicyConsole += console

		if !allowed {
			if limits.limit == 0 || len(allowedUsers) < limits.limit {
				ae.numDenied++
			}
			continue
		}

		if limits.limit == 0 || len(allowedUsers) < limits.limit {
			allowedUsers = append(allowedUsers, u)
			if len(allowedUsers) == limits.limit {
				limits.lastUser = u
				limits.hasMore = limits.hasMore || i < len(candidateUsers)-1
				if limits.maxLimit == 0 {
					break
				}
			}
		}

		limits.numAllowed++
		if limits.maxExceeded() {
			uclog.Warningf(ae.ctx, "accessor '%v' execution failed due to result limit. num allowed: %d  max limit: %d. Announce: %v", ae.accessor.ID, limits.numAllowed, limits.maxLimit, ae.thresholdAP.Thresholds.AnnounceMaxResultFailure)
			if ae.thresholdAP.Thresholds.AnnounceMaxResultFailure {
				return nil,
					nil,
					http.StatusBadRequest,
					ucerr.Friendlyf(nil, "access policy result threshold exceeded")
			}
			return nil, limits.getDefaultResponseFields(), http.StatusOK, nil
		}
	}

	allowedUsers = limits.orderUsers(allowedUsers)
	ae.numReturned = len(allowedUsers)

	respFields, err := limits.getResponseFields()
	if err != nil {
		return nil, limits.getDefaultResponseFields(), http.StatusInternalServerError, ucerr.Wrap(err)
	}

	return allowedUsers, respFields, http.StatusOK, nil
}

func (ae accessorExecutor) getLimits() (*accessorLimits, error) {
	limits := accessorLimits{
		pager: ae.pager,
		pu:    ae.pu,
	}
	if limits.pager != nil {
		limits.limit = limits.pager.GetLimit()
	}

	if !ae.isInitialQuery() || !ae.thresholdAP.HasResultThreshold() {
		return &limits, nil
	}

	limits.maxLimit = ae.thresholdAP.GetResultThreshold()
	if limits.pager != nil {
		pager, err := pagination.ApplyOptions(append(limits.pager.GetOptions(), pagination.Limit(limits.maxLimit))...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		limits.pager = pager
	}

	return &limits, nil
}

func (ae *accessorExecutor) initialize() error {
	// set up data type manager

	dtm, err := storage.NewDataTypeManager(ae.ctx, ae.s)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ae.dtm = dtm

	// load purposes and set ap context

	purposes, err := ae.s.GetPurposesForIDs(ae.ctx, true, ae.accessor.PurposeIDs...)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ae.apContext = tokenizer.BuildBaseAPContext(ae.ctx, ae.req.Context, policy.ActionExecute, purposes...)

	// load columns

	cm, err := storage.NewUserstoreColumnManager(ae.ctx, ae.s)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ae.columns = cm.GetColumns()
	ae.cm = cm

	accessorColumns, err := ae.getAccessorColumns(ae.columns, ae.accessor)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ae.accessorColumns = accessorColumns

	ae.expectedColumnIDs = set.NewUUIDSet()
	for _, accessorColumn := range accessorColumns {
		ae.expectedColumnIDs.Insert(accessorColumn.ID)
	}

	// load transformers

	transformerIDs := make([]uuid.UUID, 0, len(ae.accessor.TransformerIDs))
	for i, transformerID := range ae.accessor.TransformerIDs {
		if transformerID.IsNil() {
			transformerIDs = append(transformerIDs, ae.accessorColumns[i].DefaultTransformerID)
		} else {
			transformerIDs = append(transformerIDs, transformerID)
		}
	}
	transformerMap, err := ae.s.GetTransformersMap(ae.ctx, transformerIDs)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ae.transformerMap = transformerMap

	// load access policies and authz client

	globalAP, accessorAP, thresholdAP, err :=
		ae.s.GetAccessPolicies(
			ae.ctx,
			multitenant.MustGetTenantState(ae.ctx).ID,
			policy.AccessPolicyGlobalAccessorID,
			ae.accessor.AccessPolicyID,
		)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ae.thresholdAP = thresholdAP

	ae.clientAP = &policy.AccessPolicy{
		PolicyType: policy.PolicyTypeCompositeAnd,
		Components: []policy.AccessPolicyComponent{{Policy: &userstorePublic.ResourceID{ID: globalAP.ID}},
			{Policy: &userstorePublic.ResourceID{ID: accessorAP.ID}},
		},
	}

	// Add the column access policies to the access policy, if it is not the GetUser accessor
	if ae.accessor.ID != constants.GetUserAccessorID && !ae.accessor.AreColumnAccessPoliciesOverridden {
		columnAccessPolicyRIDs := make([]userstorePublic.ResourceID, 0, len(ae.accessorColumns))
		for _, col := range ae.accessorColumns {
			if col.AccessPolicyID != policy.AccessPolicyAllowAll.ID {
				columnAccessPolicyRIDs = append(columnAccessPolicyRIDs, userstorePublic.ResourceID{ID: col.AccessPolicyID})
			}
		}
		for _, accessPolicyRID := range columnAccessPolicyRIDs {
			ae.clientAP.Components = append(ae.clientAP.Components, policy.AccessPolicyComponent{Policy: &accessPolicyRID})
		}
	}

	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ae.ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ae.authzClient = authzClient

	return nil
}

func (ae accessorExecutor) isInitialQuery() bool {
	return ae.pager == nil || ae.pager.IsInitialQuery()
}

func (ae accessorExecutor) isArrayColumn(c storage.Column) bool {
	return c.IsArray ||
		(!c.Attributes.System &&
			ae.accessor.DataLifeCycleState == column.DataLifeCycleStateSoftDeleted)
}

func (ae accessorExecutor) newSearchExecutor() (*searchExecutor, error) {
	if ae.searchUpdateConfig == nil || ae.searchUpdateConfig.SearchCfg == nil || !ae.accessor.UseSearchIndex {
		return nil, nil
	}

	// prepare request

	searchClient, err := ucopensearch.NewClient(ae.ctx, ae.searchUpdateConfig.SearchCfg)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	sim, err := storage.NewSearchIndexManager(ae.ctx, ae.s)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	qi, err := sim.GetQueryableIndexForAccessorID(ae.accessor.ID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	searchParams, err :=
		storage.GetExecutableSearchParameters(
			ae.ctx,
			ae.cm,
			ae.accessor.SelectorConfig,
			ae.req.SelectorValues,
		)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if searchParams == nil {
		uclog.Infof(
			ae.ctx,
			"accessor '%v' not using opensearch for selector values: '%v'",
			ae.accessor.ID,
			ae.req.SelectorValues,
		)
		return nil, nil
	}

	ts := multitenant.MustGetTenantState(ae.ctx)

	se := searchExecutor{
		ctx:          ae.ctx,
		searchClient: searchClient,
		indexName:    qi.GetIndexName(ts.ID),
		searchParams: searchParams,
	}

	if err := se.setSearchQuery(qi); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &se, nil
}

func (ae *accessorExecutor) preProcessSelector() (bool, int, error) {
	se, err := ae.newSearchExecutor()
	if err != nil {
		return false, http.StatusBadRequest, ucerr.Wrap(err)
	}

	if se == nil {
		if ae.req.SelectorValues, err =
			storage.GetAdjustedSelectorValues(
				ae.ctx,
				ae.cm,
				ae.accessor.SelectorConfig,
				ae.req.SelectorValues,
			); err != nil {
			return false, http.StatusBadRequest, ucerr.Wrap(err)
		}

		return true, http.StatusOK, nil
	}

	searchTimer := timer.Start()
	userIDs, truncated, err := se.executeSearch()
	searchTime := searchTimer.Elapsed()

	if err != nil {
		return false, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if len(userIDs) == 0 {
		return false, http.StatusOK, nil
	}

	ae.numSearchRows = len(userIDs)

	userIDs, err = ae.filterSearchUserIDs(userIDs)
	if err != nil {
		return false, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	ae.accessor.SelectorConfig.WhereClause = "{id} = ANY(?)"
	ae.req.SelectorValues = []any{userIDs}
	ae.truncated = truncated

	uclog.Infof(ae.ctx, "found %d users via search: Accessor: '%v', filtered to %d userIDs: '%v', truncated: %v, elapsed: '%v'",
		ae.numSearchRows, ae.accessor.ID, len(userIDs),
		// Don't log more than 10 user IDs
		userIDs[:min(len(userIDs), 10)],
		truncated, searchTime,
	)
	return true, http.StatusOK, nil
}

func (ae *accessorExecutor) setupPagination(options *accessorPaginationOptions) error {
	if options == nil {
		ae.pager = nil
		return nil
	}

	pu := newPageableUser()

	for _, c := range ae.accessorColumns {
		if !ae.isArrayColumn(c) {
			pu.addColumn(c)
		}
	}
	ae.pu = pu

	// defaultPageSize will be superceded by accessorPaginationOptions.Limit if specified
	const defaultPageSize int = 1500

	// limitMultiplier is used to overfetch results for limit queries due to AP filtering or users in OS that have been deleted
	const limitMultiplier int = 10

	// initialize the pager
	pager, err := pagination.NewPaginatorFromQuery(
		pagination.QueryParams{
			EndingBefore:  options.EndingBefore,
			Limit:         options.Limit,
			SortKey:       options.SortKey,
			SortOrder:     options.SortOrder,
			StartingAfter: options.StartingAfter,
		},
		pagination.Limit(defaultPageSize),
		pagination.LimitMultiplier(limitMultiplier),
		pagination.ResultType(pu),
	)
	if err != nil {
		return ucerr.Wrap(err)
	}

	ae.pager = pager
	return nil
}

func (ae *accessorExecutor) transformResults(users []storage.User) ([]string, error) {
	te := tokenizer.NewTransformerExecutor(ae.s, ae.authzClient)
	defer te.CleanupExecution()

	output := make([]string, 0, len(users))
	for _, u := range users {
		// configure the transformers for the profile strings
		var transformableValues []transformableValue
		var transformerParams []tokenizer.ExecuteTransformerParameters
		for i, c := range ae.accessorColumns {
			value := u.Profile[c.Name]
			if value == nil {
				continue
			}

			transformerID := ae.accessor.TransformerIDs[i]
			tokenAccessPolicyID := ae.accessor.TokenAccessPolicyIDs[i]
			if transformerID.IsNil() {
				transformerID = c.DefaultTransformerID
				tokenAccessPolicyID = c.DefaultTokenAccessPolicyID
			}
			transformer := ae.transformerMap[transformerID]
			tv, err := newTransformableOutputValue(
				ae.ctx,
				ae.dtm,
				c,
				*transformer,
				ae.isArrayColumn(c),
				value,
			)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}

			if tv.shouldTransform {
				inputs, err := tv.getTransformableInputs(ae.ctx)
				if err != nil {
					return nil, ucerr.Wrap(err)
				}

				for _, input := range inputs {
					transformerParams = append(
						transformerParams,
						tokenizer.ExecuteTransformerParameters{
							Transformer:         transformer,
							TokenAccessPolicyID: tokenAccessPolicyID,
							Data:                input,
							DataProvenance:      &policy.UserstoreDataProvenance{UserID: u.ID, ColumnID: c.ID},
						},
					)
					tv.addValueIndex(len(transformerParams) - 1)
				}
			}

			if err := tv.Validate(); err != nil {
				return nil, ucerr.Wrap(err)
			}

			transformableValues = append(transformableValues, *tv)
		}

		var transformedValues []string
		if len(transformerParams) > 0 {
			var err error
			var transformerConsole string

			// execute the transformers
			uclog.Infof(ae.ctx, "Transforming %v values for accessor %v [%v]", len(transformerParams), ae.accessor.Name, ae.accessor.ID)
			transformedValues, transformerConsole, err = te.Execute(ae.ctx, transformerParams...)
			if err != nil {
				logAccessorTransformerError(ae.ctx, ae.accessor.ID, ae.accessor.Version)
				return nil, ucerr.Wrap(err)
			}
			ae.transformerConsole += transformerConsole
		}

		// marshal the transformed values
		profileValues := map[string]any{}
		for _, tv := range transformableValues {
			value, err := tv.getValue(ae.ctx, transformedValues)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			profileValues[tv.columnName] = value
		}
		userOutput, err := json.Marshal(profileValues)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		output = append(output, string(userOutput))
	}

	ae.succeeded = true
	return output, nil
}

type searchExecutor struct {
	ctx          context.Context
	searchClient *ucopensearch.Client
	indexName    string
	searchParams *storage.SearchParameters
	searchQuery  string
}

type searchResponse struct {
	Hits struct {
		Hits []struct {
			ID     uuid.UUID      `json:"_id"`
			Source map[string]any `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func (se searchExecutor) getTermKey() string {
	return se.searchParams.Column.ID.String()
}

func (se *searchExecutor) setSearchQuery(qi ucopensearch.QueryableIndex) error {
	searchQuery, err := qi.GetIndexQuery(se.searchParams.Term, se.searchClient.MaxResults+1)
	if err != nil {
		return ucerr.Wrap(err)
	}

	se.searchQuery = searchQuery
	return nil
}

func (se searchExecutor) executeSearch() (userIDs []uuid.UUID, truncated bool, err error) {
	// Terms can contain PII
	uclog.DebugfPII(se.ctx, "Search request index: %v truncated term %v full term %v", se.indexName, se.searchParams.Truncated, se.searchParams.TermFull)
	searchStart := time.Now().UTC()
	respBody, err := se.searchClient.SearchRequest(se.ctx, se.indexName, se.searchQuery)
	duration := time.Now().UTC().Sub(searchStart)

	if err != nil {
		return nil, false, ucerr.Wrap(err)
	}

	var searchResponse searchResponse
	if err := json.Unmarshal(respBody, &searchResponse); err != nil {
		uclog.Errorf(se.ctx, "Failed to unmarshal search response: %v", string(respBody))
		return nil, false, ucerr.Wrap(err)
	}
	searchHits := searchResponse.Hits.Hits
	if len(searchHits) == 0 {
		uclog.Debugf(se.ctx, "no hits in search response: %v", string(respBody))
	}

	truncated = false
	resultsLength := min(len(searchHits), se.searchClient.MaxResults)

	lowerCaseTermFull := strings.ToLower(se.searchParams.TermFull)
	termKey := se.getTermKey()

	addedResults := 0
	userIDs = make([]uuid.UUID, 0, resultsLength)
	for _, hit := range searchHits {
		// If we have reached the max results, we will to truncate
		if addedResults >= se.searchClient.MaxResults {
			truncated = true
			break
		}
		// If search term was truncated, we need to filter out the results which don't match the full term
		if se.searchParams.Truncated {
			val := hit.Source[termKey]
			sV, gotColumn := val.(string)
			if !gotColumn {
				uclog.Errorf(se.ctx, "Unexpected value for search column: %v (%T) skipping %v", val, val, hit)
				continue
			}
			if !strings.Contains(strings.ToLower(sV), lowerCaseTermFull) {
				continue
			}
		}
		userIDs = append(userIDs, hit.ID)
		addedResults++
	}
	hitsToDisplay := min(len(searchHits), 10)
	uclog.DebugfPII(se.ctx, "Search returned %v in %v added: %v truncated %v", hitsToDisplay, duration, addedResults, truncated)
	userIDs = userIDs[:addedResults]
	return userIDs, truncated, nil
}

// GetUsers will return user records and accompanying audit log entries for the specified user IDs
func GetUsers(
	ctx context.Context,
	checkMatchingLen bool,
	reg region.DataRegion,
	AccessPrimaryDBOnly bool,
	userIDs ...string,
) ([]string, []auditlog.Entry, error) {
	resp, _, entries, err := executeAccessor(
		ctx,
		nil, // opensearch is not used for GetUsers
		idp.ExecuteAccessorRequest{
			AccessorID:          constants.GetUserAccessorID,
			Context:             policy.ClientContext{},
			SelectorValues:      []any{userIDs},
			Region:              reg,
			AccessPrimaryDBOnly: AccessPrimaryDBOnly,
		},
		false,
		nil,
	)

	if err != nil {
		return nil, entries, ucerr.Wrap(err)
	}

	if checkMatchingLen && len(resp.Data) != len(userIDs) {
		return nil, entries, ucerr.Errorf("expected %d results but received %d", len(userIDs), len(resp.Data))
	}

	return resp.Data, entries, nil
}

// executeAccessor is an internal idp method that executes an accessor, pass in nil for p to get all results
func executeAccessor(
	ctx context.Context,
	searchUpdateConfig *config.SearchUpdateConfig,
	req idp.ExecuteAccessorRequest,
	includeDebug bool,
	paginationOptions *accessorPaginationOptions,
) (*idp.ExecuteAccessorResponse, int, []auditlog.Entry, error) {

	startTime := time.Now().UTC()

	s := storage.MustCreateStorage(ctx)

	// look up accessor

	accessor, err := s.GetLatestAccessor(ctx, req.AccessorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	defer logAccessorDuration(ctx, accessor.ID, accessor.Version, startTime)
	logAccessorCall(ctx, accessor.ID, accessor.Version)

	// initialize accessor executor

	ae := newAccessorExecutor(ctx, s, searchUpdateConfig, req, startTime, includeDebug, accessor)
	if err := ae.initialize(); err != nil {
		logAccessorConfigError(ctx, accessor.ID, accessor.Version)
		return nil, http.StatusInternalServerError, ae.auditLogInfo(), ucerr.Wrap(err)
	}

	// set up pagination

	if err := ae.setupPagination(paginationOptions); err != nil {
		return nil, http.StatusBadRequest, ae.auditLogInfo(), ucerr.Wrap(err)
	}

	// get the results for the selector

	users, respFields, code, err := ae.getAllowedUsers()
	if err != nil {
		logAccessorNotFoundError(ctx, accessor.ID, accessor.Version)
		return nil, code, ae.auditLogInfo(), ucerr.Wrap(err)
	}

	// transform results

	output, err := ae.transformResults(users)
	if err != nil {
		return nil, http.StatusInternalServerError, ae.auditLogInfo(), ucerr.Wrap(err)
	}

	logAccessorSuccess(ctx, accessor.ID, accessor.Version)

	return &idp.ExecuteAccessorResponse{
			Data:           output,
			Debug:          ae.debugInfo(),
			Truncated:      ae.truncated,
			ResponseFields: *respFields,
		},
		http.StatusOK,
		ae.auditLogInfo(),
		nil
}

type accessorPaginationOptions struct {
	StartingAfter *string `description:"A cursor value after which the returned list will start" query:"starting_after"`
	EndingBefore  *string `description:"A cursor value before which the returned list will end" query:"ending_before"`
	Limit         *string `description:"The maximum number of results to be returned per page" query:"limit"`
	SortKey       *string `description:"A comma-delimited list of column names to sort the results by - the last column name must be 'id'" query:"sort_key"`
	SortOrder     *string `description:"The order in which results should be sorted (ascending or descending)" query:"sort_order"`
}

type executeAccessorHandlerRequest struct {
	idp.ExecuteAccessorRequest
	accessorPaginationOptions
}

// OpenAPI Summary: Execute Accessor
// OpenAPI Tags: Accessors
// OpenAPI Description: This endpoint executes a specified accessor (custom read API).
func (h *handler) executeAccessorHandler(
	ctx context.Context,
	req executeAccessorHandlerRequest,
) (*idp.ExecuteAccessorResponse, int, []auditlog.Entry, error) {
	if err := h.ensureTenantMember(false); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	includeDebug := false
	if req.ExecuteAccessorRequest.Debug {
		authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		ts := multitenant.MustGetTenantState(ctx)
		resp, err := authzClient.CheckAttribute(ctx, auth.GetSubjectUUID(ctx), ts.CompanyID, "_admin")
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		if !resp.HasAttribute {
			return nil, http.StatusForbidden, nil, ucerr.Friendlyf(nil, "You must be an admin to view debug information")
		}
		includeDebug = true
	}

	resp, code, entries, err :=
		executeAccessor(
			ctx,
			h.searchUpdateConfig,
			req.ExecuteAccessorRequest,
			includeDebug,
			&req.accessorPaginationOptions,
		)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, entries, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, entries, ucerr.Wrap(err)
		case http.StatusTooManyRequests:
			return nil, http.StatusTooManyRequests, entries, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
		}
	}

	return resp, http.StatusOK, entries, nil
}
