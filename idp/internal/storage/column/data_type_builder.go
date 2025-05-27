package column

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// CanBeUnique returns true if the data type can be used for a unique column
func CanBeUnique(dataTypeID uuid.UUID) bool {
	return dts.getDataTypeForID(dataTypeID).canBeUnique()
}

// GetDisplayableUniqueDataTypes returns a displayable list of data types that can be unique
func GetDisplayableUniqueDataTypes() string {
	return dts.displayableUniqueDataTypes
}

// IsSearchIndexable returns true if columns of this data type can be indexed in search
func IsSearchIndexable(dataTypeID uuid.UUID) bool {
	return dts.getDataTypeForID(dataTypeID).isSearchIndexable()
}

var dts *dataTypeSettings

type dataTypeInfo struct {
	DataType
	boolValidator      boolValidatorMaker
	compositeValidator compositeValidatorMaker
	intValidator       intValidatorMaker
	stringValidator    stringValidatorMaker
	timestampValidator timestampValidatorMaker
	uuidValidator      uuidValidatorMaker
}

func newDataTypeInfo() *dataTypeInfo {
	return &dataTypeInfo{
		boolValidator:      invalidBool,
		compositeValidator: invalidComposite,
		intValidator:       invalidInt,
		stringValidator:    invalidString,
		timestampValidator: invalidTimestamp,
		uuidValidator:      invalidUUID,
	}
}

// Validate implements the Validateable interface
func (dti *dataTypeInfo) Validate() error {
	if dts.infoByID[dti.ID] == nil || dts.infoByID[dti.ConcreteDataTypeID] == nil {
		return ucerr.Errorf("dataTypeInfo is invalid: '%v'", dti)
	}

	return nil
}

type dataTypeSettings struct {
	infoByID                   map[uuid.UUID]*dataTypeInfo
	displayableUniqueDataTypes string
}

func (dts dataTypeSettings) getDataType(dt DataType) *dataTypeInfo {
	if dt.IsComposite() {
		return dts.getDataTypeForID(dt.ConcreteDataTypeID)
	}

	return dts.getDataTypeForID(dt.ID)
}

func (dts dataTypeSettings) getDataTypeForID(id uuid.UUID) *dataTypeInfo {
	if dti, found := dts.infoByID[id]; found {
		return dti
	}

	return newDataTypeInfo()
}

func (dts dataTypeSettings) getConcreteDataTypeForID(id uuid.UUID) uuid.UUID {
	if dti, found := dts.infoByID[id]; found {
		return dti.ConcreteDataTypeID
	}

	return datatype.Composite.ID
}

type dataTypeBuilder struct {
	dataTypeSettings
	curInfo *dataTypeInfo
}

func newDataTypeBuilder() dataTypeBuilder {
	return dataTypeBuilder{
		dataTypeSettings: dataTypeSettings{
			infoByID: map[uuid.UUID]*dataTypeInfo{},
		},
	}
}

func (dtb *dataTypeBuilder) build() *dataTypeSettings {
	dts := dtb.dataTypeSettings

	uniqueDataTypes := []string{}
	for _, dti := range dts.infoByID {
		if dti.canBeUnique() {
			uniqueDataTypes = append(uniqueDataTypes, dti.Name)
		}
	}
	dts.displayableUniqueDataTypes = strings.Join(uniqueDataTypes, ",")

	return &dts
}

func (dtb *dataTypeBuilder) registerDataType(id uuid.UUID) *dataTypeBuilder {
	if _, found := dtb.infoByID[id]; found {
		panic(fmt.Sprintf("data type id '%v' is already registered", id))
	}

	dtb.curInfo = newDataTypeInfo()
	dtb.infoByID[id] = dtb.curInfo

	if isCompositeDataTypeID(id) {
		dtb.curInfo.DataType = DataType{
			BaseModel:          ucdb.NewBaseWithID(id),
			Name:               datatype.Composite.Name,
			ConcreteDataTypeID: id,
		}
	} else {
		dt, err := GetNativeDataType(id)
		if err != nil {
			panic(fmt.Sprintf("%v", err))
		}
		dtb.curInfo.DataType = *dt
	}

	return dtb
}

func (dtb *dataTypeBuilder) withBoolValidator(v boolValidatorMaker) *dataTypeBuilder {
	dtb.curInfo.boolValidator = v
	return dtb
}

func (dtb *dataTypeBuilder) withCompositeValidator(v compositeValidatorMaker) *dataTypeBuilder {
	dtb.curInfo.compositeValidator = v
	return dtb
}

func (dtb *dataTypeBuilder) withIntValidator(v intValidatorMaker) *dataTypeBuilder {
	dtb.curInfo.intValidator = v
	return dtb
}

func (dtb *dataTypeBuilder) withStringValidator(v stringValidatorMaker) *dataTypeBuilder {
	dtb.curInfo.stringValidator = v
	return dtb
}

func (dtb *dataTypeBuilder) withTimestampValidator(v timestampValidatorMaker) *dataTypeBuilder {
	dtb.curInfo.timestampValidator = v
	return dtb
}

func (dtb *dataTypeBuilder) withUUIDValidator(v uuidValidatorMaker) *dataTypeBuilder {
	dtb.curInfo.uuidValidator = v
	return dtb
}

func init() {
	// initialize native data types

	for _, ndt := range nativeDataTypes {
		if _, found := nativeDataTypesByID[ndt.ID]; found {
			panic(fmt.Sprintf("native data type %s has conflicting id %v", ndt.Name, ndt.ID))
		}
		nativeDataTypesByID[ndt.ID] = ndt
	}

	// initialize data type validators

	dtb := newDataTypeBuilder()

	dtb.registerDataType(datatype.Composite.ID).withCompositeValidator(validateComposite)

	dtb.registerDataType(datatype.Birthdate.ID).withTimestampValidator(validateDate)
	dtb.registerDataType(datatype.Boolean.ID).withBoolValidator(validateBool)
	dtb.registerDataType(datatype.Date.ID).withTimestampValidator(validateDate)
	dtb.registerDataType(datatype.E164PhoneNumber.ID).withStringValidator(validateE164Phone)
	dtb.registerDataType(datatype.Email.ID).withStringValidator(validateEmail)
	dtb.registerDataType(datatype.Integer.ID).withIntValidator(validateInt)
	dtb.registerDataType(datatype.PhoneNumber.ID).withStringValidator(validatePhone)
	dtb.registerDataType(datatype.SSN.ID).withStringValidator(validateSSN)
	dtb.registerDataType(datatype.String.ID).withStringValidator(validateString)
	dtb.registerDataType(datatype.Timestamp.ID).withTimestampValidator(validateTimestamp)
	dtb.registerDataType(datatype.UUID.ID).withUUIDValidator(validateUUID)

	dts = dtb.build()
}
