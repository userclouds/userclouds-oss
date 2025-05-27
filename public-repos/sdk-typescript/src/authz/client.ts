import { defaultLimit, paginationStart } from '../uc/pagination';
import UCObject, { UCObjectType } from './models/ucobject';
import Edge, { Attribute, EdgeType } from './models/edge';
import BaseClient from '../uc/baseclient';
import Organization from './models/organization';

class Client extends BaseClient {
  // objects
  async listObjects(
    typeId = '',
    organizationID = '',
    startingAfter: string = paginationStart,
    limit = 100
  ): Promise<[UCObject[], boolean]> {
    const params: { [key: string]: string } = {};
    if (typeId !== '') {
      params.type_id = typeId;
    }
    if (organizationID !== '') {
      params.organization_id = organizationID;
    }
    return this.makePaginatedRequest<UCObject>(
      '/authz/objects',
      params,
      startingAfter,
      limit
    );
  }

  async createObject(
    id: string,
    typeId: string,
    alias = '',
    organizationID = ''
  ): Promise<UCObject> {
    const object: { [key: string]: string } = {
      id,
      type_id: typeId,
      alias,
    };
    if (organizationID) {
      object.organization_id = organizationID;
    }
    return this.makeRequest(
      '/authz/objects',
      'POST',
      undefined,
      JSON.stringify({ object })
    );
  }

  async getObject(objectId: string): Promise<UCObject> {
    return this.makeRequest<UCObject>(`/authz/objects/${objectId}`);
  }

  async deleteObject(objectId: string): Promise<void> {
    return this.makeRequest<void>(`/authz/objects/${objectId}`, 'DELETE');
  }

  // edges
  async listEdges(
    startingAfter: string = paginationStart,
    limit = defaultLimit
  ): Promise<[Edge[], boolean]> {
    return this.makePaginatedRequest<Edge>(
      '/authz/edges',
      {},
      startingAfter,
      limit
    );
  }

  async listEdgesOnObject(
    objectID: string,
    startingAfter: string = paginationStart,
    limit = defaultLimit
  ): Promise<[Edge[], boolean]> {
    return this.makePaginatedRequest<Edge>(
      `/authz/objects/${objectID}/edges`,
      {},
      startingAfter,
      limit
    );
  }

  async createEdge(
    id: string,
    edgeTypeId: string,
    sourceObjectId: string,
    targetObjectId: string
  ): Promise<Edge> {
    const edge: { [key: string]: string } = {
      id,
      edge_type_id: edgeTypeId,
      source_object_id: sourceObjectId,
      target_object_id: targetObjectId,
    };
    return this.makeRequest(
      '/authz/edges',
      'POST',
      undefined,
      JSON.stringify({
        edge,
      })
    );
  }

  async getEdge(edgeId: string): Promise<Edge> {
    return this.makeRequest<Edge>(`/authz/edges/${edgeId}`);
  }

  async deleteEdge(edgeId: string): Promise<void> {
    return this.makeRequest<void>(`/authz/edges/${edgeId}`, 'DELETE');
  }

  // object types
  async listObjectTypes(
    startingAfter: string = paginationStart,
    limit: number = defaultLimit
  ): Promise<[UCObjectType[], boolean]> {
    return this.makePaginatedRequest<UCObjectType>(
      '/authz/objecttypes',
      {},
      startingAfter,
      limit
    );
  }

  async createObjectType(id: string, typeName: string): Promise<UCObjectType> {
    const object_type: { [key: string]: string } = {
      id,
      type_name: typeName,
    };
    return this.makeRequest(
      '/authz/objecttypes',
      'POST',
      undefined,
      JSON.stringify({ object_type })
    );
  }

  async getObjectType(typeId: string): Promise<UCObjectType> {
    return this.makeRequest<UCObjectType>(`/authz/objecttypes/${typeId}`);
  }

  async deleteObjectType(typeId: string): Promise<void> {
    return this.makeRequest<void>(`/authz/objecttypes/${typeId}`, 'DELETE');
  }

  // edge types
  async listEdgeTypes(
    organizationID = '',
    startingAfter: string = paginationStart,
    limit: number = defaultLimit
  ): Promise<[EdgeType[], boolean]> {
    const params: { [key: string]: string } = {};
    if (organizationID !== '') {
      params.organization_id = organizationID;
    }
    return this.makePaginatedRequest<EdgeType>(
      '/authz/edgetypes',
      params,
      startingAfter,
      limit
    );
  }

  async createEdgeType(
    id: string,
    typeName: string,
    sourceObjectTypeId: string,
    targetObjectTypeId: string,
    attributes: Attribute[],
    organizationID = ''
  ): Promise<EdgeType> {
    const edge_type: { [key: string]: string | object } = {
      id,
      type_name: typeName,
      source_object_type_id: sourceObjectTypeId,
      target_object_type_id: targetObjectTypeId,
      attributes,
    };
    if (organizationID) {
      edge_type.organization_id = organizationID;
    }
    return this.makeRequest(
      '/authz/edgetypes',
      'POST',
      undefined,
      JSON.stringify({ edge_type })
    );
  }

  async getEdgeType(typeId: string): Promise<EdgeType> {
    return this.makeRequest<EdgeType>(`/authz/edgetypes/${typeId}`);
  }

  async deleteEdgeType(typeId: string): Promise<void> {
    return this.makeRequest<void>(`/authz/edgetypes/${typeId}`, 'DELETE');
  }

  async getOrganization(orgID: string): Promise<Organization> {
    return this.makeRequest<Organization>(
      `/authz/organizations/${orgID}`,
      'GET'
    );
  }

  async createOrganization(
    id: string,
    name: string,
    region = ''
  ): Promise<Organization> {
    const organization: { [key: string]: string } = {
      id,
      name,
      region,
    };
    return this.makeRequest(
      '/authz/organizations',
      'POST',
      undefined,
      JSON.stringify({ organization })
    );
  }

  async listOrganizations(
    startingAfter: string = paginationStart,
    limit: number = defaultLimit
  ): Promise<[Organization[], boolean]> {
    return this.makePaginatedRequest<Organization>(
      '/authz/organizations',
      {},
      startingAfter,
      limit
    );
  }

  async checkAttribute(
    sourceObjectID: string,
    targetObjectID: string,
    attributeName: string
  ): Promise<boolean> {
    return this.makeRequest<{ has_attribute: boolean }>(
      '/authz/checkattribute',
      'GET',
      {
        source_object_id: sourceObjectID,
        target_object_id: targetObjectID,
        attribute: attributeName,
      }
    ).then((response) => response.has_attribute);
  }
}

export default Client;
