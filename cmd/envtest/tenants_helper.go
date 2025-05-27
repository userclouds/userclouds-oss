package main

import (
	"context"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/yamlconfig"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/companyconfig"
)

// TenantRecord is a struct that tenant information needed to run a test on.
type TenantRecord struct {
	ID           uuid.UUID `yaml:"id" json:"id" validate:"notnil"`
	Name         string    `yaml:"name" json:"name" validate:"notempty"`
	TenantURL    string    `yaml:"url" json:"url" validate:"notempty"`
	ClientID     string    `yaml:"client_id" json:"client_id" validate:"notempty"`
	ClientSecret string    `yaml:"client_secret" json:"client_secret" validate:"notempty"`
}

//go:generate genvalidate TenantRecord

// TenantRecords is a struct that holds a list of Tenants we run tests on.
type TenantRecords struct {
	Tenants []TenantRecord `yaml:"tenants" json:"tenants"`
}

//go:generate genvalidate TenantRecords

func collectTenantURLs(tis []cmdline.TenantClientInfo) string {
	tenantURLs := make([]string, len(tis))
	for i, ti := range tis {
		tenantURLs[i] = ti.TenantURL
	}
	return strings.Join(tenantURLs, ", ")
}

func loadTenantFromDB(ctx context.Context, tenantIDorName string, useRegionalURLs, useEKS bool) (cmdline.TenantClientInfo, error) {
	storage := cmdline.GetCompanyStorage(ctx)
	tenant, err := cmdline.GetTenantByIDOrName(ctx, storage, tenantIDorName)
	if err != nil {
		return cmdline.TenantClientInfo{}, ucerr.Wrap(err)
	}
	company, err := storage.GetCompany(ctx, tenant.CompanyID)
	if err != nil {
		return cmdline.TenantClientInfo{}, ucerr.Wrap(err)
	}
	if universe.Current().IsProd() && company.Type == companyconfig.CompanyTypeCustomer {
		return cmdline.TenantClientInfo{}, ucerr.New("Can't run environment tests on customer tenants in prod")
	}
	tenantURL, err := cmdline.GetTenantURL(ctx, tenant.TenantURL, useRegionalURLs, useEKS)
	if err != nil {
		return cmdline.TenantClientInfo{}, ucerr.Wrap(err)

	}
	ts, err := cmdline.GetTokenSourceForTenant(ctx, storage, tenant, tenantURL)
	if err != nil {
		return cmdline.TenantClientInfo{}, ucerr.Wrap(err)
	}
	return cmdline.TenantClientInfo{
		ID:          tenant.ID,
		Name:        tenant.Name,
		TenantURL:   tenantURL,
		TokenSource: ts,
	}, nil
}

func loadTenantsFromFile(ctx context.Context, tenantsFile string, useRegionalURLs, useEKS bool) ([]cmdline.TenantClientInfo, error) {
	tenantRecords := loadTenantsRecordsFromFile(ctx, tenantsFile)
	uclog.Infof(ctx, "Loaded %v tenants from file %v", len(tenantRecords), tenantsFile)
	tis := make([]cmdline.TenantClientInfo, 0, len(tenantRecords))
	for _, tr := range tenantRecords {
		tu, err := cmdline.GetTenantURL(ctx, tr.TenantURL, useRegionalURLs, useEKS)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		ts, err := jsonclient.ClientCredentialsForURL(tu, tr.ClientID, tr.ClientSecret, nil)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		tis = append(tis, cmdline.TenantClientInfo{
			ID:          tr.ID,
			Name:        tr.Name,
			TenantURL:   tu,
			TokenSource: ts,
		})
	}
	return tis, nil
}

func loadTenantsRecordsFromFile(ctx context.Context, tenantsFile string) []TenantRecord {

	if strings.HasPrefix(tenantsFile, "s3://") {
		tenantRecords, err := getTenantsRecordsFromS3(ctx, tenantsFile)
		if err != nil {
			uclog.Fatalf(ctx, "can't load tenants file from %v: %v", tenantsFile, err)
		}
		return tenantRecords
	}
	var trs TenantRecords
	if err := yamlconfig.LoadAndDecodeFromPath(tenantsFile, &trs, false); err != nil {
		uclog.Fatalf(ctx, "can't load tenants file from %v: %v", tenantsFile, err)
	}
	if len(trs.Tenants) < 1 {
		uclog.Fatalf(ctx, "No tenants found in file %v", tenantsFile)
	}
	return trs.Tenants
}

func getTenantInfos(ctx context.Context, tenantsFile, tenantIDorName string, useRegionalURLs, useEKS bool) ([]cmdline.TenantClientInfo, error) {
	if tenantsFile != "" {
		return loadTenantsFromFile(ctx, tenantsFile, useRegionalURLs, useEKS)
	}
	ti, err := loadTenantFromDB(ctx, tenantIDorName, useRegionalURLs, useEKS)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return []cmdline.TenantClientInfo{ti}, nil
}

func getTenantsRecordsFromS3(ctx context.Context, tenantsFile string) ([]TenantRecord, error) {
	var cfg aws.Config
	var err error
	if bucketRegion := os.Getenv("ENVTEST_BUCKET_REGION"); bucketRegion == "" {
		cfg, err = ucaws.NewConfigWithDefaultRegion(ctx)
	} else {
		cfg, err = ucaws.NewConfigWithRegion(ctx, bucketRegion)
	}

	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	s3Url, err := url.Parse(tenantsFile)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	s3Service := s3.NewFromConfig(cfg)
	s3Key := s3Url.Path[1:] // remove leading '/'
	uclog.Infof(ctx, "Loading tenants from %s", tenantsFile)
	object, err := s3Service.GetObject(ctx, &s3.GetObjectInput{Bucket: &s3Url.Host, Key: &s3Key})
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	defer object.Body.Close()
	tenantRecords := TenantRecords{}
	err = yamlconfig.LoadAndDecode(object.Body, tenantsFile, &tenantRecords, false)
	return tenantRecords.Tenants, ucerr.Wrap(err)
}
