export const LAST_VIEWED_TENANT_KEY_PREFIX = 'lastViewedTenant';

/**
 * Gets the storage key for a specific company
 * @param companyId - The company ID
 */
export const getLastViewedTenantKey = (companyId: string): string => {
  return `${LAST_VIEWED_TENANT_KEY_PREFIX}-${companyId}`;
};

/**
 * Updates the last viewed tenant in localStorage based on the current URL
 * @param path - The current url or path
 */
export const updateLastViewedTenantFromURL = (path: string): void => {
  try {
    // Handle relative URLs by using the current window location
    let url: URL;
    if (path.startsWith('http')) {
      url = new URL(path);
    } else {
      // For relative paths, construct the full URL using the current window location
      url = new URL(path, window.location.origin);
    }

    const params = new URLSearchParams(url.search);
    const tenantId = params.get('tenant_id');
    const companyId = params.get('company_id');

    if (tenantId && companyId) {
      const key = getLastViewedTenantKey(companyId);
      localStorage.setItem(key, tenantId);
    }
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to update last viewed tenant from URL:', error);
  }
};

/**
 * Gets the last viewed tenant ID from localStorage for a specific company
 * @param companyId - The company ID
 */
export const getLastViewedTenant = (companyId: string): string | null => {
  try {
    const key = getLastViewedTenantKey(companyId);
    return localStorage.getItem(key);
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to get last viewed tenant from localStorage:', error);
    return null;
  }
};

/**
 * Sets the last viewed tenant ID in localStorage for a specific company
 * @param companyId - The company ID
 * @param tenantId - The tenant ID to store
 */
export const setLastViewedTenant = (
  companyId: string,
  tenantId: string
): void => {
  try {
    const key = getLastViewedTenantKey(companyId);
    localStorage.setItem(key, tenantId);
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to set last viewed tenant in localStorage:', error);
  }
};

/**
 * Removes the last viewed tenant entry for a specific company
 * @param companyId - The company ID
 */
export const removeLastViewedTenant = (companyId: string): void => {
  try {
    const key = getLastViewedTenantKey(companyId);
    localStorage.removeItem(key);
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error(
      'Failed to remove last viewed tenant from localStorage:',
      error
    );
  }
};
