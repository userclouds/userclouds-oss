export const DEFAULT_PAGE_LIMIT = 50;
export const MAX_LIMIT = 1500;

const columns = [
  'filter',
  'starting_after',
  'ending_before',
  'limit',
  'sort_key',
  'sort_order',
];

export const getParamsAsObject = (
  prefix: string,
  params: URLSearchParams
): Record<string, string> => {
  if (!params) {
    return {};
  }
  const paramsAsObject = columns
    .map((element) => prefix + element)
    .reduce((acc: Record<string, string>, paramName: string) => {
      if (params.has(paramName)) {
        acc[paramName.substring(prefix.length)] = params.get(
          paramName
        ) as string;
      }
      return acc;
    }, {});
  return paramsAsObject;
};

export const applySort = (
  prefix: string,
  params: URLSearchParams,
  columnName: string
): string => {
  const paramsAsObject = getParamsAsObject(prefix, params);
  const currentSort = (paramsAsObject.sort_key || '').split(',')[0];
  const currentOrder = paramsAsObject.sort_order;
  delete paramsAsObject.sort_key;
  delete paramsAsObject.sort_order;
  paramsAsObject[prefix + 'sort_key'] =
    columnName + (columnName === 'id' ? '' : ',id');
  if (currentSort === columnName) {
    paramsAsObject[prefix + 'sort_order'] =
      currentOrder === 'ascending' ? 'descending' : 'ascending';
  } else {
    // TODO: descending for numerical columns?
    paramsAsObject[prefix + 'sort_order'] = 'ascending';
  }
  if (typeof params.get('company_id') === 'string') {
    paramsAsObject.company_id = params.get('company_id') as string;
  }
  if (typeof params.get('tenant_id') === 'string') {
    paramsAsObject.tenant_id = params.get('tenant_id') as string;
  }

  return new URLSearchParams(paramsAsObject).toString();
};

export const columnSortDirection = (
  prefix: string,
  params: URLSearchParams,
  columnName: string
) => {
  const currentSort = params.get(prefix + 'sort_key');
  const currentOrder = params.get(prefix + 'sort_order');
  const primaryKey = (currentSort || '').split(',')[0];

  return primaryKey === columnName
    ? currentOrder === 'ascending'
      ? 'sorted-asc'
      : 'sorted-desc'
    : 'sortable';
};
