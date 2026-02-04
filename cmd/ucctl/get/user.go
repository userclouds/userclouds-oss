package get

import (
	stdcontext "context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/idp"
)

type UserCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	credentials *common.Credentials
}

func (c *UserCommand) RunE(cmd *cobra.Command, args []string) error {
	emailOrUuid := args[0]

	var err error
	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	mgmtClient, err := c.credentials.NewManagementClient()
	if err != nil {
		return err
	}

	var profiles []idp.UserBaseProfileAndAuthnResponse

	// Check to see if the argument is a uuid.
	if id, err := uuid.FromString(emailOrUuid); err == nil {
		p, err := mgmtClient.GetUserBaseProfileAndAuthN(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("failed to get user base profile and authn: %w", err)
		}
		profiles = append(profiles, *p)
	} else {
		// Get all users with this email
		users, err := common.GetUsersByEmail(cmd.Context(), mgmtClient, emailOrUuid)
		if err != nil {
			return err
		}

		// Convert to profile with authn format
		for _, user := range users {
			// Get full profile with authn if available
			userID, err := uuid.FromString(user.ID)
			if err != nil {
				return fmt.Errorf("invalid user ID: %w", err)
			}

			// Try to get the full profile with authn
			p, err := mgmtClient.GetUserBaseProfileAndAuthN(cmd.Context(), userID)
			if err != nil {
				// If authn is not available, create a basic profile
				profile := idp.UserBaseProfileAndAuthnResponse{
					UserBaseProfile: idp.UserBaseProfile{
						Email:         user.Email,
						EmailVerified: user.EmailVerified,
						Name:          user.Name,
					},
					ID:             user.ID,
					OrganizationID: user.OrganizationID,
					Authns:         []idp.UserAuthn{},
				}
				profiles = append(profiles, profile)
			} else {
				profiles = append(profiles, *p)
			}
		}
	}

	// Create AuthZ client to get user's company memberships
	authzClient, err := c.credentials.NewAuthzClient()
	if err != nil {
		return err
	}
	rbacClient := authz.NewRBACClient(authzClient)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tEMAIL\tORGANIZATION\tVERIFIED\tAUTHN TYPE\tPROVIDER\tCOMPANIES")

	for _, profile := range profiles {
		userID, err := uuid.FromString(profile.ID)
		if err != nil {
			return fmt.Errorf("invalid user ID: %w", err)
		}

		// Get user's company memberships
		companies := c.getUserCompanies(cmd.Context(), rbacClient, userID)

		if len(profile.Authns) > 0 {
			for _, authn := range profile.Authns {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%s\t%s\t%s\n",
					profile.ID, profile.Name, profile.Email, profile.OrganizationID,
					profile.EmailVerified, authn.AuthnType, authn.OIDCProvider, companies)
			}
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%s\t%s\t%s\n",
				profile.ID, profile.Name, profile.Email, profile.OrganizationID,
				profile.EmailVerified, "", "", companies)
		}
	}

	return w.Flush()
}

func (c *UserCommand) getUserCompanies(ctx stdcontext.Context, rbacClient *authz.RBACClient, userID uuid.UUID) string {
	user, err := rbacClient.GetUser(ctx, userID)
	if err != nil {
		return ""
	}

	memberships, err := user.GetMemberships(ctx)
	if err != nil {
		return ""
	}

	// Collect companies with roles
	companyRoles := make([]string, 0)
	for _, membership := range memberships {
		role := ""
		switch membership.Role {
		case ucauthz.AdminRole:
			role = "admin"
		case ucauthz.MemberRole:
			role = "member"
		default:
			role = membership.Role
		}
		// Format: company_id(role)
		companyRoles = append(companyRoles, fmt.Sprintf("%s(%s)", membership.Group.ID.String()[:8], role))
	}

	if len(companyRoles) == 0 {
		return "-"
	}

	return strings.Join(companyRoles, ", ")
}
