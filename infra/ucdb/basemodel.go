package ucdb

import (
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
)

// BaseModelable is an interface that all BaseModel instances and ancestors support
// Golang generics do not currently support accessing a struct field for a generic
// type within a method, so this interface was created to enable us to have generic
// methods that take any model object that is derived from the BaseModel struct, and
// be able to access any of the fields in BaseModel from within the generic method.
type BaseModelable interface {
	GetCreated() time.Time
	GetDeleted() time.Time
	GetID() uuid.UUID
	GetUpdated() time.Time
}

// BaseModel underlies (almost) all of our models
type BaseModel struct {
	ID uuid.UUID `db:"id" json:"id" yaml:"id"`

	Created time.Time `db:"created" json:"created" yaml:"created"`
	Updated time.Time `db:"updated" json:"updated" yaml:"updated"`

	Deleted time.Time `db:"deleted" json:"deleted" yaml:"deleted"`
}

// GetCreated is part of the BaseModelable interface
func (b BaseModel) GetCreated() time.Time {
	return b.Created
}

// GetDeleted is part of the BaseModelable interface
func (b BaseModel) GetDeleted() time.Time {
	return b.Deleted
}

// GetID is part of the BaseModelable interface
func (b BaseModel) GetID() uuid.UUID {
	return b.ID
}

// GetUpdated is part of the BaseModelable interface
func (b BaseModel) GetUpdated() time.Time {
	return b.Updated
}

// Validate implements Validateable
func (b BaseModel) Validate() error {
	if b.ID.IsNil() {
		return ucerr.Friendlyf(nil, "Can't have nil ID")
	}
	if b.Updated.IsZero() && !b.Alive() {
		return ucerr.Errorf("%v was soft-deleted before it was ever saved", b.ID)
	}
	return nil
}

// Alive returns true if the object is "alive" and false if it's been deleted
func (b BaseModel) Alive() bool {
	return b.Deleted.IsZero()
}

// NewBase initializes a new UCBase
func NewBase() BaseModel {
	// note that we don't propagate NewV4() errors because at that point the world has ended.
	return NewBaseWithID(uuid.Must(uuid.NewV4()))
}

// NewBaseWithID initializes a new BaseModel with a specific ID
func NewBaseWithID(id uuid.UUID) BaseModel {
	return BaseModel{ID: id, Deleted: time.Time{}} // lint: basemodel-safe
}

// UserBaseModel is a user-related underlying model for many of our models eg. in IDP
type UserBaseModel struct {
	BaseModel

	UserID uuid.UUID `db:"user_id" json:"user_id" yaml:"user_id"`
}

// Validate implements Validateable
func (u UserBaseModel) Validate() error {
	if u.UserID.IsNil() {
		return ucerr.Errorf("UserBaseModel %v can't have nil UserID", u.ID)
	}
	return ucerr.Wrap(u.BaseModel.Validate())
}

// NewUserBase initializes a new user base model
func NewUserBase(userID uuid.UUID) UserBaseModel {
	return UserBaseModel{BaseModel: NewBase(), UserID: userID}
}

// VersionBaseModel supports safe concurrent updates with version checks (only save if you have extant version)
type VersionBaseModel struct {
	BaseModel

	// we use _version here to indicate that it's system-managed (as distinct from eg. versioned Access Policies)
	Version int `db:"_version" json:"version" yaml:"version"`
}

//go:generate genvalidate VersionBaseModel

// NewVersionBase initializes a new VersionBaseModel
func NewVersionBase() VersionBaseModel {
	return VersionBaseModel{BaseModel: NewBase()}
}

// NewVersionBaseWithID initializes a new VersionBaseModel with a specific ID
func NewVersionBaseWithID(id uuid.UUID) VersionBaseModel {
	return VersionBaseModel{BaseModel: NewBaseWithID(id)}
}

// SystemAttributeBaseModel is for resource types where we may have system
// resources that clients should not be able to update or delete
type SystemAttributeBaseModel struct {
	BaseModel

	// note: the "description" tag is for OpenAPI spec generation, because we
	// use DB model structs as client models in a few places where it's not
	// worth maintaining separate DB and client models
	IsSystem bool `db:"is_system" json:"is_system" yaml:"is_system" description:"Whether this resource is a system resource. System resources cannot be deleted or modified. This property cannot be changed."`
}

//go:generate genvalidate SystemAttributeBaseModel

// NewSystemAttributeBase initializes a new SystemAttributeBaseModel
func NewSystemAttributeBase() SystemAttributeBaseModel {
	return SystemAttributeBaseModel{BaseModel: NewBase()}
}

// NewSystemAttributeBaseWithID initializes a new SystemAttributeBaseModel with a specific ID
func NewSystemAttributeBaseWithID(id uuid.UUID) SystemAttributeBaseModel {
	return SystemAttributeBaseModel{BaseModel: NewBaseWithID(id)}
}

// MarkAsSystem returns a copy of the given model with IsSystem set to true
func MarkAsSystem(m SystemAttributeBaseModel) SystemAttributeBaseModel {
	m.IsSystem = true
	return m
}
