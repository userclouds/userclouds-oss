import { Filter } from '../models/authz/SearchFilters';
import { NilUuid } from '../models/Uuids';
import {
  addFilterToSearchParams,
  getSearchParamsArray,
  clearSearchFilter,
  STRING_COLUMNS,
  getFormattedValue,
  UUID_COLUMNS,
  DATE_COLUMNS,
  ARRAY_PATTERN,
  UUID_PATTERN,
  DATE_PATTERN,
  STRING_PATTERN,
  getPatternFromColumnName,
} from './SearchHelper';

const prefix = 'prefix_';
describe('addFilterToSearchParams', () => {
  it('should add a filter to an empty search param', () => {
    const searchParams = new URLSearchParams();
    const filter = {
      columnName: 'alias',
      operator: 'LK',
      value: 'text',
    } as Filter;

    const newSearchParams = addFilterToSearchParams(
      prefix,
      searchParams,
      filter
    );
    expect(newSearchParams.get(prefix + 'filter')).toBe(
      "('alias',LK,'%text%')"
    );
  });
  it('should add a filter to a search param filter with one item', () => {
    const searchParams = new URLSearchParams();
    searchParams.set(prefix + 'filter', "('alias',LK,'%text%')");
    const filter = {
      columnName: 'alias',
      operator: 'LK',
      value: 'text123',
    } as Filter;

    const newSearchParams = addFilterToSearchParams(
      prefix,
      searchParams,
      filter
    );
    expect(newSearchParams.get(prefix + 'filter')).toBe(
      "(('alias',LK,'%text%'),AND,('alias',LK,'%text123%'))"
    );
  });
  it('should add a filter to a search param filter with more than one item', () => {
    const searchParams = new URLSearchParams();
    searchParams.set(prefix + 'filter', "('alias',LK,'%text%')");
    const filter1 = {
      columnName: 'alias',
      operator: 'LK',
      value: 'text123',
    } as Filter;
    const filter2 = {
      columnName: 'alias',
      operator: 'LK',
      value: 'text456',
    } as Filter;

    let newSearchParams = addFilterToSearchParams(
      prefix,
      searchParams,
      filter1
    );
    expect(newSearchParams.get(prefix + 'filter')).toBe(
      "(('alias',LK,'%text%'),AND,('alias',LK,'%text123%'))"
    );

    newSearchParams = addFilterToSearchParams(prefix, newSearchParams, filter2);
    expect(newSearchParams.get(prefix + 'filter')).toBe(
      "((('alias',LK,'%text%'),AND,('alias',LK,'%text123%')),AND,('alias',LK,'%text456%'))"
    );
  });
  it('should add a simple timestamp filter and format it to the correct epoch', () => {
    const searchParams = new URLSearchParams();
    const time = '12/1/2001';
    const filter = {
      columnName: 'created',
      operator: 'GE',
      value: '12/1/2001',
    } as Filter;
    const newSearchParams = addFilterToSearchParams(
      prefix,
      searchParams,
      filter
    );
    expect(newSearchParams.get(prefix + 'filter')).toBe(
      "('created',GE,'" + String(new Date(time).getTime() * 1000) + "')"
    );
  });
  it('should add a more complex timestamp filter and format it to the correct epoch', () => {
    const searchParams = new URLSearchParams();
    const time = '12/1/2001 5:00:00.000';
    const filter = {
      columnName: 'updated',
      operator: 'LE',
      value: time,
    } as Filter;
    const newSearchParams = addFilterToSearchParams(
      prefix,
      searchParams,
      filter
    );
    expect(newSearchParams.get(prefix + 'filter')).toBe(
      "('updated',LE,'" + String(new Date(time).getTime() * 1000) + "')"
    );
  });
});
it('should clear pagination arguments when adding a filter', () => {
  const searchParams = new URLSearchParams();
  searchParams.append(prefix + 'ending_before', '123');
  searchParams.append(prefix + 'starting_after', '456');
  expect(searchParams.get(prefix + 'ending_before')).toBe('123');
  expect(searchParams.get(prefix + 'starting_after')).toBe('456');

  const filter = {
    columnName: 'alias',
    operator: 'LK',
    value: 'text',
  } as Filter;

  const newSearchParams = addFilterToSearchParams(prefix, searchParams, filter);
  expect(newSearchParams.get(prefix + 'filter')).toBe("('alias',LK,'%text%')");
  expect(newSearchParams.get(prefix + 'ending_before')).toBeNull();
  expect(newSearchParams.get(prefix + 'starting_after')).toBeNull();
});
describe('getSearchParamArray', () => {
  it('should return an empty array when search params are empty', () => {
    const searchParams = new URLSearchParams();

    const searchParamsArray = getSearchParamsArray(
      searchParams.get(prefix + 'filter')
    );
    expect(searchParamsArray.length).toBe(0);
  });
  it('should get an array of filters from search params', () => {
    let searchParams = new URLSearchParams();
    searchParams.set(prefix + 'filter', "('alias',LK,'%text%')");
    const filter1 = {
      columnName: 'id',
      operator: 'EQ',
      value: 'id123',
    } as Filter;
    const filter2 = {
      columnName: 'date',
      operator: 'GE',
      value: '1/1/2000',
    } as Filter;
    searchParams = addFilterToSearchParams(prefix, searchParams, filter1);
    searchParams = addFilterToSearchParams(prefix, searchParams, filter2);

    const searchParamsArray = getSearchParamsArray(
      searchParams.get(prefix + 'filter')
    );

    expect(searchParamsArray.length).toBe(3);
    expect(searchParamsArray[0].columnName).toBe('alias');
    expect(searchParamsArray[0].operator).toBe('LK');
    expect(searchParamsArray[0].value).toBe('%text%');
    expect(searchParamsArray[1].columnName).toBe('id');
    expect(searchParamsArray[1].operator).toBe('EQ');
    expect(searchParamsArray[1].value).toBe('id123');
    expect(searchParamsArray[2].columnName).toBe('date');
    expect(searchParamsArray[2].operator).toBe('GE');
    expect(searchParamsArray[2].value).toBe('1/1/2000');
  });
  describe('clearSearchFilter', () => {
    it('should delete the filter when search params are empty', () => {
      let searchParams = new URLSearchParams();
      searchParams.set(prefix + 'filter', "('alias',LK,'%text%')");

      searchParams = clearSearchFilter(
        'alias',
        'LK',
        '%text%',
        prefix,
        searchParams
      );
      expect(searchParams.has(prefix + 'filter')).toBe(false);
    });
    it('should delete the correct filter when there are multiple', () => {
      let searchParams = new URLSearchParams();
      searchParams.set(
        prefix + 'filter',
        "((('alias',LK,'%text%'),AND,('alias',LK,'%text123%')),AND,('alias',LK,'%text456%'))"
      );

      searchParams = clearSearchFilter(
        'alias',
        'LK',
        '%text%',
        prefix,
        searchParams
      );

      expect(searchParams.get(prefix + 'filter')).toBe(
        "(('alias',LK,'%text123%'),AND,('alias',LK,'%text456%'))"
      );
    });
    it('should not delete a filter that is not there', () => {
      let searchParams = new URLSearchParams();
      searchParams.set(
        prefix + 'filter',
        "((('alias',LK,'%text%'),AND,('alias',LK,'%text123%')),AND,('alias',LK,'%text456%'))"
      );

      searchParams = clearSearchFilter(
        'alias',
        'LK',
        '%text123456%',
        prefix,
        searchParams
      );

      expect(searchParams.get(prefix + 'filter')).toBe(
        "((('alias',LK,'%text%'),AND,('alias',LK,'%text123%')),AND,('alias',LK,'%text456%'))"
      );
    });
  });
});
describe('getFormattedValue', () => {
  it('should do nothing for non date columns', () => {
    STRING_COLUMNS.forEach((columnName) => {
      expect(getFormattedValue(columnName, 'name')).toBe('name');
    });
    UUID_COLUMNS.forEach((columnName) => {
      expect(getFormattedValue(columnName, NilUuid)).toBe(NilUuid);
    });
  });
  it('should get a human readable timestamp for date columns from a unix epoch in microseconds', () => {
    DATE_COLUMNS.forEach((columnName) => {
      const epochDate = '1007200800000000';
      const date = new Date(Number(epochDate) / 1000);
      const value = date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
      expect(getFormattedValue(columnName, epochDate)).toBe(value);
    });
  });
});

describe('getPatternFromColumnName', () => {
  it('should return the STRING_PATTERN for string columns', () => {
    STRING_COLUMNS.forEach((columnName) => {
      expect(getPatternFromColumnName(columnName)).toBe(STRING_PATTERN.source);
    });
  });

  it('should return the DATE_PATTERN for date columns', () => {
    DATE_COLUMNS.forEach((columnName) => {
      expect(getPatternFromColumnName(columnName)).toBe(DATE_PATTERN.source);
    });
  });

  it('should return the UUID_PATTERN for UUID columns', () => {
    UUID_COLUMNS.forEach((columnName) => {
      expect(getPatternFromColumnName(columnName)).toBe(UUID_PATTERN.source);
    });
  });

  it('should return undefined for unknown columns', () => {
    expect(getPatternFromColumnName('unknown_column')).toBeUndefined();
  });
});

describe('Patterns', () => {
  it('should match non-empty strings with STRING_PATTERN', () => {
    expect(STRING_PATTERN.test('abc')).toBe(true);
    expect(STRING_PATTERN.test('')).toBe(false); // Should fail on empty string
  });

  it('should match valid dates', () => {
    const validDates = [
      '12/31/2024',
      '01/01/99',
      '2024/08/10',
      '2024/08/10 14:45:30',
      '2024/08/10 14:45:30.1234',
    ];
    validDates.forEach((date) => {
      expect(DATE_PATTERN.test(date)).toBe(true);
    });
  });

  it('should fail on invalid dates and times', () => {
    const invalidDates = [
      '13/01/2024', // Invalid month
      '2024/13/10', // Invalid month
      '2024/08/32', // Invalid day
      '2024/08/10 25:00:00', // Invalid hour
      '2024/08/10 14:60:00', // Invalid minute
    ];

    invalidDates.forEach((date) => {
      expect(DATE_PATTERN.test(date)).toBe(false);
    });
  });

  it('should match valid UUIDs with UUID_PATTERN', () => {
    const validUUIDs = [
      '123e4567-e89b-12d3-a456-426614174000',
      '550e8400-e29b-41d4-a716-446655440000',
    ];
    validUUIDs.forEach((uuid) => {
      expect(UUID_PATTERN.test(uuid)).toBe(true);
    });

    const invalidUUIDs = [
      '123e4567-e89b-12d3-a456-4266141740000', // Extra digit
      '123e4567-e89b-12d3-a456-42661417400', // Missing digit
      'g23e4567-e89b-12d3-a456-426614174000', // Invalid character
    ];
    invalidUUIDs.forEach((uuid) => {
      expect(UUID_PATTERN.test(uuid)).toBe(false);
    });
  });

  it('should match content inside parentheses with ARRAY_PATTERN', () => {
    const validArrays = ['', '(item1,item2,item3)', '(a,b,c)', '(1,2,3)'];
    validArrays.forEach((array) => {
      expect(ARRAY_PATTERN.test(array)).toBe(true);
    });

    const invalidArrays = [
      'item1,item2,item3', // No parentheses
      '(item1,item2', // Missing closing parenthesis
      'item1,item2)', // Missing opening parenthesis
    ];
    invalidArrays.forEach((array) => {
      expect(ARRAY_PATTERN.test(array)).toBe(false);
    });
  });
});
