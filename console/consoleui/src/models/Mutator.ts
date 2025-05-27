import { v4 as uuidv4 } from 'uuid';
import AccessPolicy, { blankPolicy } from './AccessPolicy';
import { Column } from './TenantUserStoreConfig';

export type MutatorSavePayload = {
  id: string; // TODO: UUID types
  name: string;
  description: string;
  columns: MutatorColumn[];
  access_policy_id?: string;
  selector_config: { where_clause: string };
  composed_access_policy?: AccessPolicy;
};

export type MutatorColumn = {
  id: string;
  name: string;
  table: string;
  data_type_id: string;
  data_type_name: string;
  is_array: boolean;
  normalizer_id: string;
  normalizer_name: string;
};

export const columnToMutatorColumn = (column: Column): MutatorColumn => {
  return {
    id: column.id,
    name: column.name,
    table: column.table,
    data_type_id: column.data_type.id,
    data_type_name: column.data_type.name,
    is_array: column.is_array,
    normalizer_id: '',
    normalizer_name: '',
  };
};

export type Mutator = {
  id: string;
  name: string;
  columns: MutatorColumn[];
  description: string;
  access_policy: AccessPolicy;
  selector_config: { where_clause: string };
  version: number;
  is_system: boolean;
};

export const blankMutator = (): Mutator => ({
  id: uuidv4(),
  name: '',
  columns: [],
  description: '',
  access_policy: blankPolicy(),
  selector_config: { where_clause: '{id} = ANY(?)' },
  version: -1,
  is_system: false,
});

export const columnsAreValid = (columns: MutatorColumn[]) => {
  if (!columns || !columns.length) {
    return false;
  }
  if (columns.some((col) => !col.normalizer_id)) {
    return false;
  }
  return true;
};

export const isValidMutator = (
  mutator: MutatorSavePayload,
  ap?: AccessPolicy
) => {
  if (!mutator.id) {
    return false;
  }
  if (!mutator.name) {
    return false;
  }
  if (!columnsAreValid(mutator.columns)) {
    return false;
  }
  if (
    !mutator.selector_config ||
    !mutator.selector_config.where_clause ||
    !mutator.selector_config.where_clause.length
  ) {
    return false;
  }
  if (
    !mutator.access_policy_id &&
    (!ap || !ap.components || !ap.components.length)
  ) {
    return false;
  }
  return true;
};

export const isValidMutatorToUpdate = (
  mutator: Mutator,
  columns: MutatorColumn[],
  ap?: AccessPolicy
) => {
  if (!mutator.id) {
    return false;
  }
  if (!mutator.name) {
    return false;
  }
  if (!columnsAreValid(mutator.columns)) {
    return false;
  }
  if (columns.length > 0 && !columnsAreValid(columns)) {
    return false;
  }
  if (
    !mutator.selector_config ||
    !mutator.selector_config.where_clause ||
    !mutator.selector_config.where_clause.length
  ) {
    return false;
  }
  if (
    !mutator.access_policy &&
    (!ap || !ap.components || !ap.components.length)
  ) {
    return false;
  }
  return true;
};

export const MUTATOR_PREFIX = 'mutators_';
export const MUTATOR_COLUMNS = ['id', 'name', 'created', 'updated'];

export default Mutator;
