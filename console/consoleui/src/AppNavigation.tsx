const PERSISTED_QUERYSTRING_VARS = ['company_id', 'tenant_id'];

export const makeCleanPageLink = (URLSearch: URLSearchParams): string => {
  const searchParams = new URLSearchParams();
  PERSISTED_QUERYSTRING_VARS.forEach((value) => {
    if (URLSearch.has(value)) {
      searchParams.set(value, URLSearch.get(value)!);
    }
  });

  return '?' + searchParams.toString();
};
