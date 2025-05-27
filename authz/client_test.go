package authz_test

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"

	"userclouds.com/authz"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
)

func (tf *testFixture) checkAttribute(t *testing.T, srcID, tgtID uuid.UUID, attr string) *authz.CheckAttributeResponse {
	t.Helper()
	resp, err := tf.client.CheckAttribute(context.Background(), srcID, tgtID, attr)
	assert.NoErr(t, err)
	return resp
}

func (tf *testFixture) listAttributes(t *testing.T, srcID, tgtID uuid.UUID) []string {
	t.Helper()
	resp, err := tf.client.ListAttributes(context.Background(), srcID, tgtID)
	assert.NoErr(t, err)
	return resp
}

func (tf *testFixture) listObjectsReachableWithAttribute(t *testing.T, srcID, tgtTypeID uuid.UUID, attr string) []uuid.UUID {
	t.Helper()
	resp, err := tf.client.ListObjectsReachableWithAttribute(context.Background(), srcID, tgtTypeID, attr)
	assert.NoErr(t, err)
	return resp
}

func uniqueName(name string) string {
	return name + "_" + uuid.Must(uuid.NewV4()).String()
}
func TestAuthzClient(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tf := newTestFixture(t)

	t.Run("test_attribute", func(t *testing.T) {
		t.Parallel()
		etID := uuid.Must(uuid.NewV4())
		etName := uniqueName("et")
		// Create edge type without attributes
		_, err := tf.client.CreateEdgeType(ctx, etID, authz.UserObjectTypeID, authz.GroupObjectTypeID, etName, nil)
		assert.NoErr(t, err)

		user := tf.newTestUser()
		group, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), authz.GroupObjectTypeID, uniqueName("group"))
		assert.NoErr(t, err)

		attributeName := "manage" // doesn't have to be unique
		// Edge not created yet, so attribute can't exist
		assert.False(t, tf.checkAttribute(t, user.ID, group.ID, attributeName).HasAttribute)

		edge, err := tf.client.CreateEdge(ctx, uuid.Must(uuid.NewV4()), user.ID, group.ID, etID)
		assert.NoErr(t, err)

		// Edge was created, but edge type doesn't (yet) have the right attribute
		assert.False(t, tf.checkAttribute(t, user.ID, group.ID, attributeName).HasAttribute)

		_, err = tf.client.UpdateEdgeType(ctx, etID, authz.UserObjectTypeID, authz.GroupObjectTypeID, etName, authz.Attributes{{
			Name:   attributeName,
			Direct: true,
		}})
		assert.NoErr(t, err)

		// Edge with attribute created, should succeed
		resp := tf.checkAttribute(t, user.ID, group.ID, attributeName)
		assert.True(t, resp.HasAttribute, assert.Must())
		assert.Equal(t, 2, len(resp.Path), assert.Must())
		assert.Equal(t, resp.Path[0].ObjectID, user.ID)
		assert.Equal(t, resp.Path[0].EdgeID, uuid.Nil)
		assert.Equal(t, resp.Path[1].ObjectID, group.ID)
		assert.Equal(t, resp.Path[1].EdgeID, edge.ID)

		// Test invalid attribute, should fail
		assert.False(t, tf.checkAttribute(t, user.ID, group.ID, "foobar").HasAttribute)

		// Test attribute in reverse (swap source & target), should fail
		assert.False(t, tf.checkAttribute(t, group.ID, user.ID, attributeName).HasAttribute)

		// Test that "manage" is included in the list of attributes between user & group
		listResp := tf.listAttributes(t, user.ID, group.ID)
		assert.Equal(t, []string{attributeName}, listResp)
	})
	t.Run("test_inherit_and_propogate", func(t *testing.T) {
		t.Parallel()

		ot := uuid.Must(uuid.NewV4())
		memberET := uuid.Must(uuid.NewV4())
		powerUserET := uuid.Must(uuid.NewV4())
		ownerET := uuid.Must(uuid.NewV4())
		parentET := uuid.Must(uuid.NewV4())

		_, err := tf.client.CreateObjectType(ctx, ot, uniqueName("obj_type"))
		assert.NoErr(t, err)
		// Members inherit read permissions only
		_, err = tf.client.CreateEdgeType(ctx, memberET, ot, ot, uniqueName("member"), authz.Attributes{
			{Name: "read", Inherit: true}})
		assert.NoErr(t, err)
		// Power users inherit read & write permissions, but not delete
		_, err = tf.client.CreateEdgeType(ctx, powerUserET, ot, ot, uniqueName("power_user"), authz.Attributes{
			{Name: "read", Inherit: true},
			{Name: "write", Inherit: true}})
		assert.NoErr(t, err)
		// Owners of resources get read + write + delete
		_, err = tf.client.CreateEdgeType(ctx, ownerET, ot, ot, uniqueName("owner"), authz.Attributes{
			{Name: "read", Direct: true},
			{Name: "write", Direct: true},
			{Name: "delete", Direct: true}})
		assert.NoErr(t, err)
		// Parent resources propagate permission to sub resources
		_, err = tf.client.CreateEdgeType(ctx, parentET, ot, ot, uniqueName("parent"), authz.Attributes{
			{Name: "read", Propagate: true},
			{Name: "write", Propagate: true},
			{Name: "delete", Propagate: true}})
		assert.NoErr(t, err)

		// Create 2 people, a team, and a resource. The team has direct read/write/delete perms on the resource.
		person1, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), ot, uniqueName("person1"))
		assert.NoErr(t, err)
		person2, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), ot, uniqueName("person2"))
		assert.NoErr(t, err)
		team, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), ot, uniqueName("team"))
		assert.NoErr(t, err)
		resource, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), ot, uniqueName("resource"))
		assert.NoErr(t, err)
		subResource, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), ot, uniqueName("sub_resource"))
		assert.NoErr(t, err)

		// Make sure people can't access the resource yet.
		assert.False(t, tf.checkAttribute(t, person1.ID, resource.ID, "read").HasAttribute)

		// Make person1 a member of team, ensure they still can't access resource.
		person1TeamEdge, err := tf.client.CreateEdge(ctx, uuid.Must(uuid.NewV4()), person1.ID, team.ID, memberET)
		assert.NoErr(t, err)
		assert.False(t, tf.checkAttribute(t, person1.ID, resource.ID, "read").HasAttribute)

		// Make the team an owner of the resource.
		teamResourceOwner, err := tf.client.CreateEdge(ctx, uuid.Must(uuid.NewV4()), team.ID, resource.ID, ownerET)
		assert.NoErr(t, err)

		// Person 1 (part of the team), can read (but not write) it via team ownership
		assert.False(t, tf.checkAttribute(t, person1.ID, resource.ID, "write").HasAttribute)
		resp := tf.checkAttribute(t, person1.ID, resource.ID, "read")
		assert.True(t, resp.HasAttribute)
		assert.Equal(t, 3, len(resp.Path), assert.Must())
		assert.Equal(t, resp.Path[0].ObjectID, person1.ID)
		assert.Equal(t, resp.Path[0].EdgeID, uuid.Nil)
		assert.Equal(t, resp.Path[1].ObjectID, team.ID)
		assert.Equal(t, resp.Path[1].EdgeID, person1TeamEdge.ID)
		assert.Equal(t, resp.Path[2].ObjectID, resource.ID)
		assert.Equal(t, resp.Path[2].EdgeID, teamResourceOwner.ID)

		// Person 2 still can't access it.
		assert.False(t, tf.checkAttribute(t, person2.ID, resource.ID, "read").HasAttribute)

		// Add person 2 to the team as an admin, ensure that they can read/write (but not delete) the resource
		// via their connection to the team.
		person2TeamEdge, err := tf.client.CreateEdge(ctx, uuid.Must(uuid.NewV4()), person2.ID, team.ID, powerUserET)
		assert.NoErr(t, err)
		assert.True(t, tf.checkAttribute(t, person2.ID, resource.ID, "read").HasAttribute)
		resp = tf.checkAttribute(t, person2.ID, resource.ID, "write")
		assert.True(t, resp.HasAttribute)
		assert.Equal(t, 3, len(resp.Path), assert.Must())
		assert.Equal(t, resp.Path[0].ObjectID, person2.ID)
		assert.Equal(t, resp.Path[0].EdgeID, uuid.Nil)
		assert.Equal(t, resp.Path[1].ObjectID, team.ID)
		assert.Equal(t, resp.Path[1].EdgeID, person2TeamEdge.ID)
		assert.Equal(t, resp.Path[2].ObjectID, resource.ID)
		assert.Equal(t, resp.Path[2].EdgeID, teamResourceOwner.ID)
		assert.False(t, tf.checkAttribute(t, person2.ID, resource.ID, "delete").HasAttribute)

		// Ensure team can delete the resource
		resp = tf.checkAttribute(t, team.ID, resource.ID, "delete")
		assert.True(t, resp.HasAttribute)
		assert.Equal(t, 2, len(resp.Path), assert.Must())
		assert.Equal(t, resp.Path[0].ObjectID, team.ID)
		assert.Equal(t, resp.Path[0].EdgeID, uuid.Nil)
		assert.Equal(t, resp.Path[1].ObjectID, resource.ID)
		assert.Equal(t, resp.Path[1].EdgeID, teamResourceOwner.ID)

		// Ensure people/team can't access sub resource yet
		assert.False(t, tf.checkAttribute(t, person1.ID, subResource.ID, "read").HasAttribute)
		assert.False(t, tf.checkAttribute(t, person2.ID, subResource.ID, "write").HasAttribute)
		assert.False(t, tf.checkAttribute(t, team.ID, subResource.ID, "delete").HasAttribute)

		// Make resource the parent of sub resource
		subResourceEdge, err := tf.client.CreateEdge(ctx, uuid.Must(uuid.NewV4()), resource.ID, subResource.ID, parentET)
		assert.NoErr(t, err)

		// Now people/team can access sub resource accordingly
		assert.True(t, tf.checkAttribute(t, person1.ID, subResource.ID, "read").HasAttribute)
		resp = tf.checkAttribute(t, person2.ID, subResource.ID, "write")
		assert.True(t, resp.HasAttribute)
		assert.Equal(t, 4, len(resp.Path), assert.Must())
		assert.Equal(t, resp.Path[0].ObjectID, person2.ID)
		assert.Equal(t, resp.Path[0].EdgeID, uuid.Nil)
		assert.Equal(t, resp.Path[1].ObjectID, team.ID)
		assert.Equal(t, resp.Path[1].EdgeID, person2TeamEdge.ID)
		assert.Equal(t, resp.Path[2].ObjectID, resource.ID)
		assert.Equal(t, resp.Path[2].EdgeID, teamResourceOwner.ID)
		assert.Equal(t, resp.Path[3].ObjectID, subResource.ID)
		assert.Equal(t, resp.Path[3].EdgeID, subResourceEdge.ID)
		// Save for later comparison
		person2SubResourceWritePath := resp

		resp = tf.checkAttribute(t, team.ID, subResource.ID, "delete")
		assert.True(t, resp.HasAttribute)
		assert.Equal(t, 3, len(resp.Path), assert.Must())
		assert.Equal(t, resp.Path[0].ObjectID, team.ID)
		assert.Equal(t, resp.Path[0].EdgeID, uuid.Nil)
		assert.Equal(t, resp.Path[1].ObjectID, resource.ID)
		assert.Equal(t, resp.Path[1].EdgeID, teamResourceOwner.ID)
		assert.Equal(t, resp.Path[2].ObjectID, subResource.ID)
		assert.Equal(t, resp.Path[2].EdgeID, subResourceEdge.ID)

		// Add a direct read edge from person 2 to the sub resource, and ensure it's the chosen path (because it's shorter)
		directReadET := uuid.Must(uuid.NewV4())
		_, err = tf.client.CreateEdgeType(ctx, directReadET, ot, ot, "direct_read", authz.Attributes{
			{Name: "read", Direct: true}})
		assert.NoErr(t, err)
		person2SubResourceEdge, err := tf.client.CreateEdge(ctx, uuid.Must(uuid.NewV4()), person2.ID, subResource.ID, directReadET)
		assert.NoErr(t, err)
		resp = tf.checkAttribute(t, person2.ID, subResource.ID, "read")
		assert.True(t, resp.HasAttribute)
		assert.Equal(t, 2, len(resp.Path), assert.Must())
		assert.Equal(t, resp.Path[0].ObjectID, person2.ID)
		assert.Equal(t, resp.Path[0].EdgeID, uuid.Nil)
		assert.Equal(t, resp.Path[1].ObjectID, subResource.ID)
		assert.Equal(t, resp.Path[1].EdgeID, person2SubResourceEdge.ID)
		// But write privileges still go through the old path (person2 -> team -> resource -> subresource)
		resp = tf.checkAttribute(t, person2.ID, subResource.ID, "write")
		assert.Equal(t, resp, person2SubResourceWritePath)

		listResp := tf.listAttributes(t, person1.ID, resource.ID)
		sort.Strings(listResp)
		assert.Equal(t, listResp, []string{"read"})

		listResp = tf.listAttributes(t, person2.ID, resource.ID)
		sort.Strings(listResp)
		assert.Equal(t, listResp, []string{"read", "write"})

		reachResp := tf.listObjectsReachableWithAttribute(t, person1.ID, ot, "read")
		assert.Equal(t, len(reachResp), 2)

		reachResp = tf.listObjectsReachableWithAttribute(t, person2.ID, ot, "read")
		assert.Equal(t, len(reachResp), 3)

		reachResp = tf.listObjectsReachableWithAttribute(t, person2.ID, ot, "write")
		assert.Equal(t, len(reachResp), 2)
	})
	t.Run("test_user_entry", func(t *testing.T) {
		t.Parallel()

		// Test ingesting multiple entries
		// NOTE: keep Entry Types alpha-sorted to make validation later easier.
		entries := []auditlog.Entry{
			{
				BaseModel: ucdb.NewBase(),
				Type:      "some_Entry1",
				Actor:     "user1",
				Payload:   auditlog.Payload{"key1": "val1"},
			},
			{
				BaseModel: ucdb.NewBase(),
				Type:      "some_Entry2",
				Actor:     "user2",
				Payload:   auditlog.Payload{"key1": "val1"},
			},
		}
		err := tf.aLClient.CreateEntry(ctx, entries[0])
		assert.NoErr(t, err)
		err = tf.aLClient.CreateEntry(ctx, entries[1])
		assert.NoErr(t, err)

		entriesRespPtr, err := tf.aLClient.ListEntries(ctx)
		assert.NoErr(t, err)

		entriesResp := *entriesRespPtr

		assert.Equal(t, len(entriesResp.Data), 2)

		// Alpha sort entries to make comparison easier
		sort.Slice(entriesResp.Data, func(i, j int) bool {
			return entriesResp.Data[i].Type < entriesResp.Data[j].Type
		})

		// Validate entry contents
		for i := range entries {
			assert.Equal(t, entriesResp.Data[i].Type, entries[i].Type)
			assert.Equal(t, entriesResp.Data[i].Actor, entries[i].Actor)
			assert.Equal(t, entriesResp.Data[i].Payload, entries[i].Payload)
		}

		// Test appending a new entry (NOTE: event Type alpha sorts to the end)
		newEntries := []auditlog.Entry{
			{
				BaseModel: ucdb.NewBase(),
				Type:      "some_event3",
				Actor:     "foo",
				Payload:   auditlog.Payload{},
			},
		}

		err = tf.aLClient.CreateEntry(ctx, newEntries[0])
		assert.NoErr(t, err)

		entriesRespPtr, err = tf.aLClient.ListEntries(ctx)
		assert.NoErr(t, err)
		entriesResp = *entriesRespPtr

		assert.Equal(t, len(entriesResp.Data), 3, assert.Must())

		sort.Slice(entriesResp.Data, func(i, j int) bool {
			return entriesResp.Data[i].Type < entriesResp.Data[j].Type
		})

		concatEntries := append(entries, newEntries...)
		for i := range concatEntries {
			assert.Equal(t, entriesResp.Data[i].Type, concatEntries[i].Type)
			assert.Equal(t, entriesResp.Data[i].Actor, concatEntries[i].Actor)
			assert.Equal(t, entriesResp.Data[i].Payload, concatEntries[i].Payload)
		}

		// Test retrieving a particular entry
		entry, err := tf.aLClient.GetEntry(ctx, newEntries[0].ID)
		assert.NoErr(t, err)
		assert.Equal(t, entry.Type, newEntries[0].Type)
		assert.Equal(t, entry.Actor, newEntries[0].Actor)
		assert.Equal(t, entry.Payload, newEntries[0].Payload)

		// Test retrieving a non-existing entry
		id := uuid.Must(uuid.NewV4())
		entry, err = tf.aLClient.GetEntry(ctx, id)
		assert.NotNil(t, err, assert.Must())
		assert.IsNil(t, entry, assert.Must())
	})
	t.Run("test_user_entry", func(t *testing.T) {
		t.Parallel()
		// Invalid identifier name
		err := tf.aLClient.CreateEntry(ctx, auditlog.Entry{
			BaseModel: ucdb.NewBase(),
			Type:      "!invalid",
			Actor:     "user1",
		},
		)
		assert.NotNil(t, err, assert.Must())

		// Missing user alias
		err = tf.aLClient.CreateEntry(ctx, auditlog.Entry{
			BaseModel: ucdb.NewBase(),
			Type:      "missing_user_id",
		},
		)
		assert.NotNil(t, err, assert.Must())
	})

	t.Run("test_object_pagination", func(t *testing.T) {
		t.Parallel()

		const limit = 5
		const numObjs = 16

		otID := uuid.Must(uuid.NewV4())
		_, err := tf.client.CreateObjectType(ctx, otID, uniqueName("obj_type"))
		assert.NoErr(t, err)

		serverObjs := enumerateObjects(ctx, t, tf.client, limit, otID, uuid.Nil)
		assert.Equal(t, 0, len(serverObjs)) // there should be now objects of this type

		for i := range numObjs {
			obj, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), otID, fmt.Sprintf("obj_%d", i))
			assert.NoErr(t, err)
			serverObjs = append(serverObjs, *obj)
		}

		clientObjs := enumerateObjects(ctx, t, tf.client, limit, otID, uuid.Nil)

		validateClientAndServerObjects(t, clientObjs, serverObjs)
	})

	t.Run("test_object_update", func(t *testing.T) {
		t.Parallel()

		obj, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), authz.UserObjectTypeID, uniqueName("userobj"))
		assert.NoErr(t, err)

		// Test updating the object
		newAlias := uniqueName("new_name")
		obj, err = tf.client.UpdateObject(ctx, obj.ID, &newAlias, authz.Source("idp"))
		assert.NoErr(t, err)
		assert.Equal(t, *obj.Alias, newAlias)

		// Test updating an object with an empty alias
		obj, err = tf.client.UpdateObject(ctx, obj.ID, nil, authz.Source("idp"))
		assert.NoErr(t, err)
		assert.IsNil(t, obj.Alias)

		// Test updating a user object from non-idp source
		_, err = tf.client.UpdateObject(ctx, obj.ID, &newAlias)
		assert.NotNil(t, err)

		// Test updating a non-existent object
		_, err = tf.client.UpdateObject(ctx, uuid.Must(uuid.NewV4()), &newAlias, authz.Source("idp"))
		assert.NotNil(t, err)

		// Test updating a non-user object
		obj2, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), authz.GroupObjectTypeID, uniqueName("groupobj"))
		assert.NoErr(t, err)
		_, err = tf.client.UpdateObject(ctx, obj2.ID, &newAlias)
		assert.IsNil(t, err)
	})
	t.Run("test_edges_pagination", func(t *testing.T) {
		// t.Parallel() - we don't run this in parallel because it depends on no other edges being created during the test to validate call counts

		const limit = 5
		const numEdges = 12
		var clientCalls int

		serverEdges, _ := enumerateEdges(ctx, t, tf.client, limit)
		et := uuid.Must(uuid.NewV4())
		_, err := tf.client.CreateEdgeType(ctx, et, authz.GroupObjectTypeID, authz.GroupObjectTypeID, uniqueName("edge_type"), nil)
		assert.NoErr(t, err)

		rootObj, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), authz.GroupObjectTypeID, uniqueName("rootobj"))
		assert.NoErr(t, err)

		for i := range numEdges {
			obj, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), authz.GroupObjectTypeID, uniqueName(fmt.Sprintf("obj_%d", i)))
			assert.NoErr(t, err)
			edge, err := tf.client.CreateEdge(ctx, uuid.Must(uuid.NewV4()), rootObj.ID, obj.ID, et)
			assert.NoErr(t, err)
			serverEdges = append(serverEdges, *edge)
		}

		clientEdges, clientCalls := enumerateEdges(ctx, t, tf.client, limit)
		targetClientCalls := int(math.Ceil(float64(len(serverEdges)) / limit))
		assert.NoErr(t, err)
		assert.Equal(t, clientCalls, targetClientCalls)

		assert.NoErr(t, validateClientAndServerEdges(clientEdges, serverEdges))
	})

	t.Run("test_edges_pagination", func(t *testing.T) {
		t.Parallel()

		const limit = 5
		const numEdges = 20
		var clientCalls int

		et := uuid.Must(uuid.NewV4())
		_, err := tf.client.CreateEdgeType(ctx, et, authz.GroupObjectTypeID, authz.GroupObjectTypeID, uniqueName("edge_type"), nil)
		assert.NoErr(t, err)

		rootObj, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), authz.GroupObjectTypeID, uniqueName("rootobj"))
		assert.NoErr(t, err)

		serverObjs := []authz.Edge{}
		for i := range numEdges {
			obj, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), authz.GroupObjectTypeID, uniqueName(fmt.Sprintf("obj_%d", i)))
			assert.NoErr(t, err)
			edge, err := tf.client.CreateEdge(ctx, uuid.Must(uuid.NewV4()), rootObj.ID, obj.ID, et)
			assert.NoErr(t, err)
			serverObjs = append(serverObjs, *edge)
		}

		clientObjs, clientCalls := enumerateEdgesOnObject(ctx, t, tf.client, rootObj.ID, limit)
		assert.Equal(t, clientCalls, 4)

		assert.NoErr(t, validateClientAndServerEdges(clientObjs, serverObjs))
	})

	t.Run("test_getobject_byname", func(t *testing.T) {
		t.Parallel()
		objName := uniqueName("testobj")
		testObj, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), authz.GroupObjectTypeID, objName)
		assert.NoErr(t, err)

		// Test IfNotExists
		obj2, err := tf.client.CreateObject(ctx, uuid.Nil, authz.GroupObjectTypeID, objName, authz.IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, testObj.ID, obj2.ID)

		_, err = tf.client.CreateObject(ctx, testObj.ID, authz.GroupObjectTypeID, objName, authz.IfNotExists())
		assert.NoErr(t, err)

		_, err = tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), authz.GroupObjectTypeID, objName, authz.IfNotExists())
		assert.NotNil(t, err, assert.Must())

		// Need to create a new AuthZ Client to test the actual request, otherwise the existing client's
		// warm cache will avoid making the HTTP request.
		host, err := tenantmap.GetHostFromTenantURL(tf.tenant.TenantURL)
		assert.NoErr(t, err)
		newClient, err := authz.NewClient(
			tf.tenant.TenantURL,
			authz.JSONClient(jsonclient.HeaderHost(host),
				jsonclient.HeaderAuthBearer(tf.jwt)))
		assert.NoErr(t, err)

		obj, err := newClient.GetObjectForName(ctx, authz.GroupObjectTypeID, objName)
		assert.NoErr(t, err)
		assert.Equal(t, obj.ID, testObj.ID)
	})

	t.Run("test_cache_tlssettings", func(t *testing.T) {
		t.Parallel()
		host, err := tenantmap.GetHostFromTenantURL(tf.tenant.TenantURL)
		assert.NoErr(t, err)

		opts := authz.JSONClient(jsonclient.HeaderHost(host),
			jsonclient.HeaderAuthBearer(tf.jwt))
		// Create a client with default timeouts
		c1, err := authz.NewClient(tf.tenant.TenantURL, opts)
		assert.NoErr(t, err)
		// Create a second client with non default timeouts that doesn't cache ObjectTypes (TTL = 0)
		c2, err := authz.NewCustomClient(0, time.Minute, time.Minute, time.Minute, tf.tenant.TenantURL, opts)
		assert.NoErr(t, err)
		// Create a third client with non default timeouts that does cache ObjectTypes (TTL = 1m)
		c3, err := authz.NewCustomClient(time.Minute, time.Minute, time.Minute, time.Minute,
			tf.tenant.TenantURL,
			opts)
		assert.NoErr(t, err)

		objType, err := c1.CreateObjectType(ctx, uuid.Must(uuid.NewV4()), "test_type")
		assert.NoErr(t, err)

		c2Obj1, err := c2.GetObjectType(ctx, objType.ID)
		assert.NoErr(t, err)
		assert.Equal(t, objType, c2Obj1)

		c3Obj1, err := c3.GetObjectType(ctx, objType.ID)
		assert.NoErr(t, err)
		assert.Equal(t, objType, c3Obj1)

		// Delete the object type from the server (also removes it from the cache in c1 but not in c2 & c3)
		err = c1.DeleteObjectType(ctx, objType.ID)
		assert.NoErr(t, err)

		// Should fail as there is no cache for object types in c2
		_, err = c2.GetObjectType(ctx, objType.ID)
		assert.NotNil(t, err)
		assert.ErrorIs(t, err, authz.ErrObjectTypeNotFound)

		// TODO this will fail with a shared cache (redis) as the delete will apply to all clients
		// Need to figure out how to run this for inline cache only

		// Should succeed as there is a cache of 1 minutes
		/*c3Obj2, err := c3.GetObjectType(ctx, objType.ID)
		assert.NoErr(t, err)
		assert.Equal(t, objType, c3Obj2)*/
	})

	t.Run("test_cache_objectparallel", func(t *testing.T) {
		t.Parallel()

		objName := uniqueName("testobjParallelTest")
		testObj, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), authz.GroupObjectTypeID, objName)
		assert.NoErr(t, err)

		// First try serially
		for range 50 {
			err = tf.client.DeleteObject(ctx, testObj.ID)
			assert.NoErr(t, err)

			_, err = tf.client.GetObjectForName(ctx, authz.GroupObjectTypeID, objName)
			assert.NotNil(t, err)
			//TODO assert.HttpError(t, err, http.StatusNotFound)

			_, err = tf.client.GetObject(ctx, testObj.ID)
			assert.NotNil(t, err)

			testObj, err = tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), authz.GroupObjectTypeID, objName)
			assert.NoErr(t, err)
		}

		// Now try in parallel
		wg := sync.WaitGroup{}

		for i := range 2 {
			wg.Add(1)

			go func(threadID int) {
				errC := 0
				defer wg.Done()
				for range 50 {
					// This will either create a new object or return the object created by the other thread
					var testObjL *authz.Object
					var err error
					retry := true

					for retry {
						retry = false
						testObjL, err = tf.client.CreateObject(ctx, uuid.Nil, authz.GroupObjectTypeID, objName, authz.IfNotExists())
						if err != nil && jsonclient.IsHTTPStatusConflict(err) {
							retry = true
						}
					}
					assert.NoErr(t, err)

					// The delete will fail if the other thread has already deleted the object
					err = tf.client.DeleteObject(ctx, testObjL.ID)
					_ = err // ignore error
					// TODO if err != nil {assert.HttpError(t, err, http.StatusNotFound)}

					// The get can succeed if the other thread has recreated the object after the delete but
					// it should never return the original object since we delete it on this thread
					objR, err := tf.client.GetObjectForName(ctx, authz.GroupObjectTypeID, objName)
					if err == nil {
						if objR.ID == testObjL.ID {
							errC = errC + 1
						}
						assert.NotEqual(t, objR.ID, testObjL.ID)
					}
					//TODO else { assert.HttpError(t, err, http.StatusNotFound) }

					_, err = tf.client.GetObject(ctx, testObjL.ID) // this should always fail because we delete testObjL on this thread
					if err == nil {
						errC = errC + 1
					}
					assert.NotNil(t, err)
				}
			}(i)

		}

		wg.Wait()
	})

	t.Run("test_cache_edges_serial", func(t *testing.T) {
		t.Parallel()
		oT, err := tf.client.CreateObjectType(ctx, uuid.Must(uuid.NewV4()), uniqueName("testedgeSerialTest"))
		assert.NoErr(t, err)
		eT, err := tf.client.CreateEdgeType(ctx, uuid.Must(uuid.NewV4()), oT.ID, oT.ID, uniqueName("testedgeSerialTest"), authz.Attributes{{
			Name:   "read",
			Direct: true,
		}})
		assert.NoErr(t, err)
		testObj1, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), oT.ID, uniqueName("testedgeSerialTest1"))
		assert.NoErr(t, err)

		testObj2Name := uniqueName("testedgeSerialTest2")
		testObj2, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), oT.ID, testObj2Name)
		assert.NoErr(t, err)

		testObj3, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), oT.ID, uniqueName("testedgeSerialTest3"))
		assert.NoErr(t, err)

		edge2, err := tf.client.CreateEdge(ctx, uuid.Must(uuid.NewV4()), testObj1.ID, testObj3.ID, eT.ID)
		assert.NoErr(t, err)

		// First try serially
		for range 5 {
			edge1, err := tf.client.CreateEdge(ctx, uuid.Must(uuid.NewV4()), testObj1.ID, testObj2.ID, eT.ID)
			assert.NoErr(t, err)

			edge3, err := tf.client.CreateEdge(ctx, uuid.Must(uuid.NewV4()), testObj2.ID, testObj3.ID, eT.ID)
			assert.NoErr(t, err)

			// Validate key for edge 1 by id and for full name (set by create)
			edgeS, err := tf.client.GetEdge(ctx, edge1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, edgeS, edge1)
			edgeS, err = tf.client.FindEdge(ctx, edge1.SourceObjectID, edge1.TargetObjectID, eT.ID)
			assert.NoErr(t, err)
			assert.Equal(t, edgeS, edge1)

			// Validate key for edge 3 by id and for full name (set by create)
			edgeS, err = tf.client.GetEdge(ctx, edge2.ID)
			assert.NoErr(t, err)
			assert.Equal(t, edgeS, edge2)
			edgeS, err = tf.client.FindEdge(ctx, edge2.SourceObjectID, edge2.TargetObjectID, eT.ID)
			assert.NoErr(t, err)
			assert.Equal(t, edgeS, edge2)

			// Validate key for edge 3 by id and for full name (set by create)
			edgeS, err = tf.client.GetEdge(ctx, edge3.ID)
			assert.NoErr(t, err)
			assert.Equal(t, edgeS, edge3)
			edgeS, err = tf.client.FindEdge(ctx, edge3.SourceObjectID, edge3.TargetObjectID, eT.ID)
			assert.NoErr(t, err)
			assert.Equal(t, edgeS, edge3)

			// Now go through make sure all possible collection keys are set so that we can validate they are cleared after a delete
			// Set key for obj1 to obj2 collection
			edgesS, err := tf.client.ListEdgesBetweenObjects(ctx, testObj1.ID, testObj2.ID)
			assert.NoErr(t, err)
			if len(edgesS) != 1 {
				assert.Equal(t, len(edgesS), 1)
			}
			// Set key for obj2 to obj3 collection
			edgesS, err = tf.client.ListEdgesBetweenObjects(ctx, testObj2.ID, testObj3.ID)
			assert.NoErr(t, err)
			assert.Equal(t, len(edgesS), 1)
			// Set key for obj1 to obj3 collection
			edgesS, err = tf.client.ListEdgesBetweenObjects(ctx, testObj1.ID, testObj3.ID)
			assert.NoErr(t, err)
			assert.Equal(t, len(edgesS), 1)
			// Set key for all edges on obj1 collection
			edges, err := tf.client.ListEdgesOnObject(ctx, testObj1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, len(edges.Data), 2)
			// Set key for all edges on obj2 collection
			edges, err = tf.client.ListEdgesOnObject(ctx, testObj2.ID)
			assert.NoErr(t, err)
			assert.Equal(t, len(edges.Data), 2)
			// Set key for all edges on obj1 collection
			edges, err = tf.client.ListEdgesOnObject(ctx, testObj3.ID)
			assert.NoErr(t, err)
			assert.Equal(t, len(edges.Data), 2)

			// Now go through make sure all possible path keys are set so that we can validate they are cleared after a delete
			resp, err := tf.client.CheckAttribute(ctx, testObj1.ID, testObj2.ID, "read")
			assert.NoErr(t, err)
			assert.True(t, resp.HasAttribute)
			resp, err = tf.client.CheckAttribute(ctx, testObj1.ID, testObj3.ID, "read")
			assert.NoErr(t, err)
			assert.True(t, resp.HasAttribute)
			resp, err = tf.client.CheckAttribute(ctx, testObj2.ID, testObj3.ID, "read")
			assert.NoErr(t, err)
			assert.True(t, resp.HasAttribute)
			resp, err = tf.client.CheckAttribute(ctx, testObj2.ID, testObj1.ID, "read")
			assert.NoErr(t, err)
			assert.False(t, resp.HasAttribute)
			resp, err = tf.client.CheckAttribute(ctx, testObj3.ID, testObj1.ID, "read")
			assert.NoErr(t, err)
			assert.False(t, resp.HasAttribute)

			// Delete object 2 this should clear out all the keys collections of edges above
			err = tf.client.DeleteObject(ctx, testObj2.ID)
			assert.NoErr(t, err)

			// Edge between 1 and 2 should should be gone
			_, err = tf.client.GetEdge(ctx, edge1.ID)
			if err == nil {
				assert.NotNil(t, err)
			}
			assert.ErrorIs(t, err, authz.ErrEdgeNotFound)

			// Edge between 1 and 3 should still exist
			edgeS, err = tf.client.GetEdge(ctx, edge2.ID)
			assert.NoErr(t, err)
			assert.Equal(t, edgeS, edge2)

			// Edge between 2 and 3 should should be gone
			_, err = tf.client.GetEdge(ctx, edge3.ID)
			assert.NotNil(t, err)
			assert.ErrorIs(t, err, authz.ErrEdgeNotFound)

			// Collection for obj1 to obj2 should be empty
			edgesS, err = tf.client.ListEdgesBetweenObjects(ctx, testObj1.ID, testObj2.ID)
			assert.NoErr(t, err)
			assert.Equal(t, len(edgesS), 0)
			// Collection for obj1 to obj3 should still have 1 edge
			edgesS, err = tf.client.ListEdgesBetweenObjects(ctx, testObj1.ID, testObj3.ID)
			assert.NoErr(t, err)
			assert.Equal(t, len(edgesS), 1)
			// Collection for obj2 to obj3 should be empty
			_, err = tf.client.ListEdgesBetweenObjects(ctx, testObj2.ID, testObj3.ID)
			assert.NotNil(t, err)
			assert.ErrorIs(t, err, authz.ErrObjectNotFound)

			// Collection for obj1 should still have 1 edge
			edges, err = tf.client.ListEdgesOnObject(ctx, testObj1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, len(edges.Data), 1)
			// Collection  shouldn't exist for obj2
			_, err = tf.client.ListEdgesOnObject(ctx, testObj2.ID)
			assert.NotNil(t, err)
			assert.ErrorIs(t, err, authz.ErrObjectNotFound)
			// Collection for obj3 should still have 1 edge
			edges, err = tf.client.ListEdgesOnObject(ctx, testObj3.ID)
			assert.NoErr(t, err)
			assert.Equal(t, len(edges.Data), 1)

			resp, err = tf.client.CheckAttribute(ctx, testObj1.ID, testObj2.ID, "read")
			assert.NoErr(t, err)
			assert.False(t, resp.HasAttribute)
			resp, err = tf.client.CheckAttribute(ctx, testObj1.ID, testObj3.ID, "read")
			assert.NoErr(t, err)
			assert.True(t, resp.HasAttribute)
			resp, err = tf.client.CheckAttribute(ctx, testObj2.ID, testObj3.ID, "read")
			assert.NoErr(t, err)
			assert.False(t, resp.HasAttribute)
			resp, err = tf.client.CheckAttribute(ctx, testObj2.ID, testObj1.ID, "read")
			assert.NoErr(t, err)
			assert.False(t, resp.HasAttribute)
			resp, err = tf.client.CheckAttribute(ctx, testObj3.ID, testObj1.ID, "read")
			assert.NoErr(t, err)
			assert.False(t, resp.HasAttribute)

			testObj2, err = tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), oT.ID, testObj2Name)
			assert.NoErr(t, err)
		}

		resp, err := tf.client.CheckAttribute(ctx, testObj1.ID, testObj3.ID, "read")
		assert.NoErr(t, err)
		assert.True(t, resp.HasAttribute)

		err = tf.client.DeleteEdge(ctx, edge2.ID)
		assert.NoErr(t, err)

		resp, err = tf.client.CheckAttribute(ctx, testObj1.ID, testObj3.ID, "read")
		assert.NoErr(t, err)
		assert.False(t, resp.HasAttribute)
	})
	t.Run("test_cache_edges_createdelete_parallel", func(t *testing.T) {
		t.Parallel()
		oT, err := tf.client.CreateObjectType(ctx, uuid.Must(uuid.NewV4()), uniqueName("testedgeParallelTest"))
		assert.NoErr(t, err)
		eT, err := tf.client.CreateEdgeType(ctx, uuid.Must(uuid.NewV4()), oT.ID, oT.ID, uniqueName("testedgeParallelTest"), authz.Attributes{{
			Name:   "read",
			Direct: true,
		}})
		assert.NoErr(t, err)
		testObj1, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), oT.ID, uniqueName("testedgeParallelTest1"))
		assert.NoErr(t, err)

		testObj2, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), oT.ID, uniqueName("testedgeParallelTest2"))
		assert.NoErr(t, err)

		wg := sync.WaitGroup{}

		// 	Test consistency of various keys across a delete of an edge
		for i := range 3 {
			wg.Add(1)

			go func(threadID int) {
				defer wg.Done()
				for range 40 {
					// This will either create a new object or return the object created by the other thread
					var edgeL *authz.Edge
					var err error
					retry := true

					for retry {
						retry = false
						edgeL, err = tf.client.CreateEdge(ctx, uuid.Nil, testObj1.ID, testObj2.ID, eT.ID, authz.IfNotExists())
						if err != nil && jsonclient.IsHTTPStatusConflict(err) {
							retry = true
						}
					}
					assert.NoErr(t, err)

					// Depending on what the other thread has done, the get can return no path, path with edgeL.ID or path with another edgeID
					// The purpose of the call is to set the cache key on success
					_, err = tf.client.CheckAttribute(ctx, testObj1.ID, testObj2.ID, "read")
					assert.NoErr(t, err)

					_, err = tf.client.ListEdgesOnObject(ctx, testObj1.ID)
					assert.NoErr(t, err)

					_, err = tf.client.ListEdgesBetweenObjects(ctx, testObj1.ID, testObj2.ID)
					assert.NoErr(t, err)

					// The delete will fail with ErrEdgeNotFound if the other thread has already deleted the edge
					retry = true
					for retry {
						retry = false
						err = tf.client.DeleteEdge(ctx, edgeL.ID)
						if err != nil && jsonclient.IsHTTPStatusConflict(err) {
							retry = true
						}
					}

					if err != nil {
						assert.ErrorIs(t, err, authz.ErrEdgeNotFound)
					}

					// The get can succeed if the other thread has recreated the edge after the delete but
					// it should never return the original edge since we delete it on this thread
					edgeR, err := tf.client.FindEdge(ctx, testObj1.ID, testObj2.ID, eT.ID)
					if err == nil {
						if edgeR.ID == edgeL.ID {
							assert.NotEqual(t, edgeR.ID, edgeL.ID)
						}
					} else {
						assert.ErrorIs(t, err, authz.ErrEdgeNotFound)
					}

					_, err = tf.client.GetEdge(ctx, edgeL.ID) // this should always fail because we delete edgeL on this thread
					assert.NotNil(t, err)

					resp, err := tf.client.CheckAttribute(ctx, testObj1.ID, testObj2.ID, "read")
					assert.NoErr(t, err)
					if resp.HasAttribute {
						if resp.Path[1].EdgeID == edgeL.ID {
							assert.NotEqual(t, resp.Path[1].EdgeID, edgeL.ID)
						}
					}

					edgesResp, err := tf.client.ListEdgesOnObject(ctx, testObj1.ID)
					assert.NoErr(t, err)
					if (len(edgesResp.Data) > 0) && (edgesResp.Data[0].ID == edgeL.ID) {
						assert.Equal(t, edgesResp.Data[0].ID, edgeL.ID)
					}

					edges, err := tf.client.ListEdgesBetweenObjects(ctx, testObj1.ID, testObj2.ID)
					assert.NoErr(t, err)
					if (len(edges) > 0) && (edges[0].ID == edgeL.ID) {
						assert.Equal(t, edges[0].ID, edgeL.ID)
					}
				}
			}(i)

		}
		wg.Wait()
	})

	t.Run("test_cache_edges_delete_serial", func(t *testing.T) {
		t.Parallel()
		oT, err := tf.client.CreateObjectType(ctx, uuid.Must(uuid.NewV4()), uniqueName("testedgeParallelTest"))
		assert.NoErr(t, err)
		eT, err := tf.client.CreateEdgeType(ctx, uuid.Must(uuid.NewV4()), oT.ID, oT.ID, uniqueName("testedgeParallelTest"), authz.Attributes{{
			Name:   "read",
			Direct: true,
		}})
		assert.NoErr(t, err)

		testObj2, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), oT.ID, uniqueName("testedgeParallelTest2"))
		assert.NoErr(t, err)

		objName := uniqueName("testobjLParallelTest")
		for range 40 {
			testObjL, err := tf.client.CreateObject(ctx, uuid.Nil, oT.ID, objName, authz.IfNotExists())
			assert.NoErr(t, err)

			edgeL, err := tf.client.CreateEdge(ctx, uuid.Nil, testObjL.ID, testObj2.ID, eT.ID)
			assert.NoErr(t, err)

			// Warm up all the cache keys that depend on this edge
			respA, err := tf.client.CheckAttribute(ctx, testObjL.ID, testObj2.ID, "read")
			assert.NoErr(t, err)
			assert.True(t, respA.HasAttribute)
			_, err = tf.client.ListEdgesOnObject(ctx, testObjL.ID)
			assert.NoErr(t, err)
			_, err = tf.client.ListEdgesOnObject(ctx, testObj2.ID)
			assert.NoErr(t, err)
			_, err = tf.client.ListEdgesBetweenObjects(ctx, testObjL.ID, testObj2.ID)
			assert.NoErr(t, err)

			// Deleting the object should delete the edges
			err = tf.client.DeleteObject(ctx, testObjL.ID)
			assert.NoErr(t, err)

			// These reads should always fail because we delete objL
			_, err = tf.client.FindEdge(ctx, testObjL.ID, testObj2.ID, eT.ID)
			assert.ErrorIs(t, err, authz.ErrEdgeNotFound)
			_, err = tf.client.GetEdge(ctx, edgeL.ID)
			assert.ErrorIs(t, err, authz.ErrEdgeNotFound)
			respA, err = tf.client.CheckAttribute(ctx, testObjL.ID, testObj2.ID, "read")
			assert.NoErr(t, err)
			assert.False(t, respA.HasAttribute)
			_, err = tf.client.ListEdgesOnObject(ctx, testObjL.ID)
			assert.ErrorIs(t, err, authz.ErrObjectNotFound)
			resp, err := tf.client.ListEdgesOnObject(ctx, testObj2.ID)
			assert.NoErr(t, err)
			assert.Equal(t, len(resp.Data), 0)
			_, err = tf.client.ListEdgesBetweenObjects(ctx, testObjL.ID, testObj2.ID)
			assert.ErrorIs(t, err, authz.ErrObjectNotFound)
		}
	})

	t.Run("test_cache_edges_delete_object_parallel", func(t *testing.T) {
		t.Parallel()
		t.Skip(`Disabling this test for now because our backend is not thread safe around edge creation and object deletion and under certain timing edge creation to a deleted object can succeed which causes this test to fail.
		TODO: Reenable it once fixed`)

		oT, err := tf.client.CreateObjectType(ctx, uuid.Must(uuid.NewV4()), uniqueName("testedgeParallelTest"))
		assert.NoErr(t, err)
		eT, err := tf.client.CreateEdgeType(ctx, uuid.Must(uuid.NewV4()), oT.ID, oT.ID, uniqueName("testedgeParallelTest"), authz.Attributes{{
			Name:   "read",
			Direct: true,
		}})
		assert.NoErr(t, err)

		testObj2, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), oT.ID, uniqueName("testedgeParallelTest2"))
		assert.NoErr(t, err)

		testObjName := uniqueName("testobjLParallelTest")
		wg := sync.WaitGroup{}

		// Test consistency of various keys for an edge across a delete of an object
		for i := range 3 {
			wg.Add(1)

			go func(threadID int) {
				defer wg.Done()
				for range 40 {
					// This will either create a new object or return the object created by the other thread
					var edgeL *authz.Edge
					var testObjL *authz.Object
					var err error
					retry := true

					for retry {
						retry = false
						testObjL, err = tf.client.CreateObject(ctx, uuid.Nil, oT.ID, testObjName, authz.IfNotExists())
						if err != nil && jsonclient.IsHTTPStatusConflict(err) {
							retry = true
						}
					}
					assert.NoErr(t, err)

					// Try to create an edge, this will succeed if the other thread(s) have not deleted the object and fail otherwise
					edgeL, err = tf.client.CreateEdge(ctx, uuid.Nil, testObjL.ID, testObj2.ID, eT.ID)
					_ = err // ignore error
					uclog.Infof(ctx, "Thread %v create edge %v with %v", threadID, edgeL, err)

					// The delete will fail if the other thread has already deleted the edge
					retry = true
					for retry {
						retry = false
						err = tf.client.DeleteObject(ctx, testObjL.ID)
						if err != nil && jsonclient.IsHTTPStatusConflict(err) {
							retry = true
						}
					}

					uclog.Infof(ctx, "Thread %v deleted %v with %v", threadID, testObjL.ID, err)
					if err != nil {
						if !jsonclient.IsHTTPNotFound(err) {
							assert.HTTPError(t, err, http.StatusNotFound)
						}
					}

					// These reads should always fail because we delete objL on this thread
					_, err = tf.client.FindEdge(ctx, testObjL.ID, testObj2.ID, eT.ID)
					if err != nil {
						if !jsonclient.IsHTTPNotFound(err) {
							assert.HTTPError(t, err, http.StatusNotFound)
						}
					}
					uclog.Infof(ctx, "Thread %v find edge %v with %v", threadID, edgeL, err)

					if edgeL != nil {
						_, err = tf.client.GetEdge(ctx, edgeL.ID)
						if !jsonclient.IsHTTPNotFound(err) {
							assert.HTTPError(t, err, http.StatusNotFound)
						}
						assert.NotNil(t, err)
					}
				}
			}(i)

		}

		wg.Wait()
	})
	t.Run("test_cache_edges_delete_edges_parallel", func(t *testing.T) {
		t.Parallel()
		t.Skip(`turn off test for now - there is some gremlin in the test that causes it to fail occasionally. TODO: fix this test`)

		oT, err := tf.client.CreateObjectType(ctx, uuid.Must(uuid.NewV4()), uniqueName("testedgeParallelTest"))
		assert.NoErr(t, err)
		eT, err := tf.client.CreateEdgeType(ctx, uuid.Must(uuid.NewV4()), oT.ID, oT.ID, uniqueName("testedgeParallelTest"), authz.Attributes{{
			Name:   "read",
			Direct: true,
		}})
		assert.NoErr(t, err)

		testObj2, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), oT.ID, uniqueName("testedgeParallelTest2"))
		assert.NoErr(t, err)

		testObjName := uniqueName("testobjLParallelTest")
		wg := sync.WaitGroup{}

		// 	Test consistency of collection, path, primary and secondary keys for an delete of all edges on an object
		for i := range 3 {
			wg.Add(1)

			go func(threadID int) {
				defer wg.Done()
				for range 40 {
					// This will either create a new object or return the object created by the other thread
					var edgeL *authz.Edge
					var testObj1 *authz.Object
					var err error
					retry := true

					// We create a new object because we don't want wait for tomstone TTL expiration from DeleteEdges
					for retry {
						retry = false
						testObj1, err = tf.client.CreateObject(ctx, uuid.Nil, oT.ID, testObjName, authz.IfNotExists())
						if err != nil && jsonclient.IsHTTPStatusConflict(err) {
							retry = true
						}
					}
					if err != nil {
						assert.NoErr(t, err)
					}
					retry = true
					for retry {
						retry = false
						edgeL, err = tf.client.CreateEdge(ctx, uuid.Nil, testObj1.ID, testObj2.ID, eT.ID, authz.IfNotExists())
						if err != nil && jsonclient.IsHTTPStatusConflict(err) {
							retry = true
						}
					}
					if err != nil {
						assert.HTTPError(t, err, http.StatusNotFound)
					}

					// Depending on what the other thread has done, the get can return no path, path with edgeL.ID or path with another edgeID
					// The purpose of the call is to set the cache key on success
					_, err = tf.client.CheckAttribute(ctx, testObj1.ID, testObj2.ID, "read")
					if err != nil {
						assert.HTTPError(t, err, http.StatusNotFound)
					}

					_, err = tf.client.ListEdgesOnObject(ctx, testObj1.ID)
					if err != nil {
						assert.HTTPError(t, err, http.StatusNotFound)
					}

					_, err = tf.client.ListEdgesOnObject(ctx, testObj2.ID)
					if err != nil {
						assert.HTTPError(t, err, http.StatusNotFound)
					}

					_, err = tf.client.ListEdgesBetweenObjects(ctx, testObj1.ID, testObj2.ID)
					if err != nil {
						assert.HTTPError(t, err, http.StatusNotFound)
					}

					// The delete will fail if the other thread has already deleted the edge
					retry = true
					for retry {
						retry = false
						err = tf.client.DeleteEdgesByObject(ctx, testObj1.ID)
						if err != nil && jsonclient.IsHTTPStatusConflict(err) {
							retry = true
						}
					}

					if err != nil {
						if !jsonclient.IsHTTPNotFound(err) {
							assert.HTTPError(t, err, http.StatusNotFound)
						}
					}

					// Because DeleteObject is not atomic we may get an edge created on thread as another thread deletes testObj1
					// This will cause DeleteEdgesByObject to fail with a 404 but the edge will still be there
					if err == nil {
						// The get can succeed if the other thread has recreated the edge after the delete but
						// it should never return the original edge since we delete it on this thread
						edgeR, err := tf.client.FindEdge(ctx, testObj1.ID, testObj2.ID, eT.ID)
						if err == nil && edgeL != nil {
							if edgeR.ID == edgeL.ID {
								assert.Equal(t, edgeR.ID != edgeL.ID, true)
							}
						} else {
							httpCode := jsonclient.GetHTTPStatusCode(err)
							assert.Equal(t, httpCode == http.StatusNotFound || httpCode == http.StatusConflict, true)
						}

						if edgeL != nil {
							_, err = tf.client.GetEdge(ctx, edgeL.ID) // this should always fail because we delete edgeL on this thread
							assert.NotNil(t, err)
						}

						resp, err := tf.client.CheckAttribute(ctx, testObj1.ID, testObj2.ID, "read")
						if err == nil && edgeL != nil {
							if resp.HasAttribute {
								if resp.Path[1].EdgeID == edgeL.ID {
									assert.Equal(t, resp.Path[1].EdgeID != edgeL.ID, true)
								}
							}
						} else {
							httpCode := jsonclient.GetHTTPStatusCode(err)
							if httpCode != http.StatusNotFound {
								assert.Equal(t, httpCode == http.StatusNotFound || httpCode == http.StatusConflict, true)
							}
						}

						edgesResp, err := tf.client.ListEdgesOnObject(ctx, testObj1.ID)
						if (err == nil && edgeL != nil && len(edgesResp.Data) > 0) && (edgesResp.Data[0].ID == edgeL.ID) {
							assert.Equal(t, edgesResp.Data[0].ID != edgeL.ID, true)
						} else if err != nil {
							assert.HTTPError(t, err, http.StatusNotFound)
						}

						edgesResp, err = tf.client.ListEdgesOnObject(ctx, testObj2.ID)
						if (err == nil && edgeL != nil && len(edgesResp.Data) > 0) && (edgesResp.Data[0].ID == edgeL.ID) {
							assert.Equal(t, edgesResp.Data[0].ID != edgeL.ID, true)
						} else if err != nil {
							assert.HTTPError(t, err, http.StatusNotFound)
						}

						edges, err := tf.client.ListEdgesBetweenObjects(ctx, testObj1.ID, testObj2.ID)
						if (err == nil && edgeL != nil && len(edges) > 0) && (edges[0].ID == edgeL.ID) {
							assert.Equal(t, edges[0].ID != edgeL.ID, true)
						} else if err != nil {
							assert.HTTPError(t, err, http.StatusNotFound)
						}
					}
					// The delete will fail if the other thread has already deleted the edge
					retry = true
					for retry {
						retry = false
						err = tf.client.DeleteObject(ctx, testObj1.ID)
						if err != nil && jsonclient.IsHTTPStatusConflict(err) {
							retry = true
						}
					}
					if err != nil {
						if !jsonclient.IsHTTPNotFound(err) {
							assert.HTTPError(t, err, http.StatusNotFound)
						}
					}
				}
			}(i)

		}
		wg.Wait()

	})

	t.Run("test_get_edges_byid", func(t *testing.T) {
		t.Parallel()
		et := uuid.Must(uuid.NewV4())
		_, err := tf.client.CreateEdgeType(ctx, et, authz.GroupObjectTypeID, authz.GroupObjectTypeID, uniqueName("edge_type"), nil)
		assert.NoErr(t, err)

		rootObj, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), authz.GroupObjectTypeID, uniqueName("rootobj"))
		assert.NoErr(t, err)

		obj, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), authz.GroupObjectTypeID, uniqueName("testobj"))
		assert.NoErr(t, err)

		edge, err := tf.client.CreateEdge(ctx, uuid.Must(uuid.NewV4()), rootObj.ID, obj.ID, et)
		assert.NoErr(t, err)

		foundEdge, err := tf.client.GetEdge(ctx, edge.ID)
		assert.NoErr(t, err)
		assert.Equal(t, edge.ID, foundEdge.ID)
		assert.Equal(t, edge.SourceObjectID, rootObj.ID)
		assert.Equal(t, edge.TargetObjectID, obj.ID)
	})

	t.Run("test_nonorg_usercreation", func(t *testing.T) {
		t.Parallel()
		user, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), authz.UserObjectTypeID, uniqueName("testuser"))
		assert.NoErr(t, err)

		obj, err := tf.client.GetObject(ctx, user.ID)
		assert.NoErr(t, err)

		// Verify that user objects created on a tenant w/o organizations enabled are assigned the tenant's company ID as org ID
		assert.Equal(t, obj.OrganizationID, tf.tenant.CompanyID)
	})

	// TODO (sgarrity 6/23): replace this with OpenAPI or equiv autogenerated tests that cover
	// all of our endpoints for this type of thing, rather than just one to ensure the plumbing is working

	t.Run("test_friendlyerrormessage", func(t *testing.T) {
		t.Parallel()
		ot, err := tf.client.CreateObjectType(ctx, uuid.Must(uuid.NewV4()), "ot")
		assert.NoErr(t, err)
		o, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), ot.ID, "o")
		assert.NoErr(t, err)
		et, err := tf.client.CreateEdgeType(ctx, uuid.Must(uuid.NewV4()), ot.ID, ot.ID, "et", nil)
		assert.NoErr(t, err)

		e, err := tf.client.CreateEdge(ctx, uuid.Must(uuid.NewV4()), o.ID, o.ID, et.ID)
		assert.NoErr(t, err)
		assert.NoErr(t, tf.client.DeleteEdge(ctx, e.ID))

		err = tf.client.DeleteEdge(ctx, e.ID)
		assert.ErrorIs(t, err, authz.ErrEdgeNotFound)
		assert.Contains(t, err.Error(), "edge not found")
	})
}

func newClientWithAuth(t *testing.T, url string) *authz.Client {
	jwt := uctest.CreateJWT(t, oidc.UCTokenClaims{}, url)
	c, err := authz.NewCustomClient(authz.DefaultObjTypeTTL, authz.DefaultEdgeTypeTTL, authz.DefaultObjTTL, authz.DefaultEdgeTTL, url, authz.JSONClient(jsonclient.HeaderAuthBearer(jwt)))
	assert.NoErr(t, err)
	return c
}

func makeObjectTypes(n int) []authz.ObjectType {
	ids := testhelpers.MakeSortedUUIDs(n)
	ots := make([]authz.ObjectType, n)
	for i := range n {
		ots[i] = authz.ObjectType{
			BaseModel: ucdb.NewBaseWithID(ids[i]),
			TypeName:  fmt.Sprintf("obj_type_%d", i),
		}
	}
	return ots
}

func makeEdgeTypes(n int) []authz.EdgeType {
	ids := testhelpers.MakeSortedUUIDs(n)
	ets := make([]authz.EdgeType, n)
	for i := range n {
		ets[i] = authz.EdgeType{
			BaseModel: ucdb.NewBaseWithID(ids[i]),
		}
	}
	return ets
}

func enumerateObjects(ctx context.Context, t *testing.T, c *authz.Client, limit int, objType uuid.UUID, orgID uuid.UUID) []authz.Object {
	cursor := pagination.CursorBegin
	objs := make([]authz.Object, 0)
	for {
		os, err := c.ListObjects(ctx, authz.Pagination(pagination.Limit(limit), pagination.StartingAfter(cursor)), authz.OrganizationID(orgID))
		assert.NoErr(t, err)
		assert.True(t, len(os.Data) <= limit, assert.Must(), assert.Errorf("ListObjects returned more objects (%d) than limit (%d)", len(os.Data), limit))
		for _, o := range os.Data {
			if o.TypeID == objType || objType.IsNil() {
				objs = append(objs, o)
			}
		}
		if !os.HasNext {
			break
		}
		cursor = os.Next
	}
	return objs
}

func validateClientAndServerObjects(t *testing.T, clientObjs, serverObjs []authz.Object) {
	assert.Equal(t, len(serverObjs), len(clientObjs),
		assert.Must(),
		assert.Errorf("client object array size (%d) doesn't match server object array size (%d)", len(clientObjs), len(serverObjs)))

	serverObjsByID := map[uuid.UUID]authz.Object{}
	for _, o := range serverObjs {
		serverObjsByID[o.ID] = o
	}
	for i := range clientObjs {
		serverObj, found := serverObjsByID[clientObjs[i].ID]
		assert.True(t, found, assert.Errorf("client obj[%d] (%+v) not found on server", i, clientObjs[i]), assert.Must())
		assert.True(t, cmp.Equal(clientObjs[i], serverObj), assert.Errorf("client object[%d] (%+v) doesn't match server object (%+v)", i, clientObjs[i], serverObj), assert.Must())

		if i > 0 {
			assert.True(t, clientObjs[i].ID.String() > clientObjs[i-1].ID.String(),
				assert.Must(),
				assert.Errorf("client object[%d]'s ID (%v) isn't greater than object[%d]'s ID (%v)", i, clientObjs[i].ID, i-1, clientObjs[i-1].ID))
		}
	}
}

func enumerateEdges(ctx context.Context, t *testing.T, c *authz.Client, limit int) ([]authz.Edge, int) {
	cursor := pagination.CursorBegin
	edges := make([]authz.Edge, 0)
	var calls int
	for {
		es, err := c.ListEdges(ctx, authz.Pagination(pagination.Limit(limit), pagination.StartingAfter(cursor)))
		assert.NoErr(t, err)
		calls++
		assert.True(t, len(es.Data) <= limit,
			assert.Must(),
			assert.Errorf("ListEdges returned more edges (%d) than limit (%d)", len(es.Data), limit))
		edges = append(edges, es.Data...)
		if !es.HasNext {
			break
		}

		cursor = es.Next
	}
	return edges, calls
}

func enumerateEdgesOnObject(ctx context.Context, t *testing.T, c *authz.Client, objID uuid.UUID, limit int) ([]authz.Edge, int) {
	cursor := pagination.CursorBegin
	edges := make([]authz.Edge, 0)
	var calls int
	for {
		es, err := c.ListEdgesOnObject(ctx, objID, authz.Pagination(pagination.Limit(limit), pagination.StartingAfter(cursor)))
		assert.NoErr(t, err)

		calls++
		assert.True(t, len(es.Data) <= limit,
			assert.Must(),
			assert.Errorf("ListEdgesOnObject returned more edges (%d) than limit (%d)", len(es.Data), limit))
		edges = append(edges, es.Data...)
		if !es.HasNext {
			break
		}

		cursor = es.Next
	}
	return edges, calls
}

func validateClientAndServerEdges(clientEdges, serverEdges []authz.Edge) error {
	if len(serverEdges) != len(clientEdges) {
		return ucerr.Errorf("client edge array size (%d) doesn't match server edge array size (%d)", len(clientEdges), len(serverEdges))
	}
	serverEdgesByID := map[uuid.UUID]authz.Edge{}
	for _, e := range serverEdges {
		serverEdgesByID[e.ID] = e
	}
	for i := range clientEdges {
		serverEdge, found := serverEdgesByID[clientEdges[i].ID]
		if !found {
			return ucerr.Errorf("client edge[%d] (%+v) not found on server", i, clientEdges[i])
		}
		if !cmp.Equal(clientEdges[i], serverEdge) {
			return ucerr.Errorf("client edge[%d] (%+v) doesn't match server edge (%+v)", i, clientEdges[i], serverEdge)
		}
		if i > 0 {
			if clientEdges[i].ID.String() <= clientEdges[i-1].ID.String() {
				return ucerr.Errorf("client edge[%d]'s ID (%v) isn't greater than edge[%d]'s ID (%v)", i, clientEdges[i].ID, i-1, clientEdges[i-1].ID)
			}
		}
	}
	return nil
}

func TestObjectTypePagination(t *testing.T) {
	// t.Parallel() - don't run this test in parallel since it uses flush
	ctx := context.Background()
	const n = pagination.DefaultLimit*2 - 1 // not divisible by page length
	serverObjs := makeObjectTypes(n)
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodGet)
		calls++
		pager, err := pagination.NewPaginatorFromRequest(r)
		assert.NoErr(t, err)

		cursor := pager.GetCursor()
		if cursor == pagination.CursorBegin {
			cursor = pagination.Cursor(fmt.Sprintf("id:%v", uuid.Nil))
		}

		firstElem := len(serverObjs)
		for i := range serverObjs {
			if fmt.Sprintf("id:%v", serverObjs[i].ID) > string(cursor) {
				firstElem = i
				break
			}
		}
		lastElem := firstElem + pager.GetLimit()
		nextCursor := pagination.CursorEnd
		if lastElem >= len(serverObjs) {
			lastElem = len(serverObjs)
		} else {
			nextCursor = pagination.Cursor(fmt.Sprintf("id:%v", serverObjs[lastElem-1].ID))
		}
		jsonapi.Marshal(w, authz.ListObjectTypesResponse{
			Data: serverObjs[firstElem:lastElem],
			ResponseFields: pagination.ResponseFields{
				HasNext: nextCursor != pagination.CursorEnd,
				Next:    nextCursor,
			},
		})
	}))

	defer server.Close()
	c := newClientWithAuth(t, server.URL)
	ots, err := c.ListObjectTypes(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, len(ots), n)
	assert.Equal(t, calls, 2)

	// Make sure we hit the cache for next get
	ots, err = c.ListObjectTypes(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, len(ots), 99)
	assert.Equal(t, calls, 2)

	err = c.FlushCache()
	assert.NoErr(t, err)

	// Ensure degenerate case (empty array) works
	serverObjs = makeObjectTypes(0)
	ots, err = c.ListObjectTypes(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, len(ots), 0)
	assert.Equal(t, calls, 3)

}

func TestEdgeTypePagination(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	const n = pagination.DefaultLimit * 2 // evenly divisible by page length
	calls := 0
	serverObjs := makeEdgeTypes(n)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodGet)
		calls++
		pager, err := pagination.NewPaginatorFromRequest(r)
		assert.NoErr(t, err)

		cursor := pager.GetCursor()
		if cursor == pagination.CursorBegin {
			cursor = pagination.Cursor(fmt.Sprintf("id:%v", uuid.Nil))
		}

		firstElem := len(serverObjs)
		for i := range serverObjs {
			if fmt.Sprintf("id:%v", serverObjs[i].ID) > string(cursor) {
				firstElem = i
				break
			}
		}
		lastElem := firstElem + pager.GetLimit()
		nextCursor := pagination.CursorEnd
		if lastElem >= len(serverObjs) {
			lastElem = len(serverObjs)
		} else {
			nextCursor = pagination.Cursor(fmt.Sprintf("id:%v", serverObjs[lastElem-1].ID))
		}
		jsonapi.Marshal(w, authz.ListEdgeTypesResponse{
			Data: serverObjs[firstElem:lastElem],
			ResponseFields: pagination.ResponseFields{
				HasNext: nextCursor != pagination.CursorEnd,
				Next:    nextCursor,
			},
		})
	}))

	defer server.Close()
	c := newClientWithAuth(t, server.URL)
	ots, err := c.ListEdgeTypes(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, len(ots), n)
	assert.Equal(t, calls, 2)
}

func TestOrgCreation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tf := newTestFixture(t, testhelpers.UseOrganizations())
	// test invalid region
	testOrgName := uniqueName("testorg")
	_, err := tf.client.CreateOrganization(ctx, uuid.Nil, testOrgName, "us-west-2")
	assert.NotNil(t, err)

	// create valid organization
	org, err := tf.client.CreateOrganization(ctx, uuid.Nil, testOrgName, "aws-us-west-2")
	assert.NoErr(t, err)
	assert.Equal(t, org.Name, testOrgName)

	// check for existence of the login app
	objs := enumerateObjects(ctx, t, tf.client, 100, authz.LoginAppObjectTypeID, org.ID)
	assert.Equal(t, len(objs), 1)

	// create a new edge type for this organization
	_, err = tf.client.CreateEdgeType(ctx, uuid.Nil, authz.GroupObjectTypeID, authz.GroupObjectTypeID, uniqueName("testorg_edge_type"), nil, authz.OrganizationID(org.ID))
	assert.NoErr(t, err)
	ets, err := tf.client.ListEdgeTypes(ctx, authz.OrganizationID(org.ID))
	assert.NoErr(t, err)
	assert.Equal(t, len(ets), 1)

	// create an object in this organization
	_, err = tf.client.CreateObject(ctx, uuid.Nil, authz.UserObjectTypeID, uniqueName("testorg_obj"), authz.OrganizationID(org.ID))
	assert.NoErr(t, err)
	objs = enumerateObjects(ctx, t, tf.client, 100, authz.UserObjectTypeID, org.ID)
	assert.Equal(t, len(objs), 1)

	// create a second organization
	testOrg2Name := uniqueName("testorg2")
	org2, err := tf.client.CreateOrganization(ctx, uuid.Nil, testOrg2Name, "aws-us-west-2")
	assert.NoErr(t, err)
	assert.Equal(t, org2.Name, testOrg2Name)

	// validate that the second organization can't see the first organization's objects or edge type
	objs = enumerateObjects(ctx, t, tf.client, 100, authz.UserObjectTypeID, org2.ID)
	assert.Equal(t, len(objs), 0)
	ets, err = tf.client.ListEdgeTypes(ctx, authz.OrganizationID(org2.ID))
	assert.NoErr(t, err)
	assert.Equal(t, len(ets), 0)

	// test IfNotExists
	org3, err := tf.client.CreateOrganization(ctx, uuid.Nil, testOrg2Name, "aws-us-west-2", authz.IfNotExists())
	assert.NoErr(t, err)
	assert.Equal(t, org3.Name, testOrg2Name)
	assert.Equal(t, org3.ID, org2.ID)

	org3, err = tf.client.CreateOrganization(ctx, org3.ID, testOrg2Name, "aws-us-west-2", authz.IfNotExists())
	assert.NoErr(t, err)
	assert.Equal(t, org3.Name, testOrg2Name)
	assert.Equal(t, org3.ID, org2.ID)

	_, err = tf.client.CreateOrganization(ctx, uuid.Must(uuid.NewV4()), testOrg2Name, "aws-us-west-2", authz.IfNotExists())
	assert.NotNil(t, err)

	// https://usercloudsworkspace.slack.com/archives/C02V94GDD88/p1700697323062339
	ot, err := tf.client.CreateObjectType(ctx, uuid.Must(uuid.NewV4()), uniqueName("ot"))
	assert.NoErr(t, err)
	o, err := tf.client.CreateObject(ctx, uuid.Must(uuid.NewV4()), ot.ID, uniqueName("o"))
	assert.NoErr(t, err)
	_, err = tf.client.CreateOrganization(ctx, o.ID, uniqueName("org"), "aws-us-west-2")
	assert.NotNil(t, err, assert.Must())

	// test GetOrganizationForName here because we've done all the setup anyway :)
	org4, err := tf.client.GetOrganizationForName(ctx, testOrgName)
	assert.NoErr(t, err)
	assert.Equal(t, org4.ID, org.ID)

	org5, err := tf.client.GetOrganizationForName(ctx, testOrg2Name)
	assert.NoErr(t, err)
	assert.Equal(t, org5.ID, org2.ID)

	// test with no UUID but conflicting name
	_, err = tf.client.CreateOrganization(ctx, uuid.Nil, testOrgName, "aws-us-west-2")
	assert.NotNil(t, err, assert.Must())
	assert.Contains(t, err.Error(), "organization already exists")
}
