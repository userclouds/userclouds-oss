package describe

import (
	stdcontext "context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/idp"
	"userclouds.com/infra/pagination"
)

// UserCommand handles the describe user subcommand
type UserCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	credentials *common.Credentials
}

// RunE executes the describe user command
func (c *UserCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("user email or ID is required")
	}

	ctx := cmd.Context()
	userIdentifier := args[0]

	var err error
	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	credOpt, err := c.credentials.GetClientCredentials()
	if err != nil {
		return fmt.Errorf("failed to create client credentials: %w", err)
	}

	mgmtClient, err := idp.NewManagementClient(c.credentials.URL, credOpt)
	if err != nil {
		return fmt.Errorf("failed to create IDP client: %w", err)
	}

	var userID uuid.UUID
	if id, err := uuid.FromString(userIdentifier); err == nil {
		userID = id
	} else {
		userID, err = common.GetSingleUserIDByEmail(ctx, mgmtClient, userIdentifier)
		if err != nil {
			return err
		}
	}

	profile, err := mgmtClient.GetUserBaseProfileAndAuthN(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user profile: %w", err)
	}

	authzClient, err := authz.NewClient(c.credentials.URL, authz.JSONClient(credOpt))
	if err != nil {
		return fmt.Errorf("failed to create authz client: %w", err)
	}
	rbacClient := authz.NewRBACClient(authzClient)

	c.printUserDetails(ctx, profile, userID, rbacClient, authzClient)

	return nil
}

func (c *UserCommand) printUserDetails(ctx stdcontext.Context, profile *idp.UserBaseProfileAndAuthnResponse, userID uuid.UUID, rbacClient *authz.RBACClient, authzClient *authz.Client) {
	// Header
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("User: %s\n", profile.Email)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Basic Information
	fmt.Println("BASIC INFORMATION")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("  ID:              %s\n", profile.ID)
	fmt.Printf("  Name:            %s\n", profile.Name)
	fmt.Printf("  Email:           %s\n", profile.Email)
	fmt.Printf("  Email Verified:  %t\n", profile.EmailVerified)
	fmt.Printf("  Organization:    %s\n", profile.OrganizationID)
	fmt.Println()

	// Authentication Information
	fmt.Println("AUTHENTICATION")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	if len(profile.Authns) == 0 {
		fmt.Println("  No authentication methods configured")
	} else {
		for i, authn := range profile.Authns {
			fmt.Printf("  Method %d:\n", i+1)
			fmt.Printf("    Type:     %s\n", authn.AuthnType)
			providerStr := authn.OIDCProvider.String()
			if providerStr != "" && providerStr != "unknown" {
				fmt.Printf("    Provider: %s\n", providerStr)
			}
			if authn.OIDCIssuerURL != "" {
				fmt.Printf("    Issuer:   %s\n", authn.OIDCIssuerURL)
			}
		}
	}
	fmt.Println()

	// Company Memberships
	fmt.Println("COMPANY MEMBERSHIPS")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	user, err := rbacClient.GetUser(ctx, userID)
	if err != nil {
		fmt.Printf("  Error fetching memberships: %v\n", err)
	} else {
		memberships, err := user.GetMemberships(ctx)
		if err != nil {
			fmt.Printf("  Error fetching memberships: %v\n", err)
		} else if len(memberships) == 0 {
			fmt.Println("  No company memberships")
		} else {
			for _, membership := range memberships {
				fmt.Printf("  • Company: %s\n", membership.Group.ID)
				fmt.Printf("    Name:    %s\n", membership.Group.Name)
				fmt.Printf("    Role:    %s\n", membership.Role)
				fmt.Println()
			}
		}
	}

	// Edges (relationships)
	fmt.Println("EDGES (RELATIONSHIPS)")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")

	// Edges where user is source
	sourceEdges := c.getEdges(ctx, authzClient, userID, true)
	if len(sourceEdges) > 0 {
		fmt.Println("  Outgoing Edges (user → target):")
		for _, edge := range sourceEdges {
			edgeType := c.getEdgeTypeName(ctx, authzClient, edge.EdgeTypeID)
			targetObj := c.getObjectInfo(ctx, authzClient, edge.TargetObjectID)
			fmt.Printf("    • %s → %s (%s)\n", edgeType, edge.TargetObjectID.String()[:8], targetObj)
		}
		fmt.Println()
	}

	// Edges where user is target
	targetEdges := c.getEdges(ctx, authzClient, userID, false)
	if len(targetEdges) > 0 {
		fmt.Println("  Incoming Edges (source → user):")
		for _, edge := range targetEdges {
			edgeType := c.getEdgeTypeName(ctx, authzClient, edge.EdgeTypeID)
			sourceObj := c.getObjectInfo(ctx, authzClient, edge.SourceObjectID)
			fmt.Printf("    • %s ← %s (%s)\n", edgeType, edge.SourceObjectID.String()[:8], sourceObj)
		}
		fmt.Println()
	}

	if len(sourceEdges) == 0 && len(targetEdges) == 0 {
		fmt.Println("  No edges found")
		fmt.Println()
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func (c *UserCommand) getEdges(ctx stdcontext.Context, client *authz.Client, objectID uuid.UUID, asSource bool) []authz.Edge {
	var filterStr string
	if asSource {
		filterStr = fmt.Sprintf("('source_object_id',EQ,'%s')", objectID)
	} else {
		filterStr = fmt.Sprintf("('target_object_id',EQ,'%s')", objectID)
	}

	edges, err := common.FetchAllPaginated(ctx, func(ctx stdcontext.Context, cursor pagination.Cursor) ([]authz.Edge, pagination.Cursor, bool, error) {
		paginationOpts := []pagination.Option{pagination.StartingAfter(cursor), pagination.Filter(filterStr)}
		resp, err := client.ListEdges(ctx, authz.Pagination(paginationOpts...))
		if err != nil {
			return nil, "", false, err
		}
		return resp.Data, resp.Next, resp.HasNext, nil
	})

	if err != nil {
		return []authz.Edge{}
	}

	return edges
}

func (c *UserCommand) getEdgeTypeName(ctx stdcontext.Context, client *authz.Client, edgeTypeID uuid.UUID) string {
	edgeType, err := client.GetEdgeType(ctx, edgeTypeID)
	if err != nil {
		return edgeTypeID.String()[:8]
	}
	if edgeType.TypeName == "" {
		return edgeTypeID.String()[:8]
	}
	return edgeType.TypeName
}

func (c *UserCommand) getObjectInfo(ctx stdcontext.Context, client *authz.Client, objectID uuid.UUID) string {
	obj, err := client.GetObject(ctx, objectID)
	if err != nil {
		return "unknown"
	}

	objType, err := client.GetObjectType(ctx, obj.TypeID)
	if err != nil {
		return obj.TypeID.String()[:8]
	}

	info := objType.TypeName
	if obj.Alias != nil && *obj.Alias != "" {
		info += fmt.Sprintf(":%s", *obj.Alias)
	}
	return info
}
