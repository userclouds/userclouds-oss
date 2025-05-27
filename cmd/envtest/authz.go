package main

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/async"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const (
	poolSizeAuthZ         int = 50
	resourcePoolSizeAuthZ int = 25
	commonPoolAuthzOps    int = 10
)

var objectTypeNames = []string{"user", "application", "role", "resource_name"}
var edgeTypeNames = []string{"user_role_assignment", "parent_resource", "entity_deleter", "entity_reader", "entity_creator", "entity_updator"}

func authzCommonPoolTest(ctx context.Context, threadNum int, azc *authz.Client, objects []uuid.UUID, objectTypes []uuid.UUID) {
	testStart := time.Now().UTC()
	uclog.Debugf(ctx, "AuthZ: Starting common pool test for thread %d", threadNum)

	// Check objects/types from common pool
	for range commonPoolAuthzOps {
		oi := rand.Intn(len(objects))

		uclog.Debugf(ctx, "Authz    : Thread %2d testing common pool object index %3d, id %v, type %v", threadNum, oi, objects[oi], objectTypes[oi])

		gotObj, err := azc.GetObject(ctx, objects[oi])
		if err != nil || gotObj.ID != objects[oi] || gotObj.TypeID != objectTypes[oi] || *gotObj.Alias != objects[oi].String() {
			logErrorf(ctx, err, "Authz: Thread %2d GetObject failed from common pool: %v", threadNum, err)
		}

		gotObj, err = azc.GetObjectForName(ctx, objectTypes[oi], objects[oi].String())
		if err != nil || gotObj.ID != objects[oi] || gotObj.TypeID != objectTypes[oi] || *gotObj.Alias != objects[oi].String() {
			logErrorf(ctx, err, "Authz: Thread %2dGetObjectForName failed from common pool: %v", threadNum, err)
		}

		gotType, err := azc.GetObjectType(ctx, objectTypes[oi])
		if err != nil || gotType.ID != objectTypes[oi] || gotType.TypeName != objectTypes[oi].String() {
			logErrorf(ctx, err, "Authz: Thread %2d GetObjectType failed for common pool: %v", threadNum, err)
		}
	}

	wallTime := time.Now().UTC().Sub(testStart)
	uclog.Debugf(ctx, "AuthZ: Finished common pool test for thread %d with ops %d in %v", threadNum, 3*commonPoolAuthzOps, wallTime)
}

func authzPerThreadTest(ctx context.Context, threadNum int, azc *authz.Client, objTypes map[string]uuid.UUID, edgeTypes map[string]uuid.UUID) {
	testStart := time.Now().UTC()
	uclog.Debugf(ctx, "AuthZ: Starting per thread objects test for thread %d", threadNum)

	// Create some resources
	var resources = make([]uuid.UUID, resourcePoolSizeAuthZ)
	for i := range resources {
		resources[i] = uuid.Must(uuid.NewV4())
		gotObj, err := azc.CreateObject(ctx, resources[i], objTypes["resource_name"], resources[i].String())
		if err != nil || gotObj.ID != resources[i] || gotObj.TypeID != objTypes["resource_name"] || *gotObj.Alias != resources[i].String() {
			logErrorf(ctx, err, "Authz: Thread %2d CreateObject failed for resource object: %v", threadNum, err)
		}
	}
	// Create folders
	folderSize := 5
	var pResources = make([]uuid.UUID, len(resources)/5)
	for i := range pResources {
		pResources[i] = resources[i*folderSize]
		for j := i*folderSize + 1; j < (i+1)*folderSize; j++ {
			gotEdge, err := azc.CreateEdge(ctx, uuid.Must(uuid.NewV4()), pResources[i], resources[j], edgeTypes["parent_resource"])
			if err != nil || gotEdge.EdgeTypeID != edgeTypes["parent_resource"] {
				logErrorf(ctx, err, "Authz: Thread %2d CreateEdge failed for folder edge: %v", threadNum, err)
			}
		}
	}
	// Create some roles
	var readerRoles = make([]uuid.UUID, len(pResources))
	var adminRoles = make([]uuid.UUID, len(pResources))
	for i := range pResources {
		readerRoles[i] = uuid.Must(uuid.NewV4())
		gotObj, err := azc.CreateObject(ctx, readerRoles[i], objTypes["role"], readerRoles[i].String())
		if err != nil || gotObj.ID != readerRoles[i] || gotObj.TypeID != objTypes["role"] || *gotObj.Alias != readerRoles[i].String() {
			logErrorf(ctx, err, "Authz: Thread %2d CreateObject failed for role object: %v", threadNum, err)
		}
		// Create entity_reader edge between the role and the parent folder
		gotEdge, err := azc.CreateEdge(ctx, uuid.Must(uuid.NewV4()), readerRoles[i], pResources[i], edgeTypes["entity_reader"])
		if err != nil || gotEdge.EdgeTypeID != edgeTypes["entity_reader"] {
			logErrorf(ctx, err, "Authz: Thread %2d CreateEdge failed for folder edge: %v", threadNum, err)
		}

		adminRoles[i] = uuid.Must(uuid.NewV4())
		gotObj, err = azc.CreateObject(ctx, adminRoles[i], objTypes["role"], adminRoles[i].String())
		if err != nil || gotObj.ID != adminRoles[i] || gotObj.TypeID != objTypes["role"] || *gotObj.Alias != adminRoles[i].String() {
			logErrorf(ctx, err, "Authz: Thread %2d CreateObject failed for role object: %v", threadNum, err)
		}

		// Create entity_delete, entity_create edges between the role and the parent folder
		gotEdge, err = azc.CreateEdge(ctx, uuid.Must(uuid.NewV4()), adminRoles[i], pResources[i], edgeTypes["entity_deleter"])
		if err != nil || gotEdge.EdgeTypeID != edgeTypes["entity_deleter"] {
			logErrorf(ctx, err, "Authz: Thread %2d CreateEdge failed for entity_delete edge: %v", threadNum, err)
		}
		gotEdge, err = azc.CreateEdge(ctx, uuid.Must(uuid.NewV4()), adminRoles[i], pResources[i], edgeTypes["entity_creator"])
		if err != nil || gotEdge.EdgeTypeID != edgeTypes["entity_creator"] {
			logErrorf(ctx, err, "Authz: Thread %2d CreateEdge failed for entity_create edge: %v", threadNum, err)
		}
	}
	// Create some users (authZ object)
	var users = make([]uuid.UUID, 2*len(pResources))
	for i := range users {
		users[i] = uuid.Must(uuid.NewV4())
		gotObj, err := azc.CreateObject(ctx, users[i], objTypes["user"], users[i].String())
		if err != nil || gotObj.ID != users[i] || gotObj.TypeID != objTypes["user"] || *gotObj.Alias != users[i].String() {
			logErrorf(ctx, err, "Authz: Thread %2d CreateObject failed for user object: %v", threadNum, err)
		}
	}
	// Give users different types of roles
	for i := range pResources {
		// Create user_role_assignment edge for readers
		gotEdge, err := azc.CreateEdge(ctx, uuid.Must(uuid.NewV4()), users[i], readerRoles[i], edgeTypes["user_role_assignment"])
		if err != nil || gotEdge.EdgeTypeID != edgeTypes["user_role_assignment"] {
			logErrorf(ctx, err, "Authz: Thread %2d CreateEdge failed for entity_delete edge: %v", threadNum, err)
		}
		// Create user_role_assignment edge for admins
		gotEdge, err = azc.CreateEdge(ctx, uuid.Must(uuid.NewV4()), users[i+len(pResources)], adminRoles[i], edgeTypes["user_role_assignment"])
		if err != nil || gotEdge.EdgeTypeID != edgeTypes["user_role_assignment"] {
			logErrorf(ctx, err, "Authz: Thread %2d CreateEdge failed for entity_create edge: %v", threadNum, err)
		}
	}
	// Check if the check attribute works as expected
	uclog.Debugf(ctx, "AuthZ: Starting check_attibute calls per thread objects test for thread %d will make %d calls", threadNum, len(pResources)*folderSize*5)
	for i := range pResources {
		for j := i * folderSize; j < (i+1)*folderSize; j++ {
			checkAttribute(ctx, threadNum, azc, users[i], resources[j], "read", true)
			checkAttribute(ctx, threadNum, azc, users[i+len(pResources)], resources[j], "delete", true)
			checkAttribute(ctx, threadNum, azc, users[i+len(pResources)], resources[j], "create", true)

			checkAttribute(ctx, threadNum, azc, users[i], resources[j], "delete", false)
			checkAttribute(ctx, threadNum, azc, users[i+len(pResources)], resources[j], "read", false)

			checkAttribute(ctx, threadNum, azc, users[i], resources[j], "read", true)
			checkAttribute(ctx, threadNum, azc, users[i+len(pResources)], resources[j], "delete", true)
			checkAttribute(ctx, threadNum, azc, users[i+len(pResources)], resources[j], "create", true)

			checkAttribute(ctx, threadNum, azc, users[i], resources[j], "delete", false)
			checkAttribute(ctx, threadNum, azc, users[i+len(pResources)], resources[j], "read", false)

			checkAttribute(ctx, threadNum, azc, users[i], resources[j], "read", true)
			checkAttribute(ctx, threadNum, azc, users[i+len(pResources)], resources[j], "delete", true)
			checkAttribute(ctx, threadNum, azc, users[i+len(pResources)], resources[j], "create", true)

			checkAttribute(ctx, threadNum, azc, users[i], resources[j], "delete", false)
			checkAttribute(ctx, threadNum, azc, users[i+len(pResources)], resources[j], "read", false)
		}
	}
	uclog.Debugf(ctx, "AuthZ: Completed check_attibute calls per thread objects test for thread %d", threadNum)

	// Cleanup user/roles/object this will cause all the edges to be deleted
	allObjects := users
	allObjects = append(allObjects, adminRoles...)
	allObjects = append(allObjects, readerRoles...)
	allObjects = append(allObjects, resources...)
	for i := range allObjects {
		if err := azc.DeleteObject(ctx, allObjects[i]); err != nil {
			logErrorf(ctx, err, "Authz: Thread %2d DeleteObject failed for authz clean up: %v", threadNum, err)
		}
	}

	wallTime := time.Now().UTC().Sub(testStart)
	uclog.Debugf(ctx, "AuthZ: Finished per thread objects test for thread %d in %v", threadNum, wallTime)
}

func checkAttribute(ctx context.Context, threadNum int, azc *authz.Client, user uuid.UUID, resource uuid.UUID, attribute string, expectValue bool) {
	resp, err := azc.CheckAttribute(ctx, user, resource, attribute)
	if err != nil {
		logErrorf(ctx, err, "Authz: Thread %2d CheckAttribute failed for user %v on resource %v : %v", threadNum, err, user, resource)
	}
	if resp == nil || resp.HasAttribute != expectValue {
		if resp != nil {
			for i := range resp.Path {
				uclog.Errorf(ctx, "path %d: %v", i, resp.Path[i])
			}
		}
		err := ucerr.Errorf("Authz: Thread %2d CheckAttribute returned unexpected value for user %v on resource %v expected %v", threadNum, user, resource, expectValue)
		logErrorf(ctx, err, "Authz: Thread %2d CheckAttribute returned unexpected value for user %v on resource %v expected %v", threadNum, user, resource, expectValue)
	}
}

func authZTypeSetup(ctx context.Context, azc *authz.Client) (map[string]uuid.UUID, map[string]uuid.UUID) {
	testStart := time.Now().UTC()
	uclog.Debugf(ctx, "AuthZ: Starting setup of common types")
	objectTypes := make([]uuid.UUID, len(objectTypeNames))
	objectTypeMap := make(map[string]uuid.UUID)

	for i := range objectTypeNames {
		objectTypes[i] = uuid.Must(uuid.NewV4())
		objectTypeMap[objectTypeNames[i]] = objectTypes[i]
		gotType, err := azc.CreateObjectType(ctx, objectTypes[i], objectTypeNames[i])
		if err != nil || gotType.ID != objectTypes[i] || gotType.TypeName != objectTypeNames[i] {
			logErrorf(ctx, err, "AuthZ: CreateObjectType failed for per thread types: %v", err)
		}
	}

	edgeAttributesDirect := []authz.Attribute{{Name: "delete", Direct: true}, {Name: "read", Direct: true}, {Name: "create", Direct: true}, {Name: "update", Direct: true}}
	edgeAttributesInherit := []authz.Attribute{{Name: "delete", Inherit: true}, {Name: "read", Inherit: true}, {Name: "create", Inherit: true}, {Name: "update", Inherit: true}}
	edgeAttributesPropagate := []authz.Attribute{{Name: "delete", Propagate: true}, {Name: "read", Propagate: true}, {Name: "create", Propagate: true}, {Name: "update", Propagate: true}}
	edgeTypesDef := []authz.EdgeType{
		{BaseModel: ucdb.NewBase(), TypeName: edgeTypeNames[0], SourceObjectTypeID: objectTypeMap["user"], TargetObjectTypeID: objectTypeMap["role"], Attributes: edgeAttributesInherit},
		{BaseModel: ucdb.NewBase(), TypeName: edgeTypeNames[1], SourceObjectTypeID: objectTypeMap["resource_name"], TargetObjectTypeID: objectTypeMap["resource_name"], Attributes: edgeAttributesPropagate},
		{BaseModel: ucdb.NewBase(), TypeName: edgeTypeNames[2], SourceObjectTypeID: objectTypeMap["role"], TargetObjectTypeID: objectTypeMap["resource_name"], Attributes: []authz.Attribute{edgeAttributesDirect[0]}},
		{BaseModel: ucdb.NewBase(), TypeName: edgeTypeNames[3], SourceObjectTypeID: objectTypeMap["role"], TargetObjectTypeID: objectTypeMap["resource_name"], Attributes: []authz.Attribute{edgeAttributesDirect[1]}},
		{BaseModel: ucdb.NewBase(), TypeName: edgeTypeNames[4], SourceObjectTypeID: objectTypeMap["role"], TargetObjectTypeID: objectTypeMap["resource_name"], Attributes: []authz.Attribute{edgeAttributesDirect[2]}},
		{BaseModel: ucdb.NewBase(), TypeName: edgeTypeNames[5], SourceObjectTypeID: objectTypeMap["role"], TargetObjectTypeID: objectTypeMap["resource_name"], Attributes: []authz.Attribute{edgeAttributesDirect[3]}},
	}
	edgeTypes := make([]uuid.UUID, len(edgeTypeNames))
	edgeTypeMap := make(map[string]uuid.UUID, len(edgeTypeNames))
	for i := range edgeTypesDef {
		edgeTypes[i] = edgeTypesDef[i].ID
		edgeTypeMap[edgeTypeNames[i]] = edgeTypes[i]
		gotType, err := azc.CreateEdgeType(ctx, edgeTypesDef[i].ID, edgeTypesDef[i].SourceObjectTypeID, edgeTypesDef[i].TargetObjectTypeID, edgeTypesDef[i].TypeName, edgeTypesDef[i].Attributes)

		if err != nil || gotType.ID != edgeTypes[i] || gotType.TypeName != edgeTypesDef[i].TypeName {
			logErrorf(ctx, err, "AuthZ: CreateEdgeType failed for per thread types: %v", err)
		}
	}

	wallTime := time.Now().UTC().Sub(testStart)
	uclog.Debugf(ctx, "AuthZ: Finished setup of common types in %v", wallTime)
	return objectTypeMap, edgeTypeMap
}
func authZTypeCleanup(ctx context.Context, threadNum int, azc *authz.Client) {
	testStart := time.Now().UTC()
	uclog.Debugf(ctx, "AuthZ: Starting clean up thread %d", threadNum)
	deleteCount := 0

	oTypes, err := azc.ListObjectTypes(ctx)
	if err != nil {
		logErrorf(ctx, err, "AuthZ: ListObjectTypes failed authz cleanup: %v", err)

	}
	uclog.Debugf(ctx, "AuthZ: Read %d object types", len(oTypes))
	for _, oT := range oTypes {
		found := false

		for _, tN := range objectTypeNames {
			if oT.TypeName == tN {
				found = true
			}
		}
		// Common pool object types (will cause delete of all objects and edges)
		if id := uuid.FromStringOrNil(oT.TypeName); !id.IsNil() && id == oT.ID {
			found = true
		}

		if !found {
			continue
		}

		if err := azc.DeleteObjectType(ctx, oT.ID); err != nil {
			logErrorf(ctx, err, "AuthZ: DeleteObjectType failed authz cleanup: %v", err)

		}
		deleteCount++
	}

	eTypes, err := azc.ListEdgeTypes(ctx)
	if err != nil {
		logErrorf(ctx, err, "AuthZ: ListObjectTypes failed authz cleanup: %v", err)
	}
	uclog.Debugf(ctx, "AuthZ: Read %d edge types", len(eTypes))
	for _, eT := range eTypes {
		found := false

		for _, tN := range edgeTypeNames {
			if eT.TypeName == tN {
				found = true
			}
		}

		if !found {
			continue
		}

		if err := azc.DeleteEdgeType(ctx, eT.ID); err != nil {
			logErrorf(ctx, err, "AuthZ: DeleteObjectType failed authz cleanup: %v", err)
		}
		deleteCount++
	}

	wallTime := time.Now().UTC().Sub(testStart)
	uclog.Debugf(ctx, "AuthZ: Finished clean up for thread %d with %d deletions in %v", threadNum, deleteCount, wallTime)
}

func authzTestWorker(ctx context.Context, threadNum int, azc *authz.Client, objects []uuid.UUID, objectTypes []uuid.UUID, sT map[string]uuid.UUID, sE map[string]uuid.UUID, numOps int) {
	for range numOps {
		// Check objects/types from common pool
		authzCommonPoolTest(ctx, threadNum, azc, objects, objectTypes)
		// Create object/edge types, some edges and run checkAttribute
		authzPerThreadTest(ctx, threadNum, azc, sT, sE)
	}
}

func authzTest(ctx context.Context, tenantURL string, tokenSource jsonclient.Option, useLocalRedis bool, iterations int) {
	for i := 1; i < iterations+1; i++ {
		// TODO the client should be reading file config and picking a different db per tenant
		opts := []authz.Option{authz.JSONClient(tokenSource), authz.BypassCache()}
		if useLocalRedis {
			rcp := cache.NewRedisClientCacheProvider(cache.NewLocalRedisClient(), "envTestCache")
			opts = append(opts, authz.CacheProvider(rcp))
		}
		azc, err := authz.NewCustomClient(time.Minute*5, time.Minute*5, time.Minute*5, time.Minute*5, tenantURL, opts...)
		if err != nil {
			logErrorf(ctx, err, "Authz: AuthZ client creation failed: %v", err)
			return
		}

		testStart := time.Now().UTC()
		uclog.Infof(ctx, "Authz: Starting an environment test for %d worker threads - %v (%d/%d)", envTestCfg.Threads, tenantURL, i, iterations)

		// Clean up in case there is left over state
		authZTypeCleanup(ctx, -1, azc)

		// Set up types for per thread operations
		objTypes, edgeTypes := authZTypeSetup(ctx, azc)

		// Create common set of object types objects to be resolved across multiple worker threads
		var objects = make([]uuid.UUID, poolSizeAuthZ)
		var objectTypes = make([]uuid.UUID, poolSizeAuthZ)
		for i := range objects {
			var err error
			// TODO DEVEXP - feels weird to not have API that sets the unique guids for you
			objectTypes[i] = uuid.Must(uuid.NewV4())
			gotType, err := azc.CreateObjectType(ctx, objectTypes[i], objectTypes[i].String())
			if err != nil || gotType.ID != objectTypes[i] || gotType.TypeName != objectTypes[i].String() {
				logErrorf(ctx, err, "Authz: CreateObjectType failed for common pool: %v", err)
			}
			objects[i] = uuid.Must(uuid.NewV4())
			gotObj, err := azc.CreateObject(ctx, objects[i], objectTypes[i], objects[i].String())
			if err != nil || gotObj.ID != objects[i] || gotObj.TypeID != objectTypes[i] || *gotObj.Alias != objects[i].String() {
				logErrorf(ctx, err, "Authz: CreateObject failed for common pool: %v", err)
			}
		}

		wg := sync.WaitGroup{}
		for i := range envTestCfg.Threads {
			wg.Add(1)
			threadNum := i
			async.Execute(func() {
				authzTestWorker(ctx, threadNum, azc, objects, objectTypes, objTypes, edgeTypes, defaultThreadOpCount)
				wg.Done()
			})
		}
		wg.Wait()

		authZTypeCleanup(ctx, -1, azc)

		wallTime := time.Now().UTC().Sub(testStart)
		uclog.Infof(ctx, "Authz: Completed environment test for %d worker threads in %v (%d/%d)", envTestCfg.Threads, wallTime, i, iterations)

	}
}
