package authz

import (
	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucdb"
)

// AuthZ object types & edge types (roles) provisioned for every tenant.
// TODO: merge the string constant with the UUID into a const-ish struct to keep them associated,
// particularly if we add more of these.
// Keep in sync with TSX constants!
// TODO: we should have a better way to sync constants between TS and Go
const (
	ObjectTypeUser     = "_user"
	ObjectTypeGroup    = "_group"
	ObjectTypeLoginApp = "_login_app"
	EdgeTypeCanLogin   = "_can_login"
	CanLoginAttribute  = "_can_login"
)

// UserObjectTypeID is the ID of a built-in object type called "_user"
var UserObjectTypeID = uuid.Must(uuid.FromString("1bf2b775-e521-41d3-8b7e-78e89427e6fe"))

// GroupObjectTypeID is the ID of a built-in object type called "_group"
var GroupObjectTypeID = uuid.Must(uuid.FromString("f5bce640-f866-4464-af1a-9e7474c4a90c"))

// LoginAppObjectTypeID is the ID of a built-in object type called "_login_app"
var LoginAppObjectTypeID = uuid.Must(uuid.FromString("9b90794f-0ed0-48d6-99a5-6fd578a9134d"))

// CanLoginEdgeTypeID is the ID of a built-in edge type called "_can_login"
var CanLoginEdgeTypeID = uuid.Must(uuid.FromString("ea723951-fb93-4a29-b977-d27c01a61f58"))

// DefaultAuthZObjectTypes is an array containing default AuthZ object types
var DefaultAuthZObjectTypes = []ObjectType{
	{BaseModel: ucdb.NewBaseWithID(UserObjectTypeID), TypeName: ObjectTypeUser},
	{BaseModel: ucdb.NewBaseWithID(GroupObjectTypeID), TypeName: ObjectTypeGroup},
	{BaseModel: ucdb.NewBaseWithID(LoginAppObjectTypeID), TypeName: ObjectTypeLoginApp},
}

// DefaultAuthZEdgeTypes is an array containing default AuthZ edge types
var DefaultAuthZEdgeTypes = []EdgeType{
	{BaseModel: ucdb.NewBaseWithID(CanLoginEdgeTypeID), TypeName: EdgeTypeCanLogin, SourceObjectTypeID: UserObjectTypeID, TargetObjectTypeID: LoginAppObjectTypeID,
		Attributes: []Attribute{
			{Name: CanLoginAttribute, Direct: true},
		},
	},
}
