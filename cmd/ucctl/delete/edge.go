package delete

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/infra/ucerr"
)

type EdgeCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	AutoApprove     bool

	credentials *common.Credentials
}

func (c *EdgeCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("edge ID is required")
	}

	edgeID, err := uuid.FromString(args[0])
	if err != nil {
		return fmt.Errorf("invalid edge ID: %w", err)
	}

	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	azClient, err := c.credentials.NewAuthzClient()
	if err != nil {
		return err
	}

	if !common.ConfirmDeletion("edge", edgeID.String(), c.AutoApprove) {
		fmt.Println("Deletion cancelled")
		return nil
	}

	err = azClient.DeleteEdge(cmd.Context(), edgeID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	fmt.Printf("Edge %s deleted successfully\n", edgeID)
	return nil
}
