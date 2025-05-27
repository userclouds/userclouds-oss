package userevent

import (
	"regexp"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// Payload is a generic key-value map
type Payload map[string]any

//go:generate gendbjson Payload

// UserEvent stores a client-defined event generated from a user's interaction.
type UserEvent struct {
	ucdb.BaseModel

	// Type is the client-defined event type name.
	Type string `db:"type" json:"type" validate:"length:1,64"`

	// User Alias is an app/client-defined string to uniquely identify a user.
	// It could also be a UserClouds-generated UUID but it doesn't need to be, just so long
	// as it is 1:1 with the client's notion of a user.
	// TODO: support UserClouds canonical user ID as UUID, in which case
	// this should be the string form of the UUID
	UserAlias string `db:"user_alias" json:"user_alias" validate:"notempty"`

	// Client-defined key-value map
	// TODO: should we enforce flatness?
	// TODO: should we enforce only specific types (int, string, etc)?
	Payload Payload `db:"payload" json:"payload"`

	// TODO: support client timestamp (since the BaseModel only includes server-side creation)
}

//go:generate genvalidate UserEvent

var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

func (ue UserEvent) extraValidate() error {
	if !validIdentifier.MatchString(ue.Type) {
		return ucerr.Errorf("type string '%s' is not a valid identifier", ue.Type)
	}
	return nil
}

func (UserEvent) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"user_alias": pagination.StringKeyType,
	}
}

//go:generate genpageable UserEvent
