package get

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"userclouds.com/cmd/ucctl/common"
)

type ObjectCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	credentials *common.Credentials
}

func (c *ObjectCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("object ID is required")
	}

	objectID, err := uuid.FromString(args[0])
	if err != nil {
		return fmt.Errorf("invalid object ID: %w", err)
	}

	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	azClient, err := c.credentials.NewAuthzClient()
	if err != nil {
		return err
	}

	obj, err := azClient.GetObject(cmd.Context(), objectID)
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}

	var alias string
	if obj.Alias != nil {
		alias = *obj.Alias
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tALIAS\tTYPE_ID\tORGANIZATION_ID\tCREATED\tUPDATED")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
		obj.ID.String(),
		alias,
		obj.TypeID.String(),
		obj.OrganizationID.String(),
		obj.Created.Format("2006-01-02 15:04:05"),
		obj.Updated.Format("2006-01-02 15:04:05"),
	)

	return w.Flush()
}
