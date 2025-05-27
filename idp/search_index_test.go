package idp_test

import (
	"fmt"
	"testing"
	"time"

	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/idp/userstore/search"
	"userclouds.com/infra/assert"
)

func TestSearchManagement(t *testing.T) {
	t.Parallel()

	tf := idptesthelpers.NewTestFixture(t)

	//// create columns

	email1Column, err := tf.IDPClient.CreateColumn(
		tf.Ctx,
		userstore.Column{
			Table:         "users",
			Name:          uniqueName("column"),
			DataType:      datatype.Email,
			IsArray:       false,
			IndexType:     userstore.ColumnIndexTypeIndexed,
			SearchIndexed: true,
		},
	)
	assert.NoErr(tf.T, err)

	email2Column, err := tf.IDPClient.CreateColumn(
		tf.Ctx,
		userstore.Column{
			Table:         "users",
			Name:          uniqueName("column"),
			DataType:      datatype.Email,
			IsArray:       false,
			IndexType:     userstore.ColumnIndexTypeIndexed,
			SearchIndexed: true,
		},
	)
	assert.NoErr(tf.T, err)

	email3Column, err := tf.IDPClient.CreateColumn(
		tf.Ctx,
		userstore.Column{
			Table:         "users",
			Name:          uniqueName("column"),
			DataType:      datatype.Email,
			IsArray:       false,
			IndexType:     userstore.ColumnIndexTypeIndexed,
			SearchIndexed: false,
		},
	)
	assert.NoErr(tf.T, err)

	email3Column.SearchIndexed = true
	email3Column, err = tf.IDPClient.UpdateColumn(tf.Ctx, email3Column.ID, *email3Column)
	assert.NoErr(tf.T, err)

	email3Column.SearchIndexed = false
	email3Column, err = tf.IDPClient.UpdateColumn(tf.Ctx, email3Column.ID, *email3Column)
	assert.NoErr(tf.T, err)

	intColumn, err := tf.IDPClient.CreateColumn(
		tf.Ctx,
		userstore.Column{
			Table:         "users",
			Name:          uniqueName("column"),
			DataType:      datatype.Integer,
			IsArray:       false,
			IndexType:     userstore.ColumnIndexTypeIndexed,
			SearchIndexed: false,
		},
	)
	assert.NoErr(tf.T, err)

	// fail changing SearchIndexed for a column with non-searchable data type

	badCol := *intColumn
	badCol.SearchIndexed = true
	_, err = tf.IDPClient.UpdateColumn(tf.Ctx, badCol.ID, badCol)
	assert.NotNil(tf.T, err)

	//// create accessors

	email1Accessor, err := tf.IDPClient.CreateAccessor(
		tf.Ctx,
		userstore.Accessor{
			Name:               uniqueName("accessor"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:            userstore.ResourceID{Name: "name"},
					Transformer:       userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
					TokenAccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
				},
			},
			AccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: fmt.Sprintf("{%s} ILIKE ?", email1Column.Name),
			},
			Purposes: []userstore.ResourceID{{Name: "operational"}},
		},
	)
	assert.NoErr(tf.T, err)

	email2Accessor, err := tf.IDPClient.CreateAccessor(
		tf.Ctx,
		userstore.Accessor{
			Name:               uniqueName("accessor"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:            userstore.ResourceID{Name: "name"},
					Transformer:       userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
					TokenAccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
				},
			},
			AccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: fmt.Sprintf("{%s} ILIKE ?", email2Column.Name),
			},
			Purposes: []userstore.ResourceID{{Name: "operational"}},
		},
	)
	assert.NoErr(tf.T, err)

	// an accessor can be associated with a column with a searchable data type that
	// is not marked searchable

	_, err = tf.IDPClient.CreateAccessor(
		tf.Ctx,
		userstore.Accessor{
			Name:               uniqueName("accessor"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:            userstore.ResourceID{Name: "name"},
					Transformer:       userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
					TokenAccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
				},
			},
			AccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: fmt.Sprintf("{%s} ILIKE ?", email3Column.Name),
			},
			Purposes: []userstore.ResourceID{{Name: "operational"}},
		},
	)
	assert.NoErr(tf.T, err)

	unsearchableAccessor, err := tf.IDPClient.CreateAccessor(
		tf.Ctx,
		userstore.Accessor{
			Name:               uniqueName("accessor"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:            userstore.ResourceID{Name: "name"},
					Transformer:       userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
					TokenAccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
				},
			},
			AccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: "{id} = ?",
			},
			Purposes: []userstore.ResourceID{{Name: "operational"}},
		},
	)
	assert.NoErr(tf.T, err)

	//// create user search indices

	// fail - missing name

	_, err = tf.IDPClient.CreateUserSearchIndex(
		tf.Ctx,
		search.UserSearchIndex{
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Type:               search.IndexTypeNgram,
			Settings:           search.NewNgramIndexSettings(3, 10),
			Columns:            []userstore.ResourceID{{ID: email1Column.ID}},
		},
	)
	assert.NotNil(tf.T, err)

	// fail - bad type

	_, err = tf.IDPClient.CreateUserSearchIndex(
		tf.Ctx,
		search.UserSearchIndex{
			Name:               uniqueName("search_index"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Type:               "foo",
			Settings:           search.NewNgramIndexSettings(3, 10),
			Columns:            []userstore.ResourceID{{ID: email1Column.ID}},
		},
	)
	assert.NotNil(tf.T, err)

	// fail - unsupported type

	_, err = tf.IDPClient.CreateUserSearchIndex(
		tf.Ctx,
		search.UserSearchIndex{
			Name:               uniqueName("search_index"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Type:               search.IndexTypeDeprecated,
			Settings:           search.IndexSettings{},
			Columns:            []userstore.ResourceID{{ID: email1Column.ID}},
		},
	)
	assert.NotNil(tf.T, err)

	// fail - column id with non-searchable data type

	_, err = tf.IDPClient.CreateUserSearchIndex(
		tf.Ctx,
		search.UserSearchIndex{
			Name:               uniqueName("search_index"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Type:               search.IndexTypeNgram,
			Settings:           search.NewNgramIndexSettings(3, 10),
			Columns:            []userstore.ResourceID{{ID: intColumn.ID}},
		},
	)
	assert.NotNil(tf.T, err)

	// fail - bad data life cycle state

	_, err = tf.IDPClient.CreateUserSearchIndex(
		tf.Ctx,
		search.UserSearchIndex{
			Name:               uniqueName("search_index"),
			DataLifeCycleState: userstore.DataLifeCycleStateSoftDeleted,
			Type:               search.IndexTypeNgram,
			Settings:           search.NewNgramIndexSettings(3, 10),
			Columns:            []userstore.ResourceID{{ID: email1Column.ID}},
		},
	)
	assert.NotNil(tf.T, err)

	// fail - cannot create in enabled state

	_, err = tf.IDPClient.CreateUserSearchIndex(
		tf.Ctx,
		search.UserSearchIndex{
			Name:               uniqueName("search_index"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Type:               search.IndexTypeNgram,
			Settings:           search.NewNgramIndexSettings(3, 10),
			Columns:            []userstore.ResourceID{{ID: email1Column.ID}},
			Enabled:            time.Now().UTC(),
		},
	)
	assert.NotNil(tf.T, err)

	// fail - cannot create in bootstrapped state

	_, err = tf.IDPClient.CreateUserSearchIndex(
		tf.Ctx,
		search.UserSearchIndex{
			Name:               uniqueName("search_index"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Type:               search.IndexTypeNgram,
			Settings:           search.NewNgramIndexSettings(3, 10),
			Columns:            []userstore.ResourceID{{ID: email1Column.ID}},
			Enabled:            time.Now().UTC(),
			Bootstrapped:       time.Now().UTC(),
		},
	)
	assert.NotNil(tf.T, err)

	// fail - cannot create in searchable state

	_, err = tf.IDPClient.CreateUserSearchIndex(
		tf.Ctx,
		search.UserSearchIndex{
			Name:               uniqueName("search_index"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Type:               search.IndexTypeNgram,
			Settings:           search.NewNgramIndexSettings(3, 10),
			Columns:            []userstore.ResourceID{{ID: email1Column.ID}},
			Enabled:            time.Now().UTC(),
			Bootstrapped:       time.Now().UTC(),
			Searchable:         time.Now().UTC(),
		},
	)
	assert.NotNil(tf.T, err)

	// create without any column IDs

	noColIndex, err := tf.IDPClient.CreateUserSearchIndex(
		tf.Ctx,
		search.UserSearchIndex{
			Name:               uniqueName("search_index"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Type:               search.IndexTypeNgram,
			Settings:           search.NewNgramIndexSettings(4, 10),
		},
	)
	assert.NoErr(tf.T, err)

	// create index for email2 column

	email2Index, err := tf.IDPClient.CreateUserSearchIndex(
		tf.Ctx,
		search.UserSearchIndex{
			Name:               uniqueName("search_index"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Type:               search.IndexTypeNgram,
			Settings:           search.NewNgramIndexSettings(3, 10),
			Columns:            []userstore.ResourceID{{ID: email2Column.ID}},
		},
	)
	assert.NoErr(tf.T, err)

	// delete and then recreate the index

	assert.NoErr(tf.T, tf.IDPClient.DeleteUserSearchIndex(tf.Ctx, email2Index.ID))

	email2Index, err = tf.IDPClient.CreateUserSearchIndex(
		tf.Ctx,
		search.UserSearchIndex{
			Name:               uniqueName("search_index"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Type:               search.IndexTypeNgram,
			Settings:           search.NewNgramIndexSettings(3, 10),
			Columns:            []userstore.ResourceID{{ID: email2Column.ID}},
		},
	)
	assert.NoErr(tf.T, err)

	// create index for email3 column

	email3Index, err := tf.IDPClient.CreateUserSearchIndex(
		tf.Ctx,
		search.UserSearchIndex{
			Name:               uniqueName("search_index"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Type:               search.IndexTypeNgram,
			Settings:           search.NewNgramIndexSettings(3, 10),
			Columns:            []userstore.ResourceID{{ID: email3Column.ID}},
		},
	)
	assert.NoErr(tf.T, err)

	// fail - create with conflicting name

	_, err = tf.IDPClient.CreateUserSearchIndex(
		tf.Ctx,
		search.UserSearchIndex{
			Name:               noColIndex.Name,
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Type:               search.IndexTypeNgram,
			Settings:           search.NewNgramIndexSettings(3, 10),
			Columns:            []userstore.ResourceID{{ID: email1Column.ID}},
		},
	)
	assert.NotNil(tf.T, err)

	//// update user search indices

	// successful update of unenabled index

	noColIndex.Name = uniqueName("search_index")
	noColIndex.Description = "new description"
	noColIndex.Settings = search.NewNgramIndexSettings(3, 10)
	noColIndex.Columns = []userstore.ResourceID{{ID: email1Column.ID}}

	email1Index, err := tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		noColIndex.ID,
		*noColIndex,
	)
	assert.NoErr(tf.T, err)

	// fail making unenabled index bootstrapped

	badIndex := *email1Index
	badIndex.Bootstrapped = time.Now().UTC()
	_, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		badIndex.ID,
		badIndex,
	)
	assert.NotNil(tf.T, err)

	// mark email1 index enabled

	email1Index.Enabled = time.Now().UTC()
	email1Index, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		email1Index.ID,
		*email1Index,
	)
	assert.NoErr(tf.T, err)

	// fail deleting enabled index

	assert.NotNil(tf.T, tf.IDPClient.DeleteUserSearchIndex(tf.Ctx, email1Index.ID))

	// fail changing name of enabled search index

	badIndex = *email1Index
	badIndex.Name = uniqueName("search_index")
	_, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		badIndex.ID,
		badIndex,
	)
	assert.NotNil(tf.T, err)

	// fail changing type settings of enabled search index

	badIndex = *email1Index
	badIndex.Settings = search.NewNgramIndexSettings(4, 10)
	_, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		badIndex.ID,
		badIndex,
	)
	assert.NotNil(tf.T, err)

	// fail changing column ids for enabled search index

	badIndex = *email1Index
	badIndex.Columns = []userstore.ResourceID{{ID: email2Column.ID}}
	_, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		badIndex.ID,
		badIndex,
	)
	assert.NotNil(tf.T, err)

	// fail changing enabled date for enabled search index

	badIndex = *email1Index
	badIndex.Enabled = time.Now().UTC()
	_, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		badIndex.ID,
		badIndex,
	)
	assert.NotNil(tf.T, err)

	// fail turning off SearchIndexed for column part of enabled search index

	badCol = *email1Column
	badCol.SearchIndexed = false
	_, err = tf.IDPClient.UpdateColumn(tf.Ctx, badCol.ID, badCol)
	assert.NotNil(tf.T, err)

	// fail making non-bootstrapped index searchable

	badIndex = *email1Index
	badIndex.Searchable = time.Now().UTC()
	_, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		badIndex.ID,
		badIndex,
	)
	assert.NotNil(tf.T, err)

	// mark email1 index bootstrapped

	email1Index.Bootstrapped = time.Now().UTC()
	email1Index, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		email1Index.ID,
		*email1Index,
	)
	assert.NoErr(tf.T, err)

	// fail changing bootstrap time

	badIndex = *email1Index
	badIndex.Bootstrapped = time.Now().UTC()
	_, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		badIndex.ID,
		badIndex,
	)
	assert.NotNil(tf.T, err)

	// mark email1 index searchable

	email1Index.Searchable = time.Now().UTC()
	email1Index, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		email1Index.ID,
		*email1Index,
	)
	assert.NoErr(tf.T, err)

	// fail changing searchable time

	badIndex = *email1Index
	badIndex.Searchable = time.Now().UTC()
	_, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		badIndex.ID,
		badIndex,
	)
	assert.NotNil(tf.T, err)

	//// accessor index management

	// fail with invalid accessor id

	assert.NotNil(
		tf.T,
		tf.IDPClient.SetAccessorUserSearchIndex(
			tf.Ctx,
			email1Index.ID,
			email1Index.ID,
			search.QueryTypeTerm),
	)

	// fail with unsearchable accessor id

	assert.NotNil(
		tf.T,
		tf.IDPClient.SetAccessorUserSearchIndex(
			tf.Ctx,
			unsearchableAccessor.ID,
			email1Index.ID,
			search.QueryTypeTerm,
		),
	)

	// fail with incompatible index

	assert.NotNil(
		tf.T,
		tf.IDPClient.SetAccessorUserSearchIndex(
			tf.Ctx,
			email2Accessor.ID,
			email1Index.ID,
			search.QueryTypeTerm,
		),
	)

	// fail with unsearchable index

	assert.NotNil(
		tf.T,
		tf.IDPClient.SetAccessorUserSearchIndex(
			tf.Ctx,
			email2Accessor.ID,
			email2Index.ID,
			search.QueryTypeTerm,
		),
	)

	// make email2 index searchable

	email2Index.Enabled = time.Now().UTC()
	email2Index.Bootstrapped = time.Now().UTC()
	email2Index.Searchable = time.Now().UTC()
	email2Index, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		email2Index.ID,
		*email2Index,
	)
	assert.NoErr(tf.T, err)

	// incompatible query type

	assert.NotNil(
		tf.T,
		tf.IDPClient.SetAccessorUserSearchIndex(
			tf.Ctx,
			email2Accessor.ID,
			email2Index.ID,
			"foo",
		),
	)

	// fail making accessor not associated with index searchable

	badAccessor := *email1Accessor
	badAccessor.UseSearchIndex = true
	_, err = tf.IDPClient.UpdateAccessor(
		tf.Ctx,
		badAccessor.ID,
		badAccessor,
	)
	assert.NotNil(tf.T, err)

	// use email1 index for email1 accessor

	assert.NoErr(
		tf.T,
		tf.IDPClient.SetAccessorUserSearchIndex(
			tf.Ctx,
			email1Accessor.ID,
			email1Index.ID,
			search.QueryTypeTerm,
		),
	)

	// make email1 accessor searchable

	email1Accessor.UseSearchIndex = true
	email1Accessor, err = tf.IDPClient.UpdateAccessor(
		tf.Ctx,
		email1Accessor.ID,
		*email1Accessor,
	)
	assert.NoErr(tf.T, err)

	// fail making email1 index unsearchable

	badIndex = *email1Index
	badIndex.Searchable = time.Time{}
	_, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		badIndex.ID,
		badIndex,
	)
	assert.NotNil(tf.T, err)

	// fail removing email1 accessor index pairing

	assert.NotNil(
		tf.T,
		tf.IDPClient.RemoveAccessorUserSearchIndex(
			tf.Ctx,
			email1Accessor.ID,
		),
	)

	// fail enabling email3 index referencing unsearchable column

	badIndex = *email3Index
	badIndex.Enabled = time.Now().UTC()
	_, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		badIndex.ID,
		badIndex,
	)
	assert.NotNil(tf.T, err)

	// change email3 index to index email1 and enable

	email3Index.Columns = []userstore.ResourceID{{ID: email1Column.ID}}
	email3Index.Enabled = time.Now().UTC()
	email3Index.Bootstrapped = time.Now().UTC()
	email3Index.Searchable = time.Now().UTC()
	email1AlternateIndex, err := tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		email3Index.ID,
		*email3Index,
	)
	assert.NoErr(tf.T, err)

	// change email1 acccessor to use email1 alternate index

	assert.NoErr(
		tf.T,
		tf.IDPClient.SetAccessorUserSearchIndex(
			tf.Ctx,
			email1Accessor.ID,
			email1AlternateIndex.ID,
			search.QueryTypeTerm,
		),
	)

	// disable and delete email1 index since it is now unused

	email1Index.Disable()
	email1Index, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		email1Index.ID,
		*email1Index,
	)
	assert.NoErr(tf.T, err)

	assert.NoErr(tf.T, tf.IDPClient.DeleteUserSearchIndex(tf.Ctx, email1Index.ID))

	// make email1 accessor unsearchable, then remove accessor
	// index pairing and disable and delete email1 alternate index

	email1Accessor.UseSearchIndex = false
	email1Accessor, err = tf.IDPClient.UpdateAccessor(
		tf.Ctx,
		email1Accessor.ID,
		*email1Accessor,
	)
	assert.NoErr(tf.T, err)

	assert.NoErr(
		tf.T,
		tf.IDPClient.RemoveAccessorUserSearchIndex(
			tf.Ctx,
			email1Accessor.ID,
		),
	)

	email1AlternateIndex.Disable()
	email1AlternateIndex, err = tf.IDPClient.UpdateUserSearchIndex(
		tf.Ctx,
		email1AlternateIndex.ID,
		*email1AlternateIndex,
	)
	assert.NoErr(tf.T, err)

	assert.NoErr(tf.T, tf.IDPClient.DeleteUserSearchIndex(tf.Ctx, email1AlternateIndex.ID))
}
