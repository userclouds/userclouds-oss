package main

import (
	"context"
	"net/url"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/gofrs/uuid"

	"userclouds.com/cmd/ucconfig/internal/cmd"
	"userclouds.com/idp"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

type cliContext struct {
	Context context.Context
}

// CLI flags for subcommands that access a tenant
type tenantConfig struct {
	TenantURL    string `env:"USERCLOUDS_TENANT_URL" required:"" help:"Tenant URL."`
	ClientID     string `env:"USERCLOUDS_CLIENT_ID" required:"" help:"Client ID."`
	ClientSecret string `env:"USERCLOUDS_CLIENT_SECRET" required:"" help:"Client secret."`
}

func (cfg tenantConfig) initTenantContext(ctx context.Context) tenantContext {
	tenantURL, err := url.Parse(cfg.TenantURL)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to parse tenant URL: %v", err)
	}
	fqtn := strings.Split(tenantURL.Hostname(), ".")[0]

	// Initialize IDP client based on env vars
	tokenSource := jsonclient.ClientCredentialsTokenSource(cfg.TenantURL+"/oidc/token", cfg.ClientID, cfg.ClientSecret, nil)
	idpClient, err := idp.NewClient(cfg.TenantURL, idp.OrganizationID(uuid.Nil), idp.JSONClient(tokenSource))
	if err != nil {
		uclog.Fatalf(ctx, "Failed to initialize IDP client: %v", err)
	}

	return tenantContext{FQTN: fqtn, IDPClient: idpClient}
}

// for subcommands that access a tenant
type tenantContext struct {
	IDPClient *idp.Client
	// fully-qualified tenant name, e.g. "mycompany-mytenant"
	FQTN string
}

type applyCmd struct {
	tenantConfig
	ManifestPath                string `arg:"" name:"manifest-path" help:"Path to UC JSON manifest file" type:"path"`
	DryRun                      bool   `help:"Don't actually apply the manifest, just print what would be done."`
	AutoApprove                 bool   `help:"Don't prompt for confirmation before applying the manifest."`
	TFProviderVersionConstraint string `help:"Version constraint that should be used for the terraform-provider-userclouds provider instantiation, e.g. \"~> 1.0\" or \"= 1.2.3\""`
	TFProviderDevDirPath        string `help:"Path to the directory containing the terraform-provider-userclouds binary for local provider development"`
}

// Run implements the apply subcommand
func (c *applyCmd) Run(ctx *cliContext) error {
	tenantCtx := c.initTenantContext(ctx.Context)
	return ucerr.Wrap(cmd.Apply(ctx.Context, c.DryRun, c.AutoApprove, tenantCtx.IDPClient, tenantCtx.FQTN, c.TenantURL, c.ClientID, c.ClientSecret, c.ManifestPath, c.TFProviderVersionConstraint, c.TFProviderDevDirPath))
}

type genManifestCmd struct {
	tenantConfig
	ManifestPath string `arg:"" name:"manifest-path" help:"Path to UC JSON manifest file" type:"path"`
}

// Run implements the gen-manifest subcommand
func (c *genManifestCmd) Run(ctx *cliContext) error {
	tenantCtx := c.initTenantContext(ctx.Context)
	return ucerr.Wrap(cmd.GenerateNewManifest(ctx.Context, tenantCtx.IDPClient, tenantCtx.FQTN, c.ManifestPath))
}

var cli struct {
	LogFile     string         `name:"logfile" help:"Path to the log file." type:"path"`
	Apply       applyCmd       `cmd:"" help:"Apply a config manifest file, modifying the live tenant to match what the manifest describes."`
	GenManifest genManifestCmd `cmd:"" help:"Generate a JSON manifest file from a live tenant."`
}

func main() {
	ctx := context.Background()
	cliCtx := kong.Parse(&cli)
	opts := []logtransports.ToolLogOption{logtransports.NoPrefix()}
	if cli.LogFile != "" {
		opts = append(opts, logtransports.Filename(cli.LogFile))
	}
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelInfo, uclog.LogLevelVerbose, "ucconfig", opts...)
	defer logtransports.Close()
	err := cliCtx.Run(&cliContext{Context: ctx})
	cliCtx.FatalIfErrorf(err)
}
