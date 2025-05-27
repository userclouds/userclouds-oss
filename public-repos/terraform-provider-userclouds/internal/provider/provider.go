package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"userclouds.com/infra/jsonclient"
)

// Ensure UserCloudsProvider satisfies various provider interfaces.
var _ provider.Provider = &UserCloudsProvider{}

// UserCloudsProvider defines the provider implementation.
type UserCloudsProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// UserCloudsProviderModel describes the provider data model.
type UserCloudsProviderModel struct {
	TenantURL    types.String `tfsdk:"tenant_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

// Metadata is used to return information about the provider.
func (p *UserCloudsProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "userclouds"
	resp.Version = p.version
}

var ucSchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"tenant_url": schema.StringAttribute{
			MarkdownDescription: "URL of the UserClouds tenant. May also be set using the `USERCLOUDS_TENANT_URL` environment variable.",
			Optional:            true,
		},
		"client_id": schema.StringAttribute{
			MarkdownDescription: "UserClouds API Client ID. May also be set using the `USERCLOUDS_CLIENT_ID` environment variable.",
			Optional:            true,
		},
		"client_secret": schema.StringAttribute{
			MarkdownDescription: "UserClouds API Client Secret. May also be set using the `USERCLOUDS_CLIENT_SECRET` environment variable.",
			Optional:            true,
			Sensitive:           true,
		},
	},
}

// Schema is used to define the schema for the provider.
func (p *UserCloudsProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = ucSchema
}

func getConfigValue(resp *provider.ConfigureResponse, envVarName string, configAttrName string, configAttrValue string) string {
	value := os.Getenv(envVarName)
	if configAttrValue != "" {
		value = configAttrValue
	}

	if value == "" {
		resp.Diagnostics.AddError(
			"Missing "+ucSchema.Attributes[configAttrName].GetMarkdownDescription(),
			"While configuring the provider, the tenant URL was not found in the "+envVarName+" environment variable or the provider "+configAttrName+" configuration attribute.",
		)
	}

	return value
}

// Configure is used to configure the provider.
func (p *UserCloudsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data UserCloudsProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tenantURL := getConfigValue(resp, "USERCLOUDS_TENANT_URL", "tenant_url", data.TenantURL.ValueString())
	clientID := getConfigValue(resp, "USERCLOUDS_CLIENT_ID", "client_id", data.ClientID.ValueString())
	clientSecret := getConfigValue(resp, "USERCLOUDS_CLIENT_SECRET", "client_secret", data.ClientSecret.ValueString())

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "tenant_url", tenantURL)
	ctx = tflog.SetField(ctx, "client_id", clientID)
	ctx = tflog.SetField(ctx, "client_secret", clientSecret)
	tflog.MaskFieldValuesWithFieldKeys(ctx, "client_secret")

	options := []jsonclient.Option{
		jsonclient.ClientCredentialsTokenSource(tenantURL+"/oidc/token", clientID, clientSecret, nil),
		jsonclient.HeaderUserAgent(fmt.Sprintf("UserClouds Terraform Provider v%v", p.version)),
	}
	client := jsonclient.New(strings.TrimSuffix(tenantURL, "/"), options...)
	if err := client.ValidateBearerTokenHeader(); err != nil {
		resp.Diagnostics.AddError(
			"Failed to create UserClouds API client",
			"An unexpected error occurred while initializing the UserClouds API client: "+err.Error())
		return
	}
	resp.DataSourceData = client
	resp.ResourceData = client
}

// Resources is used to define the resources for the provider.
func (p *UserCloudsProvider) Resources(ctx context.Context) []func() resource.Resource {
	// If we write some resources by hand, can add them here as well
	return generatedResources
}

// DataSources is used to define the data sources for the provider.
func (p *UserCloudsProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	// If we write some data sources by hand, can add them here as well
	return generatedDataSources
}

// New returns a new provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &UserCloudsProvider{
			version: version,
		}
	}
}
