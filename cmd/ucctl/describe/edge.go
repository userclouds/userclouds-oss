package describe

import (
	stdcontext "context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
)

// EdgeCommand handles the describe edge subcommand
type EdgeCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	credentials *common.Credentials
}

// RunE executes the describe edge command
func (c *EdgeCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("edge ID is required")
	}

	ctx := cmd.Context()
	edgeIDStr := args[0]

	// Parse edge ID
	edgeID, err := uuid.FromString(edgeIDStr)
	if err != nil {
		return fmt.Errorf("invalid edge ID: %w", err)
	}

	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	authzClient, err := c.credentials.NewAuthzClient()
	if err != nil {
		return err
	}

	edge, err := authzClient.GetEdge(ctx, edgeID)
	if err != nil {
		return fmt.Errorf("failed to get edge: %w", err)
	}

	edgeType, err := authzClient.GetEdgeType(ctx, edge.EdgeTypeID)
	if err != nil {
		return fmt.Errorf("failed to get edge type: %w", err)
	}

	sourceObj, err := authzClient.GetObject(ctx, edge.SourceObjectID)
	if err != nil {
		return fmt.Errorf("failed to get source object: %w", err)
	}

	targetObj, err := authzClient.GetObject(ctx, edge.TargetObjectID)
	if err != nil {
		return fmt.Errorf("failed to get target object: %w", err)
	}

	c.printEdgeDetails(ctx, edge, edgeType, sourceObj, targetObj, authzClient)

	return nil
}

func (c *EdgeCommand) printEdgeDetails(ctx stdcontext.Context, edge *authz.Edge, edgeType *authz.EdgeType, sourceObj *authz.Object, targetObj *authz.Object, client *authz.Client) {
	// Header
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Edge: %s\n", edgeType.TypeName)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Basic Information
	fmt.Println("BASIC INFORMATION")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("  ID:              %s\n", edge.ID)
	fmt.Printf("  Edge Type:       %s\n", edgeType.TypeName)
	fmt.Printf("  Edge Type ID:    %s\n", edge.EdgeTypeID)
	fmt.Println()

	// Edge Type Attributes (permissions)
	fmt.Println("EDGE TYPE ATTRIBUTES")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	if len(edgeType.Attributes) == 0 {
		fmt.Println("  No attributes defined on this edge type")
	} else {
		for _, attr := range edgeType.Attributes {
			permissions := []string{}
			if attr.Direct {
				permissions = append(permissions, "Direct")
			}
			if attr.Inherit {
				permissions = append(permissions, "Inherit")
			}
			if attr.Propagate {
				permissions = append(permissions, "Propagate")
			}
			permStr := ""
			if len(permissions) > 0 {
				permStr = fmt.Sprintf(" (%s)", strings.Join(permissions, ", "))
			}
			fmt.Printf("  • %s%s\n", attr.Name, permStr)
		}
	}
	fmt.Println()

	// Source Object
	fmt.Println("SOURCE OBJECT")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	sourceType, _ := client.GetObjectType(ctx, sourceObj.TypeID)
	fmt.Printf("  ID:              %s\n", sourceObj.ID)
	if sourceObj.Alias != nil && *sourceObj.Alias != "" {
		fmt.Printf("  Alias:           %s\n", *sourceObj.Alias)
	}
	if sourceType != nil {
		fmt.Printf("  Type:            %s\n", sourceType.TypeName)
	}
	fmt.Printf("  Organization ID: %s\n", sourceObj.OrganizationID)
	fmt.Println()

	// Target Object
	fmt.Println("TARGET OBJECT")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	targetType, _ := client.GetObjectType(ctx, targetObj.TypeID)
	fmt.Printf("  ID:              %s\n", targetObj.ID)
	if targetObj.Alias != nil && *targetObj.Alias != "" {
		fmt.Printf("  Alias:           %s\n", *targetObj.Alias)
	}
	if targetType != nil {
		fmt.Printf("  Type:            %s\n", targetType.TypeName)
	}
	fmt.Printf("  Organization ID: %s\n", targetObj.OrganizationID)
	fmt.Println()

	// Relationship Visualization
	fmt.Println("RELATIONSHIP")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	sourceName := sourceObj.ID.String()[:8]
	if sourceObj.Alias != nil && *sourceObj.Alias != "" {
		sourceName = *sourceObj.Alias
	}
	targetName := targetObj.ID.String()[:8]
	if targetObj.Alias != nil && *targetObj.Alias != "" {
		targetName = *targetObj.Alias
	}

	sourceTypeName := "unknown"
	if sourceType != nil {
		sourceTypeName = sourceType.TypeName
	}
	targetTypeName := "unknown"
	if targetType != nil {
		targetTypeName = targetType.TypeName
	}

	fmt.Printf("  %s (%s)\n", sourceName, sourceTypeName)
	fmt.Printf("    │\n")
	fmt.Printf("    │ %s\n", edgeType.TypeName)
	fmt.Printf("    ▼\n")
	fmt.Printf("  %s (%s)\n", targetName, targetTypeName)
	fmt.Println()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
