package main

import (
	"github.com/spf13/cobra"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/cmd/ucctl/context"
	"userclouds.com/cmd/ucctl/create"
	"userclouds.com/cmd/ucctl/delete"
	"userclouds.com/cmd/ucctl/describe"
	"userclouds.com/cmd/ucctl/dump"
	"userclouds.com/cmd/ucctl/get"
	"userclouds.com/cmd/ucctl/load"
	"userclouds.com/cmd/ucctl/set"
	"userclouds.com/cmd/ucctl/sync"
)

const (
	RootUsage       = "ucctl"
	RootShort       = "CLI utility for interacting with userclouds"
	RootLong        = `CLI utility for interacting with userclouds`
	SyncUsage       = "sync"
	SyncShort       = "Sync resources between environments"
	SyncLong        = `Sync UserClouds resources between different environments`
	CreateUsage     = "create"
	CreateShort     = "Create userclouds resources"
	CreateLong      = `Create userclouds resources`
	CreateUserUsage = "user"
	CreateUserShort = "Create a new user"
	CreateUserLong  = `Create a new user with password authentication, OIDC authentication, or without authentication.

The user is created in the tenant specified by the current context.
- For console employee users: set context to a console tenant context
- For regular tenant users: set context to a regular tenant context

When creating a user without authentication, the authentication will be automatically
added when the user logs in for the first time via OIDC (based on email match).`
	ContextUsage  = "context"
	ContextShort  = "Manage userclouds contexts"
	ContextLong   = `Manage userclouds contexts for different environments`
	SetUsage      = "set"
	SetShort      = "Set userclouds resources"
	SetLong       = `Set userclouds resources such as admin privileges`
	GetUsage      = "get"
	GetShort      = "Get userclouds resources"
	GetLong       = `Get userclouds resources such as users, objects, edges, etc`
	DeleteUsage   = "delete"
	DeleteShort   = "Delete userclouds resources"
	DeleteLong    = `Delete userclouds resources such as objects, edges, object types, and edge types`
	DescribeUsage = "describe"
	DescribeShort = "Describe userclouds resources with detailed information"
	DescribeLong  = `Describe userclouds resources with detailed information including relationships.

Similar to kubectl describe, this command shows extensive details about a resource including
all related resources and relationships (edges, object types, etc.).`
	DumpUsage = "dump"
	DumpShort = "Dump userclouds resources to file"
	DumpLong  = `Dump userclouds resources to a JSON file for backup or migration`
	LoadUsage = "load"
	LoadShort = "Load userclouds resources from file"
	LoadLong  = `Load userclouds resources from a JSON dump file`
)

type Root struct {
	ConfigPath string
}

func NewRoot() *Root {
	return &Root{}
}

func (r *Root) Execute() error {
	return r.Command().Execute()
}

func (r *Root) Command() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   RootUsage,
		Short: RootShort,
		Long:  RootLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	// Add persistent flag for config path
	rootCmd.PersistentFlags().StringVar(&r.ConfigPath, "uc-context", "", "Path to context config file")

	rootCmd.AddCommand(SyncCommand())
	rootCmd.AddCommand(CreateCommand())
	rootCmd.AddCommand(DeleteCommand())
	rootCmd.AddCommand(SetCommand())
	rootCmd.AddCommand(ContextCommand())
	rootCmd.AddCommand(GetCommand())
	rootCmd.AddCommand(DescribeCommand())
	rootCmd.AddCommand(DumpCommand())
	rootCmd.AddCommand(LoadCommand())
	return rootCmd
}

func SyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   SyncUsage,
		Short: SyncShort,
		Long:  SyncLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Add subcommands
	// TODO: Add more sync subcommands (tokenizer, userstore, authn, logserver)
	cmd.AddCommand(sync.NewTenantCommand())
	return cmd
}

func CreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   CreateUsage,
		Short: CreateShort,
		Long:  CreateLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(CreateUserCommand())
	return cmd
}

func GetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   GetUsage,
		Short: GetShort,
		Long:  GetLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(GetUsersCommand())
	cmd.AddCommand(GetUserCommand())
	cmd.AddCommand(GetObjectTypesCommand())
	cmd.AddCommand(GetObjectTypeCommand())
	cmd.AddCommand(GetObjectsCommand())
	cmd.AddCommand(GetObjectCommand())
	cmd.AddCommand(GetEdgeTypesCommand())
	cmd.AddCommand(GetEdgeTypeCommand())
	cmd.AddCommand(GetEdgesCommand())
	cmd.AddCommand(GetEdgeCommand())
	return cmd
}

func DeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   DeleteUsage,
		Short: DeleteShort,
		Long:  DeleteLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(DeleteObjectCommand())
	cmd.AddCommand(DeleteObjectTypeCommand())
	cmd.AddCommand(DeleteEdgeCommand())
	cmd.AddCommand(DeleteEdgeTypeCommand())
	cmd.AddCommand(DeleteUserCommand())
	return cmd
}

func SetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   SetUsage,
		Short: SetShort,
		Long:  SetLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(SetAdminCommand())
	cmd.AddCommand(SetUserCommand())
	cmd.AddCommand(SetObjectCommand())
	cmd.AddCommand(SetEdgeTypeCommand())
	return cmd
}

func CreateUserCommand() *cobra.Command {
	uc := create.UserCommand{}
	cmd := &cobra.Command{
		Use:   CreateUserUsage,
		Short: CreateUserShort,
		Long:  CreateUserLong,
		RunE:  uc.RunE,
	}

	common.AddAuthFlags(cmd, &uc.URL, &uc.ClientID, &uc.ClientSecret, &uc.ClientSecretVar, &uc.AuthnType)

	cmd.Flags().BoolVarP(&uc.Admin, "admin", "a", false, "Admin user")
	cmd.Flags().BoolVarP(&uc.Verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().StringVarP(&uc.OrganizationID, "organization-id", "", "", "organization ID for the user")
	cmd.Flags().StringVarP(&uc.Email, "email", "", "", "user email address")
	cmd.Flags().StringVarP(&uc.Name, "name", "", "", "user name")
	cmd.Flags().StringVarP(&uc.Username, "username", "", "", "username for password authentication")
	cmd.Flags().StringVarP(&uc.Password, "password", "", "", "password for password authentication")
	cmd.Flags().StringVarP(&uc.OIDCProvider, "oidc-provider", "", "", "OIDC provider (e.g., google, github)")
	cmd.Flags().StringVarP(&uc.OIDCIssuerURL, "oidc-issuer-url", "", "", "OIDC issuer URL")
	cmd.Flags().StringVarP(&uc.OIDCSubject, "oidc-subject", "", "", "OIDC subject ID")

	return cmd
}

func ContextCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     ContextUsage,
		Aliases: []string{"ctx"},
		Short:   ContextShort,
		Long:    ContextLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(ContextListCommand())
	cmd.AddCommand(ContextUseCommand())
	cmd.AddCommand(ContextSetCommand())
	cmd.AddCommand(ContextDeleteCommand())
	cmd.AddCommand(ContextShowCommand())
	return cmd
}

func ContextListCommand() *cobra.Command {
	lc := context.ListCommand{}
	return &cobra.Command{
		Use:   "list",
		Short: "List all contexts",
		Long:  "List all configured UserClouds contexts",
		RunE:  lc.RunE,
	}
}

func ContextUseCommand() *cobra.Command {
	uc := context.UseCommand{}
	return &cobra.Command{
		Use:               "use <context-name>",
		Short:             "Switch to a context",
		Long:              "Switch to a different UserClouds context",
		RunE:              uc.RunE,
		ValidArgsFunction: context.ValidContextArgs,
	}
}

func ContextSetCommand() *cobra.Command {
	sc := context.SetCommand{}
	cmd := &cobra.Command{
		Use:   "set <context-name>",
		Short: "Create or update a context",
		Long: `Create or update a UserClouds context configuration.

For global operations, use the console tenant context.`,
		RunE: sc.RunE,
	}

	cmd.Flags().StringVarP(&sc.URL, "url", "", "", "UserClouds URL (required)")
	cmd.Flags().StringVarP(&sc.ClientID, "client-id", "", "", "OAuth2 client ID (required)")
	cmd.Flags().StringVarP(&sc.ClientSecret, "client-secret", "", "", "OAuth2 client secret (required)")

	return cmd
}

func ContextDeleteCommand() *cobra.Command {
	dc := context.DeleteCommand{}
	return &cobra.Command{
		Use:               "delete <context-name>",
		Short:             "Delete a context",
		Long:              "Delete a UserClouds context configuration",
		RunE:              dc.RunE,
		ValidArgsFunction: context.ValidContextArgs,
	}
}

func ContextShowCommand() *cobra.Command {
	sc := context.ShowCommand{}
	return &cobra.Command{
		Use:               "show [context-name]",
		Short:             "Show current or specified context",
		Long:              "Display the currently active UserClouds context, or a specific context if provided",
		RunE:              sc.RunE,
		ValidArgsFunction: context.ValidContextArgs,
	}
}

func GetUsersCommand() *cobra.Command {
	gu := get.UsersCommand{}
	cmd := &cobra.Command{
		Use:   "users",
		Short: "List all users",
		Long:  "List all users",
		RunE:  gu.RunE,
	}

	common.AddAuthFlags(cmd, &gu.URL, &gu.ClientID, &gu.ClientSecret, &gu.ClientSecretVar, &gu.AuthnType)

	return cmd
}

func GetUserCommand() *cobra.Command {
	gu := get.UserCommand{}
	cmd := &cobra.Command{
		Use:   "user <email_or_uuid>",
		Short: "Get user profiles by email",
		Long:  "Get user profiles by email",
		RunE:  gu.RunE,
	}

	common.AddAuthFlags(cmd, &gu.URL, &gu.ClientID, &gu.ClientSecret, &gu.ClientSecretVar, &gu.AuthnType)

	return cmd
}

func GetObjectTypesCommand() *cobra.Command {
	got := get.ObjectTypesCommand{}
	cmd := &cobra.Command{
		Use:   "objecttypes",
		Short: "List all object types",
		Long:  "List all AuthZ object types with pagination support",
		RunE:  got.RunE,
	}

	common.AddListCommandFlags(cmd, &got.URL, &got.ClientID, &got.ClientSecret, &got.ClientSecretVar, &got.Limit, &got.Cursor, &got.NoPager, &got.Filter, &got.RawFilter, &got.Output, &got.AuthnType)

	return cmd
}

func GetObjectTypeCommand() *cobra.Command {
	got := get.ObjectTypeCommand{}
	cmd := &cobra.Command{
		Use:   "objecttype <id>",
		Short: "Get object type by ID",
		Long:  "Get detailed information about an AuthZ object type by its ID",
		RunE:  got.RunE,
	}

	common.AddAuthFlags(cmd, &got.URL, &got.ClientID, &got.ClientSecret, &got.ClientSecretVar, &got.AuthnType)

	return cmd
}

func GetObjectsCommand() *cobra.Command {
	gobj := get.ObjectsCommand{}
	cmd := &cobra.Command{
		Use:   "objects",
		Short: "List all objects",
		Long:  "List all AuthZ objects with pagination and filtering support",
		RunE:  gobj.RunE,
	}

	common.AddListCommandFlags(cmd, &gobj.URL, &gobj.ClientID, &gobj.ClientSecret, &gobj.ClientSecretVar, &gobj.Limit, &gobj.Cursor, &gobj.NoPager, &gobj.Filter, &gobj.RawFilter, &gobj.Output, &gobj.AuthnType)

	return cmd
}

func GetObjectCommand() *cobra.Command {
	gobj := get.ObjectCommand{}
	cmd := &cobra.Command{
		Use:   "object <id>",
		Short: "Get object by ID",
		Long:  "Get detailed information about an AuthZ object by its ID",
		RunE:  gobj.RunE,
	}

	common.AddAuthFlags(cmd, &gobj.URL, &gobj.ClientID, &gobj.ClientSecret, &gobj.ClientSecretVar, &gobj.AuthnType)

	return cmd
}

func GetEdgeTypesCommand() *cobra.Command {
	ge := get.EdgeTypesCommand{}
	cmd := &cobra.Command{
		Use:   "edgetypes",
		Short: "List all edge types",
		Long:  "List all AuthZ edge types with pagination and filtering support",
		RunE:  ge.RunE,
	}

	common.AddListCommandFlags(cmd, &ge.URL, &ge.ClientID, &ge.ClientSecret, &ge.ClientSecretVar, &ge.Limit, &ge.Cursor, &ge.NoPager, &ge.Filter, &ge.RawFilter, &ge.Output, &ge.AuthnType)

	return cmd
}

func GetEdgeTypeCommand() *cobra.Command {
	ge := get.EdgeTypeCommand{}
	cmd := &cobra.Command{
		Use:   "edgetype <id>",
		Short: "Get edge type by ID",
		Long:  "Get detailed information about an AuthZ edge type by its ID",
		RunE:  ge.RunE,
	}

	common.AddAuthFlags(cmd, &ge.URL, &ge.ClientID, &ge.ClientSecret, &ge.ClientSecretVar, &ge.AuthnType)

	return cmd
}

func GetEdgesCommand() *cobra.Command {
	ge := get.EdgesCommand{}
	cmd := &cobra.Command{
		Use:   "edges",
		Short: "List all edges",
		Long: `List all AuthZ edges with pagination and filtering support

Filter by source object:
  ucctl get edges --filter "source_object_id=<object-uuid>"

Filter by target object:
  ucctl get edges --filter "target_object_id=<object-uuid>"

Combine filters with AND (&) or OR (|):
  ucctl get edges --filter "source_object_id=<uuid>&target_object_id=<uuid>"

Available filter fields: id, source_object_id, target_object_id, created, updated`,
		RunE: ge.RunE,
	}

	common.AddListCommandFlags(cmd, &ge.URL, &ge.ClientID, &ge.ClientSecret, &ge.ClientSecretVar, &ge.Limit, &ge.Cursor, &ge.NoPager, &ge.Filter, &ge.RawFilter, &ge.Output, &ge.AuthnType)

	return cmd
}

func GetEdgeCommand() *cobra.Command {
	ge := get.EdgeCommand{}
	cmd := &cobra.Command{
		Use:   "edge <id>",
		Short: "Get edge by ID",
		Long:  "Get detailed information about an AuthZ edge by its ID",
		RunE:  ge.RunE,
	}

	common.AddAuthFlags(cmd, &ge.URL, &ge.ClientID, &ge.ClientSecret, &ge.ClientSecretVar, &ge.AuthnType)

	return cmd
}

func DeleteObjectCommand() *cobra.Command {
	dc := delete.ObjectCommand{}
	cmd := &cobra.Command{
		Use:   "object <id>",
		Short: "Delete an object by ID",
		Long:  "Delete an AuthZ object by its ID",
		RunE:  dc.RunE,
	}

	common.AddAuthFlags(cmd, &dc.URL, &dc.ClientID, &dc.ClientSecret, &dc.ClientSecretVar, &dc.AuthnType)
	cmd.Flags().BoolVarP(&dc.AutoApprove, "auto-approve", "y", false, "automatically approve deletion without prompting")

	return cmd
}

func DeleteObjectTypeCommand() *cobra.Command {
	dc := delete.ObjectTypeCommand{}
	cmd := &cobra.Command{
		Use:   "objecttype <id>",
		Short: "Delete an object type by ID",
		Long:  "Delete an AuthZ object type by its ID",
		RunE:  dc.RunE,
	}

	common.AddAuthFlags(cmd, &dc.URL, &dc.ClientID, &dc.ClientSecret, &dc.ClientSecretVar, &dc.AuthnType)
	cmd.Flags().BoolVarP(&dc.AutoApprove, "auto-approve", "y", false, "automatically approve deletion without prompting")

	return cmd
}

func DeleteEdgeCommand() *cobra.Command {
	dc := delete.EdgeCommand{}
	cmd := &cobra.Command{
		Use:   "edge <id>",
		Short: "Delete an edge by ID",
		Long:  "Delete an AuthZ edge by its ID",
		RunE:  dc.RunE,
	}

	common.AddAuthFlags(cmd, &dc.URL, &dc.ClientID, &dc.ClientSecret, &dc.ClientSecretVar, &dc.AuthnType)
	cmd.Flags().BoolVarP(&dc.AutoApprove, "auto-approve", "y", false, "automatically approve deletion without prompting")

	return cmd
}

func DeleteEdgeTypeCommand() *cobra.Command {
	dc := delete.EdgeTypeCommand{}
	cmd := &cobra.Command{
		Use:   "edgetype <id>",
		Short: "Delete an edge type by ID",
		Long:  "Delete an AuthZ edge type by its ID",
		RunE:  dc.RunE,
	}

	common.AddAuthFlags(cmd, &dc.URL, &dc.ClientID, &dc.ClientSecret, &dc.ClientSecretVar, &dc.AuthnType)
	cmd.Flags().BoolVarP(&dc.AutoApprove, "auto-approve", "y", false, "automatically approve deletion without prompting")

	return cmd
}

func DeleteUserCommand() *cobra.Command {
	uc := delete.UserCommand{}
	cmd := &cobra.Command{
		Use:   "user <email|user-id>",
		Short: "Delete user(s) by email or ID",
		Long: `Delete user(s) by email address or user ID.

If the argument is a UUID, the single user with that ID will be deleted.
If the argument is an email address, ALL users with that email will be deleted after confirmation.

This command will prompt for confirmation before deletion unless --auto-approve is used.`,
		RunE: uc.RunE,
	}

	common.AddAuthFlags(cmd, &uc.URL, &uc.ClientID, &uc.ClientSecret, &uc.ClientSecretVar, &uc.AuthnType)
	cmd.Flags().BoolVarP(&uc.AutoApprove, "auto-approve", "y", false, "automatically approve deletion without prompting")
	cmd.Flags().BoolVarP(&uc.Verbose, "verbose", "v", false, "verbose output")

	return cmd
}

func SetObjectCommand() *cobra.Command {
	sc := set.ObjectCommand{}
	cmd := &cobra.Command{
		Use:   "object <id>",
		Short: "Update an object's alias",
		Long:  "Update an AuthZ object's alias field by its ID",
		RunE:  sc.RunE,
	}

	common.AddAuthFlags(cmd, &sc.URL, &sc.ClientID, &sc.ClientSecret, &sc.ClientSecretVar, &sc.AuthnType)

	cmd.Flags().StringVarP(&sc.Alias, "alias", "a", "", "new alias for the object")
	cmd.Flags().BoolVarP(&sc.ClearAlias, "clear-alias", "", false, "clear the alias (set to null)")

	return cmd
}

func SetEdgeTypeCommand() *cobra.Command {
	sc := set.EdgeTypeCommand{}
	cmd := &cobra.Command{
		Use:   "edgetype <id>",
		Short: "Update an edge type",
		Long:  "Update an AuthZ edge type's properties by its ID",
		RunE:  sc.RunE,
	}

	common.AddAuthFlags(cmd, &sc.URL, &sc.ClientID, &sc.ClientSecret, &sc.ClientSecretVar, &sc.AuthnType)

	cmd.Flags().StringVarP(&sc.TypeName, "type-name", "n", "", "edge type name (required)")
	cmd.Flags().StringVarP(&sc.SourceObjectTypeID, "source-object-type-id", "s", "", "source object type ID (required)")
	cmd.Flags().StringVarP(&sc.TargetObjectTypeID, "target-object-type-id", "t", "", "target object type ID (required)")
	cmd.Flags().StringVarP(&sc.Attributes, "attributes", "", "", "attributes as JSON string")

	return cmd
}

func SetUserCommand() *cobra.Command {
	uc := set.UserCommand{}
	cmd := &cobra.Command{
		Use:   "user <email>",
		Short: "Update a user profile",
		Long: `Update a user's profile information including email, name, and authentication.

The user is identified by their email address.
This command allows you to update user profile fields and authentication credentials.
You can update the profile, password, or OIDC authentication independently.`,
		RunE: uc.RunE,
	}

	common.AddAuthFlags(cmd, &uc.URL, &uc.ClientID, &uc.ClientSecret, &uc.ClientSecretVar, &uc.AuthnType)

	cmd.Flags().BoolVarP(&uc.Verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().StringVarP(&uc.Email, "email", "", "", "user email address")
	cmd.Flags().StringVarP(&uc.Name, "name", "", "", "user name")
	cmd.Flags().BoolVarP(&uc.EmailVerified, "email-verified", "", false, "mark email as verified")
	cmd.Flags().StringVarP(&uc.Username, "username", "", "", "username for password authentication")
	cmd.Flags().StringVarP(&uc.Password, "password", "", "", "password for password authentication")
	cmd.Flags().StringVarP(&uc.OIDCProvider, "oidc-provider", "", "", "OIDC provider (e.g., google, github)")
	cmd.Flags().StringVarP(&uc.OIDCIssuerURL, "oidc-issuer-url", "", "", "OIDC issuer URL")
	cmd.Flags().StringVarP(&uc.OIDCSubject, "oidc-subject", "", "", "OIDC subject ID")

	return cmd
}

func SetAdminCommand() *cobra.Command {
	ac := set.AdminCommand{}
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Set admin privileges for a user",
		Long: `Set admin privileges for a user on a tenant or company.

The user can be specified by email or ID.
Either --tenant-id or --company-id must be specified.

For tenant operations:
  - Your current context should be the console tenant (where the user exists)
  - Use --tenant-context to specify the target tenant's context
  - Use --tenant-company-id to specify the company that owns the tenant
  - The user will be looked up in the console tenant and added as admin to the target tenant

For company operations:
  - Switch to the console tenant context first
  - The company admin role will be set in the console tenant`,
		RunE: ac.RunE,
	}

	common.AddAuthFlags(cmd, &ac.URL, &ac.ClientID, &ac.ClientSecret, &ac.ClientSecretVar, &ac.AuthnType)

	cmd.Flags().BoolVarP(&ac.Verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().StringVarP(&ac.UserEmail, "email", "e", "", "user email address")
	cmd.Flags().StringVarP(&ac.UserID, "user-id", "u", "", "user ID")
	cmd.Flags().StringVarP(&ac.TenantID, "tenant-id", "t", "", "tenant ID to set admin for")
	cmd.Flags().StringVarP(&ac.TenantContext, "tenant-context", "", "", "context name for the target tenant (required for tenant operations)")
	cmd.Flags().StringVarP(&ac.TenantCompanyID, "tenant-company-id", "", "", "company ID that owns the target tenant (required for tenant operations)")
	cmd.Flags().StringVarP(&ac.CompanyID, "company-id", "c", "", "company ID")

	return cmd
}

func DescribeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   DescribeUsage,
		Short: DescribeShort,
		Long:  DescribeLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(DescribeUserCommand())
	cmd.AddCommand(DescribeObjectCommand())
	cmd.AddCommand(DescribeObjectTypeCommand())
	cmd.AddCommand(DescribeEdgeCommand())
	cmd.AddCommand(DescribeEdgeTypeCommand())
	return cmd
}

func DescribeUserCommand() *cobra.Command {
	uc := describe.UserCommand{}
	cmd := &cobra.Command{
		Use:   "user <email|user-id>",
		Short: "Describe a user with detailed information",
		Long: `Describe a user with detailed information including:
  - Basic profile information
  - Authentication methods
  - Company memberships
  - Edges (relationships with other objects)`,
		RunE: uc.RunE,
	}

	common.AddAuthFlags(cmd, &uc.URL, &uc.ClientID, &uc.ClientSecret, &uc.ClientSecretVar, &uc.AuthnType)

	return cmd
}

func DescribeObjectCommand() *cobra.Command {
	oc := describe.ObjectCommand{}
	cmd := &cobra.Command{
		Use:   "object <object-id>",
		Short: "Describe an object with detailed information",
		Long: `Describe an AuthZ object with detailed information including:
  - Basic object information
  - Object type details
  - Outgoing edges (where object is source)
  - Incoming edges (where object is target)`,
		RunE: oc.RunE,
	}

	common.AddAuthFlags(cmd, &oc.URL, &oc.ClientID, &oc.ClientSecret, &oc.ClientSecretVar, &oc.AuthnType)

	return cmd
}

func DescribeObjectTypeCommand() *cobra.Command {
	otc := describe.ObjectTypeCommand{}
	cmd := &cobra.Command{
		Use:   "objecttype <id|name>",
		Short: "Describe an object type with detailed information",
		Long: `Describe an AuthZ object type with detailed information including:
  - Basic object type information
  - All objects of this type
  - Edge types that use this object type (as source or target)`,
		RunE: otc.RunE,
	}

	common.AddAuthFlags(cmd, &otc.URL, &otc.ClientID, &otc.ClientSecret, &otc.ClientSecretVar, &otc.AuthnType)

	return cmd
}

func DescribeEdgeCommand() *cobra.Command {
	ec := describe.EdgeCommand{}
	cmd := &cobra.Command{
		Use:   "edge <edge-id>",
		Short: "Describe an edge with detailed information",
		Long: `Describe an AuthZ edge with detailed information including:
  - Basic edge information
  - Edge type details
  - Source object details
  - Target object details
  - Relationship visualization`,
		RunE: ec.RunE,
	}

	common.AddAuthFlags(cmd, &ec.URL, &ec.ClientID, &ec.ClientSecret, &ec.ClientSecretVar, &ec.AuthnType)

	return cmd
}

func DescribeEdgeTypeCommand() *cobra.Command {
	etc := describe.EdgeTypeCommand{}
	cmd := &cobra.Command{
		Use:   "edgetype <id|name>",
		Short: "Describe an edge type with detailed information",
		Long: `Describe an AuthZ edge type with detailed information including:
  - Basic edge type information
  - Source and target object type relationship
  - All edges using this edge type`,
		RunE: etc.RunE,
	}

	common.AddAuthFlags(cmd, &etc.URL, &etc.ClientID, &etc.ClientSecret, &etc.ClientSecretVar, &etc.AuthnType)

	return cmd
}

func DumpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   DumpUsage,
		Short: DumpShort,
		Long:  DumpLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(DumpAuthzCommand())
	return cmd
}

func DumpAuthzCommand() *cobra.Command {
	dc := dump.AuthzCommand{}
	cmd := &cobra.Command{
		Use:   "authz",
		Short: "Dump all authz resources to a JSON file",
		Long: `Dump all authz resources (object types, edge types, objects, and edges) to a JSON file.

This creates a complete backup of all authorization resources that can be used for:
  - Backing up authz configuration
  - Migrating authz resources between tenants
  - Version control of authorization policies

Example:
  ucctl dump authz --output-file authz-backup.json`,
		RunE: dc.RunE,
	}

	common.AddAuthFlags(cmd, &dc.URL, &dc.ClientID, &dc.ClientSecret, &dc.ClientSecretVar, &dc.AuthnType)
	cmd.Flags().StringVarP(&dc.OutputFile, "output-file", "o", "authz-dump.json", "output file path")
	cmd.MarkFlagRequired("output-file")

	return cmd
}

func LoadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   LoadUsage,
		Short: LoadShort,
		Long:  LoadLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(LoadAuthzCommand())
	return cmd
}

func LoadAuthzCommand() *cobra.Command {
	lc := load.AuthzCommand{}
	cmd := &cobra.Command{
		Use:   "authz",
		Short: "Load authz resources from a JSON dump file",
		Long: `Load authz resources from a JSON dump file created by 'ucctl dump authz'.

This will create or update all authz resources (object types, edge types, objects, and edges)
in the target tenant based on the dump file.

Resources are loaded in dependency order:
  1. Object Types
  2. Edge Types
  3. Objects
  4. Edges

Flags:
  --dry-run         Show what would be loaded without making changes
  --skip-existing   Skip resources that already exist (default is to update them)

Example:
  ucctl load authz --input-file authz-backup.json
  ucctl load authz --input-file authz-backup.json --dry-run
  ucctl load authz --input-file authz-backup.json --skip-existing`,
		RunE: lc.RunE,
	}

	common.AddAuthFlags(cmd, &lc.URL, &lc.ClientID, &lc.ClientSecret, &lc.ClientSecretVar, &lc.AuthnType)
	cmd.Flags().StringVarP(&lc.InputFile, "input-file", "i", "", "input file path")
	cmd.Flags().BoolVar(&lc.DryRun, "dry-run", false, "show what would be loaded without making changes")
	cmd.Flags().BoolVar(&lc.SkipExisting, "skip-existing", false, "skip resources that already exist")
	cmd.MarkFlagRequired("input-file")

	return cmd
}
