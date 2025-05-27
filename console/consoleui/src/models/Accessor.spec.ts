import {
  AccessorColumn,
  getUnusedPurposes,
  transformerIsValidForAccessorColumn,
} from './Accessor';
import PaginatedResult from './PaginatedResult';
import Purpose from './Purpose';
import Transformer, { TransformType } from './Transformer';
import { NativeDataTypes } from './TenantUserStoreConfig';

describe('getUnusedPurposes', () => {
  let accessorPurposes: Purpose[] = [];
  const purposes: PaginatedResult<Purpose> = {
    data: [],
    has_next: false,
    has_prev: false,
    next: '',
    prev: '',
  };
  it('should return empty for undefined and empty', () => {
    accessorPurposes = [];
    purposes.data = [];
    let settings = getUnusedPurposes(undefined, undefined);
    expect(settings.length).toBe(0);
    settings = getUnusedPurposes(purposes, undefined);
    expect(settings.length).toBe(0);
    settings = getUnusedPurposes(undefined, accessorPurposes);
    expect(settings.length).toBe(0);
    settings = getUnusedPurposes(purposes, accessorPurposes);
    expect(settings.length).toBe(0);
  });

  it('should return empty for total overlap', () => {
    accessorPurposes = [
      { name: '1', id: '1', description: '1', is_system: false },
    ];
    purposes.data = [
      { name: '1', id: '1', description: '1', is_system: false },
    ];
    const settings = getUnusedPurposes(purposes, accessorPurposes);
    expect(settings.length).toBe(0);
    expect(accessorPurposes.length).toBe(1);
    expect(purposes.data.length).toBe(1);
  });

  it('should return correct non overlapping items', () => {
    accessorPurposes = [
      { name: '2', id: '2', description: '2', is_system: false },
    ];
    purposes.data = [
      { name: '3', id: '3', description: '3', is_system: false },
    ];
    const settings = getUnusedPurposes(purposes, accessorPurposes);
    expect(settings.length).toBe(1);
    expect(settings[0].id).toBe('3');
  });
});

describe('transformerIsValidForAccessorColumn', () => {
  const accessorColumnString: AccessorColumn = {
    id: 'string',
    name: 'string',
    table: 'string',
    data_type_id: NativeDataTypes.String.id,
    data_type_name: NativeDataTypes.String.name,
    is_array: false,
    transformer_id: '',
    transformer_name: '',
    default_access_policy_id: '',
    default_access_policy_name: '',
    token_access_policy_id: '',
    token_access_policy_name: '',
    default_transformer_name: '',
  };
  const accessorColumnDate: AccessorColumn = {
    id: 'date',
    name: 'date',
    table: 'string',
    data_type_id: NativeDataTypes.Date.id,
    data_type_name: NativeDataTypes.Date.name,
    is_array: false,
    transformer_id: '',
    transformer_name: '',
    default_access_policy_id: '',
    default_access_policy_name: '',
    token_access_policy_id: '',
    token_access_policy_name: '',
    default_transformer_name: '',
  };
  const accessorColumnBirthDate: AccessorColumn = {
    id: 'date',
    name: 'date',
    table: 'string',
    data_type_id: NativeDataTypes.Birthdate.id,
    data_type_name: NativeDataTypes.Birthdate.name,
    is_array: false,
    transformer_id: '',
    transformer_name: '',
    default_access_policy_id: '',
    default_access_policy_name: '',
    token_access_policy_id: '',
    token_access_policy_name: '',
    default_transformer_name: '',
  };
  const accessorColumnTimestamp: AccessorColumn = {
    id: 'timestamp',
    name: 'timestamp',
    table: 'string',
    data_type_id: NativeDataTypes.Timestamp.id,
    data_type_name: NativeDataTypes.Timestamp.name,
    is_array: false,
    transformer_id: '',
    transformer_name: '',
    default_access_policy_id: '',
    default_access_policy_name: '',
    token_access_policy_id: '',
    token_access_policy_name: '',
    default_transformer_name: '',
  };
  const accessorColumnUuid: AccessorColumn = {
    id: 'uuid',
    name: 'uuid',
    table: 'string',
    data_type_id: NativeDataTypes.UUID.id,
    data_type_name: NativeDataTypes.UUID.name,
    is_array: false,
    transformer_id: '',
    transformer_name: '',
    default_access_policy_id: '',
    default_access_policy_name: '',
    token_access_policy_id: '',
    token_access_policy_name: '',
    default_transformer_name: '',
  };
  const transformerString: Transformer = {
    id: 'string',
    name: '',
    description: '',
    input_data_type: NativeDataTypes.String,
    output_data_type: NativeDataTypes.String,
    reuse_existing_token: false,
    transform_type: TransformType.TokenizeByValue,
    function: '',
    parameters: '',
    version: 0,
    is_system: false,
  };

  const transformerDate: Transformer = {
    id: 'date',
    name: '',
    description: '',
    input_data_type: NativeDataTypes.Date,
    output_data_type: NativeDataTypes.Date,
    reuse_existing_token: false,
    transform_type: TransformType.TokenizeByValue,
    function: '',
    parameters: '',
    version: 0,
    is_system: false,
  };
  const transformerTimestamp: Transformer = {
    id: 'timestamp',
    name: '',
    description: '',
    input_data_type: NativeDataTypes.Timestamp,
    output_data_type: NativeDataTypes.Timestamp,
    reuse_existing_token: false,
    transform_type: TransformType.TokenizeByValue,
    function: '',
    parameters: '',
    version: 0,
    is_system: false,
  };
  const transformerUuid: Transformer = {
    id: 'uuid',
    name: '',
    description: '',
    input_data_type: NativeDataTypes.UUID,
    output_data_type: NativeDataTypes.UUID,
    reuse_existing_token: false,
    transform_type: TransformType.TokenizeByValue,
    function: '',
    parameters: '',
    version: 0,
    is_system: false,
  };
  const transformerPassthrough: Transformer = {
    id: 'passthrough',
    name: '',
    description: '',
    input_data_type: NativeDataTypes.String,
    output_data_type: NativeDataTypes.String,
    reuse_existing_token: false,
    transform_type: TransformType.PassThrough,
    function: '',
    parameters: '',
    version: 0,
    is_system: false,
  };

  it('should accept tranformers with same column type', () => {
    expect(
      transformerIsValidForAccessorColumn(
        accessorColumnString,
        transformerString
      )
    ).toBeTruthy();

    expect(
      transformerIsValidForAccessorColumn(accessorColumnUuid, transformerUuid)
    ).toBeTruthy();
    expect(
      transformerIsValidForAccessorColumn(accessorColumnDate, transformerDate)
    ).toBeTruthy();
    expect(
      transformerIsValidForAccessorColumn(
        accessorColumnTimestamp,
        transformerTimestamp
      )
    ).toBeTruthy();
  });

  it('should reject transformers with different and invalid column type', () => {
    expect(
      transformerIsValidForAccessorColumn(accessorColumnString, transformerUuid)
    ).toBeFalsy();
    expect(
      transformerIsValidForAccessorColumn(accessorColumnDate, transformerUuid)
    ).toBeFalsy();
  });

  it('should accept transformers with passthrough', () => {
    expect(
      transformerIsValidForAccessorColumn(
        accessorColumnString,
        transformerPassthrough
      )
    ).toBeTruthy();
    expect(
      transformerIsValidForAccessorColumn(
        accessorColumnUuid,
        transformerPassthrough
      )
    ).toBeTruthy();
    expect(
      transformerIsValidForAccessorColumn(
        accessorColumnDate,
        transformerPassthrough
      )
    ).toBeTruthy();
    expect(
      transformerIsValidForAccessorColumn(
        accessorColumnTimestamp,
        transformerPassthrough
      )
    ).toBeTruthy();
  });

  it('should accept transformers with string unless column is unknown', () => {
    expect(
      transformerIsValidForAccessorColumn(
        accessorColumnString,
        transformerString
      )
    ).toBeTruthy();
    expect(
      transformerIsValidForAccessorColumn(accessorColumnUuid, transformerString)
    ).toBeTruthy();
    expect(
      transformerIsValidForAccessorColumn(accessorColumnDate, transformerString)
    ).toBeTruthy();
    expect(
      transformerIsValidForAccessorColumn(
        accessorColumnTimestamp,
        transformerString
      )
    ).toBeTruthy();
  });

  it('should accept transformers based on concrete type', () => {
    expect(
      transformerIsValidForAccessorColumn(
        accessorColumnBirthDate,
        transformerDate
      )
    ).toBeTruthy();
    expect(
      transformerIsValidForAccessorColumn(
        accessorColumnBirthDate,
        transformerTimestamp
      )
    ).toBeTruthy();
  });
});
