package main

import (
	"context"
	"math/rand"
	"slices"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

func scriptTestWorker(ctx context.Context, threadNum int, azc *authz.Client, idpc *idp.Client) error {
	testStart := time.Now().UTC()
	uclog.Debugf(ctx, "AuthZ: Starting per thread objects test for thread %d", threadNum)

	objTypes, err := readAllObjectTypes(ctx, azc)
	if err != nil {
		objTypes = []authz.ObjectType{}
	}
	edgeTypes, err := readAllEdgeTypes(ctx, azc)
	if err != nil {
		edgeTypes = []authz.EdgeType{}
	}

	objs, err := readAllObjects(ctx, azc)
	if err != nil {
		objs = []authz.Object{}
	}

	userRoleAssignmentID := uuid.Nil
	userPolicyAssignmentID := uuid.Nil
	for _, edgeType := range edgeTypes {
		switch edgeType.TypeName {
		case "user_role_assignment":
			userRoleAssignmentID = edgeType.ID
		case "user_policy_assignment":
			userPolicyAssignmentID = edgeType.ID
		}
	}

	userTypeID := uuid.Nil
	resourceNameID := uuid.Nil
	applicationID := uuid.Nil
	policyID := uuid.Nil
	roleID := uuid.Nil
	for _, objType := range objTypes {
		switch objType.TypeName {
		case "_user":
			userTypeID = objType.ID
		case "role":
			roleID = objType.ID
		case "application":
			applicationID = objType.ID
		case "resource_name":
			resourceNameID = objType.ID
		case "policy":
			policyID = objType.ID
		}
	}

	users := []authz.Object{}
	resources := []authz.Object{}
	policies := []authz.Object{}
	roles := []authz.Object{}
	applications := []authz.Object{}
	for _, obj := range objs {
		switch obj.TypeID {
		case userTypeID:
			users = append(users, obj)
		case resourceNameID:
			resources = append(resources, obj)
		case applicationID:
			applications = append(applications, obj)
		case policyID:
			policies = append(policies, obj)
		case roleID:
			roles = append(roles, obj)
		}
	}

	if len(users) == 0 || len(resources) == 0 || len(applications) == 0 || len(policies) == 0 || len(roles) == 0 {
		return ucerr.New("Didn't find expected objects")
	}
	// Get resources reachable by each user and validate checkAttribute as baseline
	attribute := "read"
	resourcesReachable := make(map[uuid.UUID][]uuid.UUID)
	for i := range users {
		resourcesReachable[users[i].ID], err = azc.ListObjectsReachableWithAttribute(ctx, users[i].ID, resourceNameID, attribute)
		if err != nil {
			logErrorf(ctx, err, "Script: Thread %2d ListObjectsReachableWithAttribute failed for user %v : %v", threadNum, users[i], err)
		}

		for j := range resourcesReachable[users[i].ID] {
			if j > 5 {
				break
			}
			checkAttribute(ctx, threadNum, azc, users[i].ID, resourcesReachable[users[i].ID][j], attribute, true)
		}
	}

	// Create an organization(s)
	orgIDWest := uuid.Must(uuid.NewV4())
	if _, err := createOrganizationForCustomer(ctx, azc, orgIDWest, "aws-us-west-2", "westOrg"); err != nil {
		logErrorf(ctx, err, "Script: Thread %2d createOrganizationForCustomer failed : %v", threadNum, err)
	}

	orgIDEast := uuid.Must(uuid.NewV4())
	if _, err := createOrganizationForCustomer(ctx, azc, orgIDEast, "aws-us-east-1", "eastOrg"); err != nil {
		logErrorf(ctx, err, "Script: Thread %2d createOrganizationForCustomer failed : %v", threadNum, err)
	}

	// Create users in two orgs
	var newUsers = make([]uuid.UUID, poolSizeUserStore)
	var newEdgesRoles = make([]*authz.Edge, poolSizeUserStore)
	var newEdgesPolicies = make([]*authz.Edge, poolSizeUserStore)
	var aliases = make([]string, poolSizeUserStore)
	var profs = make([]userstore.Record, poolSizeUserStore)
	newResourcesReachable := make(map[uuid.UUID][]uuid.UUID)
	for i := range users {
		aliases[i] = uuid.Must(uuid.NewV4()).String() // This reasonably matches their format of 	00u4j7o5jllNnubIa1d7
		profs[i] = userstore.Record{
			"external_alias": aliases[i],
		}

		orgID := orgIDWest
		if i > len(users)/2 {
			orgID = orgIDEast
		}

		newUsers[i], err = idpc.CreateUser(ctx, profs[i], idp.OrganizationID(orgID))
		if err != nil {
			logErrorf(ctx, err, "Script: CreateUser failed: %v", err)
		}

		// Check if AuthZ object is correct
		authObj, err := azc.GetObject(ctx, newUsers[i])
		if authObj.Alias == nil || *authObj.Alias != aliases[i] {
			logErrorf(ctx, err, "Script: GetObject returned unexpected alias %v instead of %v", *authObj.Alias, aliases[i])

		}

		// Check if we can can grant access to an existing resource
		newEdgesRoles[i], err = azc.CreateEdge(ctx, uuid.Must(uuid.NewV4()), newUsers[i], roles[rand.Intn(len(roles))].ID, userRoleAssignmentID)
		if err != nil {
			logErrorf(ctx, err, "Script: CreateEdge (role) failed: %v", err)
		}
		newEdgesPolicies[i], err = azc.CreateEdge(ctx, uuid.Must(uuid.NewV4()), newUsers[i], policies[rand.Intn(len(policies))].ID, userPolicyAssignmentID)
		if err != nil {
			logErrorf(ctx, err, "Script: CreateEdge (policy) failed: %v", err)
		}

		newResourcesReachable[newUsers[i]], err = azc.ListObjectsReachableWithAttribute(ctx, newUsers[i], resourceNameID, attribute)
		if err != nil {
			logErrorf(ctx, err, "Script: Thread %2d ListObjectsReachableWithAttribute failed for user %v : %v", threadNum, users[i], err)
		}

		for j := range newResourcesReachable[newUsers[i]] {
			checkAttribute(ctx, threadNum, azc, newUsers[i], newResourcesReachable[newUsers[i]][j], attribute, true)
		}

		if _, err := azc.CreateObject(ctx, orgID, userTypeID, "alias", authz.OrganizationID(orgID)); err != nil {
			logErrorf(ctx, err, "Script: Thread %2d migrateToOrganization failed for user %v : %v", threadNum, newUsers[i], err)
		}

		if err := migrateToOrganization(ctx, azc, newResourcesReachable[newUsers[i]], []uuid.UUID{}, orgID); err != nil {
			logErrorf(ctx, err, "Script: Thread %2d migrateToOrganization failed for user %v : %v", threadNum, newUsers[i], err)
		}

		// Migrate the resources to the org
		for j := range newResourcesReachable[newUsers[i]] {
			if _, err := azc.AddOrganizationToObject(ctx, newResourcesReachable[newUsers[i]][j], orgID); err != nil {
				logErrorf(ctx, err, "Script: Thread %2d migrateToOrganization failed for user %v : %v", threadNum, newUsers[i], err)
			}
		}

		// Check reachability for new users (User table rows) to migrated objects
		for j := range newResourcesReachable[newUsers[i]] {
			checkAttribute(ctx, threadNum, azc, newUsers[i], newResourcesReachable[newUsers[i]][j], attribute, true)
		}

		// Check reachability for existing user (user AuthZ objects) to migrated objects
		for j := range newResourcesReachable[newUsers[i]] {
			for k := range users {
				if slices.Contains(resourcesReachable[users[k].ID], newResourcesReachable[newUsers[i]][j]) {
					checkAttribute(ctx, threadNum, azc, users[k].ID, newResourcesReachable[newUsers[i]][j], attribute, true)
					break
				}
			}
		}

	}

	for i := range newUsers {
		if err := idpc.DeleteUser(ctx, newUsers[i]); err != nil { // This should delete the AuthZ object which also deletes all th edges we created
			logErrorf(ctx, err, "Userstore: Cleanup DeleteUser failed: %v", err)
		}
	}

	// TODO azc.DeleteOrg and unmigrate objects
	wallTime := time.Now().UTC().Sub(testStart)
	uclog.Debugf(ctx, "AuthZ: Finished per thread objects test for thread %d in %v", threadNum, wallTime)

	return nil
}

// If you have an uuid that you use internally for a customer you may want to reuse it as the org id for UserClouds organization to make
// future linking easy. Otherwise pass uuid.Nil as orgID and make sure you store the returned new orgID with your customer data
func createOrganizationForCustomer(ctx context.Context, azc *authz.Client, orgID uuid.UUID, orgRegion string, orgName string) (*authz.Organization, error) {
	if orgRegion != "aws-us-west-2" && orgRegion != "aws-us-east-1" {
		return nil, ucerr.Errorf("Invalid region value %s", orgRegion)
	}

	if orgName == "" {
		return nil, ucerr.Errorf("Org name shouldn't be blank")
	}

	if orgID.IsNil() {
		orgID = uuid.Must(uuid.NewV4())
	}

	org, err := azc.CreateOrganization(ctx, orgID, orgName, region.DataRegion(orgRegion))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return org, nil
}

// migrateToOrganization will move all given objects and edgetypes associated with a customer to that customer's org
func migrateToOrganization(ctx context.Context, azc *authz.Client, objects []uuid.UUID, edgeTypes []uuid.UUID, orgID uuid.UUID) error {
	if orgID.IsNil() {
		return ucerr.Errorf("Please pass in valid org id")
	}

	if _, err := azc.GetOrganization(ctx, orgID); err != nil {
		return ucerr.Wrap(err)
	}

	for _, obj := range objects {
		_, err := azc.GetObject(ctx, obj)
		if err != nil {
			return ucerr.Wrap(err)
		}
		_, err = azc.AddOrganizationToObject(ctx, obj, orgID)
		if err != nil {
			return ucerr.Wrap(err)
		}
	}

	for _, edgeType := range edgeTypes {
		_, err := azc.GetEdgeType(ctx, edgeType)
		if err != nil {
			return ucerr.Wrap(err)
		}
		_, err = azc.AddOrganizationToEdgeType(ctx, edgeType, orgID)
		if err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}

func readAllObjectTypes(ctx context.Context, azc *authz.Client) ([]authz.ObjectType, error) {
	objTypes, err := azc.ListObjectTypes(ctx)
	if err != nil {
		uclog.Errorf(ctx, "Script: Failed to read object types: %v", err)
		return nil, ucerr.Wrap(err)
	}
	uclog.Debugf(ctx, "Script: Read %d object types from source", len(objTypes))
	return objTypes, nil
}

func readAllEdgeTypes(ctx context.Context, azc *authz.Client) ([]authz.EdgeType, error) {
	edgeTypes, err := azc.ListEdgeTypes(ctx)
	if err != nil {
		uclog.Errorf(ctx, "Script: Failed to read edge types: %v", err)
		return nil, ucerr.Wrap(err)
	}
	uclog.Debugf(ctx, "Script: Read %d edge types from source", len(edgeTypes))
	return edgeTypes, nil
}

func readAllObjects(ctx context.Context, azc *authz.Client) ([]authz.Object, error) {
	objs := []authz.Object{}
	cursor := pagination.CursorBegin
	for {
		resp, err := azc.ListObjects(ctx, authz.Pagination(pagination.StartingAfter(cursor)))
		if err != nil {
			uclog.Errorf(ctx, "Script: Failed to read objects: %v", err)
			return nil, ucerr.Wrap(err)
		}

		objs = append(objs, resp.Data...)

		if !resp.HasNext {
			break
		}
		cursor = resp.Next
	}
	uclog.Debugf(ctx, "Script: Read %d objects from source", len(objs))
	return objs, nil
}

func scriptTest(ctx context.Context, tenantURL string, tokenSource jsonclient.Option, iterations int) {
	for i := 1; i < iterations+1; i++ {
		azc, err := authz.NewClient(tenantURL, authz.JSONClient(tokenSource))
		if err != nil {
			logErrorf(ctx, err, "Script: AuthZ client creation failed: %v", err)
			return
		}

		idpc, err := idp.NewClient(tenantURL, idp.JSONClient(tokenSource))
		if err != nil {
			logErrorf(ctx, err, "Userstore: IDP client creation failed: %v", err)
			return
		}

		testStart := time.Now().UTC()
		uclog.Infof(ctx, "Script: Starting an environment test for %d worker threads - %v (%d/%d)", envTestCfg.Threads, tenantURL, i, iterations)

		wg := sync.WaitGroup{}
		for i := range envTestCfg.Threads {
			wg.Add(1)
			go func(ctx context.Context, threadNum int, azc *authz.Client) {
				if err := scriptTestWorker(ctx, threadNum, azc, idpc); err != nil {
					uclog.Errorf(ctx, "scriptTestWorker failed with %v", err)
				}
				wg.Done()
			}(ctx, i, azc)
		}
		wg.Wait()
		wallTime := time.Now().UTC().Sub(testStart)
		uclog.Infof(ctx, "Script: Completed tokenizer environment test for %d worker threads in %v (%d/%d)", envTestCfg.Threads, wallTime, i, iterations)
	}
}
