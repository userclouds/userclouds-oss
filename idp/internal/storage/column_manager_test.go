package storage_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/internal/testhelpers"
)

func TestColumnManager(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cc, lc, ccs := testhelpers.NewTestStorage(t)
	_, ct, cdb := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, ccs, cc, lc)

	s := idptesthelpers.NewStorage(ctx, t, cdb, ct.ID)
	cm, err := storage.NewUserstoreColumnManager(ctx, s)
	assert.NoErr(t, err)

	// try to create a new column w/o access policy
	cid := uuid.Must(uuid.NewV4())
	code, err := cm.CreateColumnFromClient(ctx, &userstore.Column{
		ID:                 cid,
		Name:               "first",
		DataType:           datatype.String,
		IndexType:          userstore.ColumnIndexTypeNone,
		DefaultTransformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID, Name: policy.TransformerPassthrough.Name},
	})
	assert.NotNil(t, err)
	assert.Equal(t, code, http.StatusBadRequest)

	// successfully create a column
	code, err = cm.CreateColumnFromClient(ctx, &userstore.Column{
		ID:                 cid,
		Name:               "first",
		DataType:           datatype.String,
		IndexType:          userstore.ColumnIndexTypeNone,
		AccessPolicy:       userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID, Name: policy.AccessPolicyAllowAll.Name},
		DefaultTransformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID, Name: policy.TransformerPassthrough.Name},
	})
	assert.NoErr(t, err)
	assert.Equal(t, code, http.StatusOK)

	// create an accessor that uses the column in the selector
	mm := storage.NewMethodManager(ctx, s)
	aid := uuid.Must(uuid.NewV4())
	code, err = mm.CreateAccessorFromClient(ctx, &userstore.Accessor{
		ID:   aid,
		Name: "accessor",
		Columns: []userstore.ColumnOutputConfig{{
			Column: userstore.ResourceID{ID: cid},
		}},
		AccessPolicy:   userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
		SelectorConfig: userstore.UserSelectorConfig{WhereClause: "{first} = (?)"},
	})
	assert.NoErr(t, err)
	assert.Equal(t, code, http.StatusCreated)

	// create a mutator that uses the column in the selector
	mid := uuid.Must(uuid.NewV4())
	code, err = mm.CreateMutatorFromClient(ctx, &userstore.Mutator{
		ID:   mid,
		Name: "mutator",
		Columns: []userstore.ColumnInputConfig{{
			Column:     userstore.ResourceID{ID: cid},
			Normalizer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
		}},
		AccessPolicy:   userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
		SelectorConfig: userstore.UserSelectorConfig{WhereClause: "{first} = (?)"},
	})
	assert.NoErr(t, err)
	assert.Equal(t, code, http.StatusCreated)

	// update the column name
	code, err = cm.UpdateColumnFromClient(ctx, &userstore.Column{
		ID:                 cid,
		Name:               "second",
		DataType:           datatype.String,
		IndexType:          userstore.ColumnIndexTypeNone,
		AccessPolicy:       userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID, Name: policy.AccessPolicyAllowAll.Name},
		DefaultTransformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID, Name: policy.TransformerPassthrough.Name},
	})
	assert.NoErr(t, err)
	assert.Equal(t, code, http.StatusOK)

	// try updating the default transformer of the column to one that tokenizes
	code, err = cm.UpdateColumnFromClient(ctx, &userstore.Column{
		ID:                 cid,
		Name:               "second",
		DataType:           datatype.String,
		IndexType:          userstore.ColumnIndexTypeNone,
		AccessPolicy:       userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID, Name: policy.AccessPolicyAllowAll.Name},
		DefaultTransformer: userstore.ResourceID{ID: policy.TransformerUUID.ID, Name: policy.TransformerUUID.Name},
	})
	assert.NotNil(t, err)
	assert.Equal(t, code, http.StatusBadRequest) // should fail w/o default token access policy

	// try again
	code, err = cm.UpdateColumnFromClient(ctx, &userstore.Column{
		ID:                       cid,
		Name:                     "second",
		DataType:                 datatype.String,
		IndexType:                userstore.ColumnIndexTypeNone,
		AccessPolicy:             userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID, Name: policy.AccessPolicyAllowAll.Name},
		DefaultTransformer:       userstore.ResourceID{ID: policy.TransformerUUID.ID, Name: policy.TransformerUUID.Name},
		DefaultTokenAccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
	})
	assert.NoErr(t, err)
	assert.Equal(t, code, http.StatusOK)

	// load the accessor and make sure the column name is updated in the selector
	a, err := s.GetLatestAccessor(ctx, aid)
	assert.NoErr(t, err)
	assert.Equal(t, a.SelectorConfig.WhereClause, "{second} = (?)")
	assert.Equal(t, a.Version, 1)

	// load the mutator and make sure the column name is updated in the selector
	m, err := s.GetLatestMutator(ctx, mid)
	assert.NoErr(t, err)
	assert.Equal(t, m.SelectorConfig.WhereClause, "{second} = (?)")
	assert.Equal(t, m.Version, 1)

}
