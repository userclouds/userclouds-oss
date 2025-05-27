import { Filter, Operators } from '../models/authz/SearchFilters';

export const STRING_COLUMNS = [
  'alias',
  'name',
  'description',
  'path',
  'type_name',
  'type',
  'metadata->>format',
  'metadata->>classifications',
  'metadata->>storage',
  'metadata->>owner',
];
export const DATE_COLUMNS = ['created', 'updated'];
export const UUID_COLUMNS = [
  'id',
  'type_id',
  'object_id',
  'user_id',
  'data_source_id',
  'payload->>ID',
  'actor_id',
];
export const PAGINATION_ARGUMENTS = ['ending_before', 'starting_after'];
export const ARRAY_COLUMNS = ['payload->SelectorValues'];

export const STRING_PATTERN = /.+/;
export const DATE_PATTERN =
  /^(?:(0[1-9]|1[0-2])\/(0[1-9]|[12][0-9]|3[01])\/(\d{2}|\d{4})|(\d{4})\/(0[1-9]|1[0-2])\/(0[1-9]|[12][0-9]|3[01])(?:\s([01]\d|2[0-3]):([0-5]\d):([0-5]\d)(\.\d{1,4})?)?)$/;
export const UUID_PATTERN =
  /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/;
export const ARRAY_PATTERN = /^$|(?<=\().*?(?=\))/;

export const getPatternFromColumnName = (columnName: string) => {
  if (STRING_COLUMNS.includes(columnName)) {
    return STRING_PATTERN.source;
  }
  if (DATE_COLUMNS.includes(columnName)) {
    return DATE_PATTERN.source;
  }
  if (UUID_COLUMNS.includes(columnName)) {
    return UUID_PATTERN.source;
  }
  if (ARRAY_COLUMNS.includes(columnName)) {
    return ARRAY_PATTERN.source;
  }
};

export const getHumanReadableColumnName = (param: string) => {
  const components = param.split(/->>?/g);
  const words = components[components.length - 1].split('_');
  const capitalized = words.map((w) => w.charAt(0).toUpperCase() + w.slice(1));
  return capitalized.join(' ');
};

export const getSearchParamsArray = (searchFilter: string | null) => {
  if (!searchFilter) return [];
  const filters: Array<Filter> = [];
  searchFilter = searchFilter.replaceAll(',AND,', '');
  searchFilter = searchFilter.replaceAll(',OR,', '');
  searchFilter.slice(1, searchFilter.length - 1);
  const arr = searchFilter.split('(');
  arr.map((element) => {
    const filter = getFilterFromString(element);
    if (filter) {
      filters.push(filter);
    }
    return null;
  });
  return filters;
};

export const mergeFilter = (
  newFilter: Filter,
  previousFilterString: string | null
) => {
  const filterArr = getFilterArrayFromString(previousFilterString || '');

  if (newFilter.value) {
    newFilter.value = getFormattedSearchValue(
      newFilter.columnName,
      newFilter.value
    );
  }
  if (newFilter.value2) {
    newFilter.value2 = getFormattedSearchValue(
      newFilter.columnName,
      newFilter.value2
    );
  }

  filterArr.push(...getFilterStringFromFilter(newFilter));

  return getFilterStringFromFilterArray(filterArr);
};

export const addFilterToSearchParams = (
  prefix: string,
  searchParams: URLSearchParams,
  searchFilter: Filter
) => {
  const paginationParamKeys = PAGINATION_ARGUMENTS.map((arg) => {
    return prefix + arg;
  });

  const newParams = new URLSearchParams();
  for (const [key, val] of searchParams.entries()) {
    // clear pagination vars when filter changes
    if (!paginationParamKeys.includes(key)) {
      newParams.append(key, val);
    }
  }
  const newFilter = { ...searchFilter };
  const previousFilter = newParams.get(prefix + 'filter');

  const mergedFilter = mergeFilter(newFilter, previousFilter);

  newParams.set(prefix + 'filter', mergedFilter);

  return newParams;
};

export const clearSearchFilter = (
  columnName: string,
  operator: string,
  value: string,
  prefix: string,
  searchParams: URLSearchParams
) => {
  const newParams = new URLSearchParams(searchParams);
  const filterString = newParams.get(prefix + 'filter');
  if (!filterString || filterString.length < 5) {
    return newParams;
  }
  let arr = getFilterArrayFromString(filterString);
  arr = arr.filter(
    (element) =>
      !(
        element.includes(columnName) &&
        element.includes(operator) &&
        element.includes(value)
      )
  );
  if (arr.length >= 1) {
    newParams.set(prefix + 'filter', getFilterStringFromFilterArray(arr));
  } else {
    newParams.delete(prefix + 'filter');
  }
  return newParams;
};

const getFilterFromString = (filterString: string) => {
  filterString = filterString.replaceAll(',AND,', '');
  filterString = filterString.replaceAll(',OR,', '');
  const arr = filterString.split(',');
  if (arr.length !== 3) {
    return null;
  }
  return {
    columnName: arr[0].replaceAll("'", ''),
    operator: arr[1],
    value: arr[2].replaceAll(/['()]/g, ''),
  } as Filter;
};

const getFormattedSearchValue = (columnName: string, value: string) => {
  value = value.trim();
  if (STRING_COLUMNS.includes(columnName)) {
    if (!value.startsWith('%')) {
      value = '%' + value.trim();
    }
    if (!value.endsWith('%')) {
      value += '%';
    }
  }
  if (DATE_COLUMNS.includes(columnName)) {
    // BE expects this time in microseconds.
    value = String(new Date(value).getTime() * 1000);
  }
  return value;
};

export const getFormattedValue = (columnName: string, value: string) => {
  if (DATE_COLUMNS.includes(columnName)) {
    // BE expects this time in microseconds so we store it that way but display it in a user readable ISOString.
    const date = new Date(Number(value) / 1000);
    value = date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
  }
  return value;
};

const getFilterArrayFromString = (filter: string) => {
  const filterArr = filter.split('(');
  const acc = [] as string[];
  filterArr.map((element) => {
    const filterElement = getFilterFromString(element);
    if (filterElement !== null) {
      acc.push(
        "('" +
          filterElement.columnName +
          "'," +
          filterElement.operator +
          ",'" +
          filterElement.value +
          "')"
      );
    }
    return null;
  });
  return acc;
};

const getFilterStringFromFilter = (filter: Filter) => {
  const ret = [];
  if (filter.value) {
    ret.push(
      "('" +
        filter.columnName +
        "'," +
        filter.operator +
        ",'" +
        filter.value +
        "')"
    );
  }
  if (filter.value2) {
    ret.push(
      "('" +
        filter.columnName +
        "'," +
        filter.operator2 +
        ",'" +
        filter.value2 +
        "')"
    );
  }
  return ret;
};

const getFilterStringFromFilterArray = (filterArr: string[]) => {
  filterArr = filterArr.filter((filter) => {
    return filter !== '';
  });
  let filterString = filterArr.shift();
  filterArr.map((element) => {
    filterString = '(' + filterString + ',AND,' + element + ')';
    return null;
  });

  if (filterString === undefined) {
    return '()';
  }
  return filterString;
};

export const setOperatorsForFilter = (filter: Filter) => {
  const column = filter.columnName;
  if (STRING_COLUMNS.includes(column)) {
    filter.operator = Operators.LIKE;
  } else if (UUID_COLUMNS.includes(column)) {
    filter.operator = Operators.EQUAL;
  } else if (DATE_COLUMNS.includes(column)) {
    filter.operator = Operators.GREATER_THAN_EQUAL;
    filter.operator2 = Operators.LESS_THAN_EQUAL;
  } else if (ARRAY_COLUMNS.includes(column)) {
    filter.operator = Operators.HAS;
  }
  return filter;
};
