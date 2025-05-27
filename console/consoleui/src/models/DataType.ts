import { v4 as uuidv4 } from 'uuid';
import { ResourceID, blankResourceID } from './ResourceID';

export type CompositeField = {
  data_type: ResourceID;
  name: string;
  camel_case_name: string;
  struct_name: string;
  required: boolean;
  ignore_for_uniqueness: boolean;
};

export const blankCompositeField = () => {
  return {
    data_type: blankResourceID(),
    name: '',
    camel_case_name: '',
    struct_name: '',
    required: false,
    ignore_for_uniqueness: false,
  };
};

export type CompositeAttributes = {
  include_id: boolean;
  fields: CompositeField[];
};

export const blankCompositeAttributes = () => ({
  include_id: false,
  fields: [],
});

export type DataType = {
  id: string;
  name: string;
  description: string;
  is_composite_field_type: boolean;
  is_native: boolean;
  composite_attributes: CompositeAttributes;
};

export const DATA_TYPE_PREFIX = 'data_types_';
export const DATA_TYPE_COLUMNS = ['id', 'name', 'created', 'updated'];

export const blankDataType = () => {
  return {
    id: uuidv4(),
    name: '',
    description: '',
    is_composite_field_type: false,
    is_native: false,
    composite_attributes: blankCompositeAttributes(),
  };
};

export const isValidDataType = (datatype: DataType) => {
  if (!datatype.id) {
    return false;
  }
  if (!datatype.name) {
    return false;
  }
  if (!datatype.composite_attributes) {
    return false;
  }
  return true;
};

export const getDisplayNameFieldType = (dataType: DataType) => {
  if (dataType.is_composite_field_type) {
    return 'Composite';
  }
  return dataType.composite_attributes?.fields?.length > 0
    ? dataType.composite_attributes.fields[0].camel_case_name
    : 'None';
};

export const validFieldName = (str: string): string => {
  const capitalized = str.charAt(0).toUpperCase() + str.slice(1);
  const result = capitalized.replace(/ /g, '_');
  return result;
};

export const getDataTypeIDFromName = (
  name: string,
  dataTypes: DataType[]
): string | undefined => {
  const dataType = dataTypes.find((dt) => dt.name === name);
  return dataType?.id;
};
