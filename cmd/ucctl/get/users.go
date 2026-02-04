package get

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/idp"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

type UsersCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	credentials *common.Credentials
}

func (c *UsersCommand) RunE(cmd *cobra.Command, args []string) error {
	var err error
	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	mgmtClient, err := c.credentials.NewManagementClient()
	if err != nil {
		return err
	}

	profiles, err := c.fetchProfiles(cmd.Context(), mgmtClient)
	if err != nil {
		return fmt.Errorf("failed to list user base profiles: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tEMAIL\tORGANIZATION\tVERIFIED")

	for _, profile := range profiles {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\n", profile.ID, profile.Name, profile.Email, profile.OrganizationID, profile.EmailVerified)
	}

	return w.Flush()
}

func (c *UsersCommand) fetchProfiles(ctx context.Context, mgc *idp.ManagementClient) ([]idp.UserBaseProfileResponse, error) {
	return common.FetchAllPaginated(ctx, func(ctx context.Context, cursor pagination.Cursor) ([]idp.UserBaseProfileResponse, pagination.Cursor, bool, error) {
		resp, err := mgc.ListUserBaseProfiles(ctx, idp.Pagination(pagination.StartingAfter(cursor)))
		if err != nil {
			return nil, "", false, ucerr.Wrap(err)
		}
		return resp.Data, resp.Next, resp.HasNext, nil
	})
}
