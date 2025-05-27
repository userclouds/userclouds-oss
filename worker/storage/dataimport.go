package storage

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
)

const (
	importPathRegex = `(?P<universe>[a-z]+)/tenants/(?P<tenantID>[[:xdigit:]-]{36})/(?P<importType>[a-z]+)/v(?P<version>[\d]+)/(?P<JobID>[[:xdigit:]-]{36})`
	importVersion   = 1

	// ExecuteMutatorsImportType is the import type for ExecuteMutator imports
	ExecuteMutatorsImportType = "executemutator"
)

// DataImportInfo contains information about a data import parsed from an s3 key
type DataImportInfo struct {
	TenantID   uuid.UUID
	ImportType string
	Version    int
	JobID      uuid.UUID
	Universe   universe.Universe
}

func (a *DataImportInfo) extraValidate() error {
	// very strict for now, we can modify this as needed
	if a.Version != importVersion {
		return ucerr.Errorf("Version must be 1. got %d", a.Version)
	}
	if a.Universe != universe.Current() {
		return ucerr.Errorf("Universe %v doesn't match current universe: %v", a.Universe, universe.Current())
	}
	if a.ImportType != ExecuteMutatorsImportType {
		return ucerr.Errorf("ImportType must be 'executemutator'. got %s", a.ImportType)
	}
	return nil
}

//go:generate genvalidate DataImportInfo

// GenerateDataImportPath generates an s3 key for a data import job
func GenerateDataImportPath(tenantID uuid.UUID, importType string, jobID uuid.UUID) string {
	return fmt.Sprintf("%s/tenants/%s/%s/v%d/%s", universe.Current(), tenantID, importType, importVersion, jobID)
}

// ParseDataImportPath parses an s3 key into a DataImportInfo struct
func ParseDataImportPath(path string) (*DataImportInfo, error) {
	pathRegex := regexp.MustCompile(importPathRegex)
	match := pathRegex.FindStringSubmatch(path)
	if match == nil {
		return nil, ucerr.Errorf("path '%s' does not match expected format", path)
	}
	version, err := strconv.Atoi(match[pathRegex.SubexpIndex("version")])
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	di := DataImportInfo{
		TenantID:   uuid.FromStringOrNil(match[pathRegex.SubexpIndex("tenantID")]),
		ImportType: match[pathRegex.SubexpIndex("importType")],
		Version:    version,
		JobID:      uuid.FromStringOrNil(match[pathRegex.SubexpIndex("JobID")]),
		Universe:   universe.Universe(match[pathRegex.SubexpIndex("universe")]),
	}
	if err := di.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &di, nil
}
