package column

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

var firstFieldPartRegexp = regexp.MustCompile("(^[A-Z][A-Z0-9]*$)|(^[A-Z][a-z0-9]*$)")
var secondaryFieldPartRegexp = regexp.MustCompile("(^[A-Z0-9]+$)|(^[A-Z0-9][a-z0-9]*$)")

func isValidFieldName(s string) bool {
	nameParts := strings.Split(s, "_")
	if len(nameParts) == 0 {
		return false
	}

	if !firstFieldPartRegexp.MatchString(nameParts[0]) {
		return false
	}

	for _, namePart := range nameParts[1:] {
		if !secondaryFieldPartRegexp.MatchString(namePart) {
			return false
		}
	}

	return true
}

func getCamelCaseFieldName(s string) string {
	return strings.ReplaceAll(s, "_", "")
}

func getStructFieldName(s string) string {
	return strings.ToLower(s)
}

const idFieldName = "ID"

var deprecatedIDField = CompositeField{
	DataTypeID:          datatype.String.ID,
	Name:                idFieldName,
	CamelCaseName:       getCamelCaseFieldName(idFieldName),
	StructName:          getStructFieldName(idFieldName),
	Required:            true,
	IgnoreForUniqueness: true,
}

var idField = CompositeField{
	DataTypeID:          datatype.String.ID,
	Name:                idFieldName,
	CamelCaseName:       getCamelCaseFieldName(idFieldName),
	StructName:          getStructFieldName(idFieldName),
	Required:            false,
	IgnoreForUniqueness: true,
}

// CompositeField represents the settings for a composite data type field
type CompositeField struct {
	DataTypeID          uuid.UUID `json:"data_type_id"`
	Name                string    `json:"name"`
	CamelCaseName       string    `json:"camel_case_name"`
	StructName          string    `json:"struct_name"`
	Required            bool      `json:"required"`
	IgnoreForUniqueness bool      `json:"ignore_for_uniqueness"`
}

// NewCompositeFieldFromClient creates a new composite data type field from the client counterpart
func NewCompositeFieldFromClient(f userstore.CompositeField) CompositeField {
	camelCaseName := f.CamelCaseName
	if camelCaseName == "" {
		camelCaseName = getCamelCaseFieldName(f.Name)
	}

	structName := f.StructName
	if structName == "" {
		structName = getStructFieldName(f.Name)
	}

	return CompositeField{
		DataTypeID:          f.DataType.ID,
		Name:                f.Name,
		CamelCaseName:       camelCaseName,
		StructName:          structName,
		Required:            f.Required,
		IgnoreForUniqueness: f.IgnoreForUniqueness,
	}
}

func (cf CompositeField) extraValidate() error {
	if !IsNativeDataType(cf.DataTypeID) {
		return ucerr.Friendlyf(nil, "CompositeField has invalid type id '%v'", cf.DataTypeID)
	}

	if !isValidFieldName(cf.Name) {
		return ucerr.Friendlyf(nil, "CompositeField has invalid name '%s'. %s", cf.Name, compositeFieldNameDescription)
	}

	if cf.CamelCaseName != getCamelCaseFieldName(cf.Name) {
		return ucerr.Friendlyf(nil, "CompositeField has invalid camel case name '%s'", cf.CamelCaseName)
	}

	if cf.StructName != getStructFieldName(cf.Name) {
		return ucerr.Friendlyf(nil, "Field has invalid struct name '%s'", cf.StructName)
	}

	return nil
}

func (cf CompositeField) getTransformerField() CompositeField {
	return CompositeField{
		DataTypeID:    cf.DataTypeID,
		Name:          cf.Name,
		CamelCaseName: cf.CamelCaseName,
		StructName:    cf.StructName,
	}
}

// ToClient converts the composite data type field to its client counterpart
func (cf CompositeField) ToClient() userstore.CompositeField {
	if cf == idField {
		return deprecatedIDField.ToClient()
	}

	return userstore.CompositeField{
		DataType:            userstore.ResourceID{ID: cf.DataTypeID},
		Name:                cf.Name,
		CamelCaseName:       cf.CamelCaseName,
		StructName:          cf.StructName,
		Required:            cf.Required,
		IgnoreForUniqueness: cf.IgnoreForUniqueness,
	}
}

//go:generate genvalidate CompositeField

// CompositeAttributes represents the attributes for a composite data type
type CompositeAttributes struct {
	IncludeID bool             `json:"include_id"`
	Fields    []CompositeField `json:"fields"`
}

func (ca CompositeAttributes) getTransformerAttributes() CompositeAttributes {
	var fields []CompositeField
	for _, f := range ca.Fields {
		fields = append(fields, f.getTransformerField())
	}

	return CompositeAttributes{Fields: fields}
}

func (ca CompositeAttributes) getOrderedFields() []CompositeField {
	fields := ca.Fields
	sort.Slice(fields, func(i int, j int) bool { return fields[i].Name < fields[j].Name })
	return fields
}

func (ca CompositeAttributes) equals(o CompositeAttributes) bool {
	if ca.IncludeID != o.IncludeID {
		return false
	}

	otherFields := o.getOrderedFields()
	for i, field := range ca.getOrderedFields() {
		if field != otherFields[i] {
			return false
		}
	}

	return true
}

//go:generate gendbjson CompositeAttributes
//go:generate genvalidate CompositeAttributes

// DataType represents the settings for a data type
type DataType struct {
	ucdb.BaseModel

	Name                string              `db:"name" validate:"notempty" json:"name"`
	Description         string              `db:"description" validate:"notempty" json:"description"`
	ConcreteDataTypeID  uuid.UUID           `db:"concrete_data_type_id" validate:"notnil" json:"concrete_data_type_id"`
	CompositeAttributes CompositeAttributes `db:"composite_attributes" json:"composite_attributes"`
}

func (dt *DataType) extraValidate() error {
	if isCompositeDataTypeID(dt.ID) {
		return ucerr.Friendlyf(nil, "DataType has invalid ID")
	}

	if IsNativeDataType(dt.ID) {
		ndt, err := GetNativeDataType(dt.ID)
		if err != nil {
			return ucerr.Wrap(err)
		}

		if !ndt.Equals(*dt) {
			return ucerr.Friendlyf(nil, "native data type '%v' does not match '%v'", dt, ndt)
		}

		return nil
	}

	if !dt.IsComposite() {
		return ucerr.Friendlyf(nil, "Non-native DataType must be composite")
	}

	if len(dt.CompositeAttributes.Fields) == 0 {
		return ucerr.Friendlyf(nil, "Non-native DataType must have at least one field")
	}

	names := set.NewStringSet()
	camelCaseNames := set.NewStringSet()
	structNames := set.NewStringSet()

	for _, field := range dt.CompositeAttributes.Fields {
		if names.Contains(field.Name) {
			return ucerr.Friendlyf(nil, "Non-native DataType has non-unique Field Name '%s'", field.Name)
		}
		if camelCaseNames.Contains(field.CamelCaseName) {
			return ucerr.Friendlyf(nil, "Non-native DataType has non-unique Field CamelCaseName '%s'", field.CamelCaseName)
		}
		if structNames.Contains(field.StructName) {
			return ucerr.Friendlyf(nil, "Non-native DataType has non-unique Field StructName '%s'", field.StructName)
		}
		names.Insert(field.Name)
		camelCaseNames.Insert(field.CamelCaseName)
		structNames.Insert(field.StructName)

		if field.Name == idField.Name {
			if dt.CompositeAttributes.IncludeID {
				if field != idField && field != deprecatedIDField {
					return ucerr.Friendlyf(nil, "Required DataType field '%s' is invalid", idField.Name)
				}
			}
		}
	}

	if dt.CompositeAttributes.IncludeID && !names.Contains(idField.Name) {
		return ucerr.Friendlyf(nil, "Required DataType field '%s' is missing", idField.Name)
	}

	return nil
}

//go:generate genvalidate DataType

// NewDataTypeFromClient creates a data type from the client representation
func NewDataTypeFromClient(cdt userstore.ColumnDataType) DataType {
	dt := DataType{
		Name:               cdt.Name,
		Description:        cdt.Description,
		ConcreteDataTypeID: dts.getConcreteDataTypeForID(cdt.ID),
		CompositeAttributes: CompositeAttributes{
			IncludeID: cdt.CompositeAttributes.IncludeID,
		},
	}

	if dt.IsComposite() {
		foundIDField := false
		for _, field := range cdt.CompositeAttributes.Fields {
			if field.Name == idField.Name {
				foundIDField = true
				field.Required = false
			}

			dt.CompositeAttributes.Fields = append(dt.CompositeAttributes.Fields, NewCompositeFieldFromClient(field))
		}

		if dt.CompositeAttributes.IncludeID && !foundIDField {
			dt.CompositeAttributes.Fields = append(dt.CompositeAttributes.Fields, idField)
		}
	}

	if cdt.ID.IsNil() {
		dt.BaseModel = ucdb.NewBase()
	} else {
		dt.BaseModel = ucdb.NewBaseWithID(cdt.ID)
	}

	return dt
}

// AreEquivalent checks if the two values are equivalent
func (dt DataType) AreEquivalent(left any, right any) (bool, error) {
	leftComparable, err := dt.GetComparableValue(left)
	if err != nil {
		return false, ucerr.Wrap(err)
	}
	rightComparable, err := dt.GetComparableValue(right)
	if err != nil {
		return false, ucerr.Wrap(err)
	}
	if leftComparable != rightComparable {
		return false, nil
	}
	return true, nil
}

func (dt DataType) canBeUnique() bool {
	switch dt.ConcreteDataTypeID {
	case datatype.Integer.ID, datatype.String.ID, datatype.UUID.ID:
		return true
	}

	return false
}

// Equals returns true if the data types are equal, performing a case-insentive comparison for the Name
func (dt DataType) Equals(o DataType) bool {
	return dt.ID == o.ID &&
		strings.EqualFold(dt.Name, o.Name) &&
		dt.Description == o.Description &&
		dt.ConcreteDataTypeID == o.ConcreteDataTypeID &&
		dt.CompositeAttributes.equals(o.CompositeAttributes)
}

// GetClientDataType returns the appropriate client data type
func (dt DataType) GetClientDataType() string {
	if dt.IsComposite() {
		return dts.getDataTypeForID(datatype.Composite.ID).Name
	}
	return dts.getDataTypeForID(dt.ID).Name
}

// GetColumnFields returns the column fields based on the associated data type
func (dt DataType) GetColumnFields() []userstore.ColumnField {
	var fields []userstore.ColumnField
	for _, f := range dt.CompositeAttributes.Fields {
		fields = append(
			fields,
			userstore.ColumnField{
				Type:                dts.getDataTypeForID(f.DataTypeID).Name,
				Name:                f.Name,
				CamelCaseName:       f.CamelCaseName,
				StructName:          f.StructName,
				Required:            f.Required || f == idField,
				IgnoreForUniqueness: f.IgnoreForUniqueness,
			},
		)
	}
	return fields
}

// GetComparableValue returns a comparable representation of a value
func (dt DataType) GetComparableValue(v any) (any, error) {
	v, err := dt.getComparableValue(v, false)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return v, nil
}

func (dt DataType) getComparableValue(v any, forUniqueness bool) (any, error) {
	if dt.IsComposite() {
		cv, ok := v.(userstore.CompositeValue)
		if !ok {
			return nil, ucerr.Errorf("expected '%T' but got '%v'", cv, v)
		}
		var builder strings.Builder
		for _, field := range dt.CompositeAttributes.getOrderedFields() {
			if forUniqueness && field.IgnoreForUniqueness {
				continue
			}
			fmt.Fprintf(&builder, "('%s','%v')", field.StructName, cv[field.StructName])
		}

		return builder.String(), nil
	}

	return v, nil
}

func (dt DataType) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "name,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"name:%v,id:%v",
				dt.Name,
				dt.ID,
			),
		)
	}
}

func (DataType) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"name":                  pagination.StringKeyType,
		"concrete_data_type_id": pagination.UUIDKeyType,
		"created":               pagination.TimestampKeyType,
		"updated":               pagination.TimestampKeyType,
	}
}

//go:generate genpageable DataType

// GetTransformerConstraints produces appropriate transformer constraints for the associated data type
func (dt DataType) GetTransformerConstraints() userstore.ColumnConstraints {
	var constraints userstore.ColumnConstraints

	for _, f := range dt.CompositeAttributes.Fields {
		constraints.Fields = append(
			constraints.Fields,
			userstore.ColumnField{
				Type:          dts.getDataTypeForID(f.DataTypeID).Name,
				Name:          f.Name,
				CamelCaseName: f.CamelCaseName,
				StructName:    f.StructName,
			},
		)
	}

	return constraints
}

// GetUniqueID will return the unique ID for the value, if one is specified
func (dt DataType) GetUniqueID(v any) (string, error) {
	if dt.IsComposite() {
		cv, ok := v.(userstore.CompositeValue)
		if !ok {
			return "", ucerr.Errorf("'%+v' is not a composite value, it is '%T'", v, v)
		}

		if fv, found := cv[idField.StructName]; found {
			id, ok := fv.(string)
			if !ok {
				return "",
					ucerr.Friendlyf(
						nil,
						"composite value '%+v' '%s' field is not a string, it is: '%T'",
						cv,
						idField.StructName,
						fv,
					)
			}

			return id, nil
		}
	}

	return "", nil
}

// GetTransformerDataType produces a transformer appropriate version of the data type
func (dt DataType) GetTransformerDataType() *DataType {
	dt.CompositeAttributes = dt.CompositeAttributes.getTransformerAttributes()
	return &dt
}

// IsComposite returns true if this is a composite data type
func (dt DataType) IsComposite() bool {
	return isCompositeDataTypeID(dt.ConcreteDataTypeID)
}

// IsNative returns true if this is a native data type
func (dt DataType) IsNative() bool {
	return IsNativeDataType(dt.ID)
}

func (dt DataType) isSearchIndexable() bool {
	return dt.ConcreteDataTypeID == datatype.String.ID
}

func (dt DataType) setUniqueID(v any, id string) (any, error) {
	if dt.IsComposite() {
		cv, ok := v.(userstore.CompositeValue)
		if !ok {
			return v, ucerr.Errorf("'%v' is not a composite value", v)
		}
		cv[idField.StructName] = id
		return cv, nil
	}

	return v, nil
}

// ToClient converts a data type to the client representation
func (dt DataType) ToClient() userstore.ColumnDataType {
	cdt := userstore.ColumnDataType{
		ID:                   dt.ID,
		Name:                 dt.Name,
		Description:          dt.Description,
		IsCompositeFieldType: IsNativeDataType(dt.ID),
		IsNative:             dt.IsNative(),
		CompositeAttributes: userstore.CompositeAttributes{
			IncludeID: dt.CompositeAttributes.IncludeID,
		},
	}

	for _, field := range dt.CompositeAttributes.Fields {
		cdt.CompositeAttributes.Fields = append(cdt.CompositeAttributes.Fields, field.ToClient())
	}

	return cdt
}

var compositeFieldNameDescription string

func init() {
	cf := &userstore.CompositeField{}
	nameField, found := reflect.TypeOf(cf).Elem().FieldByName("Name")
	if !found {
		panic(`userstore.CompositeField "Name" field not found`)
	}
	compositeFieldNameDescription = nameField.Tag.Get("description")
	if compositeFieldNameDescription == "" {
		panic(`userstore.CompositeField "Name" field "description" tag not found`)
	}
}
