package column

import (
	"github.com/gofrs/uuid"

	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

func isCompositeDataTypeID(id uuid.UUID) bool {
	return id == datatype.Composite.ID
}

var nativeDataTypesByID = map[uuid.UUID]DataType{}
var nativeDataTypes = []DataType{
	{
		BaseModel:          ucdb.NewBaseWithID(datatype.Birthdate.ID),
		Name:               datatype.Birthdate.Name,
		Description:        "a birthdate",
		ConcreteDataTypeID: datatype.Date.ID,
	},
	{
		BaseModel:          ucdb.NewBaseWithID(datatype.Boolean.ID),
		Name:               datatype.Boolean.Name,
		Description:        "a boolean",
		ConcreteDataTypeID: datatype.Boolean.ID,
	},
	{
		BaseModel:          ucdb.NewBaseWithID(datatype.Date.ID),
		Name:               datatype.Date.Name,
		Description:        "a date",
		ConcreteDataTypeID: datatype.Date.ID,
	},
	{
		BaseModel:          ucdb.NewBaseWithID(datatype.E164PhoneNumber.ID),
		Name:               datatype.E164PhoneNumber.Name,
		Description:        "an E.164 phone number",
		ConcreteDataTypeID: datatype.String.ID,
	},
	{
		BaseModel:          ucdb.NewBaseWithID(datatype.Email.ID),
		Name:               datatype.Email.Name,
		Description:        "an email address",
		ConcreteDataTypeID: datatype.String.ID,
	},
	{
		BaseModel:          ucdb.NewBaseWithID(datatype.Integer.ID),
		Name:               datatype.Integer.Name,
		Description:        "an integer",
		ConcreteDataTypeID: datatype.Integer.ID,
	},
	{
		BaseModel:          ucdb.NewBaseWithID(datatype.PhoneNumber.ID),
		Name:               datatype.PhoneNumber.Name,
		Description:        "a phone number",
		ConcreteDataTypeID: datatype.String.ID,
	},
	{
		BaseModel:          ucdb.NewBaseWithID(datatype.SSN.ID),
		Name:               datatype.SSN.Name,
		Description:        "a social security number",
		ConcreteDataTypeID: datatype.String.ID,
	},
	{
		BaseModel:          ucdb.NewBaseWithID(datatype.String.ID),
		Name:               datatype.String.Name,
		Description:        "a string",
		ConcreteDataTypeID: datatype.String.ID,
	},
	{
		BaseModel:          ucdb.NewBaseWithID(datatype.Timestamp.ID),
		Name:               datatype.Timestamp.Name,
		Description:        "a timestamp",
		ConcreteDataTypeID: datatype.Timestamp.ID,
	},
	{
		BaseModel:          ucdb.NewBaseWithID(datatype.UUID.ID),
		Name:               datatype.UUID.Name,
		Description:        "a UUID",
		ConcreteDataTypeID: datatype.UUID.ID,
	},
}

// GetNativeDataTypes returns the default data types
func GetNativeDataTypes() []DataType {
	var dataTypes []DataType
	dataTypes = append(dataTypes, nativeDataTypes...)
	return dataTypes
}

// GetNativeDataType returns the data type associated with id if it is a native type
func GetNativeDataType(id uuid.UUID) (*DataType, error) {
	dt, found := nativeDataTypesByID[id]
	if !found {
		return nil, ucerr.Errorf("data type '%v' is not a native data type", id)
	}

	return &dt, nil
}

// IsNativeDataType returns true if id refers to a native data type
func IsNativeDataType(id uuid.UUID) bool {
	if _, found := nativeDataTypesByID[id]; found {
		return true
	}

	return false
}
