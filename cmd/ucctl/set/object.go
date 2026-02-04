package set

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/infra/ucerr"
)

type ObjectCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	Alias           string
	ClearAlias      bool

	credentials *common.Credentials
}

func (c *ObjectCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("object ID is required")
	}

	objectID, err := uuid.FromString(args[0])
	if err != nil {
		return fmt.Errorf("invalid object ID: %w", err)
	}

	// Validate flags
	if c.Alias == "" && !c.ClearAlias {
		return fmt.Errorf("either --alias or --clear-alias must be specified")
	}

	if c.Alias != "" && c.ClearAlias {
		return fmt.Errorf("cannot specify both --alias and --clear-alias")
	}

	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	azClient, err := c.credentials.NewAuthzClient()
	if err != nil {
		return err
	}

	// Prepare alias value
	var aliasPtr *string
	if c.ClearAlias {
		aliasPtr = nil
	} else {
		aliasPtr = &c.Alias
	}

	// Update the object
	updatedObject, err := azClient.UpdateObject(cmd.Context(), objectID, aliasPtr)
	if err != nil {
		return ucerr.Wrap(err)
	}

	fmt.Printf("Object %s updated successfully\n", updatedObject.ID)
	if updatedObject.Alias != nil {
		fmt.Printf("  Alias: %s\n", *updatedObject.Alias)
	} else {
		fmt.Printf("  Alias: (none)\n")
	}

	return nil
}
