package main

import (
	"context"

	"github.com/alecthomas/kong"
	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/config"
	"userclouds.com/idp/search"
	"userclouds.com/infra/kubernetes"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/internal/ucopensearch"
)

var cli struct {
	TenantID               uuid.UUID              `name:"" help:"Tenant ID." type:"uuid"`
	AccessorRemove         accessorRemove         `cmd:"" help:"Remove an accessor index."`
	AccessorSet            accessorSet            `cmd:"" help:"Set an accessor index."`
	IndexCreate            indexCreate            `cmd:"" help:"Create an index."`
	IndexContinueBootstrap indexContinueBootstrap `cmd:"" help:"Continue index bootstrap."`
	IndexDelete            indexDelete            `cmd:"" help:"Delete an index."`
	IndexGet               indexGet               `cmd:"" help:"Get an index."`
	IndexList              indexList              `cmd:"" help:"List indices."`
	IndexQuery             indexQuery             `cmd:"" help:"Query an index."`
	IndexSetColumns        indexSetColumns        `cmd:"" help:"Set index columns."`
	IndexSetDescription    indexSetDescription    `cmd:"" help:"Set index description."`
	IndexSetDisabled       indexSetDisabled       `cmd:"" help:"Disable an index."`
	IndexSetEnabled        indexSetEnabled        `cmd:"" help:"Enable an index."`
	IndexSetName           indexSetName           `cmd:"" help:"Set index name."`
	IndexSetSearchable     indexSetSearchable     `cmd:"" help:"Mark an index as searchable."`
	IndexSetType           indexSetType           `cmd:"" help:"Set index type."`
	IndexSetUnsearchable   indexSetUnsearchable   `cmd:"" help:"Mark an index as unsearchable."`
	IndexStats             indexStats             `cmd:"" help:"Get index stats."`
	IndexMetadata          indexMetadata          `cmd:"" help:"Get index metadata."`
	IndexCountDocuments    indexCountDocuments    `cmd:"" help:"Counts documents existing by givenIDs in an index."`
}

type cmdHelper struct {
	ctx       context.Context
	searchCfg *ucopensearch.Config
	searchMgr *search.Manager
	tenantID  uuid.UUID
}

func newCmdHelper(ctx context.Context, tenantID uuid.UUID) (*cmdHelper, error) {
	uv := universe.Current()
	if !uv.IsCloud() {
		return nil, ucerr.Errorf("Must used one of the Cloud universes, %v is not supported for this tool", uv)
	}
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if cfg.OpenSearchConfig == nil {
		return nil, ucerr.Errorf("OpenSearchConfig is not set in %v (IsKubernetes: %v)", uv, kubernetes.IsKubernetes())
	}
	searchMgr, err := getSearchManager(ctx, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &cmdHelper{
		ctx:       ctx,
		searchCfg: cfg.OpenSearchConfig,
		searchMgr: searchMgr,
		tenantID:  tenantID,
	}, nil
}

func (helper *cmdHelper) getSearchClient() (*ucopensearch.Client, error) {
	return ucopensearch.NewClientForLocalTool(helper.ctx, helper.searchCfg)
}

func getSearchManager(ctx context.Context, tenantID uuid.UUID) (*search.Manager, error) {
	workerClient, err := cmdline.GetWorkerClientForTool(ctx)
	if err != nil {
		return nil, ucerr.Friendlyf(err, "error getting worker client")
	}

	companyStorage := cmdline.GetCompanyStorage(ctx)
	tenant, err := cmdline.GetTenantByIDOrName(ctx, companyStorage, tenantID.String())
	if err != nil {
		return nil, ucerr.Friendlyf(err, "error getting tenant '%v'", tenantID)
	}
	uclog.Infof(ctx, "using tenant '%s' ID: %v", tenant.Name, tenant.ID)
	creds, err := cmdline.GetTokenSourceForTenant(ctx, companyStorage, tenant, tenant.TenantURL)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	idpClient, err := idp.NewClient(tenant.TenantURL, idp.OrganizationID(uuid.Nil), idp.JSONClient(creds))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	tenantDB, _, _, err := tenantdb.Connect(ctx, companyStorage, tenant.ID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	sm, err := search.NewManager(ctx, tenantID, tenantDB, idpClient, workerClient)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return sm, nil
}
func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "opensearch")
	defer logtransports.Close()
	parsedCLI := kong.Parse(&cli, kong.UsageOnError())
	helper, err := newCmdHelper(ctx, cli.TenantID)
	if err != nil {
		uclog.Fatalf(ctx, "error initializing CliContext: %v", err)
	}
	parsedCLI.FatalIfErrorf(parsedCLI.Run(helper))
}
