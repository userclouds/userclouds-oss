package region

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
)

// RegionEnvVar is the environment variable that contains the region
const RegionEnvVar = "UC_REGION"

// MachineRegion represents a region for our systems or located
type MachineRegion string

// machineRegions is a list of regions (real or fake) UC runs in for each universe
var machineRegions = map[universe.Universe][]MachineRegion{
	universe.Prod:      {"aws-us-west-2", "aws-us-east-1", "aws-eu-west-1"},
	universe.Staging:   {"aws-us-west-2", "aws-us-east-1", "aws-eu-west-1"},
	universe.Debug:     {"aws-us-west-2", "aws-us-east-1", "aws-eu-west-1"},
	universe.Dev:       {"themoon", "mars"},
	universe.Container: {"themoon", "mars"},
	universe.CI:        {"themoon", "mars"},
	universe.Test:      {"themoon", "mars"},
	universe.OnPrem:    {"customerlocal"},
}

// MachineRegionsForUniverse returns the list of regions for a given universe
func MachineRegionsForUniverse(u universe.Universe) []MachineRegion {
	return machineRegions[u]
}

// Current returns the current region, or empty string
// TODO: error check against known list?
func Current() MachineRegion {
	r := os.Getenv(RegionEnvVar)
	return MachineRegion(r)
}

// FromAWSRegion returns a region from a aws region string. e.g. us-east-1, us-west-2
func FromAWSRegion(awsRegion string) MachineRegion {
	return MachineRegion(fmt.Sprintf("aws-%s", awsRegion))
}

// GetAWSRegion returns the AWS name of the region and blank if region is not in AWS
func GetAWSRegion(r MachineRegion) string {
	if strings.HasPrefix(string(r), "aws-") {
		return strings.TrimPrefix(string(r), "aws-")
	}
	// TODO maybe makes to error
	return ""
}

// IsValid returns true if the region is a valid region for a given universe
func IsValid(region MachineRegion, u universe.Universe) bool {
	return slices.Contains(machineRegions[u], region)
}

// Validate implements Validateable
func (r MachineRegion) Validate() error {
	if r == "" {
		return ucerr.Friendlyf(nil, "empty machine region")
	}
	if IsValid(r, universe.Current()) {
		return nil
	}
	return ucerr.Friendlyf(nil, "invalid machine region: %s for %v", r, universe.Current())
}

// DataRegion represents a region for where user data should be hosted
type DataRegion string

// dataRegions is a list of regions that user data can be hosted in
var dataRegions = map[universe.Universe][]DataRegion{
	universe.Prod:      {"aws-us-west-2", "aws-us-east-1", "aws-eu-west-1"},
	universe.Staging:   {"aws-us-west-2", "aws-us-east-1", "aws-eu-west-1"},
	universe.Debug:     {"aws-us-west-2", "aws-us-east-1", "aws-eu-west-1"},
	universe.Dev:       {""},
	universe.Container: {""},
	universe.CI:        {"aws-us-west-2", "aws-us-east-1", "aws-eu-west-1"},
	universe.Test:      {"aws-us-west-2", "aws-us-east-1", "aws-eu-west-1"},
}

// DataRegionsForUniverse returns the list of regions for a given universe
func DataRegionsForUniverse(u universe.Universe) []DataRegion {
	return dataRegions[u]
}

// Validate implements Validateable
func (r DataRegion) Validate() error {
	for _, reg := range dataRegions[universe.Current()] {
		if string(r) == string(reg) {
			return nil
		}
	}

	// We allow specifying empty data regions if the customer wants to use the default
	if r == "" {
		return nil
	}

	return ucerr.Friendlyf(nil, "invalid data region: %s", r)
}

// DefaultUserDataRegionForUniverse returns the default region for user data per universe
func DefaultUserDataRegionForUniverse(u universe.Universe) DataRegion {
	switch u {
	case universe.Prod, universe.Staging, universe.Debug, universe.CI, universe.Test:
		return "aws-us-east-1"
	case universe.Dev, universe.Container:
		return ""
	default:
		return ""
	}
}
