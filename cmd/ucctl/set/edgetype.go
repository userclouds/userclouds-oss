package set

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/infra/ucerr"
)

type EdgeTypeCommand struct {
	URL                string
	ClientID           string
	ClientSecret       string
	ClientSecretVar    string
	AuthnType          string
	TypeName           string
	SourceObjectTypeID string
	TargetObjectTypeID string
	Attributes         string

	credentials *common.Credentials
}

func (c *EdgeTypeCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("edge type ID is required")
	}

	edgeTypeID, err := uuid.FromString(args[0])
	if err != nil {
		return fmt.Errorf("invalid edge type ID: %w", err)
	}

	// Validate required flags
	if c.TypeName == "" {
		return fmt.Errorf("--type-name is required")
	}
	if c.SourceObjectTypeID == "" {
		return fmt.Errorf("--source-object-type-id is required")
	}
	if c.TargetObjectTypeID == "" {
		return fmt.Errorf("--target-object-type-id is required")
	}

	// Parse UUIDs
	sourceObjectTypeID, err := uuid.FromString(c.SourceObjectTypeID)
	if err != nil {
		return fmt.Errorf("invalid source object type ID: %w", err)
	}

	targetObjectTypeID, err := uuid.FromString(c.TargetObjectTypeID)
	if err != nil {
		return fmt.Errorf("invalid target object type ID: %w", err)
	}

	// Parse attributes if provided
	var attributes authz.Attributes
	if c.Attributes != "" {
		if err := json.Unmarshal([]byte(c.Attributes), &attributes); err != nil {
			return fmt.Errorf("invalid attributes JSON: %w", err)
		}
	}

	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	azClient, err := c.credentials.NewAuthzClient()
	if err != nil {
		return err
	}

	// Update the edge type
	updatedEdgeType, err := azClient.UpdateEdgeType(
		cmd.Context(),
		edgeTypeID,
		sourceObjectTypeID,
		targetObjectTypeID,
		c.TypeName,
		attributes,
	)
	if err != nil {
		return ucerr.Wrap(err)
	}

	fmt.Printf("Edge type %s updated successfully\n", updatedEdgeType.ID)
	fmt.Printf("  Type Name: %s\n", updatedEdgeType.TypeName)
	fmt.Printf("  Source Object Type ID: %s\n", updatedEdgeType.SourceObjectTypeID)
	fmt.Printf("  Target Object Type ID: %s\n", updatedEdgeType.TargetObjectTypeID)
	if len(updatedEdgeType.Attributes) > 0 {
		attrsJSON, _ := json.MarshalIndent(updatedEdgeType.Attributes, "  ", "  ")
		fmt.Printf("  Attributes: %s\n", string(attrsJSON))
	}

	return nil
}
