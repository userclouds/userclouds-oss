package get

import (
	"fmt"
	"os"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"userclouds.com/cmd/ucctl/common"
)

type ObjectTypeCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	credentials *common.Credentials
}

func (c *ObjectTypeCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: ucctl get objecttype <id>")
	}

	objectTypeID, err := uuid.FromString(args[0])
	if err != nil {
		return fmt.Errorf("invalid object type ID: %w", err)
	}

	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	azClient, err := c.credentials.NewAuthzClient()
	if err != nil {
		return err
	}

	objectType, err := azClient.GetObjectType(cmd.Context(), objectTypeID)
	if err != nil {
		return fmt.Errorf("failed to get object type: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Object Type Details:\n")
	fmt.Fprintf(os.Stdout, "  ID:         %s\n", objectType.ID.String())
	fmt.Fprintf(os.Stdout, "  Type Name:  %s\n", objectType.TypeName)
	fmt.Fprintf(os.Stdout, "  Created:    %s\n", objectType.Created.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(os.Stdout, "  Updated:    %s\n", objectType.Updated.Format("2006-01-02 15:04:05"))

	return nil
}
