package delete

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/infra/ucerr"
)

type ObjectTypeCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	AutoApprove     bool

	credentials *common.Credentials
}

func (c *ObjectTypeCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("object type ID is required")
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

	if !common.ConfirmDeletion("object type", objectTypeID.String(), c.AutoApprove) {
		fmt.Println("Deletion cancelled")
		return nil
	}

	err = azClient.DeleteObjectType(cmd.Context(), objectTypeID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	fmt.Printf("Object type %s deleted successfully\n", objectTypeID)
	return nil
}
