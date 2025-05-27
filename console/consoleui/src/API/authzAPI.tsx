import {
  JSONValue,
  HTTPError,
  extractErrorMessage,
  tryGetJSON,
  makeAPIError,
} from '@userclouds/sharedui';

import { fetchTenantUser } from './users';

import { UserProfile } from '../models/UserProfile';
import { ObjectType } from '../models/authz/ObjectType';
import EdgeType from '../models/authz/EdgeType';
import UCObject from '../models/authz/Object';
import Edge from '../models/authz/Edge';
import PaginatedResult from '../models/PaginatedResult';
import {
  AuthorizationRequest,
  CheckAttributeResponse,
} from '../models/authz/CheckAttribute';
import { makeCompanyConfigURL, PAGINATION_API_VERSION } from '../API';

// Keep in sync with authz.UserObjectTypeID
export const UserTypeID = '1bf2b775-e521-41d3-8b7e-78e89427e6fe';

export const runTenantAuthZAuthorizationCheck = (
  tenantID: string,
  authorizationRequest: AuthorizationRequest
): Promise<CheckAttributeResponse> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/authz/checkattribute`,
      {
        source_object_id: authorizationRequest.sourceObjectID,
        attribute: authorizationRequest.attribute,
        target_object_id: authorizationRequest.targetObjectID,
      }
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
        reject(makeAPIError(error, `Error checking attribute`));
      });
  });
};

export const fetchTenantAuthZObjectTypes = (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<ObjectType>> => {
  return new Promise(async (resolve, reject) => {
    // TODO: we might eventually want to support pagination arguments to this method, but for now we assume
    // there aren't that many object types and rely on the console endpoint to return them all.
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/authz/objecttypes`,
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
        reject(
          makeAPIError(
            e,
            `error fetching authz object types (tenant uuid: ${tenantID})`
          )
        );
      });
  });
};

export const fetchTenantAuthZObjectType = (
  tenantID: string,
  objectTypeID: string
): Promise<ObjectType> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/authz/objecttypes/${objectTypeID}`
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
        reject(
          makeAPIError(
            e,
            `error fetching authz object type (tenant uuid: ${tenantID}, object type uuid: ${objectTypeID})`
          )
        );
      });
  });
};

export const createTenantAuthZObjectType = (
  tenantID: string,
  objectType: ObjectType
): Promise<ObjectType> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      '/api/tenants/' + tenantID + '/authz/objecttypes'
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({ object_type: objectType }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      })
      .catch((e) => {
        reject(makeAPIError(e, 'error creating object type'));
      });
  });
};

export const deleteTenantAuthZObjectType = (
  tenantID: string,
  objectID: string
): Promise<void> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/authz/objecttypes/${encodeURIComponent(objectID)}`
    );
    return fetch(url, { method: 'DELETE' })
      .then(async (response) => {
        if (response.ok) {
          resolve();
        } else {
          reject(
            makeAPIError(
              response.statusText,
              `error deleting object type (tenant uuid: ${tenantID}, object uuid: ${objectID})`
            )
          );
        }
      })
      .catch((e) => {
        reject(
          makeAPIError(
            e,
            `error deleting object type (tenant uuid: ${tenantID}, object uuid: ${objectID})`
          )
        );
      });
  });
};

export const fetchTenantAuthZEdgeTypes = (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<EdgeType>> => {
  return new Promise(async (resolve, reject) => {
    // TODO: we might eventually want to support pagination arguments to this method, but for now we assume
    // there aren't that many edge types and rely on the console endpoint to return them all.
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/authz/edgetypes`,
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
        reject(
          makeAPIError(
            e,
            `error fetching authz edge types (tenant uuid: ${tenantID})`
          )
        );
      });
  });
};

export const fetchTenantAuthZEdgeType = (
  tenantID: string,
  edgeTypeID: string
): Promise<EdgeType> => {
  return new Promise(async (resolve, reject) => {
    // TODO: we might eventually want to support pagination arguments to this method, but for now we assume
    // there aren't that many edge types and rely on the console endpoint to return them all.
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/authz/edgetypes/${edgeTypeID}`
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
        reject(
          makeAPIError(
            e,
            `error fetching authz edge type (tenant uuid: ${tenantID}, edge type uuid: ${edgeTypeID})`
          )
        );
      });
  });
};

export const createTenantAuthZEdgeType = (
  tenantID: string,
  edgeType: EdgeType
): Promise<EdgeType> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      '/api/tenants/' + tenantID + '/authz/edgetypes'
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({ edge_type: edgeType }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      })
      .catch((e) => {
        reject(makeAPIError(e, 'error creating edge type'));
      });
  });
};

export const updateTenantAuthZEdgeType = (
  tenantID: string,
  edgeType: EdgeType
): Promise<EdgeType> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      '/api/tenants/' + tenantID + '/authz/edgetypes/' + edgeType.id
    );
    return fetch(url, {
      method: 'PUT',
      body: JSON.stringify({ edge_type: edgeType }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      })
      .catch((e) => {
        reject(makeAPIError(e, 'error updating edge type'));
      });
  });
};

export const deleteTenantAuthZEdgeType = (
  tenantID: string,
  edgeTypeID: string
): Promise<void> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/authz/edgetypes/${encodeURIComponent(edgeTypeID)}`
    );
    return fetch(url, { method: 'DELETE' })
      .then(async (response) => {
        if (response.ok) {
          resolve();
        } else {
          reject(
            makeAPIError(
              response.statusText,
              `error deleting edge type (tenant uuid: ${tenantID}, object uuid: ${edgeTypeID})`
            )
          );
        }
      })
      .catch((e) => {
        reject(
          makeAPIError(
            e,
            `error deleting edge type (tenant uuid: ${tenantID}, object uuid: ${edgeTypeID})`
          )
        );
      });
  });
};

export const deleteTenantAuthZEdge = (
  tenantID: string,
  edgeID: string
): Promise<void> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/authz/edges/${encodeURIComponent(edgeID)}`
    );
    return fetch(url, { method: 'DELETE' })
      .then(async (response) => {
        if (response.ok) {
          resolve();
        } else {
          reject(
            makeAPIError(
              response.statusText,
              `error deleting edge (tenant uuid: ${tenantID}, edge uuid: ${edgeID})`
            )
          );
        }
      })
      .catch((e) => {
        reject(
          makeAPIError(
            e,
            `error deleting edge (tenant uuid: ${tenantID}, edge uuid: ${edgeID})`
          )
        );
      });
  });
};

export const fetchTenantAuthZObjectPage = (
  tenantID: string,
  queryParams: Record<string, string>,
  objectTypeID?: string
): Promise<PaginatedResult<UCObject>> => {
  return new Promise(async (resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }
    if (objectTypeID) {
      queryParams.type_id = objectTypeID;
    }

    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/authz/objects`,
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
        reject(
          makeAPIError(
            error,
            `error fetching authz objects (tenant uuid: ${tenantID})`
          )
        );
      });
  });
};

export const createTenantAuthZObject = (
  tenantID: string,
  object: UCObject
): Promise<UCObject> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      '/api/tenants/' + tenantID + '/authz/objects'
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({ object: object }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      })
      .catch((e) => {
        reject(makeAPIError(e, 'error creating object'));
      });
  });
};

export const deleteTenantAuthZObject = (
  tenantID: string,
  objectID: string
): Promise<void> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/authz/objects/${encodeURIComponent(objectID)}`
    );
    return fetch(url, { method: 'DELETE' })
      .then(async (response) => {
        if (response.ok) {
          resolve();
        } else {
          reject(
            makeAPIError(
              response.statusText,
              `error deleting object (tenant uuid: ${tenantID}, object uuid: ${objectID})`
            )
          );
        }
      })
      .catch((e) => {
        reject(
          makeAPIError(
            e,
            `error deleting object (tenant uuid: ${tenantID}, object uuid: ${objectID})`
          )
        );
      });
  });
};

export const fetchTenantAuthZObject = (
  tenantID: string,
  objectID: string
): Promise<UCObject> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/authz/objects/${encodeURIComponent(objectID)}`
    );
    return fetch(url)
      .then(async (response) => {
        const jsonResponse = await tryGetJSON(response);
        const obj = jsonResponse as UCObject;
        // TODO: ugly client side hack to render users-as-objects; figure out
        // how to represent users-as-objects without hacks in multiple places.
        if (obj.type_id === UserTypeID) {
          try {
            // if a user is created as an object and not as a tenant user
            // we will get an error but can still resolve the object.
            const user = await fetchTenantUser(tenantID, objectID);
            const userProfile = UserProfile.fromJSON(user);
            obj.alias = userProfile.name() || userProfile.email() || objectID;
          } catch {
            resolve(obj);
          }
        }
        resolve(obj);
      })
      .catch((e) => {
        reject(
          makeAPIError(
            e,
            `error fetching object (tenant uuid: ${tenantID}, object uuid: ${objectID})`
          )
        );
      });
  });
};

export const fetchTenantAuthZEdges = (
  tenantID: string,
  objectID: string
): Promise<Array<Edge>> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/authz/edges`,
      { object_id: objectID, version: PAGINATION_API_VERSION }
    );
    return fetch(url)
      .then(async (response) => {
        const jsonResponse = await tryGetJSON(response);

        // TODO: this only fetches the first page of results. We aren't using this for much yet,
        // and need to revisit the client side of pagination anyways, so we will fix this (Issue #666).
        const result = jsonResponse as PaginatedResult<JSONValue>;
        const esJSON = (result?.data || []) as JSONValue[];
        const edges = new Array<Edge>();
        esJSON.forEach((eJSON) => {
          edges.push(eJSON as Edge);
        });
        resolve(edges);
      })
      .catch((e) => {
        reject(
          makeAPIError(
            e,
            `error fetching authz objects (tenant uuid: ${tenantID})`
          )
        );
      });
  });
};

export const fetchTenantAuthZEdgePage = (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<Edge>> => {
  return new Promise(async (resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }

    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/authz/edges`,
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
        reject(
          makeAPIError(
            error,
            `error fetching authz edges (tenant uuid: ${tenantID})`
          )
        );
      });
  });
};

export const fetchTenantAuthZEdge = (
  tenantID: string,
  edgeID: string
): Promise<Edge> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/authz/edges/${encodeURIComponent(edgeID)}`
    );
    return fetch(url)
      .then(async (response) => {
        const jsonResponse = await tryGetJSON(response);

        resolve(jsonResponse as Edge);
      })
      .catch((e) => {
        reject(
          makeAPIError(
            e,
            `error fetching edge (tenant uuid: ${tenantID}, edge uuid: ${edgeID})`
          )
        );
      });
  });
};

export const createTenantAuthZEdge = (
  tenantID: string,
  edge: Edge
): Promise<Edge> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      '/api/tenants/' + tenantID + '/authz/edges'
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({ edge: edge }),
    })
      .then(async (response) => {
        const jsonResponse = await tryGetJSON(response);
        resolve(jsonResponse as Edge);
      })
      .catch((e) => {
        reject(makeAPIError(e, 'error creating object'));
      });
  });
};
