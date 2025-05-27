package main

import (
	"context"
	"log"
	"os"

	"github.com/gofrs/uuid"
	"github.com/joho/godotenv"

	"userclouds.com/authz"
	"userclouds.com/idp"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
)

// Object type IDs
var fileTypeID, teamTypeID uuid.UUID

// Edge type IDs (i.e. roles)
var teamMemberID uuid.UUID
var fileContainerTypeID uuid.UUID
var fileEditorTypeID, fileViewerTypeID uuid.UUID
var fileTeamEditorTypeID, fileTeamViewerTypeID uuid.UUID

var aliceUserID, bobUserID uuid.UUID

var edgeTypeNames = []string{"team_member", "file_editor", "file_viewer", "file_team_editor", "file_team_viewer", "file_container"}

func oneTimeProvision(ctx context.Context, idpClient *idp.Client, authZClient *authz.Client) error {
	// NB: these Create commands are only here because this is a self-contained sample app; normally
	// you would do one-time provisioning via the Console or via the AuthZ API ahead of time, not in every single app.
	//
	// You can either generate some UUIDs ahead of time and compile them in to your app as constants for each tenant,
	// or query them at runtime by name,
	fileTypeID = mustID(provisionObjectType(ctx, authZClient, "file"))
	teamTypeID = mustID(provisionObjectType(ctx, authZClient, "team"))

	// Team members inherit any direct permissions of their team.
	// NB: you can recursively inherit through multiple edges, so you could nest teams within teams
	teamMemberID = mustID(provisionEdgeType(ctx, authZClient, authz.UserObjectTypeID, teamTypeID, "team_member", authz.Attributes{
		{Name: "read", Inherit: true},
		{Name: "write", Inherit: true},
	}))
	// Users can be editors or viewers of a file with direct permissions.
	fileEditorTypeID = mustID(provisionEdgeType(ctx, authZClient, authz.UserObjectTypeID, fileTypeID, "file_editor", authz.Attributes{
		{Name: "read", Direct: true},
		{Name: "write", Direct: true},
	}))
	fileViewerTypeID = mustID(provisionEdgeType(ctx, authZClient, authz.UserObjectTypeID, fileTypeID, "file_viewer", authz.Attributes{
		{Name: "read", Direct: true},
	}))
	// Teams can be editors or viewers of a file with direct permissions.
	fileTeamEditorTypeID = mustID(provisionEdgeType(ctx, authZClient, teamTypeID, fileTypeID, "file_team_editor", authz.Attributes{
		{Name: "read", Direct: true},
		{Name: "write", Direct: true},
	}))
	fileTeamViewerTypeID = mustID(provisionEdgeType(ctx, authZClient, teamTypeID, fileTypeID, "file_team_viewer", authz.Attributes{
		{Name: "read", Direct: true},
	}))
	// Files can contain other files, and permissions on a container propagate to children.
	fileContainerTypeID = mustID(provisionEdgeType(ctx, authZClient, fileTypeID, fileTypeID, "file_container", authz.Attributes{
		{Name: "read", Propagate: true},
		{Name: "write", Propagate: true},
	}))

	// Create a few test users (normally users would sign up via the UI, so this is slightly contrived)
	aliceUserID = mustID(provisionUser(ctx, idpClient, "Alice"))

	bobUserID = mustID(provisionUser(ctx, idpClient, "Bob"))

	return nil
}

func initClients() (*idp.Client, *authz.Client, *ucjwt.Config) {
	ctx := context.Background()

	err := godotenv.Load()
	if err != nil {
		log.Printf("error loading .env file: %v\n(did you forget to copy `.env.example` to `.env` and substitute values?)", err)
	}

	tenantURL := os.Getenv("USERCLOUDS_TENANT_URL")
	clientID := os.Getenv("USERCLOUDS_CLIENT_ID")
	clientSecret := os.Getenv("USERCLOUDS_CLIENT_SECRET")
	tenantID := uuid.FromStringOrNil(os.Getenv("USERCLOUDS_TENANT_ID"))

	if tenantURL == "" || clientID == "" || clientSecret == "" || tenantID.IsNil() {
		log.Fatal("missing one or more required environment variables: USERCLOUDS_TENANT_URL, USERCLOUDS_CLIENT_ID, USERCLOUDS_CLIENT_SECRET, USERCLOUDS_TENANT_ID")
	}

	tokenSource := jsonclient.ClientCredentialsTokenSource(tenantURL+"/oidc/token", clientID, clientSecret, nil)

	orgID := uuid.Nil
	idpClient, err := idp.NewClient(tenantURL, idp.OrganizationID(orgID), idp.JSONClient(tokenSource))
	if err != nil {
		log.Fatalf("error initializing idp client: %v", err)
	}

	authZClient, err := authz.NewClient(tenantURL, authz.JSONClient(tokenSource))
	if err != nil {
		log.Fatalf("error initializing authz client: %v", err)
	}

	if err := oneTimeProvision(ctx, idpClient, authZClient); err != nil {
		log.Fatalf("error provisioning basic authz types: %v", err)
	}

	uclog.Debugf(ctx, "Test message")

	config := ucjwt.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TenantURL:    tenantURL,
		TenantID:     tenantID,
	}

	return idpClient, authZClient, &config
}

func main() {
	ctx := context.Background()
	idpClient, authZClient, config := initClients()
	fm := NewFileManager(authZClient)
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("error getting hostname: %v", err)
	}

	// Optional to initialize logging from SDK
	logtransports.InitLoggingSDK(config, true, hostname)
	defer uclog.Close()

	// Alice creates the root directory and has full permissions
	rootDir := mustFile(fm.NewRoot(ctx, aliceUserID))
	// Alice can create '/dir1' and '/dir2'
	dir1 := mustFile(fm.NewDir(ctx, "dir1", rootDir, aliceUserID))
	dir2 := mustFile(fm.NewDir(ctx, "dir2", rootDir, aliceUserID))

	// Bob cannot create files in '/dir1'
	if _, err = fm.NewFile(ctx, "file1", dir1, bobUserID); err == nil {
		log.Fatalf("expected Bob to fail to create file under dir1")
	}

	// Alice can create files in '/dir1'
	file1 := mustFile(fm.NewFile(ctx, "file1", dir1, aliceUserID))

	// Bob cannot read '/dir1/file1'
	if _, err = fm.ReadFile(ctx, file1, bobUserID); err == nil {
		log.Fatalf("expected Bob to fail to read file1")
	}

	// Grant Bob viewer permissions in 'dir1'
	_ = mustEdge(authZClient.CreateEdge(ctx, uuid.Nil, bobUserID, dir1.id, fileViewerTypeID))

	// Now Bob can read '/dir1/file1'
	if _, err := fm.ReadFile(ctx, file1, bobUserID); err != nil {
		log.Fatalf("expected Bob to succeed reading dir1/file1: %v", err)
	}

	// Bob cannot (yet) create subdirectories under '/dir2'
	if _, err = fm.NewDir(ctx, "dir3", dir2, bobUserID); err == nil {
		log.Fatalf("expected Bob to fail to create dir3 under /dir2")
	}

	// Create a team, add Bob to it, give that team write permissions to '/dir2'
	aliceReportsTeamID := mustID(provisionObject(ctx, authZClient, teamTypeID, "Alice's Direct Reports"))
	_ = mustEdge(authZClient.CreateEdge(ctx, uuid.Nil, bobUserID, aliceReportsTeamID, teamMemberID, authz.IfNotExists()))
	_ = mustEdge(authZClient.CreateEdge(ctx, uuid.Nil, aliceReportsTeamID, dir2.id, fileTeamEditorTypeID, authz.IfNotExists()))

	// Now Bob can create subdirectories under '/dir2'
	dir3, err := fm.NewDir(ctx, "dir3", dir2, bobUserID)
	if err != nil {
		log.Fatalf("expected Bob to succeed creating dir3 under /dir2: %v", err)
	}

	// But still not under '/dir1'
	if _, err = fm.NewDir(ctx, "dir3", dir1, bobUserID); err == nil {
		log.Fatalf("expected Bob to fail to create dir3 under /dir1")
	}

	renderFileTree(ctx, idpClient, authZClient, rootDir, 0)

	log.Printf("successfully ran sample")

	objectsToDelete := []uuid.UUID{
		rootDir.id,
		dir1.id,
		dir2.id,
		dir3.id,
		file1.id,
		aliceReportsTeamID,
	}
	for _, id := range objectsToDelete {
		if err := authZClient.DeleteObject(ctx, id); err != nil {
			log.Printf("failed to delete object %s: %v", id, err)
		}
	}

	usersToDelete := []uuid.UUID{
		aliceUserID,
		bobUserID,
	}
	for _, id := range usersToDelete {
		if err := idpClient.DeleteUser(ctx, id); err != nil {
			log.Printf("failed to delete user %s: %v", id, err)
		}
	}
}
