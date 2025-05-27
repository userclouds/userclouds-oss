from .models import ResourceID

# Note: these need to stay in sync with idp/userstore/datatype/constants.go
ColumnDataTypeBirthdate = ResourceID(name="birthdate")
ColumnDataTypeBoolean = ResourceID(name="boolean")
ColumnDataTypeDate = ResourceID(name="date")
ColumnDataTypeE164PhoneNumber = ResourceID(name="e164_phonenumber")
ColumnDataTypeEmail = ResourceID(name="email")
ColumnDataTypeInteger = ResourceID(name="integer")
ColumnDataTypePhoneNumber = ResourceID(name="phonenumber")
ColumnDataTypeSSN = ResourceID(name="ssn")
ColumnDataTypeString = ResourceID(name="string")
ColumnDataTypeTimestamp = ResourceID(name="timestamp")
ColumnDataTypeUUID = ResourceID(name="uuid")
