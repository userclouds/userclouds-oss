package ucdb

import (
	"context"
	"os"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/secret"
)

func TestGetPostgresURL(t *testing.T) {
	ctx := context.Background()
	type testCase struct {
		cfg               Config
		region            region.MachineRegion
		universe          universe.Universe
		expectedURL       string
		expectedMasterURL string
		DBHostOverride    string
		DBPortOverride    string
	}
	cases := []testCase{
		{
			cfg:            Config{Host: "free-tier4.aws-us-west-2.cockroachlabs.cloud", Port: "5432", DBName: "jerry", User: "seinfeld", Password: secret.NewTestString("cosmo"), DBProduct: AWSAuroraPostgres},
			DBPortOverride: "8888",
			DBHostOverride: "hellonewman.cloud",
			region:         region.MachineRegion("aws-us-west-2"),
			universe:       universe.Debug,
			expectedURL:    "postgresql://seinfeld:cosmo@free-tier4.aws-us-west-2.cockroachlabs.cloud:5432/jerry",
		},
		{
			cfg:            Config{Host: "production-7qd.aws-us-west-2.cockroachlabs.cloud", Port: "5432", DBName: "jerry", User: "seinfeld", Password: secret.NewTestString("kramer"), DBProduct: AWSAuroraPostgres},
			DBPortOverride: "8888",
			DBHostOverride: "hellonewman.cloud",
			region:         region.MachineRegion("aws-us-east-1"),
			universe:       universe.Prod,
			expectedURL:    "postgresql://seinfeld:kramer@production-7qd.aws-us-east-1.cockroachlabs.cloud:5432/jerry",
		},
		{
			cfg:            Config{Host: "production-7qd.aws-us-west-2.cockroachlabs.cloud", Port: "5432", DBName: "frank", User: "costanza", Password: secret.NewTestString("rogers"), DBProduct: AWSAuroraPostgres},
			DBPortOverride: "8888",
			DBHostOverride: "hellonewman.cloud",
			region:         region.MachineRegion("aws-eu-west-1"),
			universe:       universe.Prod,
			expectedURL:    "postgresql://costanza:rogers@production-7qd.aws-eu-west-1.cockroachlabs.cloud:5432/frank",
		},
		{
			cfg:            Config{Host: "status-postgre.c3cgcihdah0v.us-west-2.rds.amazonaws.com", Port: "5432", DBName: "kenny", User: "bania", Password: secret.NewTestString("ovaltine"), DBProduct: AWSRDSPostgres},
			DBPortOverride: "8888",
			DBHostOverride: "hellonewman.cloud",
			region:         region.MachineRegion("aws-us-east-1"),
			universe:       universe.Prod,
			expectedURL:    "postgresql://bania:ovaltine@status-postgre.c3cgcihdah0v.us-west-2.rds.amazonaws.com:5432/kenny",
		},
		{
			cfg:            Config{Host: "staging-mgs.aws-us-west-2.cockroachlabs.cloud", Port: "5432", DBName: "david", User: "puddy", Password: secret.NewTestString("newman"), DBProduct: AWSAuroraPostgres},
			DBPortOverride: "8888",
			DBHostOverride: "hellonewman.cloud",
			region:         region.MachineRegion("aws-us-west-2"),
			universe:       universe.Staging,
			expectedURL:    "postgresql://puddy:newman@staging-mgs.aws-us-west-2.cockroachlabs.cloud:5432/david",
		},
		{
			cfg:            Config{Host: "staging-mgs.aws-us-west-2.cockroachlabs.cloud", Port: "5432", DBName: "david", User: "eightball", Password: secret.NewTestString("mailman"), DBProduct: AWSAuroraPostgres},
			DBPortOverride: "8888",
			DBHostOverride: "hellonewman.cloud",
			region:         region.MachineRegion("aws-us-east-1"),
			universe:       universe.Staging,
			expectedURL:    "postgresql://eightball:mailman@staging-mgs.aws-us-east-1.cockroachlabs.cloud:5432/david",
		},
		{
			cfg:            Config{Host: "staging-mgs.aws-us-west-2.cockroachlabs.cloud", Port: "5432", DBName: "jackie", User: "chiles", Password: secret.NewTestString("sue-allen"), DBProduct: AWSAuroraPostgres},
			DBPortOverride: "8888",
			DBHostOverride: "hellonewman.cloud",
			region:         region.MachineRegion("aws-eu-west-1"),
			universe:       universe.Staging,
			expectedURL:    "postgresql://chiles:sue-allen@staging-mgs.aws-eu-west-1.cockroachlabs.cloud:5432/jackie",
		},
		{
			cfg:            Config{Host: "status-postgre.c3cgcihdah0v.us-west-2.rds.amazonaws.com", Port: "5432", DBName: "jp", User: "peterman", Password: secret.NewTestString("catalog"), DBProduct: AWSRDSPostgres},
			DBPortOverride: "8888",
			DBHostOverride: "hellonewman.cloud",
			region:         region.MachineRegion("aws-eu-west-1"),
			universe:       universe.Staging,
			expectedURL:    "postgresql://peterman:catalog@status-postgre.c3cgcihdah0v.us-west-2.rds.amazonaws.com:5432/jp",
		},
		{
			cfg:            Config{Host: "production-7qd.aws-us-west-2.cockroachlabs.cloud", Port: "5432", DBName: "jerry", User: "seinfeld", Password: secret.NewTestString("puddy"), DBProduct: AWSAuroraPostgres},
			DBPortOverride: "8888",
			DBHostOverride: "hellonewman.cloud",
			region:         region.MachineRegion("aws-us-east-1"),
			universe:       universe.Prod,
			expectedURL:    "postgresql://seinfeld:puddy@production-7qd.aws-us-east-1.cockroachlabs.cloud:5432/jerry",
		},
		{
			cfg:            Config{Host: "jerry.aws-us-west-2.cockroachlabs.cloud", Port: "5432", DBName: "jerry", User: "seinfeld", Password: secret.NewTestString("password"), DBProduct: AWSAuroraPostgres},
			DBPortOverride: "8888",
			DBHostOverride: "hellonewman.cloud",
			region:         region.MachineRegion("aws-us-east-1"),
			universe:       universe.Dev,
			expectedURL:    "postgresql://seinfeld:password@hellonewman.cloud:8888/jerry",
		},
		{
			cfg:         Config{Host: "jerry.aws-us-west-2.cockroachlabs.cloud", Port: "5432", DBName: "jerry", User: "seinfeld", Password: secret.NewTestString("password"), DBProduct: AWSAuroraPostgres},
			region:      region.MachineRegion("themoon"),
			universe:    universe.Dev,
			expectedURL: "postgresql://seinfeld:password@jerry.aws-us-west-2.cockroachlabs.cloud:5432/jerry",
		},
		{
			cfg:               Config{Host: "aurora-global-staging.global-ghfsmzcj8xvp.global.rds.amazonaws.com", Port: "5432", DBName: "idp_17c0ac547abb46d0852c06405da0957c", User: "tenant_17c0ac547abb46d0852c06405da0957c", Password: secret.NewTestString("password"), DBProduct: AWSAuroraPostgres},
			region:            region.MachineRegion("aws-us-west-2"),
			universe:          universe.Dev,
			expectedURL:       "postgresql://tenant_17c0ac547abb46d0852c06405da0957c:password@aurora-global-staging.global-ghfsmzcj8xvp.global.rds.amazonaws.com:5432/idp_17c0ac547abb46d0852c06405da0957c",
			expectedMasterURL: "postgresql://tenant_17c0ac547abb46d0852c06405da0957c:password@aurora-global-staging.global-ghfsmzcj8xvp.global.rds.amazonaws.com:5432/idp_17c0ac547abb46d0852c06405da0957c",
		},
		{
			cfg: Config{Host: "aurora-global-staging.global-ghfsmzcj8xvp.global.rds.amazonaws.com", Port: "5432", DBName: "idp_17c0ac547abb46d0852c06405da0957c", User: "tenant_17c0ac547abb46d0852c06405da0957c", Password: secret.NewTestString("password"), DBProduct: AWSAuroraPostgres,
				RegionalHosts: map[string]string{"aws-us-west-2": "aurora-cluster-staging-us-west-2.cluster-ro-crjyptgsmywz.us-west-2.rds.amazonaws.com", "aws-us-east-1": "aurora-cluster-staging-us-east-1.cluster-ro-cly2gyqwkgwv.us-east-1.rds.amazonaws.com"}},
			region:            region.MachineRegion("aws-us-west-2"),
			universe:          universe.Dev,
			expectedURL:       "postgresql://tenant_17c0ac547abb46d0852c06405da0957c:password@aurora-cluster-staging-us-west-2.cluster-ro-crjyptgsmywz.us-west-2.rds.amazonaws.com:5432/idp_17c0ac547abb46d0852c06405da0957c",
			expectedMasterURL: "postgresql://tenant_17c0ac547abb46d0852c06405da0957c:password@aurora-global-staging.global-ghfsmzcj8xvp.global.rds.amazonaws.com:5432/idp_17c0ac547abb46d0852c06405da0957c",
		},
		{
			cfg: Config{Host: "aurora-global-staging.global-ghfsmzcj8xvp.global.rds.amazonaws.com", Port: "5432", DBName: "idp_17c0ac547abb46d0852c06405da0957c", User: "tenant_17c0ac547abb46d0852c06405da0957c", Password: secret.NewTestString("password"), DBProduct: AWSAuroraPostgres,
				RegionalHosts: map[string]string{"aws-us-west-2": "aurora-cluster-staging-us-west-2.cluster-ro-crjyptgsmywz.us-west-2.rds.amazonaws.com", "aws-us-east-1": "aurora-cluster-staging-us-east-1.cluster-ro-cly2gyqwkgwv.us-east-1.rds.amazonaws.com"}},
			region:            region.MachineRegion("aws-us-east-1"),
			universe:          universe.Dev,
			expectedURL:       "postgresql://tenant_17c0ac547abb46d0852c06405da0957c:password@aurora-cluster-staging-us-east-1.cluster-ro-cly2gyqwkgwv.us-east-1.rds.amazonaws.com:5432/idp_17c0ac547abb46d0852c06405da0957c",
			expectedMasterURL: "postgresql://tenant_17c0ac547abb46d0852c06405da0957c:password@aurora-global-staging.global-ghfsmzcj8xvp.global.rds.amazonaws.com:5432/idp_17c0ac547abb46d0852c06405da0957c",
		},
	}
	for i, c := range cases {
		if c.DBHostOverride != "" {
			assert.NoErr(t, os.Setenv("UC_DB_HOST_OVERRIDE", c.DBHostOverride))
		}
		if c.DBPortOverride != "" {
			assert.NoErr(t, os.Setenv("UC_DB_PORT_OVERRIDE", c.DBPortOverride))
		}
		pgURL, masterURL, err := GetPostgresURLs(ctx, false, &c.cfg, c.universe, c.region)
		assert.NoErr(t, err, assert.Errorf("case %d", i))
		assert.Equal(t, pgURL, c.expectedURL, assert.Errorf("case %d", i))
		if c.expectedMasterURL != "" {
			assert.Equal(t, masterURL, c.expectedMasterURL, assert.Errorf("case %d", i))
		}
		if c.DBHostOverride != "" {
			assert.NoErr(t, os.Unsetenv("UC_DB_HOST_OVERRIDE"))
		}
		if c.DBPortOverride != "" {
			assert.NoErr(t, os.Unsetenv("UC_DB_PORT_OVERRIDE"))
		}
	}
}
