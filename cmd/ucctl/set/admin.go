package set

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/cmd/ucctl/common"
)

// AdminCommand handles the set admin subcommand
type AdminCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	UserEmail       string
	UserID          string
	TenantID        string
	TenantContext   string
	TenantCompanyID string
	CompanyID       string
	Verbose         bool

	credentials       *common.Credentials
	tenantCredentials *common.Credentials
}

// RunE executes the set admin command
func (c *AdminCommand) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	if err := c.validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := c.setAdmin(ctx); err != nil {
		return fmt.Errorf("failed to set admin privileges: %w", err)
	}

	return nil
}

func (c *AdminCommand) validate() error {
	// Load console tenant credentials from context or flags (for user lookup)
	var err error
	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	// User must be specified by either email or ID
	if c.UserEmail == "" && c.UserID == "" {
		return fmt.Errorf("user must be specified by either --email or --user-id")
	}

	if c.UserEmail != "" && c.UserID != "" {
		return fmt.Errorf("specify only one of --email or --user-id, not both")
	}

	// Must specify either tenant or company
	if c.TenantID == "" && c.CompanyID == "" {
		return fmt.Errorf("either --tenant-id or --company-id must be specified")
	}

	if c.TenantID != "" && c.CompanyID != "" {
		return fmt.Errorf("specify only one of --tenant-id or --company-id, not both")
	}

	// If setting tenant admin, require tenant context and company ID
	// TODO: This is not my favorite way to approach this, but it does work for now.
	if c.TenantID != "" {
		if c.TenantContext == "" {
			return fmt.Errorf("--tenant-context is required when setting tenant admin")
		}
		if c.TenantCompanyID == "" {
			return fmt.Errorf("--tenant-company-id is required when setting tenant admin (the company that owns this tenant)")
		}

		// Load tenant credentials for target tenant operations by context name
		tenantCreds, err := common.LoadCredentialsFromContextName(c.TenantContext, "")
		if err != nil {
			return fmt.Errorf("failed to load tenant context credentials: %w", err)
		}
		c.tenantCredentials = tenantCreds
	}

	return nil
}

func (c *AdminCommand) setAdmin(ctx context.Context) error {
	// Get the user ID if email was provided
	var userID uuid.UUID
	var err error

	if c.UserEmail != "" {
		userID, err = c.getUserIDByEmail(ctx)
		if err != nil {
			return fmt.Errorf("failed to find user by email: %w", err)
		}
	} else {
		userID, err = uuid.FromString(c.UserID)
		if err != nil {
			return fmt.Errorf("invalid user ID: %w", err)
		}
	}

	// Set admin based on tenant or company
	if c.TenantID != "" {
		err = c.setAdminForTenant(ctx, userID)
	} else {
		err = c.setAdminForCompany(ctx, userID)
	}

	if err != nil {
		return err
	}

	fmt.Printf("Admin privileges set for user %s\n", userID)

	return nil
}

func (c *AdminCommand) getUserIDByEmail(ctx context.Context) (uuid.UUID, error) {
	// Create IDP management client (connects to console tenant by default via context)
	mgmtClient, err := c.credentials.NewManagementClient()
	if err != nil {
		return uuid.Nil, err
	}

	// Get single user ID by email
	userID, err := common.GetSingleUserIDByEmail(ctx, mgmtClient, c.UserEmail)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w (make sure your context is set to console tenant)", err)
	}

	return userID, nil
}

func (c *AdminCommand) setAdminForTenant(ctx context.Context, userID uuid.UUID) error {
	tenantID, err := uuid.FromString(c.TenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID: %w", err)
	}

	// Parse the company ID that owns this tenant (used as organization ID)
	tenantCompanyID, err := uuid.FromString(c.TenantCompanyID)
	if err != nil {
		return fmt.Errorf("invalid tenant company ID: %w", err)
	}

	// Connect to the target tenant
	tenantCredOpt, err := c.tenantCredentials.GetClientCredentials()
	if err != nil {
		return fmt.Errorf("failed to create tenant client credentials: %w", err)
	}

	// Create authz client for the target tenant using tenant credentials
	authzClient, err := authz.NewClient(c.tenantCredentials.URL, authz.JSONClient(tenantCredOpt), authz.TenantID(tenantID))
	if err != nil {
		return fmt.Errorf("failed to create authz client: %w", err)
	}

	// First, ensure the user object exists in this tenant's authz system
	_, err = authzClient.GetObject(ctx, userID)
	if err != nil {
		// User object doesn't exist, create it
		// Create user object with the company ID as organization ID
		// This is how authz works: the organization ID is the company that owns the tenant
		_, err = authzClient.CreateObject(ctx, userID, authz.UserObjectTypeID, "", authz.OrganizationID(tenantCompanyID))
		if err != nil {
			return fmt.Errorf("failed to create user object: %w", err)
		}
	}

	// Create admin edge from user to company (organization)
	// This is the edge that grants admin privileges in the tenant
	edgeID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("failed to generate edge ID: %w", err)
	}

	_, err = authzClient.CreateEdge(ctx, edgeID, userID, tenantCompanyID, ucauthz.AdminEdgeTypeID)
	if err != nil {
		return fmt.Errorf("failed to create admin edge: %w", err)
	}

	return nil
}

func (c *AdminCommand) setAdminForCompany(ctx context.Context, userID uuid.UUID) error {
	companyID, err := uuid.FromString(c.CompanyID)
	if err != nil {
		return fmt.Errorf("invalid company ID: %w", err)
	}

	// Create authz client connected to console tenant
	authzClient, err := c.credentials.NewAuthzClient()
	if err != nil {
		return err
	}

	rbacClient := authz.NewRBACClient(authzClient)

	// First, ensure the user object exists in console tenant's authz system
	user, err := rbacClient.GetUser(ctx, userID)
	if err != nil {
		// User object doesn't exist in console tenant, create it
		// Get the company to find its organization ID
		companyObj, err := authzClient.GetObject(ctx, companyID)
		if err != nil {
			return fmt.Errorf("failed to get company object: %w", err)
		}

		// Create user object with the company's organization ID
		_, err = authzClient.CreateObject(ctx, userID, authz.UserObjectTypeID, "", authz.OrganizationID(companyObj.OrganizationID))
		if err != nil {
			return fmt.Errorf("failed to create user object in console tenant: %w", err)
		}

		// Now get the user
		user, err = rbacClient.GetUser(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to get user after creation: %w", err)
		}
	}

	// Get the company group (company is a group in console tenant)
	group, err := rbacClient.GetGroup(ctx, companyID)
	if err != nil {
		return fmt.Errorf("failed to get company group: %w", err)
	}

	// Add user to company with admin role
	_, err = group.AddUserRole(ctx, *user, ucauthz.AdminRole)
	if err != nil {
		return fmt.Errorf("failed to add admin role: %w", err)
	}

	return nil
}
