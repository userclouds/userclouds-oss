package describe

import (
	stdcontext "context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/infra/pagination"
)

// ObjectCommand handles the describe object subcommand
type ObjectCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	credentials *common.Credentials
}

// RunE executes the describe object command
func (c *ObjectCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("object ID is required")
	}

	ctx := cmd.Context()
	objectIDStr := args[0]

	objectID, err := uuid.FromString(objectIDStr)
	if err != nil {
		return fmt.Errorf("invalid object ID: %w", err)
	}

	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	authzClient, err := c.credentials.NewAuthzClient()
	if err != nil {
		return err
	}

	object, err := authzClient.GetObject(ctx, objectID)
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}

	objectType, err := authzClient.GetObjectType(ctx, object.TypeID)
	if err != nil {
		return fmt.Errorf("failed to get object type: %w", err)
	}

	c.printObjectDetails(ctx, object, objectType, authzClient)

	return nil
}

func (c *ObjectCommand) printObjectDetails(ctx stdcontext.Context, object *authz.Object, objectType *authz.ObjectType, client *authz.Client) {
	// Header
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	objectName := object.ID.String()
	if object.Alias != nil && *object.Alias != "" {
		objectName = *object.Alias
	}
	fmt.Printf("Object: %s\n", objectName)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Basic Information
	fmt.Println("BASIC INFORMATION")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("  ID:              %s\n", object.ID)
	fmt.Printf("  Type:            %s\n", objectType.TypeName)
	fmt.Printf("  Type ID:         %s\n", object.TypeID)
	if object.Alias != nil && *object.Alias != "" {
		fmt.Printf("  Alias:           %s\n", *object.Alias)
	}
	fmt.Printf("  Organization ID: %s\n", object.OrganizationID)
	fmt.Println()

	// Outgoing Edges (object is source)
	fmt.Println("OUTGOING EDGES (object → target)")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	outgoingEdges := c.getEdges(ctx, client, object.ID, true)
	if len(outgoingEdges) == 0 {
		fmt.Println("  No outgoing edges")
	} else {
		for _, edge := range outgoingEdges {
			edgeType := c.getEdgeTypeName(ctx, client, edge.EdgeTypeID)
			targetInfo := c.getObjectInfo(ctx, client, edge.TargetObjectID)
			fmt.Printf("  • %s → %s (%s)\n", edgeType, edge.TargetObjectID.String()[:8], targetInfo)
		}
	}
	fmt.Println()

	// Incoming Edges (object is target)
	fmt.Println("INCOMING EDGES (source → object)")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	incomingEdges := c.getEdges(ctx, client, object.ID, false)
	if len(incomingEdges) == 0 {
		fmt.Println("  No incoming edges")
	} else {
		for _, edge := range incomingEdges {
			edgeType := c.getEdgeTypeName(ctx, client, edge.EdgeTypeID)
			sourceInfo := c.getObjectInfo(ctx, client, edge.SourceObjectID)
			fmt.Printf("  • %s ← %s (%s)\n", edgeType, edge.SourceObjectID.String()[:8], sourceInfo)
		}
	}
	fmt.Println()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func (c *ObjectCommand) getEdges(ctx stdcontext.Context, client *authz.Client, objectID uuid.UUID, asSource bool) []authz.Edge {
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

func (c *ObjectCommand) getEdgeTypeName(ctx stdcontext.Context, client *authz.Client, edgeTypeID uuid.UUID) string {
	edgeType, err := client.GetEdgeType(ctx, edgeTypeID)
	if err != nil {
		return edgeTypeID.String()[:8]
	}
	if edgeType.TypeName == "" {
		return edgeTypeID.String()[:8]
	}
	return edgeType.TypeName
}

func (c *ObjectCommand) getObjectInfo(ctx stdcontext.Context, client *authz.Client, objectID uuid.UUID) string {
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
