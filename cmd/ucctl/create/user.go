package create

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/oidc"
)

// UserCommand handles the create user subcommand
type UserCommand struct {
	Admin           bool
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	OrganizationID  string
	Email           string
	Name            string
	Username        string
	Password        string
	OIDCProvider    string
	OIDCIssuerURL   string
	OIDCSubject     string
	Verbose         bool

	credentials *common.Credentials
}

// RunE executes the create user command
func (c *UserCommand) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	if err := c.validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := c.createUser(ctx); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (c *UserCommand) validate() error {
	if c.OrganizationID == "" {
		return fmt.Errorf("organization id is required")
	}

	var err error
	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	// Validate authentication method (optional - can create user without authn)
	hasPassword := c.Username != "" && c.Password != ""
	hasOIDC := c.OIDCProvider != "" && c.OIDCIssuerURL != "" && c.OIDCSubject != ""

	// Validate partial OIDC parameters
	hasPartialOIDC := c.OIDCProvider != "" || c.OIDCIssuerURL != "" || c.OIDCSubject != ""
	if hasPartialOIDC && !hasOIDC {
		return fmt.Errorf("when using OIDC authentication, all three flags are required: --oidc-provider, --oidc-issuer-url, and --oidc-subject")
	}

	// Validate partial password parameters
	hasPartialPassword := c.Username != "" || c.Password != ""
	if hasPartialPassword && !hasPassword {
		return fmt.Errorf("when using password authentication, both --username and --password are required")
	}

	if hasPassword && hasOIDC {
		return fmt.Errorf("cannot provide both password and OIDC authentication - choose one or omit both to create user without authentication")
	}

	return nil
}

func (c *UserCommand) createUser(ctx context.Context) error {
	mgmtClient, err := c.credentials.NewManagementClient()
	if err != nil {
		return err
	}

	// Build user profile
	profile := userstore.Record{}
	if c.Email != "" {
		profile["email"] = c.Email
	}
	if c.Name != "" {
		profile["name"] = c.Name
	}
	// Mark email as verified by default
	profile["email_verified"] = true

	var opts []idp.Option
	orgID, err := uuid.FromString(c.OrganizationID)
	if err != nil {
		return fmt.Errorf("invalid organization ID: %w", err)
	}
	opts = append(opts, idp.OrganizationID(orgID))

	var userID uuid.UUID
	var authnType string
	var provider string

	// Create user with appropriate authn method (or without authn)
	if c.Username != "" && c.Password != "" {
		userID, err = mgmtClient.CreateUserWithPassword(ctx, c.Username, c.Password, profile, opts...)
		if err != nil {
			return fmt.Errorf("failed to create user with password: %w", err)
		}
		authnType = "password"
		provider = ""
	} else if c.OIDCProvider != "" {
		var providerType oidc.ProviderType
		if err := providerType.UnmarshalText([]byte(c.OIDCProvider)); err != nil {
			return fmt.Errorf("invalid OIDC provider: %w", err)
		}

		userID, err = mgmtClient.CreateUserWithOIDC(ctx, providerType, c.OIDCIssuerURL, c.OIDCSubject, profile, opts...)
		if err != nil {
			return fmt.Errorf("failed to create user with OIDC: %w", err)
		}
		authnType = "oidc"
		provider = c.OIDCProvider
	} else {
		userID, err = mgmtClient.CreateUser(ctx, profile, opts...)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		authnType = ""
		provider = ""
	}

	// Display user in table format (same as get user)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tEMAIL\tORGANIZATION\tVERIFIED\tAUTHN TYPE\tPROVIDER")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%s\t%s\n",
		userID.String(),
		c.Name,
		c.Email,
		c.OrganizationID,
		true, // email_verified is always true for newly created users
		authnType,
		provider)

	return w.Flush()
}
