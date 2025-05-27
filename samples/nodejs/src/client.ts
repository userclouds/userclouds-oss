import { v4 as uuidv4 } from 'uuid';

import {
  AuthZClient,
  UserstoreClient,
  PlexClient,
  Edge,
  UCObject,
} from '@userclouds/sdk-typescript';

import {
  userAccessorID,
  userMutatorID,
  datasetObjTypeID,
  orgAdminEdgeTypeID,
} from './constants';

class Client {
  private authZClient: AuthZClient;

  private userstoreClient: UserstoreClient;

  private plexClient: PlexClient;

  constructor(
    authZClient: AuthZClient,
    userstoreClient: UserstoreClient,
    plexClient: PlexClient
  ) {
    this.authZClient = authZClient;
    this.userstoreClient = userstoreClient;
    this.plexClient = plexClient;
  }

  //
  // AuthZ-based methods
  //
  async isEmployee(userId: string): Promise<boolean> {
    return this.authZClient
      .getObject(userId)
      .then((obj: UCObject) => obj.organization_id === process.env.COMPANY_ID); // for the purpose of this app, "employees" are users in the company org
  }

  async getOrganization(orgID: string) {
    return this.authZClient.getOrganization(orgID);
  }

  async getOrgIDForUser(userId: string) {
    return this.authZClient
      .getObject(userId)
      .then((obj: UCObject) => obj.organization_id);
  }

  async listUsersInOrg(orgId: string) {
    return this.userstoreClient.listUsers(orgId).then((resp) => resp[0]);
  }

  async listOrganizations() {
    return this.authZClient.listOrganizations();
  }

  async listDatasets(orgId: string) {
    return this.authZClient
      .listObjects(datasetObjTypeID, orgId)
      .then((resp) => resp[0]);
  }

  async getDataset(datasetId: string) {
    return this.authZClient.getObject(datasetId);
  }

  async createOrganization(orgId: string, name: string, region?: string) {
    await this.authZClient.createOrganization(orgId, name, region);
  }

  async inviteUser(
    inviteeEmail: string,
    inviterUserId: string,
    inviterName: string,
    inviterEmail: string,
    state: string,
    clientId: string,
    redirectUrl: string,
    inviteText: string,
    expires: string
  ) {
    return this.plexClient.inviteUser(
      inviteeEmail,
      inviterUserId,
      inviterName,
      inviterEmail,
      clientId,
      state,
      redirectUrl,
      inviteText,
      expires
    );
  }

  async listAdmins(orgId: string) {
    return this.authZClient
      .listEdgesOnObject(orgId)
      .then(
        ([edges]) =>
          edges &&
          edges.length &&
          edges.filter((edge: Edge) => edge.edge_type_id === orgAdminEdgeTypeID)
      );
  }

  async isAdmin(userId: string, orgId: string): Promise<boolean> {
    return this.authZClient
      .listEdgesOnObject(userId)
      .then(([edges]): boolean => {
        if (
          edges &&
          edges.find(
            (edge: Edge) =>
              edge.edge_type_id === orgAdminEdgeTypeID &&
              edge.target_object_id === orgId
          )
        ) {
          return true;
        }
        return false;
      });
  }

  async makeAdmin(userID: string, orgId: string) {
    const isAdmin = await this.authZClient
      .listEdgesOnObject(userID)
      .then(([edges]): boolean => {
        if (
          edges &&
          edges.find(
            (edge: Edge) =>
              edge.edge_type_id === orgAdminEdgeTypeID &&
              edge.target_object_id === orgId
          )
        ) {
          return true;
        }
        return false;
      });
    if (!isAdmin) {
      await this.authZClient.createEdge(
        uuidv4(),
        orgAdminEdgeTypeID,
        userID,
        orgId
      );
    }
  }

  async revokeAdmin(userID: string, orgId: string) {
    await this.authZClient.listEdgesOnObject(userID).then(async ([edges]) => {
      const edge = edges.find(
        (e: Edge) =>
          e.edge_type_id === orgAdminEdgeTypeID && e.target_object_id === orgId
      );
      if (edge) {
        await this.authZClient.deleteEdge(edge.id);
      }
    });
  }

  //
  // Userstore-based methods
  //
  async getUserProfileForApp(userId: string) {
    return this.userstoreClient.executeAccessor(userAccessorID, {}, [userId]);
  }

  async updateUserProfileForApp(
    userId: string,
    phoneNumber: string,
    addresses: object[]
  ) {
    return this.userstoreClient.executeMutator(userMutatorID, {}, [userId], {
      PhoneNumber: {
        value: phoneNumber,
        purpose_additions: [{ Name: 'operational' }],
      },
      HomeAddresses: {
        value: addresses,
        purpose_additions: [{ Name: 'operational' }],
      },
    });
  }
}

export default Client;
