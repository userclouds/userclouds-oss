package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/async"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/timer"
)

const (
	commonPoolUserOps int = 10
	poolSizeUserStore int = 50
)

var passthroughTransformer = userstore.ResourceID{ID: policy.TransformerPassthrough.ID}

type userStoreTest struct {
	entityPrefix                   string
	idp                            *idp.Client
	tc                             *idp.TokenizerClient
	vg                             *valueGenerator
	threadCount                    int
	addressColumnName              string
	addressDataTypeName            string
	listUsersAccessorName          string
	phoneNumberAddressAccessorName string
	phoneNumberColumnName          string
	phoneNumberTransformerName     string
}

func newUserStoreTest(ctx context.Context, threadCount int, tenantURL string, tokenSource jsonclient.Option) (*userStoreTest, error) {
	entityPrefix := fmt.Sprintf("EnvTest%d", rand.Intn(1000000))

	idpc, err := idp.NewClient(tenantURL, idp.JSONClient(tokenSource))
	if err != nil {
		logErrorf(ctx, err, "Userstore: IDP client creation failed: %v", err)
		return nil, ucerr.Wrap(err)
	}

	return &userStoreTest{
		entityPrefix:                   entityPrefix,
		idp:                            idpc,
		tc:                             idpc.TokenizerClient,
		threadCount:                    threadCount,
		vg:                             newValueGenerator(),
		addressColumnName:              entityPrefix + "Address",
		addressDataTypeName:            entityPrefix + "Address",
		listUsersAccessorName:          entityPrefix + "ListUsersForCleanup",
		phoneNumberAddressAccessorName: entityPrefix + "GetPhoneNumberAndAddress",
		phoneNumberColumnName:          entityPrefix + "PhoneNumber",
		phoneNumberTransformerName:     entityPrefix + "TransformPhoneNumber",
	}, nil
}

func (ut *userStoreTest) cleanup(ctx context.Context) {
	tmr := timer.Start()

	uclog.Debugf(ctx, "Userstore: Clean up started for %d threads", ut.threadCount)

	ut.cleanupUsers(ctx)
	ut.cleanupAccessors(ctx)
	ut.cleanupTransformers(ctx)
	ut.cleanupAccessPolicies(ctx)
	ut.cleanupAccessPolicyTemplates(ctx)
	ut.cleanupColumns(ctx)
	ut.cleanupDataTypes(ctx)

	uclog.Debugf(ctx, "Userstore: Finished clean up in %v", tmr.Elapsed())
}

func getUnpaginatedCollection[T any](ctx context.Context, name string, m func(ctx context.Context, opts ...idp.Option) ([]T, bool, pagination.Cursor, error)) []T {
	var items = make([]T, 0)
	cursor := pagination.CursorBegin
	for {
		itemsResp, hasNext, nextCursor, err := m(ctx, idp.Pagination(pagination.StartingAfter(cursor), pagination.Limit(pagination.MaxLimit)))
		if err != nil {
			logErrorf(ctx, err, "Userstore: %v failed during cleanup: %v", name, err)
			return nil
		}

		items = append(items, itemsResp...)
		if !hasNext {
			break
		}
		cursor = nextCursor
	}
	return items
}

func (ut *userStoreTest) cleanupAccessors(ctx context.Context) {
	accessors := getUnpaginatedCollection(ctx, "ListAccessors",
		func(ctx context.Context, opts ...idp.Option) ([]userstore.Accessor, bool, pagination.Cursor, error) {
			policyResp, err := ut.idp.ListAccessors(ctx, false, opts...)
			if err != nil {
				return nil, false, pagination.CursorBegin, ucerr.Wrap(err)
			}
			return policyResp.Data, policyResp.HasNext, policyResp.Next, nil
		})

	totalDeleted := 0
	for _, a := range accessors {
		if !ut.isEnvTestName(a.Name) {
			continue
		}

		if err := ut.idp.DeleteAccessor(ctx, a.ID); err != nil {
			logErrorf(ctx, err, "Userstore: DeleteAccessor failed for '%v' during cleanup: %v", a.ID, err)
			continue
		}

		totalDeleted++
	}
	uclog.Debugf(ctx, "Userstore: Finished accessor clean up. Deleted %d accessors", totalDeleted)
}

func (ut *userStoreTest) cleanupAccessPolicies(ctx context.Context) {
	policies := getUnpaginatedCollection(ctx, "ListAccessPolicies",
		func(ctx context.Context, opts ...idp.Option) ([]policy.AccessPolicy, bool, pagination.Cursor, error) {
			policyResp, err := ut.idp.ListAccessPolicies(ctx, false, opts...)
			if err != nil {
				return nil, false, pagination.CursorBegin, ucerr.Wrap(err)
			}
			return policyResp.Data, policyResp.HasNext, policyResp.Next, nil
		})

	totalDeleted := 0
	for _, ap := range policies {
		if !ut.isEnvTestName(ap.Name) {
			continue
		}

		if err := ut.tc.DeleteAccessPolicy(ctx, ap.ID, 0); err != nil {
			logErrorf(ctx, err, "Userstore: DeleteAccessPolicy failed for '%v' during cleanup: %v", ap.ID, err)
			continue
		}

		totalDeleted++
	}
	uclog.Debugf(ctx, "Userstore: Finished access policy clean up. Deleted %d access policies", totalDeleted)
}

func (ut *userStoreTest) cleanupAccessPolicyTemplates(ctx context.Context) {
	policies := getUnpaginatedCollection(ctx, "ListAccessPolicyTemplates",
		func(ctx context.Context, opts ...idp.Option) ([]policy.AccessPolicyTemplate, bool, pagination.Cursor, error) {
			policyResp, err := ut.idp.ListAccessPolicyTemplates(ctx, false, opts...)
			if err != nil {
				return nil, false, pagination.CursorBegin, ucerr.Wrap(err)
			}
			return policyResp.Data, policyResp.HasNext, policyResp.Next, nil
		})

	totalDeleted := 0
	for _, apt := range policies {
		if !ut.isEnvTestName(apt.Name) {
			continue
		}

		if err := ut.tc.DeleteAccessPolicyTemplate(ctx, apt.ID, 0); err != nil {
			logErrorf(ctx, err, "Userstore: DeleteAccessPolicyTemplate failed for '%v' during cleanup: %v", apt.ID, err)
			continue
		}

		totalDeleted++
	}
	uclog.Debugf(ctx, "Userstore: Finished access policy template clean up. Deleted %d access policy templates", totalDeleted)
}

func (ut *userStoreTest) cleanupColumns(ctx context.Context) {
	columns := getUnpaginatedCollection(ctx, "ListColumns",
		func(ctx context.Context, opts ...idp.Option) ([]userstore.Column, bool, pagination.Cursor, error) {
			columnsResp, err := ut.idp.ListColumns(ctx, opts...)
			if err != nil {
				return nil, false, pagination.CursorBegin, ucerr.Wrap(err)
			}
			return columnsResp.Data, columnsResp.HasNext, columnsResp.Next, nil
		})

	totalDeleted := 0
	for _, c := range columns {
		if !ut.isEnvTestName(c.Name) {
			continue
		}

		if err := ut.idp.DeleteColumn(ctx, c.ID); err != nil {
			logErrorf(ctx, err, "Userstore: DeleteColumn failed for '%v' during cleanup: %v", c.ID, err)
			continue
		}

		totalDeleted++
	}
	uclog.Debugf(ctx, "Userstore: Finished column clean up. Deleted %d columns", totalDeleted)
}

func (ut *userStoreTest) cleanupDataTypes(ctx context.Context) {
	dataTypes := getUnpaginatedCollection(ctx, "ListDataTypes",
		func(ctx context.Context, opts ...idp.Option) ([]userstore.ColumnDataType, bool, pagination.Cursor, error) {
			dataTypeResp, err := ut.idp.ListDataTypes(ctx, opts...)
			if err != nil {
				return nil, false, pagination.CursorBegin, ucerr.Wrap(err)
			}
			return dataTypeResp.Data, dataTypeResp.HasNext, dataTypeResp.Next, nil
		})

	totalDeleted := 0
	for _, dt := range dataTypes {
		if !ut.isEnvTestName(dt.Name) {
			continue
		}

		if err := ut.idp.DeleteDataType(ctx, dt.ID); err != nil {
			logErrorf(ctx, err, "Userstore: DeleteDataType failed for '%v' during cleanup: %v", dt.ID, err)
			continue
		}

		totalDeleted++
	}
	uclog.Debugf(ctx, "Userstore: Finished data type clean up. Deleted %d data types", totalDeleted)
}

func (ut *userStoreTest) cleanupTransformers(ctx context.Context) {
	transformers := getUnpaginatedCollection(ctx, "ListTransformers",
		func(ctx context.Context, opts ...idp.Option) ([]policy.Transformer, bool, pagination.Cursor, error) {
			transformerResp, err := ut.idp.ListTransformers(ctx, opts...)
			if err != nil {
				return nil, false, pagination.CursorBegin, ucerr.Wrap(err)
			}
			return transformerResp.Data, transformerResp.HasNext, transformerResp.Next, nil
		})

	totalDeleted := 0
	for _, t := range transformers {
		if !ut.isEnvTestName(t.Name) {
			continue
		}

		if err := ut.tc.DeleteTransformer(ctx, t.ID); err != nil {
			logErrorf(ctx, err, "Userstore: DeleteTransformer failed for '%v' during cleanup: %v", t.ID, err)
			continue
		}

		totalDeleted++
	}
	uclog.Debugf(ctx, "Userstore: Finished transformer clean up. Deleted %d transformers", totalDeleted)
}

func (ut *userStoreTest) cleanupUsers(ctx context.Context) {
	accessors := getUnpaginatedCollection(ctx, "ListAccessors",
		func(ctx context.Context, opts ...idp.Option) ([]userstore.Accessor, bool, pagination.Cursor, error) {
			transformerResp, err := ut.idp.ListAccessors(ctx, false, opts...)
			if err != nil {
				return nil, false, pagination.CursorBegin, ucerr.Wrap(err)
			}
			return transformerResp.Data, transformerResp.HasNext, transformerResp.Next, nil
		})

	accessorID := uuid.Nil
	for _, a := range accessors {
		if a.Name == ut.listUsersAccessorName {
			accessorID = a.ID
			break
		}
	}

	if accessorID.IsNil() {
		logErrorf(ctx, nil, "Userstore: Could not find accessor '%s' during user cleanup", ut.listUsersAccessorName)
		return
	}

	users, err := ut.idp.ExecuteAccessor(
		ctx,
		accessorID,
		map[string]any{},
		nil,
	)
	if err != nil {
		logErrorf(ctx, err, "Userstore: Could not execute accessor '%s' during user cleanup: %v", ut.listUsersAccessorName, err)
		return
	}

	totalDeleted := 0
	for _, userData := range users.Data {
		record := map[string]string{}
		if err = json.Unmarshal([]byte(userData), &record); err != nil {
			logErrorf(ctx, err, "Userstore: ExecuteAccessor return value %s for user cleanup could not be parsed: %v", userData, err)
			continue
		}

		uid, err := uuid.FromString(record["id"])
		if err != nil {
			logErrorf(ctx, err, "Userstore: user record did not contain valid 'id' in record '%v' during user cleanup: %v", record, err)
			continue
		}

		if err := ut.idp.DeleteUser(ctx, uid); err != nil {
			logErrorf(ctx, err, "Userstore: DeleteUser failed for '%v' during user cleanup: %v", uid, err)
			continue
		}

		totalDeleted++
	}
	uclog.Debugf(ctx, "Userstore: Finished user clean up. Deleted %d users", totalDeleted)
}

func (ut userStoreTest) isEnvTestName(name string) bool {
	return strings.HasPrefix(name, ut.entityPrefix)
}

func (ut userStoreTest) newUserAlias() string {
	return fmt.Sprintf("%s_%v", ut.entityPrefix, uuid.Must(uuid.NewV4()))
}

func (ut *userStoreTest) runCommonPoolTests(
	ctx context.Context,
	threadNum int,
	users []uuid.UUID,
	aliases []string,
	records []userstore.Record,
) {
	tmr := timer.Start()
	uclog.Debugf(ctx, "Userstore: Thread %2d testing common pool", threadNum)

	// Check objects/types from common pool
	for range commonPoolAuthzOps {
		ui := rand.Intn(len(users))

		uclog.Debugf(
			ctx,
			"Userstore: Thread %2d testing common pool user index %3d, id %v, alias %v, phone %v, address %v",
			threadNum,
			ui,
			users[ui],
			aliases[ui],
			records[ui][ut.phoneNumberColumnName],
			records[ui][ut.addressColumnName],
		)

		user, err := ut.idp.GetUser(ctx, users[ui])
		if err != nil || user.ID != users[ui] || user.Profile["external_alias"] != aliases[ui] {
			logErrorf(ctx, err, "Userstore: Thread %d GetUser failed from common pool: %v", threadNum, err)
		}

		if user != nil {
			for columnName := range records[ui] {

				if ut.valueAsString(user.Profile[columnName]) != ut.valueAsString(records[ui][columnName]) {
					logErrorf(ctx, nil,
						"Userstore: Thread %2d GetUser difference in expected profile for %s from common pool got %v expected %v",
						threadNum,
						columnName,
						user.Profile[columnName],
						records[ui][columnName],
					)
				}
			}
		}

		accessors, err := ut.idp.ListAccessors(ctx, false)
		if err != nil {
			logErrorf(ctx, err, "Userstore: Thread %2d ListAccessors failed from common pool: %v", threadNum, err)
			continue
		}

		for _, a := range accessors.Data {
			if a.Name == ut.phoneNumberAddressAccessorName {
				var fieldsStr *idp.ExecuteAccessorResponse
				retryCount := 0
				maxRetryCount := envTestCfg.Threads
				for {
					fieldsStr, err = ut.idp.ExecuteAccessor(ctx,
						a.ID,
						map[string]any{"api_key": "security_api_key"},
						[]any{aliases[ui]},
					)

					if err == nil {
						uclog.Debugf(ctx, "[runCommonPoolTests] Execute accessor %v [%v] success", a.Name, a.ID)
						break
					}
					if retryCount >= maxRetryCount {
						logErrorf(ctx, err, "Userstore: Thread %2d ExecuteAccessor failed for per thread accessor: %v", threadNum, err)
						break
					}
					retryCount++
				}
				// TODO DEVEXP is this really a map[string]any if so why doesn't SDK just return that
				fields := map[string]string{}
				if len(fieldsStr.Data) > 0 {
					if err = json.Unmarshal([]byte(fieldsStr.Data[0]), &fields); err != nil {
						logErrorf(ctx, err, "Userstore: Thread %2d ExecuteAccessor return value %s from common pool couldn't be parsed: %v", threadNum, fieldsStr.Data, err)
					}
				}

				if len(fields) != 2 {
					logErrorf(ctx, err, "Userstore: Thread %2d ExecuteAccessor returned unexpected field count: %d", threadNum, len(fieldsStr.Data))
				}

				if len(fields) != 2 ||
					ut.valueAsString(fields[ut.phoneNumberColumnName]) != ut.valueAsString(records[ui][ut.phoneNumberColumnName]) ||
					ut.valueAsString(fields[ut.addressColumnName]) != ut.valueAsString(records[ui][ut.addressColumnName]) {
					logErrorf(ctx, err, "Userstore: Thread %2d ExecuteAccessor returned unexpected values: %v expected %v", threadNum, fieldsStr, records[ui])
				}
			}
		}
	}

	uclog.Debugf(ctx, "Userstore: Thread %2d  finished testing common pool in %v", threadNum, tmr.Elapsed())
}

func (ut *userStoreTest) runPerThreadTests(ctx context.Context, threadNum int) error {
	tmr := timer.Start()
	uclog.Debugf(ctx, "Userstore: Thread %2d testing per thread", threadNum)

	ssnColumnName := ut.threadSSNColumnName(threadNum)

	userAlias := ut.newUserAlias()
	userRecord := userstore.Record{
		"external_alias": userAlias,
		"email":          ut.vg.newEmail(),
		"name":           ut.vg.newName(),
		ssnColumnName:    []string{ut.vg.newSSN()},
	}
	uid, err := ut.idp.CreateUser(ctx, userRecord)
	if err != nil {
		logErrorf(ctx, err, "Userstore: Thread %2d CreateUser failed for per thread user: %v", threadNum, err)
		return ucerr.Wrap(err)
	}

	aptName := ut.threadAccessPolicyTemplateName(threadNum)
	apt, err := ut.tc.CreateAccessPolicyTemplate(
		ctx,
		policy.AccessPolicyTemplate{
			Name:     aptName,
			Function: fmt.Sprintf(`function policy(context, params) { return context.client.api_key === params.api_key; } // %s`, aptName),
		},
		idp.IfNotExists(),
	)
	if err != nil {
		logErrorf(ctx, err, "Userstore: Thread %2d CreateAccessPolicyTemplate failed for per thread policy: %v", threadNum, err)
		return ucerr.Wrap(err)
	}

	ap, err := ut.tc.CreateAccessPolicy(
		ctx,
		policy.AccessPolicy{
			Name: ut.threadAccessPolicyName(threadNum),
			Components: []policy.AccessPolicyComponent{
				{
					Template:           &userstore.ResourceID{ID: apt.ID},
					TemplateParameters: `{"api_key": "security_api_key"}`,
				},
			},
			PolicyType: policy.PolicyTypeCompositeAnd,
		},
		idp.IfNotExists(),
	)
	if err != nil {
		logErrorf(ctx, err, "Userstore: Thread %2d CreateAccessPolicy failed for per thread policy: %v", threadNum, err)
		return ucerr.Wrap(err)
	}

	a, err := ut.idp.CreateAccessor(
		ctx,
		userstore.Accessor{
			Name:               ut.threadAccessorName(threadNum),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:      userstore.ResourceID{Name: ssnColumnName},
					Transformer: passthroughTransformer,
				},
			},
			AccessPolicy:   userstore.ResourceID{ID: ap.ID},
			SelectorConfig: userstore.UserSelectorConfig{WhereClause: "{external_alias} = ?"},
			Purposes: []userstore.ResourceID{
				{Name: "operational"},
			},
		},
		idp.IfNotExists(),
	)
	if err != nil {
		logErrorf(ctx, err, "Userstore: Thread %2d CreateAccessor for per thread accessor: %v", threadNum, err)
		return ucerr.Wrap(err)
	}

	ut.runPerThreadAccessors(ctx, threadNum, a, ssnColumnName, userRecord, userAlias)

	if err := ut.idp.DeleteUser(ctx, uid); err != nil {
		logErrorf(ctx, err, "Userstore: Thread %2d DeleteUser failed for per thread user: %v", threadNum, err)
	}

	if err := ut.idp.DeleteAccessor(ctx, a.ID); err != nil {
		logErrorf(ctx, err, "Userstore: Thread %2d DeleteAccessor failed for per thread accessor: %v", threadNum, err)
	}

	if err := ut.tc.DeleteAccessPolicy(ctx, ap.ID, 0); err != nil {
		logErrorf(ctx, err, "Userstore: Thread %2d DeleteAccessPolicy failed for per thread accessor: %v", threadNum, err)
	}

	if err := ut.tc.DeleteAccessPolicyTemplate(ctx, apt.ID, 0); err != nil {
		logErrorf(ctx, err, "Userstore: Thread %2d DeleteAccessPolicyTemplate failed for per thread accessor: %v", threadNum, err)
	}

	uclog.Debugf(ctx, "Userstore: Thread %2d  finished testing per thread in %v", threadNum, tmr.Elapsed())
	return nil
}

func (ut *userStoreTest) runPerThreadAccessors(
	ctx context.Context,
	threadNum int,
	accessor *userstore.Accessor,
	columnName string,
	user userstore.Record,
	userAlias string,
) {
	accessors, err := ut.idp.ListAccessors(ctx, false)
	if err != nil {
		logErrorf(ctx, err, "Userstore: Thread %2d ListAccessors failed from common pool: %v", threadNum, err)
		return
	}

	for _, a := range accessors.Data {
		if a.Name != accessor.Name {
			continue
		}

		if a.ID != accessor.ID {
			logErrorf(ctx, err, "Userstore: Thread %2d ListAccessors ID error expected %v got %v", threadNum, accessor.ID, a.ID)
		}

		retryCount := 0
		maxRetryCount := envTestCfg.Threads
		var fieldsStr *idp.ExecuteAccessorResponse
		for {
			// TODO DEVEXP - we should return an array not a single flat string
			// TODO DEVEXP - we should return better error when unrelated column deletion collides with accessor execution
			fieldsStr, err = ut.idp.ExecuteAccessor(
				ctx,
				a.ID,
				map[string]any{"api_key": "security_api_key"},
				[]any{userAlias},
			)
			if err == nil {
				uclog.Debugf(ctx, "[runPerThreadTests] Execute accessor %v [%v] success", a.Name, a.ID)
				break
			}
			if retryCount >= maxRetryCount {
				logErrorf(ctx, err, "Userstore: Thread %2d ExecuteAccessor failed for per thread accessor: %v", threadNum, err)
				break
			}
			retryCount++
		}

		// TODO DEVEXP is this really a map[string]any if so why doesn't SDK just return that
		fields := map[string]string{}
		if len(fieldsStr.Data) > 0 {
			if err = json.Unmarshal([]byte(fieldsStr.Data[0]), &fields); err != nil {
				logErrorf(ctx, err, "Userstore: Thread %2d ExecuteAccessor return value %s couldn't be parsed: %v", threadNum, fieldsStr.Data, err)
			}
		}

		if len(fields) != 1 {
			logErrorf(ctx, err, "Userstore: Thread %2d ExecuteAccessor returned unexpected field count: %d", threadNum, len(fields))
		}
		if len(fields) != 1 || ut.valueAsString(fields[columnName]) != ut.valueAsString(user[columnName]) {
			logErrorf(ctx, err, "Userstore: Thread %2d ExecuteAccessor returned unexpected values: %v expected %v", threadNum, fieldsStr, user[columnName])
		}

		// Make sure this call fails
		fieldsStr, err = ut.idp.ExecuteAccessor(
			ctx,
			a.ID,
			map[string]any{"purpose": "marketing"},
			[]any{userAlias},
		)
		if err == nil {
			uclog.Debugf(ctx, "[runPerThreadTests] Execute accessor %v [%v] success", a.Name, a.ID)
			break
		}
		if len(fieldsStr.Data) > 0 {
			logErrorf(ctx, err, "Userstore: Thread %2d ExecuteAccessor succeeded for Marketing for per thread accessor: %v", threadNum, fieldsStr)
		}
	}
}

func (ut *userStoreTest) runTestWorker(
	ctx context.Context,
	threadNum int,
	users []uuid.UUID,
	aliases []string,
	records []userstore.Record,
	numOps int,
) {
	for range numOps {
		// Execute a series of operations against a common pool of users, shared across all worker threads
		ut.runCommonPoolTests(ctx, threadNum, users, aliases, records)
		// Create accessors and do operations on objects specific to this thread
		if err := ut.runPerThreadTests(ctx, threadNum); err != nil {
			logErrorf(ctx, err, "Userstore: Thread %2d failed perThreadTest", threadNum)
		}
	}
}

func (ut *userStoreTest) setup(
	ctx context.Context,
) (users []uuid.UUID, aliases []string, records []userstore.Record, err error) {
	// create composite address data type

	dt, err := ut.idp.CreateDataType(
		ctx,
		userstore.ColumnDataType{
			Name:        ut.addressDataTypeName,
			Description: "test address",
			CompositeAttributes: userstore.CompositeAttributes{
				IncludeID: true,
				Fields: []userstore.CompositeField{
					{
						Name:     "Street",
						DataType: datatype.String,
					},
				},
			},
		},
		idp.IfNotExists(),
	)
	if err != nil {
		logErrorf(ctx, err, "Userstore: CreateDataType for '%s' failed: %v", ut.addressDataTypeName, err)
		return nil, nil, nil, ucerr.Wrap(err)
	}

	// create columns used for common pool

	if _, err := ut.idp.CreateColumn(
		ctx,
		userstore.Column{
			Table:     "users",
			Name:      "external_alias",
			DataType:  datatype.String,
			IsArray:   false,
			IndexType: userstore.ColumnIndexTypeUnique,
		},
		idp.IfNotExists(),
	); err != nil {
		logErrorf(ctx, err, "Userstore: CreateColumn for 'external_alias' failed: %v", err)
		return nil, nil, nil, ucerr.Wrap(err)
	}

	if _, err = ut.idp.CreateColumn(
		ctx,
		userstore.Column{
			Table:     "users",
			Name:      ut.phoneNumberColumnName,
			DataType:  datatype.PhoneNumber,
			IndexType: userstore.ColumnIndexTypeIndexed,
		},
		idp.IfNotExists(),
	); err != nil {
		logErrorf(ctx, err, "Userstore: CreateColumn for '%s' failed: %v", ut.phoneNumberColumnName, err)
		return nil, nil, nil, ucerr.Wrap(err)
	}

	if _, err = ut.idp.CreateColumn(
		ctx,
		userstore.Column{
			Table:     "users",
			Name:      ut.addressColumnName,
			DataType:  userstore.ResourceID{ID: dt.ID},
			IndexType: userstore.ColumnIndexTypeIndexed,
		},
		idp.IfNotExists(),
	); err != nil {
		logErrorf(ctx, err, "Userstore: CreateColumn for '%s' failed: %v", ut.addressColumnName, err)
		return nil, nil, nil, ucerr.Wrap(err)
	}

	// create per thread columns
	// TODO: doing this in main thread because doing so in worker threads causes idp and accessor failures - verify this is still true?

	for i := range envTestCfg.Threads {
		threadColName := ut.threadSSNColumnName(i)
		if _, err = ut.idp.CreateColumn(
			ctx,
			userstore.Column{
				Table:     "users",
				Name:      threadColName,
				DataType:  datatype.SSN,
				IsArray:   true,
				IndexType: userstore.ColumnIndexTypeIndexed,
				Constraints: userstore.ColumnConstraints{
					PartialUpdates: true,
					UniqueRequired: true,
				},
			},
			idp.IfNotExists(),
		); err != nil {
			logErrorf(ctx, err, "Userstore:  CreateColumn for per thread column '%s' failed: %v", threadColName, err)
			return nil, nil, nil, ucerr.Wrap(err)
		}
	}

	// Create transformers that we will use

	tf, err := ut.tc.CreateTransformer(
		ctx,
		policy.Transformer{
			Name:           ut.phoneNumberTransformerName,
			TransformType:  policy.TransformTypeTransform,
			InputDataType:  datatype.PhoneNumber,
			OutputDataType: datatype.PhoneNumber,
			Function:       fmt.Sprintf(`function transform(data, params) { return data; } // %s`, ut.phoneNumberTransformerName),
		},
		idp.IfNotExists(),
	)
	if err != nil {
		logErrorf(ctx, err, "Userstore: CreateTransformer for '%s' failed: %v", ut.phoneNumberTransformerName, err)
		return nil, nil, nil, ucerr.Wrap(err)
	}

	// Create accessors that we will use

	if _, err = ut.idp.CreateAccessor(
		ctx,
		userstore.Accessor{
			Name:               ut.phoneNumberAddressAccessorName,
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:      userstore.ResourceID{Name: ut.phoneNumberColumnName},
					Transformer: userstore.ResourceID{ID: tf.ID},
				},
				{
					Column:      userstore.ResourceID{Name: ut.addressColumnName},
					Transformer: passthroughTransformer,
				},
			},
			AccessPolicy:   userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			SelectorConfig: userstore.UserSelectorConfig{WhereClause: "{external_alias} = ?"},
			Purposes: []userstore.ResourceID{
				{Name: "operational"},
			},
		},
		idp.IfNotExists(),
	); err != nil {
		logErrorf(ctx, err, "Userstore: CreateAccessor for '%s' failed: %v", ut.phoneNumberAddressAccessorName, err)
		return nil, nil, nil, ucerr.Wrap(err)
	}

	_, err = ut.idp.CreateAccessor(
		ctx,
		userstore.Accessor{
			Name:               ut.listUsersAccessorName,
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			AccessPolicy:       userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:      userstore.ResourceID{Name: "id"},
					Transformer: passthroughTransformer,
				},
			},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: fmt.Sprintf("{external_alias} LIKE '%s'", ut.entityPrefix+`%`),
			},
			Purposes: []userstore.ResourceID{
				{Name: "operational"},
			},
		},
		idp.IfNotExists(),
	)
	if err != nil {
		logErrorf(ctx, err, "Userstore: CreateAccessor for '%s' failed: %v", ut.listUsersAccessorName, err)
		return nil, nil, nil, ucerr.Wrap(err)
	}

	userRegions, err := ut.idp.ListUserRegions(ctx)
	if err != nil {
		logErrorf(ctx, err, "Userstore: ListUserRegions failed: %v", err)
		return nil, nil, nil, ucerr.Wrap(err)
	}

	// Create common set of users to be resolved across multiple worker threads

	users = make([]uuid.UUID, poolSizeUserStore)
	aliases = make([]string, poolSizeUserStore)
	records = make([]userstore.Record, poolSizeUserStore)
	for i := range users {
		aliases[i] = ut.newUserAlias()
		records[i] = userstore.Record{
			"external_alias":         aliases[i],
			"email":                  ut.vg.newEmail(),
			"name":                   ut.vg.newName(),
			ut.phoneNumberColumnName: ut.vg.newPhoneNumber(),
			ut.addressColumnName:     ut.vg.newAddress(),
		}

		reg := userRegions[rand.Intn(len(userRegions))]
		users[i], err = ut.idp.CreateUser(ctx, records[i], idp.DataRegion(reg))
		if err != nil {
			logErrorf(ctx, err, "Userstore: CreateUser failed for common pool: %v", err)
			return nil, nil, nil, ucerr.Wrap(err)
		}
	}

	return users, aliases, records, nil
}

func (ut userStoreTest) threadAccessorName(threadNum int) string {
	return fmt.Sprintf("%s%s_%d", ut.entityPrefix, "ThreadAccessor", threadNum)
}

func (ut userStoreTest) threadAccessPolicyName(threadNum int) string {
	return fmt.Sprintf("%s%s_%d", ut.entityPrefix, "ThreadAccessPolicy", threadNum)
}

func (ut userStoreTest) threadAccessPolicyTemplateName(threadNum int) string {
	return fmt.Sprintf("%s%s_%d", ut.entityPrefix, "ThreadAccessPolicyTemplate", threadNum)
}

func (ut userStoreTest) threadSSNColumnName(threadNum int) string {
	return fmt.Sprintf("%s%s_%d", ut.entityPrefix, "SSNs", threadNum)
}

func (userStoreTest) valueAsString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}

	bytes, err := json.Marshal(v)
	if err != nil {
		return ""
	}

	return string(bytes)
}

func userstoreTest(ctx context.Context, tenantURL string, tokenSource jsonclient.Option, iterations int) {
	tmr := timer.Start()
	for i := 1; i < iterations+1; i++ {
		uclog.Infof(ctx, "Userstore: Starting an environment test for %d worker threads - %v (%d/%d)", envTestCfg.Threads, tenantURL, i, iterations)
		ut, err := newUserStoreTest(ctx, envTestCfg.Threads, tenantURL, tokenSource)
		if err != nil {
			logErrorf(ctx, err, "Userstore: IDP client creation failed: %v", err)
			return
		}

		users, aliases, records, err := ut.setup(ctx)
		if err != nil {
			logErrorf(ctx, err, "Userstore: Test setup failed: %v", err)
			return
		}

		wg := sync.WaitGroup{}
		for i := range envTestCfg.Threads {
			wg.Add(1)
			threadNum := i
			async.Execute(func() {
				ut.runTestWorker(ctx, threadNum, users, aliases, records, defaultThreadOpCount)
				wg.Done()
			})
		}
		wg.Wait()
		ut.cleanup(ctx)
		uclog.Infof(ctx, "Userstore: Completed environment test for %d worker threads in %v (%d/%d)", envTestCfg.Threads, tmr.Reset(), i, iterations)
	}
}

type valueGenerator struct {
	emailIDs     []string
	emailDomains []string
	firstNames   []string
	lastNames    []string
	streetNames  []string
	streetTypes  []string
}

func newValueGenerator() *valueGenerator {
	return &valueGenerator{
		emailIDs:     []string{"Bob", "John", "Larry", "Greg"},
		emailDomains: []string{"gmail.com", "hotmail.com", "ymail.com", "apple.com"},
		firstNames:   []string{"Bob", "John", "Larry", "Greg"},
		lastNames:    []string{"Smith", "Johnson", "Wild", "Rogers"},
		streetNames:  []string{"Creek", "Lake", "Bay", "Shore"},
		streetTypes:  []string{"Street", "Way", "Lane", "Avenue"},
	}
}

func (vg valueGenerator) newAddress() userstore.CompositeValue {
	return userstore.CompositeValue{
		"street": fmt.Sprintf(
			"%d %s %s",
			rand.Intn(10000),
			vg.streetNames[rand.Intn(len(vg.streetNames))],
			vg.streetTypes[rand.Intn(len(vg.streetTypes))],
		),
		"id": fmt.Sprintf("id%d", rand.Intn(100000)),
	}
}

func (vg valueGenerator) newEmail() string {
	return fmt.Sprintf(
		"%s@%s",
		vg.emailIDs[rand.Intn(len(vg.emailIDs))],
		vg.emailDomains[rand.Intn(len(vg.emailDomains))],
	)
}

func (vg valueGenerator) newName() string {
	return fmt.Sprintf(
		"%s %s",
		vg.firstNames[rand.Intn(len(vg.firstNames))],
		vg.lastNames[rand.Intn(len(vg.lastNames))],
	)
}

func (valueGenerator) newPhoneNumber() string {
	rD := make([]int, 10)
	for i := range rD {
		rD[i] = rand.Intn(10)
	}
	return fmt.Sprintf("%d%d%d-%d%d%d-%d%d%d%d", rD[0], rD[1], rD[2], rD[3], rD[4], rD[5], rD[6], rD[7], rD[8], rD[9])
}

func (valueGenerator) newSSN() string {
	rD := make([]int, 9)
	for i := range rD {
		rD[i] = rand.Intn(10)
	}
	return fmt.Sprintf("%d%d%d-%d%d-%d%d%d%d", rD[0], rD[1], rD[2], rD[3], rD[4], rD[5], rD[6], rD[7], rD[8])
}
