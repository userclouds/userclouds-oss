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

// ObjectTypeCommand handles the describe objecttype subcommand
type ObjectTypeCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	credentials *common.Credentials
}

// RunE executes the describe objecttype command
func (c *ObjectTypeCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("objecttype ID or name is required")
	}

	ctx := cmd.Context()
	objectTypeIdentifier := args[0]

	var err error
	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	authzClient, err := c.credentials.NewAuthzClient()
	if err != nil {
		return err
	}

	var objectType *authz.ObjectType
	if objectTypeID, err := uuid.FromString(objectTypeIdentifier); err == nil {
		objectType, err = authzClient.GetObjectType(ctx, objectTypeID)
		if err != nil {
			return fmt.Errorf("failed to get object type: %w", err)
		}
	} else {
		// Search by name
		objectType, err = c.findObjectTypeByName(ctx, authzClient, objectTypeIdentifier)
		if err != nil {
			return err
		}
	}

	c.printObjectTypeDetails(ctx, objectType, authzClient)

	return nil
}

func (c *ObjectTypeCommand) findObjectTypeByName(ctx stdcontext.Context, client *authz.Client, name string) (*authz.ObjectType, error) {
	objectTypes, err := client.ListObjectTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list object types: %w", err)
	}

	for _, ot := range objectTypes {
		if ot.TypeName == name {
			return &ot, nil
		}
	}

	return nil, fmt.Errorf("object type not found: %s", name)
}

func (c *ObjectTypeCommand) printObjectTypeDetails(ctx stdcontext.Context, objectType *authz.ObjectType, client *authz.Client) {
	// Header
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Object Type: %s\n", objectType.TypeName)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Basic Information
	fmt.Println("BASIC INFORMATION")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("  ID:              %s\n", objectType.ID)
	fmt.Printf("  Type Name:       %s\n", objectType.TypeName)
	fmt.Println()

	// Objects of this type
	fmt.Println("OBJECTS OF THIS TYPE")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	objects := c.getObjectsByType(ctx, client, objectType.ID)
	if len(objects) == 0 {
		fmt.Println("  No objects found")
	} else {
		for _, obj := range objects {
			objName := obj.ID.String()[:8]
			if obj.Alias != nil && *obj.Alias != "" {
				objName = *obj.Alias
			}
			fmt.Printf("  • %s (%s)\n", objName, obj.ID)
		}
	}
	fmt.Println()

	// Edge types using this object type
	fmt.Println("EDGE TYPES USING THIS OBJECT TYPE")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	edgeTypesAsSource, edgeTypesAsTarget := c.getEdgeTypesUsingObjectType(ctx, client, objectType.ID)

	if len(edgeTypesAsSource) == 0 && len(edgeTypesAsTarget) == 0 {
		fmt.Println("  No edge types found")
	} else {
		if len(edgeTypesAsSource) > 0 {
			fmt.Println("  As Source:")
			for _, et := range edgeTypesAsSource {
				targetType := c.getObjectTypeName(ctx, client, et.TargetObjectTypeID)
				fmt.Printf("    • %s → %s\n", et.TypeName, targetType)
			}
		}
		if len(edgeTypesAsTarget) > 0 {
			fmt.Println("  As Target:")
			for _, et := range edgeTypesAsTarget {
				sourceType := c.getObjectTypeName(ctx, client, et.SourceObjectTypeID)
				fmt.Printf("    • %s ← %s\n", et.TypeName, sourceType)
			}
		}
	}
	fmt.Println()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func (c *ObjectTypeCommand) getObjectsByType(ctx stdcontext.Context, client *authz.Client, objectTypeID uuid.UUID) []authz.Object {
	filterStr := fmt.Sprintf("('type_id',EQ,'%s')", objectTypeID)

	objects, err := common.FetchAllPaginated(ctx, func(ctx stdcontext.Context, cursor pagination.Cursor) ([]authz.Object, pagination.Cursor, bool, error) {
		paginationOpts := []pagination.Option{pagination.StartingAfter(cursor), pagination.Filter(filterStr)}
		resp, err := client.ListObjects(ctx, authz.Pagination(paginationOpts...))
		if err != nil {
			return nil, "", false, err
		}
		return resp.Data, resp.Next, resp.HasNext, nil
	})

	if err != nil {
		return []authz.Object{}
	}

	return objects
}

func (c *ObjectTypeCommand) getEdgeTypesUsingObjectType(ctx stdcontext.Context, client *authz.Client, objectTypeID uuid.UUID) ([]authz.EdgeType, []authz.EdgeType) {
	asSource := []authz.EdgeType{}
	asTarget := []authz.EdgeType{}

	edgeTypes, err := client.ListEdgeTypes(ctx)
	if err != nil {
		return asSource, asTarget
	}

	for _, et := range edgeTypes {
		if et.SourceObjectTypeID == objectTypeID {
			asSource = append(asSource, et)
		}
		if et.TargetObjectTypeID == objectTypeID {
			asTarget = append(asTarget, et)
		}
	}

	return asSource, asTarget
}

func (c *ObjectTypeCommand) getObjectTypeName(ctx stdcontext.Context, client *authz.Client, objectTypeID uuid.UUID) string {
	objectType, err := client.GetObjectType(ctx, objectTypeID)
	if err != nil {
		return objectTypeID.String()[:8]
	}
	if objectType.TypeName == "" {
		return objectTypeID.String()[:8]
	}
	return objectType.TypeName
}
