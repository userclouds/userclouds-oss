import { v4 as uuidv4 } from 'uuid';
import { DataType } from './DataType';
import { ResourceID, blankResourceID } from './ResourceID';

export const CompositeDataTypeUUID = 'd81658a7-848a-4504-9c6e-5fa17f90f1a6';
// NOTE: keep string values in sync with DataType defined in Go server code (idp/userstore/datatype/constants.go).
export const NativeDataTypes = {
  Birthdate: {
    id: '76f0685b-dd42-4b3f-8c33-4c72e4eff73e',
    name: 'birthdate',
  },
  Boolean: {
    id: 'e16b5ead-54db-4b42-a55f-f21907cda9e4',
    name: 'boolean',
  },
  Date: {
    id: '3e2546c0-14d6-49d3-8b95-a5000bb4ad6a',
    name: 'date',
  },
  E164PhoneNumber: {
    id: '97f0ab8a-f2fd-43da-9feb-3d1f8aacc042',
    name: 'e164_phonenumber',
  },
  Email: {
    id: '8a84f041-c605-4ebf-b552-9e14f51c9e54',
    name: 'email',
  },
  Integer: {
    id: '22b8a1b6-e5a2-4c3c-9a99-746f0345b727',
    name: 'integer',
  },
  PhoneNumber: {
    id: 'ae962c31-2ca7-42e1-814b-32e6493dba82',
    name: 'phonenumber',
  },
  SSN: {
    id: 'fba9f9bb-b9e0-4258-9fb8-6777792dbeba',
    name: 'ssn',
  },
  String: {
    id: 'd26b6d52-a8d7-4c2f-9efc-394eb90a3294',
    name: 'string',
  },
  Timestamp: {
    id: '66a87f97-32c4-4ccc-91da-d8c880e21e5a',
    name: 'timestamp',
  },
  UUID: {
    id: 'd036bbba-6012-4d74-b7c4-9a2bbc09a749',
    name: 'uuid',
  },
} as const;
export type DataTypeKey = keyof typeof NativeDataTypes;

export type DataTypeEnumFunc = {
  [key in DataTypeKey]: ResourceID;
};

export const DataTypeEnum: DataTypeEnumFunc = NativeDataTypes;

// NOTE: keep string values in sync with dataTypeInfo in idp/internal/storage/column/data_type.go:dataTypeByInternalType
export const ConcreteTypeFromSchemaDataType: { [key: string]: string } = {
  [NativeDataTypes.Boolean.id]: NativeDataTypes.Boolean.id,
  [NativeDataTypes.Birthdate.id]: NativeDataTypes.Date.id,
  [NativeDataTypes.Date.id]: NativeDataTypes.Date.id,
  [NativeDataTypes.E164PhoneNumber.id]: NativeDataTypes.String.id,
  [NativeDataTypes.Email.id]: NativeDataTypes.String.id,
  [NativeDataTypes.Integer.id]: NativeDataTypes.Integer.id,
  [NativeDataTypes.PhoneNumber.id]: NativeDataTypes.String.id,
  [NativeDataTypes.SSN.id]: NativeDataTypes.String.id,
  [NativeDataTypes.String.id]: NativeDataTypes.String.id,
  [NativeDataTypes.Timestamp.id]: NativeDataTypes.Timestamp.id,
  [NativeDataTypes.UUID.id]: NativeDataTypes.UUID.id,
};

export const getConcreteTypeFromSchemaDataType = (dataType: string) => {
  if (ConcreteTypeFromSchemaDataType[dataType]) {
    return ConcreteTypeFromSchemaDataType[dataType];
  }
  return CompositeDataTypeUUID;
};

export const NativeDataTypeFriendly: { [key: string]: string } = {
  [NativeDataTypes.Boolean.id]: 'Boolean',
  [NativeDataTypes.Birthdate.id]: 'Birthdate',
  [NativeDataTypes.Date.id]: 'Date',
  [NativeDataTypes.E164PhoneNumber.id]: 'E.164 Phone Number',
  [NativeDataTypes.Email.id]: 'Email',
  [NativeDataTypes.Integer.id]: 'Integer',
  [NativeDataTypes.PhoneNumber.id]: 'Phone Number',
  [NativeDataTypes.SSN.id]: 'SSN',
  [NativeDataTypes.String.id]: 'String',
  [NativeDataTypes.Timestamp.id]: 'Timestamp',
  [NativeDataTypes.UUID.id]: 'UUID',
};

export enum ColumnIndexType {
  None = 'none',
  Indexed = 'indexed',
  Unique = 'unique',
}

export const nativeDataTypes = [
  {
    key: NativeDataTypes.Boolean.id,
    label: NativeDataTypeFriendly[NativeDataTypes.Boolean.id],
    payload: NativeDataTypes.Boolean,
  },
  {
    key: NativeDataTypes.Birthdate.id,
    label: NativeDataTypeFriendly[NativeDataTypes.Birthdate.id],
    payload: NativeDataTypes.Birthdate,
  },
  {
    key: NativeDataTypes.Date.id,
    label: NativeDataTypeFriendly[NativeDataTypes.Date.id],
    payload: NativeDataTypes.Date,
  },
  {
    key: NativeDataTypes.E164PhoneNumber.id,
    label: NativeDataTypeFriendly[NativeDataTypes.E164PhoneNumber.id],
    payload: NativeDataTypes.E164PhoneNumber,
  },
  {
    key: NativeDataTypes.Email.id,
    label: NativeDataTypeFriendly[NativeDataTypes.Email.id],
    payload: NativeDataTypes.Email,
  },
  {
    key: NativeDataTypes.Integer.id,
    label: NativeDataTypeFriendly[NativeDataTypes.Integer.id],
    payload: NativeDataTypes.Integer,
  },
  {
    key: NativeDataTypes.PhoneNumber.id,
    label: NativeDataTypeFriendly[NativeDataTypes.PhoneNumber.id],
    payload: NativeDataTypes.PhoneNumber,
  },
  {
    key: NativeDataTypes.SSN.id,
    label: NativeDataTypeFriendly[NativeDataTypes.SSN.id],
    payload: NativeDataTypes.SSN,
  },
  {
    key: NativeDataTypes.String.id,
    label: NativeDataTypeFriendly[NativeDataTypes.String.id],
    payload: NativeDataTypes.String,
  },
  {
    key: NativeDataTypes.Timestamp.id,
    label: NativeDataTypeFriendly[NativeDataTypes.Timestamp.id],
    payload: NativeDataTypes.Timestamp,
  },
  {
    key: NativeDataTypes.UUID.id,
    label: NativeDataTypeFriendly[NativeDataTypes.UUID.id],
    payload: NativeDataTypes.UUID,
  },
];

export type ColumnConstraints = {
  immutable_required: boolean;
  partial_updates: boolean;
  unique_id_required: boolean;
  unique_required: boolean;
};

export const blankColumnConstraints = () => ({
  immutable_required: false,
  partial_updates: false,
  unique_id_required: false,
  unique_required: false,
});

export const columnConstraintsEqual = (
  constraints1: ColumnConstraints,
  constraints2: ColumnConstraints
) => {
  if (
    constraints1.immutable_required !== constraints2.immutable_required ||
    constraints1.partial_updates !== constraints2.partial_updates ||
    constraints1.unique_id_required !== constraints2.unique_id_required ||
    constraints1.unique_required !== constraints2.unique_required
  ) {
    return false;
  }
  return true;
};

export type Column = {
  id: string;
  table: string;
  name: string;
  data_type: ResourceID;
  access_policy: ResourceID;
  default_transformer: ResourceID;
  default_token_access_policy: ResourceID;
  is_array: boolean;
  index_type: ColumnIndexType;
  is_system?: boolean;
  search_indexed: boolean;
  constraints: ColumnConstraints;
};

export const blankColumn = () => ({
  id: uuidv4(),
  table: 'users',
  name: '',
  data_type: blankResourceID(),
  access_policy: blankResourceID(),
  default_transformer: blankResourceID(),
  default_token_access_policy: blankResourceID(),
  is_array: false,
  index_type: ColumnIndexType.None,
  is_system: false,
  search_indexed: false,
  constraints: blankColumnConstraints(),
});

export const COLUMN_PREFIX = 'columns_';
export const COLUMN_COLUMNS = ['id', 'name', 'created', 'updated'];

export type Schema = {
  columns: Column[];
};

export const defaultUserStoreEmailColumnID =
  '2c7a7c9b-90e8-47e4-8f6e-ec73bd2dec16';
export const defaultUserStoreNameColumnID =
  'fe20fd48-a006-4ad8-9208-4aad540d8794';

export const userStoreHasEmailColumn = (columns: Column[]) => {
  if (
    columns.filter((c) => c.id === defaultUserStoreEmailColumnID).length > 0
  ) {
    return true;
  }
  return false;
};

export const userStoreHasNameColumn = (columns: Column[]) => {
  if (columns.filter((c) => c.id === defaultUserStoreNameColumnID).length > 0) {
    return true;
  }
  return false;
};

type TenantUserStoreConfig = {
  schema: Schema;
};

export const columnsAreEqual = (columnA: Column, columnB: Column) => {
  return (
    columnA.id === columnB.id &&
    columnA.name === columnB.name &&
    columnA.data_type.id === columnB.data_type.id &&
    columnA.data_type.name === columnB.data_type.name &&
    columnA.is_array === columnB.is_array &&
    columnA.index_type === columnB.index_type &&
    columnA.search_indexed === columnB.search_indexed &&
    columnConstraintsEqual(columnA.constraints, columnB.constraints)
  );
};

export const columnNameAlphabeticalComparator = (a: Column, b: Column) => {
  const nameA = a.name.toUpperCase();
  const nameB = b.name.toUpperCase();
  if (nameA < nameB) {
    return -1;
  }
  if (nameA > nameB) {
    return 1;
  }
  return 0;
};

export const uniqueIDsAvailable = (col: Column, datatypes: DataType[]) => {
  const datatype = datatypes.find(
    (dataType) => dataType.id === col.data_type.id
  );
  if (!datatype) {
    return false;
  }
  return datatype.is_composite_field_type;
};

export const immutableAvailable = (col: Column, datatypes: DataType[]) => {
  const datatype = datatypes.find(
    (dataType) => dataType.id === col.data_type.id
  );
  if (!datatype) {
    return false;
  }
  if (col.constraints.unique_id_required && col.constraints.unique_required) {
    return true;
  }
  return false;
};

export const partialUpdatesAvailable = (col: Column) => {
  return (
    col.is_array &&
    (col.constraints.unique_id_required || col.constraints.unique_required)
  );
};

export const uniqueValuesAvailable = (col: Column, datatypes: DataType[]) => {
  const datatype = datatypes.find(
    (dataType) => dataType.id === col.data_type.id
  );
  if (!datatype) {
    return false;
  }
  if (
    !datatype.is_composite_field_type &&
    [
      NativeDataTypes.Integer.id as string,
      NativeDataTypes.UUID.id as string,
      NativeDataTypes.String.id as string,
    ].includes(datatype.id)
  ) {
    return true;
  }
  if (
    !datatype.composite_attributes.include_id &&
    datatype.composite_attributes.fields?.length === 1 &&
    (datatype.composite_attributes.fields[0].data_type.id ===
      NativeDataTypes.Integer.id ||
      datatype.composite_attributes.fields[0].data_type.id ===
        NativeDataTypes.UUID.id ||
      datatype.composite_attributes.fields[0].data_type.id ===
        NativeDataTypes.String.id)
  ) {
    return true;
  }
  return false;
};

export const getSelectedDataType = (name: string, dataTypes: DataType[]) => {
  const dataType = dataTypes.find((dt) => name === dt.name);
  return dataType;
};

export default TenantUserStoreConfig;
