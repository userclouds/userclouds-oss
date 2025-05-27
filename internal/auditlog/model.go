package auditlog

import (
	"fmt"
	"regexp"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// Storage defines the interface for storing the audit log
type Storage struct {
	db *ucdb.DB
}

// NewStorage returns a new DB-backed auditlog.Storage object to access auditlog
func NewStorage(db *ucdb.DB) *Storage {
	return &Storage{db}
}

// Payload is a generic key-value map
type Payload map[string]any

//go:generate gendbjson Payload

// EventType identifies the type of audit log entry
type EventType string

// Different audit event types levels
const (
	LoginAttempt         EventType = "LoginAttempt"
	LoginSuccess         EventType = "LoginSuccess"
	LogoutSuccess        EventType = "LogoutSuccess"
	LoginFailure         EventType = "LoginFailure"
	InviteSent           EventType = "InviteSent"
	InviteRedemeed       EventType = "InviteRedemeed"
	PasswordReset        EventType = "PasswordReset"
	AccountCreated       EventType = "AccountCreated"
	AccountImpersonation EventType = "AccountImpersonation"
	TenantCreated        EventType = "TenantCreated"

	// AuthZ Events
	CreateObjectType   EventType = "ObjectTypeCreated"
	DeleteObjectType   EventType = "ObjectTypeDeleted"
	CreateObject       EventType = "ObjectCreated"
	UpdateObject       EventType = "ObjectUpdated"
	DeleteObject       EventType = "ObjectDeleted"
	CreateEdgeType     EventType = "EdgeTypeCreated"
	UpdateEdgeType     EventType = "UpdateEdgeType"
	DeleteEdgeType     EventType = "EdgeTypeDeleted"
	CreateEdge         EventType = "EdgeCreated"
	DeleteEdge         EventType = "EdgeDeleted"
	CreateOrganization EventType = "OrganizationCreated"
	UpdateOrganization EventType = "OrganizationUpdated"
	DeleteOrganization EventType = "OrganizationDeleted"

	// Custom Events
	AccessPolicyCustom EventType = "AccessPolicyCustom"
	TransformerCustom  EventType = "TransformerCustom"
)

const (
	// BasePathSegment is base path segment for the URL for the handlers
	BasePathSegment string = "/auditlog"
	// EntryPathSegment is path segment for audit log entries
	EntryPathSegment string = "/entries"
)

// Entry describes a single entry in tenants audit log
type Entry struct {
	ucdb.BaseModel
	// Type is the type name of audit log entry.
	Type EventType `db:"type" json:"type" validate:"length:1,64"`
	// Actor is the entity that performed the action/operation for which the entry is being made
	Actor string `db:"actor_id" json:"actor_id" validate:"notempty"`
	// Payload
	Payload Payload `db:"payload" json:"payload"`
}

func (u Entry) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "created,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"created:%v,id:%v",
				u.Created.UnixMicro(),
				u.ID,
			),
		)
	}
}

func (Entry) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"created":                   pagination.TimestampKeyType,
		"type":                      pagination.StringKeyType,
		"actor_id":                  pagination.UUIDKeyType,
		"payload->'SelectorValues'": pagination.ArrayKeyType,
		"payload->>'ID'":            pagination.UUIDKeyType,
		"payload->>'Version'":       pagination.IntKeyType,
	}
}

//go:generate genpageable Entry

var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

func (u Entry) extraValidate() error {
	if !validIdentifier.MatchString(string(u.Type)) {
		return ucerr.Errorf("type string '%s' is not a valid identifier", u.Type)
	}

	return nil
}

//go:generate genvalidate Entry

//go:generate genorm Entry auditlog tenantdb

// NewEntry creates a new audit log entry
func NewEntry(actor string, eventType EventType, payload Payload) *Entry {
	return &Entry{
		BaseModel: ucdb.NewBase(),
		Actor:     actor,
		Type:      eventType,
		Payload:   payload,
	}
}

// NewEntryArray creates a new audit log entry array
func NewEntryArray(actor string, eventType EventType, payload Payload) []Entry {
	return []Entry{*NewEntry(actor, eventType, payload)}
}
