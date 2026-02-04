package load

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
)

// AuthzCommand handles the load authz subcommand
type AuthzCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	InputFile       string
	DryRun          bool
	SkipExisting    bool

	credentials *common.Credentials
}

// AuthzDump represents all authz resources
type AuthzDump struct {
	ObjectTypes []authz.ObjectType `json:"object_types"`
	EdgeTypes   []authz.EdgeType   `json:"edge_types"`
	Objects     []authz.Object     `json:"objects"`
	Edges       []authz.Edge       `json:"edges"`
}

// RunE executes the load authz command
func (c *AuthzCommand) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var err error
	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	authzClient, err := c.credentials.NewAuthzClient()
	if err != nil {
		return err
	}

	// Read dump file
	fmt.Printf("Reading from %s...\n", c.InputFile)
	data, err := os.ReadFile(c.InputFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var dump AuthzDump
	if err := json.Unmarshal(data, &dump); err != nil {
		return fmt.Errorf("failed to parse dump file: %w", err)
	}

	fmt.Printf("Found:\n")
	fmt.Printf("  %d object types\n", len(dump.ObjectTypes))
	fmt.Printf("  %d edge types\n", len(dump.EdgeTypes))
	fmt.Printf("  %d objects\n", len(dump.Objects))
	fmt.Printf("  %d edges\n", len(dump.Edges))

	if c.DryRun {
		fmt.Println("\nDry run mode - no changes will be made")
		return nil
	}

	fmt.Println("\nLoading authz resources...")

	// Load resources in order of dependencies
	if err := c.loadObjectTypes(ctx, authzClient, dump.ObjectTypes); err != nil {
		return fmt.Errorf("failed to load object types: %w", err)
	}

	if err := c.loadEdgeTypes(ctx, authzClient, dump.EdgeTypes); err != nil {
		return fmt.Errorf("failed to load edge types: %w", err)
	}

	if err := c.loadObjects(ctx, authzClient, dump.Objects); err != nil {
		return fmt.Errorf("failed to load objects: %w", err)
	}

	if err := c.loadEdges(ctx, authzClient, dump.Edges); err != nil {
		return fmt.Errorf("failed to load edges: %w", err)
	}

	fmt.Println("\nSuccessfully loaded all authz resources")
	return nil
}

func (c *AuthzCommand) loadObjectTypes(ctx context.Context, client *authz.Client, objectTypes []authz.ObjectType) error {
	fmt.Printf("\nLoading %d object types...\n", len(objectTypes))
	created := 0
	skipped := 0

	for _, ot := range objectTypes {
		// Check if it already exists
		existing, err := client.GetObjectType(ctx, ot.ID)
		if err == nil && existing != nil {
			if c.SkipExisting {
				skipped++
				continue
			}
			// Object types cannot be updated in place, skip
			skipped++
			fmt.Printf("  Skipped existing object type: %s (%s)\n", ot.TypeName, ot.ID)
		} else {
			// Create new
			_, err = client.CreateObjectType(ctx, ot.ID, ot.TypeName)
			if err != nil {
				return fmt.Errorf("failed to create object type %s: %w", ot.TypeName, err)
			}
			created++
			fmt.Printf("  Created object type: %s (%s)\n", ot.TypeName, ot.ID)
		}
	}

	fmt.Printf("  Created: %d, Skipped: %d\n", created, skipped)
	return nil
}

func (c *AuthzCommand) loadEdgeTypes(ctx context.Context, client *authz.Client, edgeTypes []authz.EdgeType) error {
	fmt.Printf("\nLoading %d edge types...\n", len(edgeTypes))
	created := 0
	skipped := 0
	updated := 0

	for _, et := range edgeTypes {
		// Check if it already exists
		existing, err := client.GetEdgeType(ctx, et.ID)
		if err == nil && existing != nil {
			if c.SkipExisting {
				skipped++
				continue
			}
			// Update existing
			_, err = client.UpdateEdgeType(ctx, et.ID, et.SourceObjectTypeID, et.TargetObjectTypeID, et.TypeName, et.Attributes, authz.OrganizationID(et.OrganizationID))
			if err != nil {
				return fmt.Errorf("failed to update edge type %s: %w", et.ID, err)
			}
			updated++
			fmt.Printf("  Updated edge type: %s (%s)\n", et.TypeName, et.ID)
		} else {
			// Create new
			_, err = client.CreateEdgeType(ctx, et.ID, et.SourceObjectTypeID, et.TargetObjectTypeID, et.TypeName, et.Attributes, authz.OrganizationID(et.OrganizationID))
			if err != nil {
				return fmt.Errorf("failed to create edge type %s: %w", et.TypeName, err)
			}
			created++
			fmt.Printf("  Created edge type: %s (%s)\n", et.TypeName, et.ID)
		}
	}

	fmt.Printf("  Created: %d, Updated: %d, Skipped: %d\n", created, updated, skipped)
	return nil
}

func (c *AuthzCommand) loadObjects(ctx context.Context, client *authz.Client, objects []authz.Object) error {
	fmt.Printf("\nLoading %d objects...\n", len(objects))
	created := 0
	skipped := 0
	updated := 0

	for _, obj := range objects {
		// Check if it already exists
		existing, err := client.GetObject(ctx, obj.ID)
		if err == nil && existing != nil {
			if c.SkipExisting {
				skipped++
				continue
			}
			// Update existing (only alias can be updated)
			_, err = client.UpdateObject(ctx, obj.ID, obj.Alias)
			if err != nil {
				return fmt.Errorf("failed to update object %s: %w", obj.ID, err)
			}
			updated++
			objName := obj.ID.String()[:8]
			if obj.Alias != nil && *obj.Alias != "" {
				objName = *obj.Alias
			}
			fmt.Printf("  Updated object: %s (%s)\n", objName, obj.ID)
		} else {
			// Create new
			alias := ""
			if obj.Alias != nil {
				alias = *obj.Alias
			}
			_, err = client.CreateObject(ctx, obj.ID, obj.TypeID, alias, authz.OrganizationID(obj.OrganizationID))
			if err != nil {
				return fmt.Errorf("failed to create object %s: %w", obj.ID, err)
			}
			created++
			objName := obj.ID.String()[:8]
			if obj.Alias != nil && *obj.Alias != "" {
				objName = *obj.Alias
			}
			fmt.Printf("  Created object: %s (%s)\n", objName, obj.ID)
		}
	}

	fmt.Printf("  Created: %d, Updated: %d, Skipped: %d\n", created, updated, skipped)
	return nil
}

func (c *AuthzCommand) loadEdges(ctx context.Context, client *authz.Client, edges []authz.Edge) error {
	fmt.Printf("\nLoading %d edges...\n", len(edges))
	created := 0
	skipped := 0
	errors := 0

	for _, edge := range edges {
		// Check if it already exists by ID
		existing, err := client.GetEdge(ctx, edge.ID)
		if err == nil && existing != nil {
			skipped++
			continue
		}

		// Try to create the edge
		_, err = client.CreateEdge(ctx, edge.ID, edge.SourceObjectID, edge.TargetObjectID, edge.EdgeTypeID)
		if err != nil {
			// Check if it's an "already exists" error
			// This can happen when:
			// 1. Edge with same ID exists but GetEdge failed to find it (timing/consistency issue)
			// 2. Edge with same (edge_type, source, target) exists with different ID
			errMsg := err.Error()
			if contains(errMsg, "already exists") {
				// This is expected when syncing between tenants
				skipped++
				continue
			}
			// Other error - report it but continue
			errors++
			fmt.Printf("  Error creating edge %s: %v\n", edge.ID.String()[:8], err)
			continue
		}
		created++
		fmt.Printf("  Created edge: %s → %s (%s)\n",
			edge.SourceObjectID.String()[:8],
			edge.TargetObjectID.String()[:8],
			edge.ID.String()[:8])
	}

	fmt.Printf("  Created: %d, Skipped: %d", created, skipped)
	if errors > 0 {
		fmt.Printf(", Errors: %d", errors)
	}
	fmt.Println()

	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
