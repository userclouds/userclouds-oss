import {
  columnsAreValid,
  isValidMutator,
  isValidMutatorToUpdate,
  blankMutator,
  MutatorSavePayload,
  MutatorColumn,
  Mutator,
} from './Mutator';
import { blankPolicy as blankAccessPolicy } from './AccessPolicy';

describe('columnsAreValid', () => {
  it('should return false if columns are empty or undefined', () => {
    expect(columnsAreValid([])).toBe(false);
  });

  it('should return false if any column does not have a normalizer_id', () => {
    const columns: MutatorColumn[] = [
      {
        id: '1',
        name: 'column1',
        table: 'table1',
        data_type_id: 'data1',
        data_type_name: 'data1',
        is_array: false,
        normalizer_id: '',
        normalizer_name: 'normalizer1',
      },
    ];
    expect(columnsAreValid(columns)).toBe(false);
  });

  it('should return true if all columns are valid', () => {
    const columns: MutatorColumn[] = [
      {
        id: '1',
        name: 'column1',
        table: 'table1',
        data_type_id: 'data1',
        data_type_name: 'data1',
        is_array: false,
        normalizer_id: 'norm1',
        normalizer_name: 'normalizer1',
      },
    ];
    expect(columnsAreValid(columns)).toBe(true);
  });
});

describe('isValidMutator', () => {
  it('should return false if mutator id is missing', () => {
    const mutator: MutatorSavePayload = {
      id: '',
      name: 'mutator1',
      description: 'desc1',
      columns: [],
      selector_config: { where_clause: '' },
    };
    expect(isValidMutator(mutator)).toBe(false);
  });

  it('should return false if mutator name is missing', () => {
    const mutator: MutatorSavePayload = {
      id: '1',
      name: '',
      description: 'desc1',
      columns: [],
      selector_config: { where_clause: '' },
    };
    expect(isValidMutator(mutator)).toBe(false);
  });

  it('should return false if columns are invalid', () => {
    const mutator: MutatorSavePayload = {
      id: '1',
      name: 'mutator1',
      description: 'desc1',
      columns: [],
      selector_config: { where_clause: '' },
    };
    expect(isValidMutator(mutator)).toBe(false);
  });

  it('should return false if selector_config is invalid', () => {
    const mutator: MutatorSavePayload = {
      id: '1',
      name: 'mutator1',
      description: 'desc1',
      columns: [],
      selector_config: { where_clause: '' },
    };
    expect(isValidMutator(mutator)).toBe(false);
  });

  it('should return true if all required fields are valid', () => {
    const mutator: MutatorSavePayload = {
      id: '1',
      name: 'mutator1',
      description: 'desc1',
      columns: [
        {
          id: '1',
          name: 'column1',
          table: 'table1',
          data_type_id: 'data1',
          data_type_name: 'data1',
          is_array: false,
          normalizer_id: 'norm1',
          normalizer_name: 'normalizer1',
        },
      ],
      access_policy_id: '1',
      selector_config: { where_clause: 'clause' },
    };
    expect(isValidMutator(mutator)).toBe(true);
  });
});

describe('isValidMutatorToUpdate', () => {
  it('should return false if mutator id is missing', () => {
    const mutator: Mutator = {
      id: '',
      name: 'mutator1',
      description: 'desc1',
      columns: [],
      access_policy: blankAccessPolicy(),
      selector_config: { where_clause: '' },
      version: 1,
      is_system: false,
    };
    const columns: MutatorColumn[] = [];
    expect(isValidMutatorToUpdate(mutator, columns)).toBe(false);
  });

  it('should return false if mutator name is missing', () => {
    const mutator: Mutator = {
      id: '1',
      name: '',
      description: 'desc1',
      columns: [],
      access_policy: blankAccessPolicy(),
      selector_config: { where_clause: '' },
      version: 1,
      is_system: false,
    };
    const columns: MutatorColumn[] = [];
    expect(isValidMutatorToUpdate(mutator, columns)).toBe(false);
  });

  it('should return false if columns are invalid', () => {
    const mutator: Mutator = {
      id: '1',
      name: 'mutator1',
      description: 'desc1',
      columns: [],
      access_policy: blankAccessPolicy(),
      selector_config: { where_clause: '' },
      version: 1,
      is_system: false,
    };
    const columns: MutatorColumn[] = [];
    expect(isValidMutatorToUpdate(mutator, columns)).toBe(false);
  });

  it('should return true if all required fields are valid', () => {
    const mutator: Mutator = {
      id: '1',
      name: 'mutator1',
      description: 'desc1',
      columns: [
        {
          id: '1',
          name: 'column1',
          table: 'table1',
          data_type_id: 'data1',
          data_type_name: 'data1',
          is_array: false,
          normalizer_id: 'norm1',
          normalizer_name: 'normalizer1',
        },
      ],
      access_policy: blankAccessPolicy(),
      selector_config: { where_clause: 'clause' },
      version: 1,
      is_system: false,
    };
    const columns: MutatorColumn[] = [
      {
        id: '1',
        name: 'column1',
        table: 'table1',
        data_type_id: 'data1',
        data_type_name: 'data1',
        is_array: false,
        normalizer_id: 'norm1',
        normalizer_name: 'normalizer1',
      },
    ];
    expect(isValidMutatorToUpdate(mutator, columns)).toBe(true);
  });
});

describe('blankMutator', () => {
  it('should create a new mutator with default values', () => {
    const mutator = blankMutator();
    expect(mutator.id).toBeTruthy();
    expect(mutator.name).toBe('');
    expect(mutator.description).toBe('');
    expect(mutator.columns).toEqual([]);
    expect(mutator.selector_config).toEqual({ where_clause: '{id} = ANY(?)' });
  });
});
