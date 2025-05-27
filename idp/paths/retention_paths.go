package paths

import (
	"fmt"

	"github.com/gofrs/uuid"
)

// RetentionPath manages building a path for interacting with retention durations
type RetentionPath struct {
	basePath         string
	durationTypePath string
	durationPath     string
}

// NewRetentionPath creates a new retention path
func NewRetentionPath(isLive bool) *RetentionPath {
	var rp RetentionPath
	rp.initIfNecessary(isLive)
	return &rp
}

func (rp *RetentionPath) initIfNecessary(isLive bool) {
	if rp.basePath == "" {
		rp.ForTenant()
	}
	if rp.durationTypePath == "" {
		rp.forDurationType(isLive)
	}
}

// Build returns a path string for the retention path
func (rp *RetentionPath) Build() string {
	rp.initIfNecessary(true)
	return fmt.Sprintf("%s%s%s", rp.basePath, rp.durationTypePath, rp.durationPath)
}

// ForColumn configures the retention path to be for a column
func (rp *RetentionPath) ForColumn(columnID uuid.UUID) *RetentionPath {
	rp.basePath = fmt.Sprintf("%s/%v", BaseConfigColumnsPath, columnID)
	return rp
}

// ForDuration configures the retention path to be for a duration ID
func (rp *RetentionPath) ForDuration(durationID uuid.UUID) *RetentionPath {
	rp.durationPath = fmt.Sprintf("/%v", durationID)
	return rp
}

func (rp *RetentionPath) forDurationType(isLive bool) *RetentionPath {
	if isLive {
		rp.durationTypePath = "/liveretentiondurations"
	} else {
		rp.durationTypePath = "/softdeletedretentiondurations"
	}
	return rp
}

// ForPurpose configures the retention path to be for a purpose
func (rp *RetentionPath) ForPurpose(purposeID uuid.UUID) *RetentionPath {
	rp.basePath = fmt.Sprintf("%s/%v", BaseConfigPurposePath, purposeID)
	return rp
}

// ForTenant configures the retention path to be for a tenant
func (rp *RetentionPath) ForTenant() *RetentionPath {
	rp.basePath = BaseConfigPath
	return rp
}
