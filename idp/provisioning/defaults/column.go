package defaults

import (
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucdb"
)

var defaultColumnsByID = map[uuid.UUID]storage.Column{}
var defaultColumns = []storage.Column{
	{
		BaseModel:            ucdb.NewBaseWithID(column.IDColumnID),
		Table:                "users",
		Name:                 "id",
		DataTypeID:           datatype.UUID.ID,
		IsArray:              false,
		IndexType:            storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeUnique),
		Attributes:           storage.ColumnAttributes{System: true, Immutable: true},
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	},
	{
		BaseModel:            ucdb.NewBaseWithID(column.CreatedColumnID),
		Table:                "users",
		Name:                 "created",
		DataTypeID:           datatype.Timestamp.ID,
		IsArray:              false,
		IndexType:            storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeNone),
		Attributes:           storage.ColumnAttributes{System: true, Immutable: true},
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	},
	{
		BaseModel:            ucdb.NewBaseWithID(column.UpdatedColumnID),
		Table:                "users",
		Name:                 "updated",
		DataTypeID:           datatype.Timestamp.ID,
		IsArray:              false,
		IndexType:            storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeNone),
		Attributes:           storage.ColumnAttributes{System: true, Immutable: true},
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	},
	{
		BaseModel:            ucdb.NewBaseWithID(column.OrganizationColumnID),
		Table:                "users",
		Name:                 "organization_id",
		DataTypeID:           datatype.UUID.ID,
		IsArray:              false,
		IndexType:            storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeNone),
		Attributes:           storage.ColumnAttributes{System: true, Immutable: true},
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	},
	{
		BaseModel:            ucdb.NewBaseWithID(column.VersionColumnID),
		Table:                "users",
		Name:                 "version",
		DataTypeID:           datatype.Integer.ID,
		IsArray:              false,
		IndexType:            storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeNone),
		Attributes:           storage.ColumnAttributes{System: true, Immutable: true, SystemName: "_version"},
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	},
	{
		BaseModel:            ucdb.NewBaseWithID(column.NameColumnID),
		Table:                "users",
		Name:                 "name",
		DataTypeID:           datatype.String.ID,
		IsArray:              false,
		IndexType:            storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeIndexed),
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	},
	{
		BaseModel:            ucdb.NewBaseWithID(column.NicknameColumnID),
		Table:                "users",
		Name:                 "nickname",
		DataTypeID:           datatype.String.ID,
		IsArray:              false,
		IndexType:            storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeNone),
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	},
	{
		BaseModel:            ucdb.NewBaseWithID(column.PictureColumnID),
		Table:                "users",
		Name:                 "picture",
		DataTypeID:           datatype.String.ID,
		IsArray:              false,
		IndexType:            storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeNone),
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	},
	{
		BaseModel:            ucdb.NewBaseWithID(column.EmailColumnID),
		Table:                "users",
		Name:                 "email",
		DataTypeID:           datatype.String.ID,
		IsArray:              false,
		IndexType:            storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeIndexed),
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	},
	{
		BaseModel:            ucdb.NewBaseWithID(column.EmailVerifiedColumnID),
		Table:                "users",
		Name:                 "email_verified",
		DataTypeID:           datatype.Boolean.ID,
		IsArray:              false,
		IndexType:            storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeNone),
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	},
	{
		BaseModel:            ucdb.NewBaseWithID(column.ExternalAliasColumnID),
		Table:                "users",
		Name:                 "external_alias",
		DataTypeID:           datatype.String.ID,
		IsArray:              false,
		IndexType:            storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeUnique),
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	},
}

// GetDefaultColumns returns the default columns
func GetDefaultColumns() []storage.Column {
	var columns []storage.Column
	columns = append(columns, defaultColumns...)
	return columns
}

// IsDefaultColumn returns true if id refers to a default column
func IsDefaultColumn(id uuid.UUID) bool {
	if _, found := defaultColumnsByID[id]; found {
		return true
	}

	return false
}

func init() {
	for _, dc := range defaultColumns {
		if _, found := defaultColumnsByID[dc.ID]; found {
			panic(fmt.Sprintf("column %s has conflicting id %v", dc.Name, dc.ID))
		}
		defaultColumnsByID[dc.ID] = dc
	}
}
