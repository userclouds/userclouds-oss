package logeventmetadata

import (
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	logServerClient "userclouds.com/logserver/client"
)

// MetricAttributes contains set of attribues
type MetricAttributes struct {
	// Ignore indicate that the client shouldn't send this event to the server
	Ignore bool `db:"ignore" json:"ignore"`
	// System indicate that this is a system (UC) event and not per object event
	System bool `db:"system" json:"system"`
	// AnyService indicate that this is event can be triggered by any service
	AnyService bool `db:"anyservice" json:"anyservice"`
}

// MetricMetadata describes metadata for an event
type MetricMetadata struct {
	ucdb.BaseModel
	// Service in which this event occurs
	Service service.Service `db:"service" json:"service"`
	// Category is the category of the event
	Category uclog.EventCategory `db:"category" json:"category"`
	// StringID is the name the client will pass if it doesn't have the code
	StringID string `db:"string_id" json:"string_id" validate:"notempty"`
	// Unique numeric code for the event, stays same across namestring changes
	Code uclog.EventCode `db:"code" json:"code"`
	// Human readable name for the event
	Name string `db:"name" json:"name"`
	// URL to object this events relates to if available
	ReferenceURL string `db:"url" json:"url"`
	// Description for what the event represents
	Description string `db:"description" json:"description"`
	// Attributes for the event
	Attributes MetricAttributes `db:"attributes" json:"attributes"`
}

func (u MetricMetadata) extraValidate() error {
	if u.Service != "" && !service.IsValid(u.Service) {
		return ucerr.Errorf("service string '%s' is not a valid service", u.Service)
	}
	return nil
}

// NewMetricMetadataFromLogClientMetricMetadata creates MetricMetadata from LogClientMetricMetadata
func NewMetricMetadataFromLogClientMetricMetadata(l logServerClient.MetricMetadata) MetricMetadata {
	return MetricMetadata{BaseModel: l.BaseModel, Service: l.Service,
		Category: l.Category, StringID: l.StringID, Code: l.Code, Name: l.Name,
		ReferenceURL: l.ReferenceURL, Description: l.Description,
		Attributes: MetricAttributes{Ignore: l.Attributes.Ignore, System: l.Attributes.System,
			AnyService: l.Attributes.AnyService}}
}

// EqualLogClientMetricMetadata validates that the contents of MetricMetadata are same as contents of logServerClient.MetricMetadata
func (u MetricMetadata) EqualLogClientMetricMetadata(l logServerClient.MetricMetadata, ignoreBase bool, allowCodeZero bool) bool {
	return (ignoreBase || u.BaseModel == l.BaseModel) &&
		(u.Attributes.AnyService == l.Attributes.AnyService && u.Attributes.Ignore == l.Attributes.Ignore &&
			u.Attributes.System == l.Attributes.System) &&
		(u.Code == l.Code || l.Code == 0 && allowCodeZero) &&
		(u.Service == l.Service && u.Category == l.Category && u.StringID == l.StringID &&
			u.Name == l.Name && u.ReferenceURL == l.ReferenceURL && u.Description == l.Description)
}

// Equal validates that the contents of MetricMetadata are same as contents of another  MetricMetadata
func (u MetricMetadata) Equal(l MetricMetadata, ignoreBase bool) bool {
	return (ignoreBase || u.BaseModel == l.BaseModel) &&
		(u.Attributes.AnyService == l.Attributes.AnyService && u.Attributes.Ignore == l.Attributes.Ignore &&
			u.Attributes.System == l.Attributes.System) &&
		(u.Service == l.Service && u.Category == l.Category && u.StringID == l.StringID && u.Code == l.Code &&
			u.Name == l.Name && u.ReferenceURL == l.ReferenceURL && u.Description == l.Description)
}

//go:generate genvalidate MetricAttributes

//go:generate genpageable MetricMetadata

//go:generate genvalidate MetricMetadata

//go:generate gendbjson MetricAttributes

//go:generate genorm MetricMetadata event_metadata logdb,companyconfig
