package dump

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/infra/pagination"
)

// AuthzCommand handles the dump authz subcommand
type AuthzCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	OutputFile      string

	credentials *common.Credentials
}

// AuthzDump represents all authz resources
type AuthzDump struct {
	ObjectTypes []authz.ObjectType `json:"object_types"`
	EdgeTypes   []authz.EdgeType   `json:"edge_types"`
	Objects     []authz.Object     `json:"objects"`
	Edges       []authz.Edge       `json:"edges"`
}

// RunE executes the dump authz command
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

	fmt.Println("Dumping authz resources...")

	dump := &AuthzDump{}

	fmt.Println("Fetching object types...")
	dump.ObjectTypes, err = c.fetchAllObjectTypes(ctx, authzClient)
	if err != nil {
		return fmt.Errorf("failed to fetch object types: %w", err)
	}
	fmt.Printf("  Found %d object types\n", len(dump.ObjectTypes))

	fmt.Println("Fetching edge types...")
	dump.EdgeTypes, err = c.fetchAllEdgeTypes(ctx, authzClient)
	if err != nil {
		return fmt.Errorf("failed to fetch edge types: %w", err)
	}
	fmt.Printf("  Found %d edge types\n", len(dump.EdgeTypes))

	fmt.Println("Fetching objects...")
	dump.Objects, err = c.fetchAllObjects(ctx, authzClient)
	if err != nil {
		return fmt.Errorf("failed to fetch objects: %w", err)
	}
	fmt.Printf("  Found %d objects\n", len(dump.Objects))

	fmt.Println("Fetching edges...")
	dump.Edges, err = c.fetchAllEdges(ctx, authzClient)
	if err != nil {
		return fmt.Errorf("failed to fetch edges: %w", err)
	}
	fmt.Printf("  Found %d edges\n", len(dump.Edges))

	// Write to file
	fmt.Printf("Writing to %s...\n", c.OutputFile)
	data, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dump: %w", err)
	}

	if err := os.WriteFile(c.OutputFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Successfully dumped authz resources to %s\n", c.OutputFile)
	return nil
}

func (c *AuthzCommand) fetchAllObjectTypes(ctx context.Context, client *authz.Client) ([]authz.ObjectType, error) {
	objectTypes, err := client.ListObjectTypes(ctx)
	if err != nil {
		return nil, err
	}
	return objectTypes, nil
}

func (c *AuthzCommand) fetchAllEdgeTypes(ctx context.Context, client *authz.Client) ([]authz.EdgeType, error) {
	edgeTypes, err := client.ListEdgeTypes(ctx)
	if err != nil {
		return nil, err
	}
	return edgeTypes, nil
}

func (c *AuthzCommand) fetchAllObjects(ctx context.Context, client *authz.Client) ([]authz.Object, error) {
	return common.FetchAllPaginated(ctx, func(ctx context.Context, cursor pagination.Cursor) ([]authz.Object, pagination.Cursor, bool, error) {
		paginationOpts := []pagination.Option{pagination.StartingAfter(cursor), pagination.Limit(100)}
		resp, err := client.ListObjects(ctx, authz.Pagination(paginationOpts...))
		if err != nil {
			return nil, "", false, err
		}
		return resp.Data, resp.Next, resp.HasNext, nil
	})
}

func (c *AuthzCommand) fetchAllEdges(ctx context.Context, client *authz.Client) ([]authz.Edge, error) {
	return common.FetchAllPaginated(ctx, func(ctx context.Context, cursor pagination.Cursor) ([]authz.Edge, pagination.Cursor, bool, error) {
		paginationOpts := []pagination.Option{pagination.StartingAfter(cursor), pagination.Limit(100)}
		resp, err := client.ListEdges(ctx, authz.Pagination(paginationOpts...))
		if err != nil {
			return nil, "", false, err
		}
		return resp.Data, resp.Next, resp.HasNext, nil
	})
}
