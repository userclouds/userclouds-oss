package events

import (
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/paths"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uclog"
	logServerClient "userclouds.com/logserver/client"
)

const (
	// AccessorPrefix is prefix for accessor events
	AccessorPrefix string = "acc"
	// MutatorPrefix is prefix for mutator events
	MutatorPrefix string = "mut"
	// SubCategoryConfig is name of error event
	SubCategoryConfig string = "config"
	// SubCategoryAccessDenied is name of error event
	SubCategoryAccessDenied string = "denied"
	//SubCategoryNotFound is name of error event
	SubCategoryNotFound string = "notfound"
	//SubCategoryTransformError is name of error event
	SubCategoryTransformError string = "transformerror"
	// SubCategoryValidationError is name of error event
	SubCategoryValidationError string = "validationerror"
	// APPrefix is prefix for access policy events
	APPrefix string = "ap"
	// TransformerPrefix is prefix for transformer events
	TransformerPrefix string = "gp"
	// SubCategoryConflict is name of error event
	SubCategoryConflict string = "conflict"
)

// GetEventName generates the names of the custom events for tracking individual accessor/mutator
func GetEventName(id uuid.UUID, c uclog.EventCategory, prefix string, subcategory string, v int) string {
	s := fmt.Sprintf("%s.%s", prefix, id.String())
	op := "cus"
	switch c {
	case uclog.EventCategoryCall:
		op = "call"
	case uclog.EventCategoryDuration:
		op = "dur"
	case uclog.EventCategoryInputError:
		op = "err"
	case uclog.EventCategoryResultSuccess:
		op = "succ"
	case uclog.EventCategoryResultFailure:
		op = "fail"
	}
	if subcategory != "" {
		op = fmt.Sprintf("%s.%s", op, subcategory)
	}
	return fmt.Sprintf("%s.%d.%s", s, v, op)
}

// GetEventsForAccessor returns events that should be defined per instance of accessor
func GetEventsForAccessor(id uuid.UUID, v int) []logServerClient.MetricMetadata {
	return []logServerClient.MetricMetadata{
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryCall, Name: "Calls",
			StringID: GetEventName(id, uclog.EventCategoryCall, AccessorPrefix, "", v), ReferenceURL: paths.GetReferenceURLForAccessor(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryDuration, Name: "Duration",
			StringID: GetEventName(id, uclog.EventCategoryDuration, AccessorPrefix, "", v), ReferenceURL: paths.GetReferenceURLForAccessor(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryResultSuccess, Name: "Success",
			StringID: GetEventName(id, uclog.EventCategoryResultSuccess, AccessorPrefix, "", v), ReferenceURL: paths.GetReferenceURLForAccessor(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryInputError, Name: "Accessor Configuration Error",
			StringID: GetEventName(id, uclog.EventCategoryInputError, AccessorPrefix, SubCategoryConfig, v), ReferenceURL: paths.GetReferenceURLForAccessor(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryInputError, Name: "Access Denied Error",
			StringID: GetEventName(id, uclog.EventCategoryInputError, AccessorPrefix, SubCategoryAccessDenied, v), ReferenceURL: paths.GetReferenceURLForAccessor(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryInputError, Name: "Not Found Error",
			StringID: GetEventName(id, uclog.EventCategoryInputError, AccessorPrefix, SubCategoryNotFound, v), ReferenceURL: paths.GetReferenceURLForAccessor(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryInputError, Name: "Transform Error",
			StringID: GetEventName(id, uclog.EventCategoryInputError, AccessorPrefix, SubCategoryTransformError, v), ReferenceURL: paths.GetReferenceURLForAccessor(id, v)},
	}
}

// GetEventsForMutator returns events that should be defined per instance of mutator
func GetEventsForMutator(id uuid.UUID, v int) []logServerClient.MetricMetadata {
	return []logServerClient.MetricMetadata{
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryCall, Name: "Calls",
			StringID: GetEventName(id, uclog.EventCategoryCall, MutatorPrefix, "", v), ReferenceURL: paths.GetReferenceURLForMutator(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryDuration, Name: "Duration",
			StringID: GetEventName(id, uclog.EventCategoryDuration, MutatorPrefix, "", v), ReferenceURL: paths.GetReferenceURLForMutator(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryResultSuccess, Name: "Success",
			StringID: GetEventName(id, uclog.EventCategoryResultSuccess, MutatorPrefix, "", v), ReferenceURL: paths.GetReferenceURLForMutator(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryInputError, Name: "Mutator Configuration Error",
			StringID: GetEventName(id, uclog.EventCategoryInputError, MutatorPrefix, SubCategoryConfig, v), ReferenceURL: paths.GetReferenceURLForMutator(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryInputError, Name: "Access Denied Error",
			StringID: GetEventName(id, uclog.EventCategoryInputError, MutatorPrefix, SubCategoryAccessDenied, v), ReferenceURL: paths.GetReferenceURLForMutator(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryInputError, Name: "Not Found Error",
			StringID: GetEventName(id, uclog.EventCategoryInputError, MutatorPrefix, SubCategoryNotFound, v), ReferenceURL: paths.GetReferenceURLForMutator(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryInputError, Name: "Validation Error",
			StringID: GetEventName(id, uclog.EventCategoryInputError, MutatorPrefix, SubCategoryValidationError, v), ReferenceURL: paths.GetReferenceURLForMutator(id, v)},
	}
}

// GetEventsForTransformer returns events that should be defined per instance of transformer
func GetEventsForTransformer(id uuid.UUID, v int) []logServerClient.MetricMetadata {
	return []logServerClient.MetricMetadata{
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryCall, Name: "Calls",
			StringID: GetEventName(id, uclog.EventCategoryCall, TransformerPrefix, "", v), ReferenceURL: paths.GetReferenceURLForTransformer(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryDuration, Name: "Duration",
			StringID: GetEventName(id, uclog.EventCategoryDuration, TransformerPrefix, "", v), ReferenceURL: paths.GetReferenceURLForTransformer(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryInputError, Name: "Runtime Error",
			StringID: GetEventName(id, uclog.EventCategoryInputError, TransformerPrefix, "", v), ReferenceURL: paths.GetReferenceURLForTransformer(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryInputError, Name: "Token Conflict",
			StringID: GetEventName(id, uclog.EventCategoryInputError, TransformerPrefix, SubCategoryConflict, v), ReferenceURL: paths.GetReferenceURLForTransformer(id, v)},
	}
}

// GetEventsForAccessPolicy returns events that should be defined per instance of access policy
func GetEventsForAccessPolicy(id uuid.UUID, v int) []logServerClient.MetricMetadata {
	return []logServerClient.MetricMetadata{
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryCall, Name: "Calls",
			StringID: GetEventName(id, uclog.EventCategoryCall, APPrefix, "", v), ReferenceURL: paths.GetReferenceURLForAccessPolicy(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryDuration, Name: "Duration",
			StringID: GetEventName(id, uclog.EventCategoryDuration, APPrefix, "", v), ReferenceURL: paths.GetReferenceURLForAccessPolicy(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryInputError, Name: "Runtime Error",
			StringID: GetEventName(id, uclog.EventCategoryInputError, APPrefix, "", v), ReferenceURL: paths.GetReferenceURLForAccessPolicy(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryResultSuccess, Name: "Access Allowed",
			StringID: GetEventName(id, uclog.EventCategoryResultSuccess, APPrefix, "", v), ReferenceURL: paths.GetReferenceURLForAccessPolicy(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryResultFailure, Name: "Access Denied",
			StringID: GetEventName(id, uclog.EventCategoryResultFailure, APPrefix, "", v), ReferenceURL: paths.GetReferenceURLForAccessPolicy(id, v)},
	}
}

// GetEventsForAccessPolicyTemplate returns events that should be defined per instance of access policy template
func GetEventsForAccessPolicyTemplate(id uuid.UUID, v int) []logServerClient.MetricMetadata {
	return []logServerClient.MetricMetadata{
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryCall, Name: "Calls",
			StringID: GetEventName(id, uclog.EventCategoryCall, APPrefix, "", v), ReferenceURL: paths.GetReferenceURLForAccessPolicy(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryDuration, Name: "Duration",
			StringID: GetEventName(id, uclog.EventCategoryDuration, APPrefix, "", v), ReferenceURL: paths.GetReferenceURLForAccessPolicy(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryInputError, Name: "Runtime Error",
			StringID: GetEventName(id, uclog.EventCategoryInputError, APPrefix, "", v), ReferenceURL: paths.GetReferenceURLForAccessPolicy(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryResultSuccess, Name: "Access Allowed",
			StringID: GetEventName(id, uclog.EventCategoryResultSuccess, APPrefix, "", v), ReferenceURL: paths.GetReferenceURLForAccessPolicy(id, v)},
		{BaseModel: ucdb.NewBase(), Service: service.IDP, Category: uclog.EventCategoryResultFailure, Name: "Access Denied",
			StringID: GetEventName(id, uclog.EventCategoryResultFailure, APPrefix, "", v), ReferenceURL: paths.GetReferenceURLForAccessPolicy(id, v)},
	}
}
