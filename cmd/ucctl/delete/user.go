package delete

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"userclouds.com/cmd/ucctl/common"
)

// UserCommand handles the delete user subcommand
type UserCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	AutoApprove     bool
	Verbose         bool

	credentials *common.Credentials
}

// RunE executes the delete user command
func (c *UserCommand) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	if len(args) != 1 {
		return fmt.Errorf("user email or ID is required")
	}

	userIdentifier := args[0]

	if err := c.validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := c.deleteUser(ctx, userIdentifier); err != nil {
		return fmt.Errorf("failed to delete user(s): %w", err)
	}

	return nil
}

func (c *UserCommand) validate() error {
	var err error
	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	return nil
}

func (c *UserCommand) deleteUser(ctx context.Context, userIdentifier string) error {
	// Create client credentials option
	// Create IDP management client
	mgmtClient, err := c.credentials.NewManagementClient()
	if err != nil {
		return err
	}

	var userIDs []uuid.UUID

	// Check if the argument is a UUID
	if id, err := uuid.FromString(userIdentifier); err == nil {
		// It's a UUID - delete this single user
		userIDs = append(userIDs, id)
	} else {
		// It's an email - look up all users with this email
		// Get all user IDs with this email
		userIDs, err = common.GetUserIDsByEmail(ctx, mgmtClient, userIdentifier)
		if err != nil {
			return err
		}
	}

	// Prompt for confirmation
	var message string
	if len(userIDs) == 1 {
		message = fmt.Sprintf("user %s", userIDs[0])
	} else {
		message = fmt.Sprintf("ALL users with email %s (%d users)", userIdentifier, len(userIDs))
	}

	if !common.ConfirmDeletion(message, "", c.AutoApprove) {
		fmt.Println("Deletion cancelled")
		return nil
	}

	// Delete all users
	deletedCount := 0
	var lastErr error

	for _, userID := range userIDs {
		err = mgmtClient.DeleteUser(ctx, userID)
		if err != nil {
			if c.Verbose {
				fmt.Printf("Warning: Failed to delete user %s: %v\n", userID, err)
			}
			lastErr = err
		} else {
			deletedCount++
			if c.Verbose {
				fmt.Printf("Deleted user %s\n", userID)
			}
		}
	}

	if deletedCount == 0 {
		return fmt.Errorf("failed to delete users: %w", lastErr)
	}

	if deletedCount < len(userIDs) {
		fmt.Printf("Warning: Deleted %d of %d users (some failures occurred)\n", deletedCount, len(userIDs))
		return fmt.Errorf("partially failed: deleted %d of %d users", deletedCount, len(userIDs))
	}

	if len(userIDs) == 1 {
		fmt.Printf("Deleted user %s\n", userIDs[0])
	} else {
		fmt.Printf("Deleted %d user(s) with email %s\n", deletedCount, userIdentifier)
	}

	return nil
}
