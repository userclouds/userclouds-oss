import { config as dotenvConfig } from 'dotenv';
import { v4 as uuidv4, NIL as NIL_UUID } from 'uuid';

/* eslint-disable import/no-unresolved */
import {
  AuthZClient,
  getClientCredentialsToken,
  UCObject,
  UCObjectType,
  EdgeType,
  Edge,
} from '@userclouds/sdk-typescript';
/* eslint-enable import/no-unresolved */

const docUserObjectType: UCObjectType = {
  id: '755410e3-97da-4acc-8173-4a10cab2c861',
  type_name: 'DocUser',
  organization_id: NIL_UUID,
};

const folderObjectType: UCObjectType = {
  id: 'f7478d4c-4001-4735-80bc-da136f22b5ac',
  type_name: 'Folder',
  organization_id: NIL_UUID,
};

const documentObjectType: UCObjectType = {
  id: 'a9460374-2431-4771-a760-840a62e5566e',
  type_name: 'Document',
  organization_id: NIL_UUID,
};

const userViewFolderEdgeType: EdgeType = {
  id: '4c3a7c7b-aae4-4d58-8094-7a9f3d7da7c6',
  type_name: 'UserViewFolder',
  source_object_type_id: docUserObjectType.id,
  target_object_type_id: folderObjectType.id,
  attributes: [
    {
      name: 'view',
      direct: true,
      inherit: false,
      propagate: false,
    },
  ],
};

const folderViewFolderEdgeType: EdgeType = {
  id: 'a2fcd885-f763-4a68-8733-3084631d2fbe',
  type_name: 'FolderViewFolder',
  source_object_type_id: folderObjectType.id,
  target_object_type_id: folderObjectType.id,
  attributes: [
    {
      name: 'view',
      direct: false,
      inherit: false,
      propagate: true,
    },
  ],
};

const folderViewDocEdgeType: EdgeType = {
  id: '0765a607-a933-4e6b-9c07-4566fa8c2944',
  type_name: 'FolderViewDoc',
  source_object_type_id: folderObjectType.id,
  target_object_type_id: documentObjectType.id,
  attributes: [
    {
      name: 'view',
      direct: false,
      inherit: false,
      propagate: true,
    },
  ],
};

const expectedObjTypes: UCObjectType[] = [
  docUserObjectType,
  folderObjectType,
  documentObjectType,
];

const expectedEdgeTypes: EdgeType[] = [
  userViewFolderEdgeType,
  folderViewFolderEdgeType,
  folderViewDocEdgeType,
];

const setupAuthZ = async (client: AuthZClient) => {
  const [objTypes] = await client.listObjectTypes();
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

  const [edgeTypes] = await client.listEdgeTypes();
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
};

const testAuthZ = async (client: AuthZClient) => {
  const objects: UCObject[] = [];
  const edges: Edge[] = [];

  try {
    const user = await client.createObject(
      uuidv4(),
      docUserObjectType.id,
      'user'
    );
    objects.push(user);

    const folder1 = await client.createObject(
      uuidv4(),
      folderObjectType.id,
      'folder1'
    );

    objects.push(folder1);

    const folder2 = await client.createObject(
      uuidv4(),
      folderObjectType.id,
      'folder2'
    );
    objects.push(folder2);

    const doc1 = await client.createObject(
      uuidv4(),
      documentObjectType.id,
      'doc1'
    );
    objects.push(doc1);

    const doc2 = await client.createObject(
      uuidv4(),
      documentObjectType.id,
      'doc2'
    );
    objects.push(doc2);

    edges.push(
      await client.createEdge(
        uuidv4(),
        userViewFolderEdgeType.id,
        user.id,
        folder1.id
      )
    );

    edges.push(
      await client.createEdge(
        uuidv4(),
        folderViewFolderEdgeType.id,
        folder1.id,
        folder2.id
      )
    );

    edges.push(
      await client.createEdge(
        uuidv4(),
        folderViewDocEdgeType.id,
        folder2.id,
        doc2.id
      )
    );

    // user can view folder1
    if (!(await client.checkAttribute(user.id, folder1.id, 'view'))) {
      throw new Error('user cannot view folder1 but should be able to');
    }

    // user can view folder2
    if (!(await client.checkAttribute(user.id, folder2.id, 'view'))) {
      throw new Error('user cannot view folder2 but should be able to');
    }

    // user cannot view doc1
    if (await client.checkAttribute(user.id, doc1.id, 'view')) {
      throw new Error('user can view doc1 but should not be able to');
    }

    // user can view doc2
    if (!(await client.checkAttribute(user.id, doc2.id, 'view'))) {
      throw new Error('user cannot view doc2 but should be able to');
    }
  } catch (e) {
    console.error(e);
    throw e;
  } finally {
    await Promise.all(
      edges.map(async (edge) => {
        await client.deleteEdge(edge.id);
      })
    );

    await Promise.all(
      objects.map(async (object) => {
        await client.deleteObject(object.id);
      })
    );
  }
};

const cleanupAuthZ = async (client: AuthZClient) => {
  const deleteEdgeTypes = expectedEdgeTypes.map(
    (edgeType): Promise<void> => client.deleteEdgeType(edgeType.id)
  );
  await Promise.all(deleteEdgeTypes);

  const deleteObjTypes = expectedObjTypes.map(
    (objType): Promise<void> => client.deleteObjectType(objType.id)
  );
  await Promise.all(deleteObjTypes);
};

const main = async (
  tenantURL: string,
  clientID: string,
  clientSecret: string
) => {
  const authHeader = await getClientCredentialsToken(
    tenantURL,
    clientID,
    clientSecret
  );

  const client = new AuthZClient(tenantURL, authHeader);

  await setupAuthZ(client);
  await testAuthZ(client);
  await cleanupAuthZ(client);
};

dotenvConfig();

if (
  process.env.TENANT_URL === undefined ||
  process.env.CLIENT_ID === undefined ||
  process.env.CLIENT_SECRET === undefined
) {
  throw new Error('Missing environment variables');
}

main(
  process.env.TENANT_URL,
  process.env.CLIENT_ID,
  process.env.CLIENT_SECRET
).catch((e) => {
  console.error(e);
  process.exit(1);
});
