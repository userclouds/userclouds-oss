import { v4 as uuidv4 } from 'uuid';
import AccessPolicy, { blankPolicy } from './AccessPolicy';
import {
  Column,
  getConcreteTypeFromSchemaDataType,
  NativeDataTypes,
} from './TenantUserStoreConfig';
import { DurationType } from './ColumnRetentionDurations';
import Purpose from './Purpose';
import Transformer, {
  TokenizingTransformerTypes,
  TransformType,
} from './Transformer';
import PaginatedResult from './PaginatedResult';

export const DEFAULT_PASSTHROUGH_ACCESSOR_ID =
  'c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a';

export const isTokenizingTransformer = (
  transformers: Transformer[] | undefined,
  transformer_id: string
) => {
  return (
    transformers &&
    transformers.find(
      (t) =>
        t.id === transformer_id &&
        TokenizingTransformerTypes.includes(t.transform_type)
    )
  );
};

export type AccessorSavePayload = {
  id: string; // TODO: UUID types
  name: string;
  description: string;
  data_life_cycle_state: DurationType;
  columns: AccessorColumn[];
  access_policy_id?: string;
  selector_config: { where_clause: string };
  purposes: Purpose[];
  composed_access_policy?: AccessPolicy;
  composed_token_access_policy?: AccessPolicy;
  is_audit_logged: boolean;
  are_column_access_policies_overridden: boolean;
  use_search_index: boolean;
};

export type AccessorColumn = {
  id: string;
  name: string;
  table: string;
  data_type_id: string;
  data_type_name: string;
  is_array: boolean;
  transformer_id: string;
  transformer_name: string;
  token_access_policy_id: string;
  token_access_policy_name: string;
  default_access_policy_id: string;
  default_access_policy_name: string;
  default_transformer_name: string;
};

export const columnToAccessorColumn = (column: Column): AccessorColumn => {
  return {
    id: column.id,
    name: column.name,
    table: column.table,
    data_type_id: column.data_type.id,
    data_type_name: column.data_type.name,
    is_array: column.is_array,
    transformer_id: column.default_transformer.id,
    transformer_name: column.default_transformer.name,
    token_access_policy_id: column.default_token_access_policy.id,
    token_access_policy_name: column.default_token_access_policy.name,
    default_access_policy_id: column.access_policy.id,
    default_access_policy_name: column.access_policy.name,
    default_transformer_name: column.default_transformer.name,
  };
};

export type Accessor = {
  id: string;
  name: string;
  description: string;
  data_life_cycle_state: DurationType;
  columns: AccessorColumn[];
  access_policy: AccessPolicy;
  selector_config: { where_clause: string };
  purposes: Purpose[];
  version: number;
  is_system: boolean;
  is_audit_logged: boolean;
  are_column_access_policies_overridden: boolean;
  use_search_index: boolean;
};

export const blankAccessor = (): Accessor => ({
  id: uuidv4(),
  name: '',
  description: '',
  data_life_cycle_state: DurationType.Live,
  columns: [],
  access_policy: blankPolicy(),
  selector_config: { where_clause: '{id} = ANY(?)' },
  purposes: [],
  version: -1,
  is_system: false,
  is_audit_logged: false,
  are_column_access_policies_overridden: false,
  use_search_index: false,
});

export const columnsAreValid = (columns: AccessorColumn[]) => {
  if (!columns || !columns.length) {
    return false;
  }
  if (columns.some((col) => !col.transformer_id)) {
    return false;
  }
  return true;
};

export const isValidAccessor = (
  accessor: AccessorSavePayload,
  ap: AccessPolicy
) => {
  if (!accessor.id) {
    return false;
  }
  if (!accessor.name) {
    return false;
  }
  if (!columnsAreValid(accessor.columns)) {
    return false;
  }
  if (!accessor.purposes || !accessor.purposes.length) {
    return false;
  }
  if (
    !accessor.selector_config ||
    !accessor.selector_config.where_clause ||
    !accessor.selector_config.where_clause.length
  ) {
    return false;
  }
  if (
    !accessor.access_policy_id &&
    (!ap || !ap.components || !ap.components.length)
  ) {
    return false;
  }

  return true;
};

export const isValidAccessorToUpdate = (
  accessor: Accessor,
  ap: AccessPolicy
) => {
  if (!accessor.id) {
    return false;
  }
  if (!accessor.name) {
    return false;
  }
  if (!columnsAreValid(accessor.columns)) {
    return false;
  }
  if (!accessor.purposes || !accessor.purposes.length) {
    return false;
  }
  if (
    !accessor.selector_config ||
    !accessor.selector_config.where_clause ||
    !accessor.selector_config.where_clause.length
  ) {
    return false;
  }
  if (
    !accessor.access_policy &&
    (!ap || !ap.components || !ap.components.length)
  ) {
    return false;
  }

  return true;
};

export const getUnusedPurposes = (
  purposes: PaginatedResult<Purpose> | undefined,
  modifiedAccessorPurposes: Purpose[] | undefined
) => {
  let displayPurposes: Purpose[] = [];
  if (purposes) {
    displayPurposes = purposes.data.filter((purpose) => {
      if (modifiedAccessorPurposes) {
        return !(
          modifiedAccessorPurposes.findIndex((purpose1) => {
            return purpose1.id === purpose.id;
          }) >= 0
        );
      }
      return true;
    });
  }
  return displayPurposes;
};

// Should be kept in sync with  idp/internal/storage/tokenizer_models.go:Transfomer.CanInput()
export const transformerIsValidForAccessorColumn = (
  column: AccessorColumn,
  transformer: Transformer
) => {
  if (transformer.transform_type === TransformType.PassThrough) {
    return true;
  }
  if (column.data_type_id === transformer.input_data_type.id) {
    return true;
  }
  if (transformer.input_data_type.id === NativeDataTypes.String.id) {
    return true;
  }
  if (transformer.input_data_type.id === NativeDataTypes.Date.id) {
    return (
      getConcreteTypeFromSchemaDataType(column.data_type_id) ===
      NativeDataTypes.Date.id
    );
  }
  if (transformer.input_data_type.id === NativeDataTypes.Timestamp.id) {
    return (
      getConcreteTypeFromSchemaDataType(column.data_type_id) ===
        NativeDataTypes.Timestamp.id ||
      getConcreteTypeFromSchemaDataType(column.data_type_id) ===
        NativeDataTypes.Date.id
    );
  }
  return false;
};

export type ExecuteAccessorResponse = {
  data: string[];
  debug: Record<string, any>;
};

export type ParsedExecuteAccessorData = Record<string, string>[];

export const ACCESSOR_PREFIX = 'accessors_';
export const ACCESSOR_COLUMNS = ['id', 'name', 'created', 'updated'];

export default Accessor;
