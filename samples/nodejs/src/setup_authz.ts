import { v4 as uuidv4, NIL as NIL_UUID } from 'uuid';

import {
  AuthZClient,
  getClientCredentialsToken,
  UCObject,
  UCObjectType,
  EdgeType,
  Organization,
  Region,
} from '@userclouds/sdk-typescript';

import {
  userObjTypeID,
  groupObjTypeID,
  datasetObjTypeID,
  orgAdminEdgeTypeID,
  org1ID,
  org2ID,
} from './constants';

const checkTypes = async (
  tenantUrl: string,
  clientId: string,
  clientSecret: string
) => {
  const token = await getClientCredentialsToken(
    tenantUrl,
    clientId,
    clientSecret
  );

  const client = new AuthZClient(tenantUrl, token);

  // ensure our custom object types exist

  const [objTypes] = await client.listObjectTypes();
  const expectedObjTypes: UCObjectType[] = [
    { id: datasetObjTypeID, type_name: 'Dataset', organization_id: NIL_UUID },
  ];

  const createObjTypes = expectedObjTypes.map(
    (expectedObjType): Promise<UCObjectType> | null => {
      if (
        objTypes.filter(
          (foundObjType: UCObjectType) =>
            foundObjType.id === expectedObjType.id &&
            foundObjType.type_name === expectedObjType.type_name
        ).length === 0
      ) {
        return client.createObjectType(
          expectedObjType.id,
          expectedObjType.type_name
        );
      }
      return null;
    }
  );

  await Promise.all(createObjTypes);

  // ensure our custom edge types exist

  const [edgeTypes] = await client.listEdgeTypes();

  const expectedEdgeTypes: EdgeType[] = [
    {
      id: orgAdminEdgeTypeID,
      type_name: 'OrgAdmin',
      source_object_type_id: userObjTypeID,
      target_object_type_id: groupObjTypeID,
      attributes: [],
    },
  ];

  const createEdgeTypes = expectedEdgeTypes.map(
    (expectedEdgeType): Promise<EdgeType> | null => {
      if (
        edgeTypes.filter(
          (foundEdgeType: EdgeType) =>
            foundEdgeType.id === expectedEdgeType.id &&
            foundEdgeType.type_name === expectedEdgeType.type_name &&
            foundEdgeType.source_object_type_id ===
              expectedEdgeType.source_object_type_id &&
            foundEdgeType.target_object_type_id ===
              expectedEdgeType.target_object_type_id &&
            foundEdgeType.attributes === expectedEdgeType.attributes
        ).length === 0
      ) {
        return client.createEdgeType(
          expectedEdgeType.id,
          expectedEdgeType.type_name,
          expectedEdgeType.source_object_type_id,
          expectedEdgeType.target_object_type_id,
          expectedEdgeType.attributes
        );
      }
      return null;
    }
  );

  await Promise.all(createEdgeTypes);

  // pre-populate with some organizations

  const [orgs] = await client.listOrganizations();

  const expectedOrgs: Organization[] = [
    { id: org1ID, name: 'Incognito', region: Region.Default },
    { id: org2ID, name: 'Simpsons', region: Region.Default },
  ];

  const createOrgs = expectedOrgs.map(
    (expectedOrg): Promise<Organization> | null => {
      if (
        orgs.filter(
          (foundOrg: Organization) => foundOrg.name === expectedOrg.name
        ).length === 0
      ) {
        return client.createOrganization(
          expectedOrg.id,
          expectedOrg.name,
          expectedOrg.region
        );
      }
      return null;
    }
  );

  await Promise.all(createOrgs);

  // pre-populate some objects

  const [objects] = await client.listObjects();

  const expectedObjs: UCObject[] = [
    {
      id: uuidv4(),
      type_id: datasetObjTypeID,
      alias: 'Sierra Forest',
      organization_id: org1ID,
    },
    {
      id: uuidv4(),
      type_id: datasetObjTypeID,
      alias: 'Somewhere smoky',
      organization_id: org1ID,
    },
    {
      id: uuidv4(),
      type_id: datasetObjTypeID,
      alias: 'Somewhere else',
      organization_id: org1ID,
    },
    {
      id: uuidv4(),
      type_id: datasetObjTypeID,
      alias: 'Mountain Fire',
      organization_id: org2ID,
    },
    {
      id: uuidv4(),
      type_id: datasetObjTypeID,
      alias: 'Camp Fire',
      organization_id: org2ID,
    },
  ];

  const createObjs = expectedObjs.map(
    (expectedObj): Promise<UCObject> | null => {
      if (
        objects.filter(
          (foundObj: UCObject) =>
            foundObj.alias === expectedObj.alias &&
            foundObj.type_id === expectedObj.type_id
        ).length === 0
      ) {
        return client.createObject(
          expectedObj.id,
          expectedObj.type_id,
          expectedObj.alias,
          expectedObj.organization_id
        );
      }
      return null;
    }
  );

  await Promise.all(createObjs);
};

export default checkTypes;
