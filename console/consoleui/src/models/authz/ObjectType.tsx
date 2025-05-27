import { v4 as uuidv4 } from 'uuid';

export const blankObjectType = () => {
  return {
    id: uuidv4(),
    type_name: '',
  };
};

export type ObjectType = {
  id: string; // TODO: UUID
  type_name: string;
};

export const OBJECT_TYPE_COLUMNS = [
  'id',
  'type_name',
  'source_object_type_id',
  'target_object_type_id',
  'created',
  'updated',
];

export const OBJECT_TYPE_PREFIX = 'object_types_';
