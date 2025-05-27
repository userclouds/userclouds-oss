import { v4 as uuidv4 } from 'uuid';
import { ResourceID } from './ResourceID';
import { NativeDataTypes } from './TenantUserStoreConfig';

export enum TransformType {
  PassThrough = 'passthrough',
  Transform = 'transform',
  TokenizeByValue = 'tokenizebyvalue',
  TokenizeByReference = 'tokenizebyreference',
}

export const TransformTypeFriendly = {
  [TransformType.PassThrough]: 'Passthrough',
  [TransformType.Transform]: 'Transform',
  [TransformType.TokenizeByValue]: 'Tokenize by value',
  [TransformType.TokenizeByReference]: 'Tokenize by reference',
};

export const TokenizingTransformerTypes = [
  TransformType.TokenizeByReference.toString(),
  TransformType.TokenizeByValue.toString(),
];

type Transformer = {
  id: string; // TODO: UUID
  name: string;
  description: string;
  input_data_type: ResourceID;
  output_data_type: ResourceID;
  reuse_existing_token: boolean;
  transform_type: TransformType;
  function: string;
  parameters: string;
  version: number;
  is_system: boolean;
};

export const TRANSFORMER_COLUMNS = [
  'id',
  'name',
  'description',
  'created',
  'updated',
];
export const TRANSFORMERS_PREFIX = 'transformers_';

export const DEFAULT_TRANSFORMER_NAME = 'PassthroughUnchangedData';

export const findByName = (transformers?: Transformer[], name?: string) => {
  if (!transformers || !name) return null;

  return transformers.find((t) => t.name === name);
};

export const blankTransformer = () => ({
  id: uuidv4(),
  name: '',
  description: '',
  input_data_type: NativeDataTypes.String,
  output_data_type: NativeDataTypes.String,
  transform_type: TransformType.TokenizeByValue,
  reuse_existing_token: false,
  function: 'function transform(data, params) {\n  return data;\n}',
  parameters: '{}',
  version: 0,
  is_system: false,
});

export type TransformerTestResult = {
  value: string;
  debug: Record<string, any>;
};

export default Transformer;
