package authz_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	. "userclouds.com/authz"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/uctest"
)

func newCachedClientWithAuth(t *testing.T, url string) *Client {
	jwt := uctest.CreateJWT(t, oidc.UCTokenClaims{}, url)
	c, err := NewCustomClient(DefaultObjTypeTTL, DefaultEdgeTypeTTL, DefaultObjTTL, DefaultEdgeTTL, url, JSONClient(jsonclient.HeaderAuthBearer(jwt)))
	assert.NoErr(t, err)
	return c
}

func TestCache(t *testing.T) {
	ctx := context.Background()
	// make sure that when test ran in parallel, it doesn't interfere with other tests
	randSuffix := uuid.Must(uuid.NewV4()).String()
	ot1 := &ObjectType{
		BaseModel: ucdb.NewBase(),
		TypeName:  "objtype1" + randSuffix,
	}
	ot2 := &ObjectType{
		BaseModel: ucdb.NewBase(),
		TypeName:  "objtype2" + randSuffix,
	}
	ot3 := &ObjectType{
		BaseModel: ucdb.NewBase(),
		TypeName:  "objtype3" + randSuffix,
	}

	et1 := &EdgeType{
		BaseModel:          ucdb.NewBase(),
		TypeName:           "edgetype1" + randSuffix,
		SourceObjectTypeID: ot1.ID,
		TargetObjectTypeID: ot2.ID,
	}
	et2 := &EdgeType{
		BaseModel:          ucdb.NewBase(),
		TypeName:           "edgetype2" + randSuffix,
		SourceObjectTypeID: ot2.ID,
		TargetObjectTypeID: ot1.ID,
	}
	et3 := &EdgeType{
		BaseModel:          ucdb.NewBase(),
		TypeName:           "edgetype3" + randSuffix,
		SourceObjectTypeID: ot1.ID,
		TargetObjectTypeID: ot2.ID,
	}

	alias1 := "obj1" + randSuffix
	obj1 := &Object{
		BaseModel: ucdb.NewBase(),
		Alias:     &alias1,
		TypeID:    ot1.ID,
	}
	alias2 := "obj2" + randSuffix
	obj2 := &Object{
		BaseModel: ucdb.NewBase(),
		Alias:     &alias2,
		TypeID:    ot2.ID,
	}
	alias3 := "obj3" + randSuffix
	obj3 := &Object{
		BaseModel: ucdb.NewBase(),
		Alias:     &alias3,
		TypeID:    ot3.ID,
	}

	e1 := &Edge{
		BaseModel:      ucdb.NewBase(),
		EdgeTypeID:     uuid.Must(uuid.NewV4()),
		SourceObjectID: obj1.ID,
		TargetObjectID: obj2.ID,
	}
	e2 := &Edge{
		BaseModel:      ucdb.NewBase(),
		EdgeTypeID:     uuid.Must(uuid.NewV4()),
		SourceObjectID: obj2.ID,
		TargetObjectID: obj1.ID,
	}

	org1 := &Organization{
		BaseModel: ucdb.NewBase(),
		Name:      "org1" + uuid.Must(uuid.NewV4()).String(),
	}

	org2 := &Organization{
		BaseModel: ucdb.NewBase(),
		Name:      "org2" + uuid.Must(uuid.NewV4()).String(),
	}

	org3 := &Organization{
		BaseModel: ucdb.NewBase(),
		Name:      "org3" + uuid.Must(uuid.NewV4()).String(),
		Region:    "aws-us-west-2",
	}
	tnInvalid := "typeInvalid"

	t.Run("TestObjectTypeCache", func(t *testing.T) {
		// This can't run in parallel as it depends on other tests not flushing the cache  t.Parallel()

		var calls int
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.Method == http.MethodPost {
				// Read the body to figure out which object was posted
				b, err := io.ReadAll(r.Body)
				if err != nil {
					jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
					return
				}
				br := bytes.NewReader(b)
				dec := json.NewDecoder(br)

				var req CreateObjectTypeRequest
				assert.NoErr(t, dec.Decode(&req))

				if req.ObjectType.ID == ot2.ID {
					err := ucerr.WrapWithFriendlyStructure(nil, jsonclient.SDKStructuredError{
						Error:     "This object type already exists",
						ID:        ot2.ID,
						Identical: true,
					})
					jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusConflict))
					return
				}
				assert.Equal(t, req.ObjectType.TypeName, ot3.TypeName)
				jsonapi.Marshal(w, ot3, jsonapi.Code(http.StatusCreated))
			} else {
				jsonapi.Marshal(w, ListObjectTypesResponse{Data: []ObjectType{*ot1, *ot2}, ResponseFields: pagination.ResponseFields{HasNext: false}})
			}
		}))
		defer mockServer.Close()

		c := newCachedClientWithAuth(t, mockServer.URL)

		// this should miss (and then warm) the cache
		got1, err := c.FindObjectTypeID(ctx, ot1.TypeName)
		assert.NoErr(t, err)
		assert.Equal(t, got1, ot1.ID)
		assert.Equal(t, calls, 1)

		// this hits the cache
		gotNothing, err := c.FindObjectTypeID(ctx, tnInvalid)
		assert.NotNil(t, err)
		assert.Equal(t, gotNothing, uuid.Nil)
		assert.Equal(t, calls, 1)

		// this should have been cached already
		got2, err := c.FindObjectTypeID(ctx, ot2.TypeName)
		assert.NoErr(t, err)
		assert.Equal(t, got2, ot2.ID)
		assert.Equal(t, calls, 1)

		// try getting by ID this should have been cached already
		objType2, err := c.GetObjectType(ctx, ot2.ID)
		assert.NoErr(t, err)
		assert.Equal(t, objType2.ID, ot2.ID)
		assert.Equal(t, calls, 1)

		// modify the local copy and refetch which should hit the cache and return unmodified value
		objType2.ID = uuid.Nil
		objType2r, err := c.GetObjectType(ctx, ot2.ID)
		assert.NoErr(t, err)
		assert.Equal(t, objType2r.ID, ot2.ID)
		assert.Equal(t, calls, 1)

		// test if not exist creation for existing object doesn't clear the cache
		created, err := c.CreateObjectType(ctx, ot2.ID, ot2.TypeName, IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, created.ID, ot2.ID)
		assert.Equal(t, calls, 1)

		// test if not exist creation for existing object doesn't clear the cache
		created, err = c.CreateObjectType(ctx, uuid.Nil, ot2.TypeName, IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, created.ID, ot2.ID)
		assert.Equal(t, calls, 1)

		// this should have been cached already in the collection
		_, err = c.ListObjectTypes(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, calls, 1)

		// test creation
		created, err = c.CreateObjectType(ctx, ot3.ID, ot3.TypeName)
		assert.NoErr(t, err)
		assert.Equal(t, created, ot3)
		assert.Equal(t, calls, 2)

		// creation should have cached this
		// NB: any cache miss after this would invalidate the cache and make this fail,
		// since our mock is currently too lazy to store created object types :)
		got3, err := c.FindObjectTypeID(ctx, ot3.TypeName)
		assert.NoErr(t, err)
		assert.Equal(t, got3, ot3.ID)
		assert.Equal(t, calls, 2)

		// try getting by ID this should have been cached already
		objType3, err := c.GetObjectType(ctx, ot3.ID)
		assert.NoErr(t, err)
		assert.Equal(t, objType3.ID, ot3.ID)
		assert.Equal(t, calls, 2)

		// delete the object type to get it out of the cache
		err = c.DeleteObjectType(ctx, ot3.ID)
		assert.NoErr(t, err)
		assert.Equal(t, calls, 3)

		// this should miss the cache and fail because ot3 doesn't exist in the server response
		gotNothing, err = c.FindObjectTypeID(ctx, ot3.TypeName)
		assert.NotNil(t, err)
		assert.Equal(t, gotNothing, uuid.Nil)
		assert.Equal(t, calls, 4)

		// delete the object type to get it out of the cache
		err = c.DeleteObjectType(ctx, ot3.ID)
		assert.NoErr(t, err)
		assert.Equal(t, calls, 5)

		// this should miss the cache because the deletion of ot3 should have invalidated the global collection
		_, err = c.ListObjectTypes(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, calls, 6)

		// confirm that global collection is cached
		_, err = c.ListObjectTypes(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, calls, 6)

		// try read bypassing the cache
		_, err = c.ListObjectTypes(ctx, BypassCache())
		assert.NoErr(t, err)
		assert.Equal(t, calls, 7)

		// create a new object type
		created, err = c.CreateObjectType(ctx, ot3.ID, ot3.TypeName)
		assert.NoErr(t, err)
		assert.Equal(t, created, ot3)
		assert.Equal(t, calls, 8)

		// this should miss the cache
		_, err = c.ListObjectTypes(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, calls, 9)

		// delete the object type to get it out of the cache
		err = c.DeleteObjectType(ctx, ot3.ID)
		assert.NoErr(t, err)
		assert.Equal(t, calls, 10)

		// this should miss the cache and fail because ot3 doesn't exist in the server response
		_, err = c.GetObjectType(ctx, ot3.ID)
		assert.NoErr(t, err)
		assert.Equal(t, calls, 11)

		got1, err = c.FindObjectTypeID(ctx, ot1.TypeName)
		assert.NoErr(t, err)
		assert.Equal(t, got1, ot1.ID)
		assert.Equal(t, calls, 12)

		// verify that this should have been cached already
		got1, err = c.FindObjectTypeID(ctx, ot1.TypeName)
		assert.NoErr(t, err)
		assert.Equal(t, got1, ot1.ID)
		assert.Equal(t, calls, 12)
		// verify that this bypass hit the server
		got1, err = c.FindObjectTypeID(ctx, ot1.TypeName, BypassCache())
		assert.NoErr(t, err)
		assert.Equal(t, got1, ot1.ID)
		assert.Equal(t, calls, 13)

		// try getting by ID this should have been cached already
		objType2, err = c.GetObjectType(ctx, ot2.ID)
		assert.NoErr(t, err)
		assert.Equal(t, objType2.ID, ot2.ID)
		assert.Equal(t, calls, 13)
		// verify that this bypass hit the server
		_, err = c.GetObjectType(ctx, ot2.ID, BypassCache())
		assert.NoErr(t, err)
		assert.Equal(t, calls, 14)
	})

	t.Run("TestEdgeTypeCache", func(t *testing.T) {
		// This can't run in parallel as it depends on other tests not flushing the cache  t.Parallel()
		var calls int
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.Method == http.MethodPost { // create
				// Read the body to figure out which object was posted
				b, err := io.ReadAll(r.Body)
				if err != nil {
					jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
					return
				}
				br := bytes.NewReader(b)
				dec := json.NewDecoder(br)

				var req CreateEdgeTypeRequest
				assert.NoErr(t, dec.Decode(&req))

				if req.EdgeType.ID == et2.ID {
					err := ucerr.WrapWithFriendlyStructure(nil, jsonclient.SDKStructuredError{
						Error:     "This edge type already exists",
						ID:        et2.ID,
						Identical: true,
					})
					jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusConflict))
					return
				}
				assert.Equal(t, req.EdgeType.TypeName, et3.TypeName)
				jsonapi.Marshal(w, *et3, jsonapi.Code(http.StatusCreated))
			} else if strings.Count(r.URL.Path, "/") == 3 { // get edge type
				jsonapi.Marshal(w, *et2)
			} else { // list edge types
				jsonapi.Marshal(w, ListEdgeTypesResponse{Data: []EdgeType{*et1, *et2}, ResponseFields: pagination.ResponseFields{HasNext: false}})
			}
		}))
		defer mockServer.Close()

		c := newCachedClientWithAuth(t, mockServer.URL)

		// this should miss (and warm) the cache
		got1, err := c.FindEdgeTypeID(ctx, et1.TypeName)
		assert.NoErr(t, err)
		assert.Equal(t, got1, et1.ID)
		assert.Equal(t, calls, 1)

		// this should hit the cache since we all the edge types above
		gotNothing, err := c.FindEdgeTypeID(ctx, tnInvalid)
		assert.NotNil(t, err)
		assert.Equal(t, gotNothing, uuid.Nil)
		assert.Equal(t, calls, 1)

		// this should have been cached already
		got2, err := c.FindEdgeTypeID(ctx, et2.TypeName)
		assert.NoErr(t, err)
		assert.Equal(t, got2, et2.ID)
		assert.Equal(t, calls, 1)

		// this should have been cached already
		edgeType2, err := c.GetEdgeType(ctx, et2.ID)
		assert.NoErr(t, err)
		assert.Equal(t, edgeType2.ID, et2.ID)
		assert.Equal(t, calls, 1)

		// Make sure cache returns a copy by modifying the value and refetching
		edgeType2.ID = uuid.Nil
		edgeType2f, err := c.GetEdgeType(ctx, et2.ID)
		assert.NoErr(t, err)
		assert.Equal(t, edgeType2f.ID, et2.ID)
		assert.Equal(t, calls, 1)

		// and this hits the cache since we we fetched the edge types above
		got3, err := c.GetEdgeType(ctx, et1.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got3, et1)
		assert.Equal(t, calls, 1)

		// test if not exist creation for existing object doesn't clear the cache
		created, err := c.GetEdgeType(ctx, et2.ID, IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, created.ID, et2.ID)
		assert.Equal(t, calls, 1)

		// this should have been cached already in the collection
		_, err = c.ListEdgeTypes(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, calls, 1)

		// test if not exist creation for existing edge type doesn't clear the cache
		created, err = c.CreateEdgeType(ctx, et2.ID, et2.SourceObjectTypeID, et2.TargetObjectTypeID, et2.TypeName, et2.Attributes, IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, created.ID, et2.ID)
		assert.Equal(t, calls, 1)

		// test if not exist creation for existing edge type doesn't clear the cache
		created, err = c.CreateEdgeType(ctx, uuid.Nil, et2.SourceObjectTypeID, et2.TargetObjectTypeID, et2.TypeName, et2.Attributes, IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, created.ID, et2.ID)
		assert.Equal(t, calls, 1)

		// test if not exist nop update for existing edge type doesn't clear the cache
		created, err = c.UpdateEdgeType(ctx, et2.ID, et2.SourceObjectTypeID, et2.TargetObjectTypeID, et2.TypeName, et2.Attributes)
		assert.NoErr(t, err)
		assert.Equal(t, created.ID, et2.ID)
		assert.Equal(t, calls, 1)

		// this should have been cached already in the collection
		_, err = c.ListEdgeTypes(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, calls, 1)

		// test creation
		created1, err := c.CreateEdgeType(ctx, et3.ID, et3.SourceObjectTypeID, et3.TargetObjectTypeID, et3.TypeName, nil)
		assert.NoErr(t, err)
		assert.Equal(t, created1, et3)
		assert.Equal(t, calls, 2)

		// creation should have cached this
		got4, err := c.FindEdgeTypeID(ctx, et3.TypeName)
		assert.NoErr(t, err)
		assert.Equal(t, got4, created1.ID)
		assert.Equal(t, calls, 2)

		_, err = c.FindEdgeTypeID(ctx, et3.TypeName, BypassCache())
		assert.NotNil(t, err)
		assert.Equal(t, calls, 3)

		got6, err := c.GetEdgeType(ctx, et2.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got6, et2)
		assert.Equal(t, calls, 3)

		got7, err := c.GetEdgeType(ctx, et2.ID, BypassCache())
		assert.NoErr(t, err)
		assert.Equal(t, got7, et2)
		assert.Equal(t, calls, 4)

		_, err = c.ListEdgeTypes(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, calls, 5)

		/* TODO re-enable but setting tombstone time to 0
		_, err = c.ListEdgeTypes(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, calls, 5)

		_, err = c.ListEdgeTypes(ctx, BypassCache())
		assert.NoErr(t, err)
		assert.Equal(t, calls, 6)

		// poke to reset the cache to test one last miss path
		err = c.FlushCache()
		assert.NoErr(t, err)
		got8, err := c.GetEdgeType(ctx, et2.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got8, et2)
		assert.Equal(t, calls, 7)*/
	})

	t.Run("TestObjectCache", func(t *testing.T) {
		// This can't run in parallel as it depends on other tests not flushing the cache t.Parallel()
		var calls int
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.Method == http.MethodPost { // create
				if strings.Contains(r.URL.Path, "edges") {
					jsonapi.Marshal(w, *e1, jsonapi.Code(http.StatusCreated))
				} else {
					// Read the body to figure out which object was posted
					b, err := io.ReadAll(r.Body)
					if err != nil {
						jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
						return
					}
					br := bytes.NewReader(b)
					dec := json.NewDecoder(br)

					var req CreateObjectRequest
					assert.NoErr(t, dec.Decode(&req))

					if req.Object.ID == obj1.ID {
						err := ucerr.WrapWithFriendlyStructure(nil, jsonclient.SDKStructuredError{
							Error:     "This object already exists",
							ID:        obj1.ID,
							Identical: true,
						})
						jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusConflict))
						return
					}
					jsonapi.Marshal(w, *obj3, jsonapi.Code(http.StatusCreated))
				}
			} else if r.URL.RawQuery != "" { // GetObjectForName
				jsonapi.Marshal(w, ListObjectsResponse{Data: []Object{*obj1}, ResponseFields: pagination.ResponseFields{HasNext: false}})
			} else { // GetObject
				if strings.Contains(r.URL.Path, obj1.ID.String()) {
					jsonapi.Marshal(w, *obj1)
				} else if strings.Contains(r.URL.Path, obj2.ID.String()) {
					jsonapi.Marshal(w, *obj2)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}
		}))
		defer mockServer.Close()

		c := newCachedClientWithAuth(t, mockServer.URL)

		// this should miss (and warm) the cache
		got1, err := c.GetObject(ctx, obj1.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got1, obj1)
		assert.Equal(t, calls, 1)

		// this should miss the cache
		invalidID := uuid.Must(uuid.NewV4())
		gotNothing, err := c.GetObject(ctx, invalidID)
		assert.NotNil(t, err)
		assert.IsNil(t, gotNothing)
		assert.Equal(t, calls, 2)

		// this should have been cached already
		got2, err := c.GetObject(ctx, obj1.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got2, obj1)
		assert.Equal(t, calls, 2)

		// this won't be cached
		got3, err := c.GetObject(ctx, obj2.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got3, obj2)
		assert.Equal(t, calls, 3)

		// this will be cached
		got4, err := c.GetObjectForName(ctx, obj2.TypeID, *obj2.Alias)
		assert.NoErr(t, err)
		assert.Equal(t, got4, obj2)
		assert.Equal(t, calls, 3)

		// test if not exist creation for existing object doesn't clear the cache
		created, err := c.CreateObject(ctx, obj1.ID, obj1.TypeID, *obj1.Alias, IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, created.ID, obj1.ID)
		assert.Equal(t, calls, 3)

		// test if not exist creation for existing object doesn't clear the cache
		created, err = c.CreateObject(ctx, uuid.Nil, obj1.TypeID, *obj1.Alias, IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, created.ID, obj1.ID)
		assert.Equal(t, calls, 3)

		// test creation
		created1, err := c.CreateObject(ctx, obj3.ID, obj3.TypeID, *obj3.Alias)
		assert.NoErr(t, err)
		assert.Equal(t, created1, obj3)
		assert.Equal(t, calls, 4)

		// creation should have cached this
		got5, err := c.GetObject(ctx, obj3.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got5, obj3)
		assert.Equal(t, calls, 4)
		// verify the bypassing cache calls the server
		_, err = c.GetObject(ctx, obj3.ID, BypassCache())
		assert.NotNil(t, err)
		assert.Equal(t, calls, 5)
		// check that obj1 is still cached
		got5, err = c.GetObjectForName(ctx, obj1.TypeID, *obj1.Alias)
		assert.NoErr(t, err)
		assert.Equal(t, got5, obj1)
		assert.Equal(t, calls, 5)
		// verify the bypassing cache calls the server
		got5, err = c.GetObjectForName(ctx, obj1.TypeID, *obj1.Alias, BypassCache())
		assert.NoErr(t, err)
		assert.Equal(t, got5, obj1)
		assert.Equal(t, calls, 6)

		_, err = c.ListObjects(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, calls, 7)

		// Verify the bypassing cache calls the server. This API doesn't currently cache so just verifying the flag this also covers ListObjectsWithQuery
		_, err = c.ListObjects(ctx, BypassCache())
		assert.NoErr(t, err)
		assert.Equal(t, calls, 8)

		// poke to reset the cache to test one last miss path
		err = c.FlushCache()
		assert.NoErr(t, err)
		got6, err := c.GetObjectForName(ctx, obj1.TypeID, *obj1.Alias)
		assert.NoErr(t, err)
		assert.Equal(t, got6, obj1)
		assert.Equal(t, calls, 9)

		// set up some edges to test deletion
		got7, err := c.CreateEdge(ctx, e1.ID, e1.SourceObjectID, e1.TargetObjectID, e1.EdgeTypeID)
		assert.NoErr(t, err)
		assert.Equal(t, got7.ID, e1.ID)
		assert.Equal(t, calls, 10)
		got8, err := c.CreateEdge(ctx, e2.ID, e2.SourceObjectID, e2.TargetObjectID, e2.EdgeTypeID)
		assert.NoErr(t, err)
		assert.Equal(t, got8.ID, e1.ID)
		assert.Equal(t, calls, 11)

		// delete
		assert.IsNil(t, c.DeleteObject(ctx, obj1.ID))
		assert.Equal(t, calls, 12)

		// and make sure we miss the cache reading it
		// (note our test handler doesn't actually implement delete so we can read it again)
		got99, err := c.GetObject(ctx, obj1.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got99, obj1)
		assert.Equal(t, calls, 13)

		// Flush the cache
		assert.NoErr(t, c.FlushCacheObjectsAndEdges())

		// this should miss (and warm) the cache for next call
		gotByName, err := c.GetObjectForName(ctx, obj1.TypeID, *obj1.Alias)
		assert.NoErr(t, err)
		assert.Equal(t, gotByName, obj1)
		assert.Equal(t, calls, 14)

		gotByName, err = c.GetObjectForName(ctx, obj1.TypeID, *obj1.Alias)
		assert.NoErr(t, err)
		assert.Equal(t, gotByName, obj1)
		assert.Equal(t, calls, 14)
	})

	t.Run("TestOrganizationCache", func(t *testing.T) {
		// This can't run in parallel as it depends on other tests not flushing the cache t.Parallel()
		var calls int
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.Method == http.MethodPost { // create
				jsonapi.Marshal(w, *org3, jsonapi.Code(http.StatusCreated))
			} else if r.URL.RawQuery != "" { // ListOrgs
				jsonapi.Marshal(w, ListOrganizationsResponse{Data: []Organization{*org1, *org2}, ResponseFields: pagination.ResponseFields{HasNext: false}})
			} else { // GetOrg
				if strings.Contains(r.URL.Path, org1.ID.String()) {
					jsonapi.Marshal(w, *org1)
				} else if strings.Contains(r.URL.Path, org2.ID.String()) {
					jsonapi.Marshal(w, *org2)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}
		}))
		defer mockServer.Close()

		c := newCachedClientWithAuth(t, mockServer.URL)

		// this should miss (and warm) the cache
		got1, err := c.GetOrganization(ctx, org1.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got1, org1)
		assert.Equal(t, calls, 1)

		// this should miss the cache
		invalidID := uuid.Must(uuid.NewV4())
		gotNothing, err := c.GetOrganization(ctx, invalidID)
		assert.NotNil(t, err)
		assert.IsNil(t, gotNothing)
		assert.Equal(t, calls, 2)

		// this should have been cached already
		got2, err := c.GetOrganization(ctx, org1.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got2, org1)
		assert.Equal(t, calls, 2)

		// this won't be cached
		got3, err := c.GetOrganization(ctx, org2.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got3, org2)
		assert.Equal(t, calls, 3)

		// this will be cached
		got4, err := c.GetOrganization(ctx, org2.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got4, org2)
		assert.Equal(t, calls, 3)

		// test creation
		created1, err := c.CreateOrganization(ctx, org3.ID, org3.Name, org3.Region)
		assert.NoErr(t, err)
		assert.Equal(t, created1, org3)
		assert.Equal(t, calls, 4)

		got5, err := c.GetOrganization(ctx, org3.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got5, org3)
		assert.Equal(t, calls, 4)

		resp, err := c.ListOrganizations(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, len(resp), 2)
		assert.Equal(t, calls, 5)

		/* re-enable by setting tombstone to 0
		resp, err = c.ListOrganizations(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, len(resp), 2)
		assert.Equal(t, calls, 5)*/
	})
}

func TestEdgeCache(t *testing.T) {
	ctx := context.Background()

	e1 := &Edge{
		BaseModel:      ucdb.NewBase(),
		EdgeTypeID:     uuid.Must(uuid.NewV4()),
		SourceObjectID: uuid.Must(uuid.NewV4()),
		TargetObjectID: uuid.Must(uuid.NewV4()),
	}
	e2 := &Edge{
		BaseModel:      ucdb.NewBase(),
		EdgeTypeID:     uuid.Must(uuid.NewV4()),
		SourceObjectID: uuid.Must(uuid.NewV4()),
		TargetObjectID: uuid.Must(uuid.NewV4()),
	}

	var calls int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == http.MethodPost { // create
			// Read the body to figure out which object was posted
			b, err := io.ReadAll(r.Body)
			if err != nil {
				jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
				return
			}
			br := bytes.NewReader(b)
			dec := json.NewDecoder(br)

			var req CreateEdgeRequest
			assert.NoErr(t, dec.Decode(&req))

			if req.Edge.ID == e2.ID {
				err := ucerr.WrapWithFriendlyStructure(nil, jsonclient.SDKStructuredError{
					Error:     "This edge already exists",
					ID:        e2.ID,
					Identical: true,
				})
				jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusConflict))
				return
			}
			assert.Equal(t, req.Edge.ID, e1.ID)
			jsonapi.Marshal(w, *e1, jsonapi.Code(http.StatusCreated))
		} else if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
		} else if strings.Contains(r.URL.RawQuery, e2.EdgeTypeID.String()) { // list edge types
			jsonapi.Marshal(w, ListEdgesResponse{Data: []Edge{*e2}})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	c := newCachedClientWithAuth(t, mockServer.URL)

	// this should miss the cache and populate it for e2
	got3, err := c.FindEdge(ctx, e2.SourceObjectID, e2.TargetObjectID, e2.EdgeTypeID)
	assert.NoErr(t, err)
	assert.Equal(t, got3, e2)
	assert.Equal(t, calls, 1)

	// test if not exist creation for existing edge doesn't clear the cache
	created, err := c.CreateEdge(ctx, e2.ID, e2.SourceObjectID, e2.TargetObjectID, e2.EdgeTypeID, IfNotExists())
	assert.NoErr(t, err)
	assert.Equal(t, created.ID, e2.ID)
	assert.Equal(t, calls, 1)

	// test if not exist creation for existing edge doesn't clear the cache
	created, err = c.CreateEdge(ctx, uuid.Nil, e2.SourceObjectID, e2.TargetObjectID, e2.EdgeTypeID, IfNotExists())
	assert.NoErr(t, err)
	assert.Equal(t, created.ID, e2.ID)
	assert.Equal(t, calls, 1)

	got1, err := c.CreateEdge(ctx, e1.ID, e1.SourceObjectID, e1.TargetObjectID, e1.EdgeTypeID)
	assert.NoErr(t, err)
	assert.Equal(t, got1.ID, e1.ID)
	assert.Equal(t, calls, 2)

	// this should be cached after create
	got2, err := c.FindEdge(ctx, e1.SourceObjectID, e1.TargetObjectID, e1.EdgeTypeID)
	assert.NoErr(t, err)
	assert.Equal(t, got2, e1)
	assert.Equal(t, calls, 2)

	/* renenable by setting tombstone to 0
	// and this should hit
	got4, err := c.FindEdge(ctx, e2.SourceObjectID, e2.TargetObjectID, e2.EdgeTypeID)
	assert.NoErr(t, err)
	assert.Equal(t, got4, e2)
	assert.Equal(t, calls, 2)

	// this should bypass the cache
	got4, err = c.FindEdge(ctx, e2.SourceObjectID, e2.TargetObjectID, e2.EdgeTypeID, BypassCache())
	assert.NoErr(t, err)
	assert.Equal(t, got4, e2)
	assert.Equal(t, calls, 3)

	// and this should hit
	got4, err = c.GetEdge(ctx, e2.ID)
	assert.NoErr(t, err)
	assert.Equal(t, got4, e2)
	assert.Equal(t, calls, 3)

	// this should bypass the cache
	_, err = c.GetEdge(ctx, e2.ID, BypassCache())
	assert.NotNil(t, err)
	assert.Equal(t, calls, 4)

	assert.IsNil(t, c.DeleteEdge(ctx, e2.ID))
	assert.Equal(t, calls, 5)

	// this should miss the cache again after delete
	got5, err := c.FindEdge(ctx, e2.SourceObjectID, e2.TargetObjectID, e2.EdgeTypeID)
	assert.NoErr(t, err)
	assert.Equal(t, got5, e2)
	assert.Equal(t, calls, 6)

	// and this should miss
	got6, err := c.FindEdge(ctx, uuid.Nil, uuid.Nil, uuid.Nil)
	assert.NotNil(t, err)
	assert.IsNil(t, got6)
	assert.Equal(t, calls, 7)
	*/
}
