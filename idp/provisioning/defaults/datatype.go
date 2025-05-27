package defaults

import (
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucdb"
)

var defaultDataTypesByID = map[uuid.UUID]column.DataType{}
var defaultDataTypes = []column.DataType{
	{
		BaseModel:          ucdb.NewBaseWithID(datatype.CanonicalAddress.ID),
		Name:               datatype.CanonicalAddress.Name,
		Description:        "a canonical address",
		ConcreteDataTypeID: datatype.Composite.ID,
		CompositeAttributes: column.CompositeAttributes{
			Fields: []column.CompositeField{
				{
					CamelCaseName: "AdministrativeArea",
					DataTypeID:    datatype.String.ID,
					Name:          "Administrative_Area",
					StructName:    "administrative_area",
				},
				{
					CamelCaseName: "Country",
					DataTypeID:    datatype.String.ID,
					Name:          "Country",
					StructName:    "country",
				},
				{
					CamelCaseName: "DependentLocality",
					DataTypeID:    datatype.String.ID,
					Name:          "Dependent_Locality",
					StructName:    "dependent_locality",
				},
				{
					CamelCaseName:       "ID",
					DataTypeID:          datatype.String.ID,
					IgnoreForUniqueness: true,
					Name:                "ID",
					StructName:          "id",
				},
				{
					CamelCaseName: "Locality",
					DataTypeID:    datatype.String.ID,
					Name:          "Locality",
					StructName:    "locality",
				},
				{
					CamelCaseName: "Name",
					DataTypeID:    datatype.String.ID,
					Name:          "Name",
					StructName:    "name",
				},
				{
					CamelCaseName: "Organization",
					DataTypeID:    datatype.String.ID,
					Name:          "Organization",
					StructName:    "organization",
				},
				{
					CamelCaseName: "PostCode",
					DataTypeID:    datatype.String.ID,
					Name:          "Post_Code",
					StructName:    "post_code",
				},
				{
					CamelCaseName: "SortingCode",
					DataTypeID:    datatype.String.ID,
					Name:          "Sorting_Code",
					StructName:    "sorting_code",
				},
				{
					CamelCaseName: "StreetAddressLine1",
					DataTypeID:    datatype.String.ID,
					Name:          "Street_Address_Line_1",
					StructName:    "street_address_line_1",
				},
				{
					CamelCaseName: "StreetAddressLine2",
					DataTypeID:    datatype.String.ID,
					Name:          "Street_Address_Line_2",
					StructName:    "street_address_line_2",
				},
			},
			IncludeID: true,
		},
	},
}

// GetDefaultDataTypes returns the default data types
func GetDefaultDataTypes() []column.DataType {
	var dataTypes []column.DataType
	dataTypes = append(dataTypes, defaultDataTypes...)
	return dataTypes
}

// IsDefaultDataType returns true if id refers to a default data type
func IsDefaultDataType(id uuid.UUID) bool {
	if _, found := defaultDataTypesByID[id]; found {
		return true
	}

	return false
}

func init() {
	for _, ddt := range defaultDataTypes {
		if _, found := defaultDataTypesByID[ddt.ID]; found {
			panic(fmt.Sprintf("default data type %s has conflicting id %v", ddt.Name, ddt.ID))
		}
		defaultDataTypesByID[ddt.ID] = ddt
	}

	for _, ndt := range column.GetNativeDataTypes() {
		if _, found := defaultDataTypesByID[ndt.ID]; found {
			panic(fmt.Sprintf("native data type %s has conflicting id %v", ndt.Name, ndt.ID))
		}
		defaultDataTypesByID[ndt.ID] = ndt
		defaultDataTypes = append(defaultDataTypes, ndt)
	}
}
