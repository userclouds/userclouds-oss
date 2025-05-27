import {
  APIError,
  HTTPError,
  extractErrorMessage,
  tryValidate,
  tryGetJSON,
  makeAPIError,
} from '@userclouds/sharedui';
import PaginatedResult from './models/PaginatedResult';
import Company from './models/Company';
import { Roles, UserRoles } from './models/UserRoles';
import { UserProfile, MyProfile, UserBaseProfile } from './models/UserProfile';
import ActiveInstance from './chart/ActiveInstance';
import ServiceInfo from './ServiceInfo';
import LogRow from './chart/LogRow';

import { UserInvite } from './models/UserInvite';

export const PAGINATION_API_VERSION = '3';
// makeCompanyConfigURL constructs URLs to CompanyConfig endpoints for AJAX requests
// and makes it easy to centrally fix up URLs if things change in the future.
// path: URL path string
// query: javascript object/dict, query string, or URLSearchParams object
export function makeCompanyConfigURL(
  path: string,
  query?: string | Record<string, string>
): string {
  if (!query) {
    return path;
  }
  // Since this is running on the same protocol & host, use relative paths.
  return `${path}?${new URLSearchParams(query).toString()}`;
}

// Keep in sync with authz.UserObjectTypeID
export const UserTypeID = '1bf2b775-e521-41d3-8b7e-78e89427e6fe';

class InternalClient {
  async forceFetchCompanies(): Promise<Company[]> {
    return new Promise((resolve, reject) => {
      const companyURL = makeCompanyConfigURL('/api/companies');
      return fetch(companyURL)
        .then((response) => {
          if (!response.ok) {
            reject(new Error('Error fetching companies'));
          }
          return response.json();
        }, reject)
        .then(resolve, reject);
    });
  }

  async createCompany(company: Company): Promise<Company> {
    return new Promise((resolve, reject) => {
      const url = makeCompanyConfigURL('/api/companies');
      const req = {
        company: company,
      };
      return fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      })
        .then(async (response) => {
          if (!response.ok) {
            const message = await extractErrorMessage(response);
            throw new HTTPError(message, response.status);
          }

          resolve(response.json());
        }, reject)
        .catch((e) => {
          reject(
            makeAPIError(e, `Error creating company named ${company.name}`)
          );
        });
    });
  }

  // This method has auth to ensure only UserClouds company admins can call it.
  async fetchAllCompanies(): Promise<Company[]> {
    return new Promise(async (resolve, reject) => {
      const companyURL = makeCompanyConfigURL('/api/allcompanies');
      try {
        const rawResponse = await fetch(companyURL);
        const jsonResponse = await tryGetJSON(rawResponse);
        const typedResponse = jsonResponse as Company[];
        resolve(typedResponse);
      } catch (error) {
        reject(makeAPIError(error, 'error fetching all companies'));
      }
    });
  }

  async listTenantRolesForEmployees(tenantID: string): Promise<UserRoles[]> {
    return new Promise((resolve, reject) => {
      const url = makeCompanyConfigURL(
        `/api/tenants/${encodeURIComponent(tenantID)}/employeeroles`
      );
      return fetch(url)
        .then(async (response) => {
          if (!response.ok) {
            const message = await extractErrorMessage(response);
            throw new HTTPError(message, response.status);
          }

          resolve(response.json());
        })
        .catch((error) => {
          reject(
            makeAPIError(error, `error listing tenant user roles (${tenantID})`)
          );
        });
    });
  }

  async listCompanyRolesForEmployees(companyID: string): Promise<UserRoles[]> {
    return new Promise((resolve, reject) => {
      const url = makeCompanyConfigURL(
        `/api/companies/${encodeURIComponent(companyID)}/employeeroles`
      );
      return fetch(url)
        .then(async (response) => {
          if (!response.ok) {
            const message = await extractErrorMessage(response);
            throw new HTTPError(message, response.status);
          }

          resolve(response.json());
        })
        .catch((error) => {
          reject(
            makeAPIError(
              error,
              `error listing company user roles (${companyID})`
            )
          );
        });
    });
  }

  async addTenantRoleForUser(
    tenantID: string,
    userID: string,
    role: string
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      const req = {
        user_id: userID,
        organization_role: role,
      };
      const url = makeCompanyConfigURL(
        `/api/tenants/${encodeURIComponent(tenantID)}/employeeroles`
      );
      return fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      })
        .then(async (response) => {
          if (!response.ok) {
            const message = await extractErrorMessage(response);
            throw new HTTPError(message, response.status);
          }

          resolve();
        })
        .catch((error) => {
          reject(
            makeAPIError(
              error,
              `error adding company role (${JSON.stringify(req)})`
            )
          );
        });
    });
  }

  async updateTenantRolesForEmployee(
    tenantID: string,
    userRoles: UserRoles
  ): Promise<void> {
    return new Promise(async (resolve, reject) => {
      try {
        const req = {
          organization_role:
            userRoles.organization_role === Roles.NoRole
              ? null
              : userRoles.organization_role,
          policy_role:
            userRoles.policy_role === Roles.UserGroupPolicyNoRole
              ? null
              : userRoles.policy_role,
        };
        const url = makeCompanyConfigURL(
          `/api/tenants/${encodeURIComponent(
            tenantID
          )}/employeeroles/${encodeURIComponent(userRoles.id)}`
        );
        const response = await fetch(url, {
          method: 'PUT',
          body: JSON.stringify(req),
        });
        await tryValidate(response);
        resolve();
      } catch (error) {
        reject(makeAPIError(error, `error updating tenant role`));
      }
    });
  }

  async updateCompanyRolesForEmployee(
    companyID: string,
    userRoles: UserRoles
  ): Promise<void> {
    return new Promise(async (resolve, reject) => {
      try {
        const req = {
          organization_role: userRoles.organization_role,
          policy_role:
            userRoles.policy_role === Roles.UserGroupPolicyNoRole
              ? null
              : userRoles.policy_role,
        };
        const url = makeCompanyConfigURL(
          `/api/companies/${encodeURIComponent(
            companyID
          )}/employeeroles/${encodeURIComponent(userRoles.id)}`
        );
        const response = await fetch(url, {
          method: 'PUT',
          body: JSON.stringify(req),
        });
        await tryValidate(response);
        resolve();
      } catch (error) {
        reject(makeAPIError(error, `error updating company role`));
      }
    });
  }

  async removeTenantRolesForEmployee(
    tenantID: string,
    userID: string
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      const url = makeCompanyConfigURL(
        `/api/tenants/${encodeURIComponent(
          tenantID
        )}/employeeroles/${encodeURIComponent(userID)}`
      );
      return fetch(url, {
        method: 'DELETE',
      })
        .then(async (response) => {
          if (!response.ok) {
            const message = await extractErrorMessage(response);
            throw new HTTPError(message, response.status);
          }

          resolve();
        })
        .catch((error) => {
          reject(makeAPIError(error, `error removing tenant roles for user`));
        });
    });
  }

  async removeCompanyRolesForEmployee(
    companyID: string,
    userID: string
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      const url = makeCompanyConfigURL(
        `/api/companies/${encodeURIComponent(
          companyID
        )}/employeeroles/${encodeURIComponent(userID)}`
      );
      return fetch(url, {
        method: 'DELETE',
      })
        .then(async (response) => {
          if (!response.ok) {
            const message = await extractErrorMessage(response);
            throw new HTTPError(message, response.status);
          }

          resolve();
        })
        .catch((error) => {
          reject(makeAPIError(error, `error removing company roles for user`));
        });
    });
  }

  async fetchActiveInstances(tenantID: string): Promise<ActiveInstance[]> {
    return new Promise(async (resolve, reject) => {
      const url = makeCompanyConfigURL(
        `/api/tenants/${encodeURIComponent(tenantID)}/counters/sources`
      );

      try {
        const rawResponse = await fetch(url);
        const jsonResponse = await tryGetJSON(rawResponse);
        const instances = jsonResponse as ActiveInstance[];
        resolve(instances);
      } catch (e) {
        reject(
          makeAPIError(
            e,
            `error fetching active instance (tenant uuid: ${tenantID})`
          )
        );
      }
    });
  }

  async fetchEventLog(tenantID: string): Promise<LogRow[]> {
    return new Promise(async (resolve, reject) => {
      const url = makeCompanyConfigURL(
        `/api/tenants/${encodeURIComponent(
          tenantID
        )}/counters/activity?event_count=120`
      );

      try {
        const rawResponse = await fetch(url);
        const jsonResponse = await tryGetJSON(rawResponse);
        const logRows = jsonResponse as LogRow[];
        resolve(logRows);
      } catch (e) {
        reject(
          makeAPIError(e, `error fetching event log (tenant uuid: ${tenantID})`)
        );
      }
    });
  }

  async fetchTenantUserPage(
    tenantID: string,
    organizationID: string,
    queryParams: Record<string, string>
  ): Promise<PaginatedResult<UserBaseProfile>> {
    return new Promise((resolve, reject) => {
      queryParams.organization_id = organizationID;
      if (!queryParams.version) {
        queryParams.version = PAGINATION_API_VERSION;
      }
      const url = makeCompanyConfigURL(
        `/api/tenants/${encodeURIComponent(tenantID)}/users`,
        queryParams
      );
      return fetch(url)
        .then(async (response) => {
          if (!response.ok) {
            const message = await extractErrorMessage(response);
            throw new HTTPError(message, response.status);
          }
          resolve(response.json());
        })
        .catch((e) => {
          // this will catch 500 errors,
          // JSON parse errors,
          // and the thrown HTTPError above
          reject(
            makeAPIError(e, `error fetching users (tenant uuid: ${tenantID})`)
          );
        });
    });
  }

  async updateUser(
    tenantID: string,
    user: UserProfile
  ): Promise<UserProfile | APIError> {
    try {
      const url = makeCompanyConfigURL(
        `/api/tenants/${encodeURIComponent(
          tenantID
        )}/users/${encodeURIComponent(user.id)}`
      );
      const req = user;
      // TODO: use PATCH instead of PUT?
      const rawResponse = await fetch(url, {
        method: 'PUT',
        body: JSON.stringify(req),
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      const updatedUser = UserProfile.fromJSON(jsonResponse);
      return updatedUser;
    } catch (e) {
      return makeAPIError(
        e,
        `error saving user ${user.id} in tenant ${tenantID}`
      );
    }
  }

  async deleteUser(tenantID: string, userID: string): Promise<void> {
    return new Promise(async (resolve, reject) => {
      const url = makeCompanyConfigURL(
        `/api/tenants/${encodeURIComponent(
          tenantID
        )}/users/${encodeURIComponent(userID)}`
      );
      try {
        const response = await fetch(url, {
          method: 'DELETE',
        });
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve();
      } catch (e) {
        reject(
          makeAPIError(e, `error deleting user ${userID} in tenant ${tenantID}`)
        );
      }
    });
  }

  async fetchCompanyInvites(
    companyID: string,
    queryParams: Record<string, string>
  ): Promise<PaginatedResult<UserInvite>> {
    return new Promise((resolve, reject) => {
      if (!queryParams.version) {
        queryParams.version = PAGINATION_API_VERSION;
      }
      const url = makeCompanyConfigURL(
        `/api/companies/${encodeURIComponent(companyID)}/actions/listinvites`,
        queryParams
      );
      return fetch(url)
        .then(async (response) => {
          if (!response.ok) {
            const message = await extractErrorMessage(response);
            throw new HTTPError(message, response.status);
          }

          resolve(response.json());
        })
        .catch((error) => {
          reject(makeAPIError(error, `Error listing invites`));
        });
    });
  }

  async inviteUserToExistingCompany(
    companyID: string,
    inviteeEmails: string,
    company_role: string,
    tenant_roles: Record<string, string>
  ): Promise<void | APIError> {
    try {
      const url = makeCompanyConfigURL(
        `/api/companies/${encodeURIComponent(companyID)}/actions/inviteuser`
      );
      const req = {
        invitee_emails: inviteeEmails,
        company_role,
        tenant_roles,
      };
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      await tryValidate(rawResponse);
      return undefined;
    } catch (e) {
      return makeAPIError(e, `error sending invite(s)`);
    }
  }

  async fetchMyProfile(): Promise<MyProfile> {
    return new Promise(async (resolve, reject) => {
      const url = makeCompanyConfigURL('/auth/userinfo');
      try {
        const rawResponse = await fetch(url);
        const jsonResponse = await tryGetJSON(rawResponse);
        resolve(MyProfile.fromJSON(jsonResponse));
      } catch (e) {
        reject(makeAPIError(e, 'error fetching logged-in user profile'));
      }
    });
  }

  async fetchServiceInfo(): Promise<ServiceInfo> {
    return new Promise(async (resolve, reject) => {
      const url = makeCompanyConfigURL('/api/serviceinfo');
      return fetch(url)
        .then(async (response) => {
          if (!response.ok) {
            const message = await extractErrorMessage(response);
            throw new HTTPError(message, response.status);
          }

          resolve(response.json());
        })
        .catch((error) => {
          reject(makeAPIError(error, `Error fetching console service info`));
        });
    });
  }

  impersonateUser(tenantID: string, userID: string): Promise<void> {
    return new Promise(async (resolve, reject) => {
      try {
        const rawResponse = await fetch(`/auth/impersonateuser`, {
          body: JSON.stringify({ tenant_id: tenantID, target_user_id: userID }),
          method: 'POST',
        });
        const jsonResponse = await tryGetJSON(rawResponse);
        const typedResponse = jsonResponse as {
          redirect_to: string;
        };
        window.location.href = typedResponse.redirect_to;
        resolve();
      } catch (e) {
        reject(
          makeAPIError(
            e,
            `error logging in as user (tenant uuid: ${tenantID}, user uuid: ${userID})`
          )
        );
      }
    });
  }

  unimpersonateUser(): Promise<void> {
    return new Promise(async (resolve, reject) => {
      try {
        await fetch(`/auth/unimpersonateuser`, { method: 'GET' });
        resolve();
      } catch (e) {
        reject(makeAPIError(e, `error unimpersonating`));
      }
    });
  }
}

const API = new InternalClient();

export default API;
