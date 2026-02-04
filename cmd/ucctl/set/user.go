package set

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/oidc"
)

// UserCommand handles the set user subcommand
type UserCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	Email           string
	Name            string
	EmailVerified   bool
	Username        string
	Password        string
	OIDCProvider    string
	OIDCIssuerURL   string
	OIDCSubject     string
	Verbose         bool

	credentials *common.Credentials
}

// RunE executes the set user command
func (c *UserCommand) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	if err := c.validate(args); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// First argument is the email to identify the user
	userEmail := args[0]

	if err := c.updateUser(ctx, cmd, userEmail); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (c *UserCommand) validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("user email is required")
	}

	var err error
	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	// Validate authentication method (optional - can update user without changing authn)
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
		return fmt.Errorf("cannot provide both password and OIDC authentication - choose one")
	}

	return nil
}

func (c *UserCommand) updateUser(ctx context.Context, cmd *cobra.Command, userEmail string) error {
	// Create IDP management client
	mgmtClient, err := c.credentials.NewManagementClient()
	if err != nil {
		return err
	}

	// Get single user ID by email
	userID, err := common.GetSingleUserIDByEmail(ctx, mgmtClient, userEmail)
	if err != nil {
		return err
	}

	// Build user profile with fields to update
	profile := userstore.Record{}
	if c.Email != "" {
		profile["email"] = c.Email
	}
	if c.Name != "" {
		profile["name"] = c.Name
	}

	// Check if email_verified flag was explicitly set
	if cmd.Flags().Changed("email-verified") {
		profile["email_verified"] = c.EmailVerified
	}

	// Update user profile if any fields are set
	if len(profile) > 0 {
		req := idp.UpdateUserRequest{
			Profile: profile,
		}
		_, err = mgmtClient.UpdateUser(ctx, userID, req)
		if err != nil {
			return fmt.Errorf("failed to update user profile: %w", err)
		}
	}

	// Update password if provided
	if c.Username != "" && c.Password != "" {
		err = mgmtClient.UpdateUsernamePassword(ctx, c.Username, c.Password)
		if err != nil {
			return fmt.Errorf("failed to update password: %w", err)
		}
	}

	// Update OIDC authentication if provided
	if c.OIDCProvider != "" {
		var provider oidc.ProviderType
		if err := provider.UnmarshalText([]byte(c.OIDCProvider)); err != nil {
			return fmt.Errorf("invalid OIDC provider: %w", err)
		}
		// Note: There's no direct update OIDC method in the management client
		// This would typically require deleting and recreating the OIDC authentication
		// or using a lower-level API call
		return fmt.Errorf("updating OIDC authentication is not currently supported via this command")
	}

	fmt.Printf("User updated: %s\n", userID)
	return nil
}
