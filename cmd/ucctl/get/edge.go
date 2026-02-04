package get

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"userclouds.com/cmd/ucctl/common"
)

type EdgeCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	credentials *common.Credentials
}

func (c *EdgeCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
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

	edge, err := azClient.GetEdge(cmd.Context(), edgeID)
	if err != nil {
		return fmt.Errorf("failed to get edge: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tSOURCE_OBJECT_ID\tTARGET_OBJECT_ID\tEDGE_TYPE_ID\tCREATED\tUPDATED")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
		edge.ID.String(),
		edge.SourceObjectID.String(),
		edge.TargetObjectID.String(),
		edge.EdgeTypeID.String(),
		edge.Created.Format("2006-01-02 15:04:05"),
		edge.Updated.Format("2006-01-02 15:04:05"),
	)

	return w.Flush()
}
