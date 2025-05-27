package ucauthz

// The equivalent package in every other service is called "authz", but to avoid authz.authz sequence this one is named ucauthz
import (
	"github.com/gofrs/uuid"
)

// These types are used in Console app to define roles of console users
const (
	// EdgeTypeAdmin a tenant admin
	EdgeTypeAdmin = "_admin"
	// EdgeTypeMember a tenant member
	EdgeTypeMember = "_member"
	AdminRole      = EdgeTypeAdmin
	MemberRole     = EdgeTypeMember
)

// AdminEdgeTypeID is the ID of a built-in edge type called "_adminOf"
var AdminEdgeTypeID = uuid.Must(uuid.FromString("237aba41-90bd-47de-a6cd-bf75c3c76b74"))

// MemberEdgeTypeID is the ID of a built-in edge type called "_memberOf"
var MemberEdgeTypeID = uuid.Must(uuid.FromString("e5eb5062-8f08-4cd8-a43e-b4a81ab55f50"))

// AdminRoleTypeID is the ID of a built-in edge type called "_admin_deprecated"
var AdminRoleTypeID = uuid.Must(uuid.FromString("60b69666-4a8a-4eb3-94dd-621298fb365d"))

// MemberRoleTypeID is the ID of a built-in edge type called "_member_deprecated"
var MemberRoleTypeID = uuid.Must(uuid.FromString("1eec16ec-6130-4f9e-a51f-21bc19b20d8f"))
