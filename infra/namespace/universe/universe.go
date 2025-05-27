package universe

import (
	"fmt"
	"os"
	"slices"

	"userclouds.com/infra/ucerr"
)

// Universe represents a universe (or environment) that UC code runs in
type Universe string

// Environment keys for config settings
// We use these instead of command line args because it works better with `go test`
const (
	EnvKeyUniverse = "UC_UNIVERSE"
)

// Supported universes.
const (
	Undefined Universe = "undefined" // undefined universe
	Dev       Universe = "dev"       // local dev laptops
	Test      Universe = "test"      // automated tests on localhost
	CI        Universe = "ci"        // AWS continuous integration env
	Debug     Universe = "debug"     // AWS EB universe to debug off master
	Staging   Universe = "staging"   // cloud hosted staging universe (similar to prod)
	Prod      Universe = "prod"      // user-facing prod deployment
	Container Universe = "container" //  container (dev only for now)
	OnPrem    Universe = "onprem"    // on-premises deployment
)

//go:generate genstringconstenum Universe

// Current checks the current application environment.
func Current() Universe {
	value, isDefined := os.LookupEnv(EnvKeyUniverse)
	var u Universe
	if isDefined {
		u = Universe(value)
	} else {
		u = Undefined
	}
	if err := u.Validate(); err != nil {
		panic(fmt.Sprintf("invalid universe from environment: %v", u))
	}
	return u
}

// IsUndefined returns true if universe is undefined
func (u Universe) IsUndefined() bool {
	return u == Undefined
}

// IsProd returns true if universe is prod
func (u Universe) IsProd() bool {
	return u == Prod
}

// IsProdOrStaging returns true if universe is prod or staging
func (u Universe) IsProdOrStaging() bool {
	return u == Prod || u == Staging
}

// IsDebug returns true if universe is debug
func (u Universe) IsDebug() bool {
	return u == Debug
}

// IsContainer true if universe is container
func (u Universe) IsContainer() bool {
	return u == Container
}

// IsDev returns true if universe is dev
func (u Universe) IsDev() bool {
	return u == Dev
}

// IsCloud returns true if universe is one of the cloud envs (prod, staging debug)
func (u Universe) IsCloud() bool {
	return u.IsProdOrStaging() || u.IsDebug()
}

// IsOnPrem returns true if universe is on-premises
func (u Universe) IsOnPrem() bool {
	return u == OnPrem
}

// IsKubernetes true if universe cloud or on prem, both are running in k8s
func (u Universe) IsKubernetes() bool {
	return u.IsCloud() || u.IsOnPrem()
}

// IsOnPremOrContainer returns true if universe is on-premises or container
func (u Universe) IsOnPremOrContainer() bool {
	return u.IsOnPrem() || u.IsContainer()
}

// IsTestOrCI returns true if universe is CI or tests
func (u Universe) IsTestOrCI() bool {
	return u == CI || u == Test
}

// AllUniverses returns a list of all known universes
// Useful for config testing etc
func AllUniverses() []Universe {
	return allUniverseValues
}

// Validate implements Validateable
func (u Universe) Validate() error {
	if slices.Contains(allUniverseValues, u) {
		return nil
	}
	return ucerr.Errorf("unknown Universe value %v", u)
}
