import { APIError } from '@userclouds/sharedui';
import {
  fetchTenantAuthZEdge,
  fetchTenantAuthZObjectTypes,
  fetchTenantAuthZObjectType,
  fetchTenantAuthZEdgeTypes,
  fetchTenantAuthZEdgeType,
  fetchTenantAuthZObject,
  fetchTenantAuthZEdges,
  fetchTenantAuthZObjectPage,
  fetchTenantAuthZEdgePage,
  createTenantAuthZEdgeType,
  updateTenantAuthZEdgeType,
  createTenantAuthZObjectType,
} from '../API/authzAPI';
import { AppDispatch } from '../store';
import PaginatedResult from '../models/PaginatedResult';
import UCObject from '../models/authz/Object';
import { ObjectType, OBJECT_TYPE_PREFIX } from '../models/authz/ObjectType';
import EdgeType, { EDGE_TYPE_PREFIX } from '../models/authz/EdgeType';
import Edge from '../models/authz/Edge';
import {
  GET_EDGE_REQUEST,
  GET_EDGE_SUCCESS,
  GET_EDGE_ERROR,
  GET_OBJECT_TYPES_REQUEST,
  GET_OBJECT_TYPES_SUCCESS,
  GET_OBJECT_TYPES_ERROR,
  GET_OBJECT_TYPE_REQUEST,
  GET_OBJECT_TYPE_SUCCESS,
  GET_OBJECT_TYPE_ERROR,
  GET_EDGE_TYPES_REQUEST,
  GET_EDGE_TYPES_SUCCESS,
  GET_EDGE_TYPES_ERROR,
  GET_EDGE_TYPE_REQUEST,
  GET_EDGE_TYPE_SUCCESS,
  GET_EDGE_TYPE_ERROR,
  GET_OBJECT_REQUEST,
  GET_OBJECT_SUCCESS,
  GET_OBJECT_ERROR,
  GET_EDGES_REQUEST,
  GET_EDGES_SUCCESS,
  GET_EDGES_ERROR,
  GET_DISPLAY_OBJECTS_REQUEST,
  GET_DISPLAY_OBJECTS_SUCCESS,
  GET_DISPLAY_OBJECTS_ERROR,
  GET_DISPLAY_EDGES_REQUEST,
  GET_DISPLAY_EDGES_SUCCESS,
  GET_DISPLAY_EDGES_ERROR,
  CREATE_EDGE_TYPE_REQUEST,
  CREATE_EDGE_TYPE_SUCCESS,
  CREATE_EDGE_TYPE_ERROR,
  UPDATE_EDGE_TYPE_REQUEST,
  UPDATE_EDGE_TYPE_SUCCESS,
  UPDATE_EDGE_TYPE_ERROR,
  CREATE_OBJECT_TYPE_REQUEST,
  CREATE_OBJECT_TYPE_SUCCESS,
  CREATE_OBJECT_TYPE_ERROR,
} from '../actions/authz';
import { postSuccessToast } from './notifications';
import { redirect } from '../routing';
import {
  getParamsAsObject,
  DEFAULT_PAGE_LIMIT,
} from '../controls/PaginationHelper';
import { PAGINATION_API_VERSION } from '../API';

export const fetchAuthZEdge =
  (tenantID: string, edgeID: string) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_EDGE_REQUEST,
    });
    fetchTenantAuthZEdge(tenantID, edgeID).then(
      (edge: Edge) => {
        dispatch({
          type: GET_EDGE_SUCCESS,
          data: edge,
        });
      },
      (error: Error) => {
        dispatch({
          type: GET_EDGE_ERROR,
          data: error.message,
        });
      }
    );
  };
export const fetchAuthZObjectTypes =
  (tenantID: string, queryParams: URLSearchParams, limit?: number) =>
  (dispatch: AppDispatch) => {
    const paramsAsObject = getParamsAsObject(OBJECT_TYPE_PREFIX, queryParams);
    if (limit) {
      paramsAsObject.limit = String(limit);
    } else if (!paramsAsObject.limit) {
      paramsAsObject.limit = String(DEFAULT_PAGE_LIMIT);
    }
    if (!paramsAsObject.sort_order) {
      paramsAsObject.sort_order = 'ascending';
    }
    if (!paramsAsObject.sort_key) {
      paramsAsObject.sort_key = 'type_name,id';
    }
    if (!paramsAsObject.version) {
      paramsAsObject.version = PAGINATION_API_VERSION;
    }
    dispatch({
      type: GET_OBJECT_TYPES_REQUEST,
    });
    fetchTenantAuthZObjectTypes(tenantID, paramsAsObject).then(
      (objectTypes: PaginatedResult<ObjectType>) => {
        dispatch({
          type: GET_OBJECT_TYPES_SUCCESS,
          data: objectTypes,
        });
      },
      (error: Error) => {
        dispatch({
          type: GET_OBJECT_TYPES_ERROR,
          data: error.message,
        });
      }
    );
  };

export const fetchAuthZObjectType =
  (tenantID: string, objectTypeID: string) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_OBJECT_TYPE_REQUEST,
    });
    fetchTenantAuthZObjectType(tenantID, objectTypeID).then(
      (objectType: ObjectType) => {
        dispatch({
          type: GET_OBJECT_TYPE_SUCCESS,
          data: objectType,
        });
      },
      (error: APIError) => {
        dispatch({
          type: GET_OBJECT_TYPE_ERROR,
          data: error.message,
        });
      }
    );
  };

export const createObjectType =
  (tenantID: string, objectType: ObjectType) => (dispatch: AppDispatch) => {
    dispatch({
      type: CREATE_OBJECT_TYPE_REQUEST,
    });
    createTenantAuthZObjectType(tenantID, objectType).then(
      (response: ObjectType) => {
        dispatch({
          type: CREATE_OBJECT_TYPE_SUCCESS,
          data: response,
        });
        dispatch(postSuccessToast('Successfully created object type'));
        redirect(`/objecttypes/${response.id}/?tenant_id=${tenantID}`);
      },
      (error) => {
        dispatch({
          type: CREATE_OBJECT_TYPE_ERROR,
          data: error.message,
        });
      }
    );
  };
export const fetchAuthZEdgeTypes =
  (tenantID: string, queryParams: URLSearchParams, limit?: number) =>
  (dispatch: AppDispatch) => {
    const paramsAsObject = getParamsAsObject(EDGE_TYPE_PREFIX, queryParams);
    if (limit) {
      paramsAsObject.limit = String(limit);
    } else if (!paramsAsObject.limit) {
      paramsAsObject.limit = String(DEFAULT_PAGE_LIMIT);
    }
    if (!paramsAsObject.sort_order) {
      paramsAsObject.sort_order = 'ascending';
    }
    if (!paramsAsObject.sort_key) {
      paramsAsObject.sort_key = 'type_name,id';
    }
    if (!paramsAsObject.version) {
      paramsAsObject.version = PAGINATION_API_VERSION;
    }
    dispatch({
      type: GET_EDGE_TYPES_REQUEST,
    });
    fetchTenantAuthZEdgeTypes(tenantID, paramsAsObject).then(
      (edgeTypes: PaginatedResult<EdgeType>) => {
        dispatch({
          type: GET_EDGE_TYPES_SUCCESS,
          data: edgeTypes,
        });
      },
      (error: Error) => {
        dispatch({
          type: GET_EDGE_TYPES_ERROR,
          data: error.message,
        });
      }
    );
  };

export const fetchAuthZEdgeType =
  (tenantID: string, edgeTypeID: string) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_EDGE_TYPE_REQUEST,
    });
    fetchTenantAuthZEdgeType(tenantID, edgeTypeID).then(
      (edgeType: EdgeType) => {
        dispatch({
          type: GET_EDGE_TYPE_SUCCESS,
          data: edgeType,
        });
      },
      (error: Error) => {
        dispatch({
          type: GET_EDGE_TYPE_ERROR,
          data: error.message,
        });
      }
    );
  };

export const fetchAuthZObject =
  (tenantID: string, objectID: string) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_OBJECT_REQUEST,
    });
    fetchTenantAuthZObject(tenantID, objectID).then(
      (object: UCObject) => {
        dispatch({
          type: GET_OBJECT_SUCCESS,
          data: object,
        });
      },
      (error: Error) => {
        dispatch({
          type: GET_OBJECT_ERROR,
          data: error.message,
        });
      }
    );
  };

export const fetchAuthZEdgesForObject =
  (tenantID: string, objectID: string) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_EDGES_REQUEST,
    });
    fetchTenantAuthZEdges(tenantID, objectID).then(
      async (edges: Array<Edge>) => {
        // For each edge, fetch the other object.
        const promises = new Array<Promise<UCObject>>();
        const objectIDs = new Array<string>();
        edges.forEach((edge) => {
          const otherID =
            edge.source_object_id === objectID
              ? edge.target_object_id
              : edge.source_object_id;
          promises.push(fetchTenantAuthZObject(tenantID, otherID));
          objectIDs.push(otherID);
        });

        let objs = new Array<UCObject>();
        let lastError = '';
        Promise.all(promises).then(
          (objects: Array<UCObject>) => {
            objs = objects;
          },
          (error: Error) => {
            lastError = error.message;
          }
        );

        dispatch({
          type: GET_EDGES_SUCCESS,
          data: {
            edgesForObject: edges,
            objectsForEdgesForObject: objs,
            error: lastError,
          },
        });
      },
      (error: Error) => {
        dispatch({
          type: GET_EDGES_ERROR,
          data: error.message,
        });
      }
    );
  };

export const fetchAuthZObjects =
  (
    tenantID: string,
    queryParams: Record<string, string>,
    objectTypeID?: string
  ) =>
  (dispatch: AppDispatch) => {
    dispatch({
      type: GET_DISPLAY_OBJECTS_REQUEST,
    });
    fetchTenantAuthZObjectPage(tenantID, queryParams, objectTypeID).then(
      (response: PaginatedResult<UCObject>) => {
        dispatch({
          type: GET_DISPLAY_OBJECTS_SUCCESS,
          data: response,
        });
      },
      (error: Error) => {
        dispatch({
          type: GET_DISPLAY_OBJECTS_ERROR,
          data: error.message,
        });
      }
    );
  };

export const fetchAuthZEdges =
  (tenantID: string, queryParams: Record<string, string>) =>
  (dispatch: AppDispatch) => {
    dispatch({
      type: GET_DISPLAY_EDGES_REQUEST,
    });
    fetchTenantAuthZEdgePage(tenantID, queryParams).then(
      (response: PaginatedResult<Edge>) => {
        dispatch({
          type: GET_DISPLAY_EDGES_SUCCESS,
          data: response,
        });
      },
      (error: Error) => {
        dispatch({
          type: GET_DISPLAY_EDGES_ERROR,
          data: error.message,
        });
      }
    );
  };

export const createEdgeType =
  (tenantID: string, edgeType: EdgeType) => (dispatch: AppDispatch) => {
    dispatch({
      type: CREATE_EDGE_TYPE_REQUEST,
    });
    createTenantAuthZEdgeType(tenantID, edgeType).then(
      (response: EdgeType) => {
        dispatch({
          type: CREATE_EDGE_TYPE_SUCCESS,
          data: edgeType,
        });
        dispatch(postSuccessToast('Successfully created edge type'));
        redirect(`/edgetypes/${response.id}/?tenant_id=${tenantID}`);
      },
      (error) => {
        dispatch({
          type: CREATE_EDGE_TYPE_ERROR,
          data: error.message,
        });
      }
    );
  };

export const updateEdgeType =
  (tenantID: string, edgeType: EdgeType) => (dispatch: AppDispatch) => {
    dispatch({
      type: UPDATE_EDGE_TYPE_REQUEST,
    });
    updateTenantAuthZEdgeType(tenantID, edgeType).then(
      (response: EdgeType) => {
        dispatch({
          type: UPDATE_EDGE_TYPE_SUCCESS,
          data: response,
        });
        dispatch(fetchAuthZEdgeTypes(tenantID, new URLSearchParams()));
      },
      (error) => {
        dispatch({
          type: UPDATE_EDGE_TYPE_ERROR,
          data: error.message,
        });
      }
    );
  };
