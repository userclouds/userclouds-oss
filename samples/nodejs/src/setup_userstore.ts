import { v4 as uuidv4 } from 'uuid';

/* eslint-disable no-multi-str */
import {
  UserstoreClient,
  getClientCredentialsToken,
  Column,
  Purpose,
  ResourceID,
  AccessPolicyComponent,
  User,
  COLUMN_INDEX_TYPE_INDEXED,
  COLUMN_INDEX_TYPE_NONE,
  POLICY_TYPE_COMPOSITE_AND,
  DATA_TYPE_STRING,
  TRANSFORM_TYPE_PASSTHROUGH,
} from '@userclouds/sdk-typescript';
import { userAccessorID, userMutatorID, org1ID, org2ID } from './constants';

const setupUserstore = async (
  tenantUrl: string,
  clientId: string,
  clientSecret: string
) => {
  const token = await getClientCredentialsToken(
    tenantUrl,
    clientId,
    clientSecret
  );

  const client = new UserstoreClient(tenantUrl, token);
  const [columns] = await client.listColumns();
  const expectedColumns: Column[] = [
    {
      id: null,
      name: 'PhoneNumber',
      type: 'string',
      is_array: false,
      default_value: '',
      index_type: COLUMN_INDEX_TYPE_INDEXED,
    },
    {
      id: null,
      name: 'HomeAddresses',
      type: 'address',
      is_array: true,
      default_value: '',
      index_type: COLUMN_INDEX_TYPE_NONE,
    },
  ];

  const createColumns = expectedColumns.map(
    (expectedColumn): Promise<Column> | null => {
      if (
        columns.filter(
          (existingColumn: Column) =>
            existingColumn.name === expectedColumn.name &&
            existingColumn.type === expectedColumn.type &&
            existingColumn.is_array === expectedColumn.is_array &&
            existingColumn.default_value === expectedColumn.default_value &&
            existingColumn.index_type === expectedColumn.index_type
        ).length === 0
      ) {
        return client.createColumn(expectedColumn, true);
      }
      return null;
    }
  );

  await Promise.all(createColumns);

  const [purposes] = await client.listPurposes();
  const expectedPurposes: Purpose[] = [
    {
      id: null,
      name: 'operational',
      description: 'Purpose is for basic operation of the site',
    },
    {
      id: null,
      name: 'marketing', // this purpose is created as an example, not used for rest of demo
      description: 'Purpose is for marketing team to use for outreach',
    },
  ];

  const createPurposes = expectedPurposes.map(
    (expectedPurpose): Promise<Purpose> | null => {
      if (
        purposes.filter(
          (existingPurpose: Purpose) =>
            existingPurpose.name === expectedPurpose.name
        ).length === 0
      ) {
        return client.createPurpose(expectedPurpose, true);
      }
      return null;
    }
  );

  await Promise.all(createPurposes);

  const [accessPolicyTemplates] = await client.listAccessPolicyTemplates();
  let appAccessPolicyTemplate = accessPolicyTemplates.find(
    (template) => template.name === 'AccessPolicyTemplateForApp'
  );
  if (!appAccessPolicyTemplate) {
    appAccessPolicyTemplate = await client.createAccessPolicyTemplate({
      id: null,
      name: 'AccessPolicyTemplateForApp',
      description: 'Access policy template for app',
      function: 'function policy(context, params) { return true; }',
      version: 0,
    });
  }

  const [accessPolicies] = await client.listAccessPolicies();
  let appAccessPolicy = accessPolicies.find(
    (policy) => policy.name === 'AccessPolicyForApp'
  );
  if (!appAccessPolicy) {
    appAccessPolicy = await client.createAccessPolicy(
      {
        id: null,
        name: 'AccessPolicyForApp',
        description: 'Access policy for app',
        policy_type: POLICY_TYPE_COMPOSITE_AND,
        version: 0,
        components: [
          new AccessPolicyComponent(
            null,
            new ResourceID(appAccessPolicyTemplate.id, null),
            '{}'
          ),
        ],
        required_context: {},
      },
      true
    );
  }

  const [transformers] = await client.listTransformers();
  let passThroughPolicy = transformers.find(
    (policy) => policy.name === 'PassthroughUnchangedData'
  );
  if (!passThroughPolicy) {
    passThroughPolicy = await client.createTransformer(
      {
        id: null,
        name: 'PassthroughUnchangedData',
        description: 'Passthrough unchanged data',
        input_type: DATA_TYPE_STRING,
        transform_type: TRANSFORM_TYPE_PASSTHROUGH,
        function: 'function transform(data, params) {\n\treturn data;\n}',
        parameters: '{}',
      },
      true
    );
  }

  const accessors = await client.listAccessors();
  let accessor = accessors.find((a) => a.name === 'PIIAccessorForApp');
  if (accessor && accessor.id !== userAccessorID) {
    await client.deleteAccessor(accessor.id);
    accessor = null;
  }
  if (!accessor) {
    accessor = await client.createAccessor(
      {
        id: userAccessorID,
        name: 'PIIAccessorForApp',
        description: 'Accessor for main app',
        columns: [
          {
            column: new ResourceID(null, 'id'),
            transformer: new ResourceID(passThroughPolicy.id, null),
          },
          {
            column: new ResourceID(null, 'email'),
            transformer: new ResourceID(passThroughPolicy.id, null),
          },
          {
            column: new ResourceID(null, 'name'),
            transformer: new ResourceID(passThroughPolicy.id, null),
          },
          {
            column: new ResourceID(null, 'PhoneNumber'),
            transformer: new ResourceID(passThroughPolicy.id, null),
          },
          {
            column: new ResourceID(null, 'HomeAddresses'),
            transformer: new ResourceID(passThroughPolicy.id, null),
          },
        ],
        access_policy: new ResourceID(appAccessPolicy.id, null),
        selector_config: {
          where_clause: '{id} = ?',
        },
        token_access_policy: new ResourceID(null, null),
        purposes: [new ResourceID(null, 'operational')],
        version: 0,
      },
      true
    );
  }

  const mutators = await client.listMutators();
  let mutator = mutators.find((m) => m.name === 'PIIMutatorForApp');
  if (mutator && mutator.id !== userMutatorID) {
    await client.deleteMutator(mutator.id);
    mutator = null;
  }
  if (!mutator) {
    mutator = await client.createMutator(
      {
        id: userMutatorID,
        name: 'PIIMutatorForApp',
        description: 'Mutator for main app',
        columns: [
          {
            column: new ResourceID(null, 'PhoneNumber'),
            normalizer: new ResourceID(passThroughPolicy.id, null),
          },
          {
            column: new ResourceID(null, 'HomeAddresses'),
            normalizer: new ResourceID(passThroughPolicy.id, null),
          },
        ],
        access_policy: new ResourceID(appAccessPolicy.id, null),
        selector_config: {
          where_clause: '{id} = ?',
        },
        version: 0,
      },
      true
    );
  }

  // pre-populate some users in Org1 and Org2

  const expectedOrg1Users = [
    {
      id: uuidv4(),
      name: 'John Doe',
      email: 'johndoe@gmail.com',
      PhoneNumber: '123-456-7890',
      HomeAddresses: [
        {
          street_address_line_1: '123 Main St',
          locality: 'San Francisco',
          country: 'USA',
        },
      ],
    },
    {
      id: uuidv4(),
      name: 'Jane Doe',
      email: 'janedoe@gmail.com',
      PhoneNumber: '321-654-0987',
      HomeAddresses: [
        {
          street_address_line_1: '5 Crooked Way',
          locality: 'Ontario',
          country: 'Canada',
        },
      ],
    },
  ];
  const [org1Users] = await client.listUsers(org1ID);
  const createOrg1Users = expectedOrg1Users.map(
    (expectedUser): Promise<string> | null => {
      if (
        org1Users.filter(
          (foundUser: User) =>
            foundUser.profile.name === expectedUser.name &&
            foundUser.profile.email === expectedUser.email
        ).length === 0
      ) {
        return client.createUser(expectedUser, expectedUser.id, org1ID);
      }
      return null;
    }
  );

  const expectedOrg2Users = [
    {
      id: uuidv4(),
      name: 'Bart Simpson',
      email: 'elbarto@hotmail.com',
      PhoneNumber: '555-555-5555',
      HomeAddresses: [
        {
          street_address_line_1: '742 Evergreen Terrace',
          locality: 'Springfield',
          country: 'USA',
        },
      ],
    },
  ];
  const [org2Users] = await client.listUsers(org2ID);
  const createOrg2Users = expectedOrg2Users.map(
    (expectedUser): Promise<string> | null => {
      if (
        org2Users.filter(
          (foundUser: User) =>
            foundUser.profile.name === expectedUser.name &&
            foundUser.profile.email === expectedUser.email
        ).length === 0
      ) {
        return client.createUser(expectedUser, expectedUser.id, org2ID);
      }
      return null;
    }
  );

  await Promise.all([...createOrg1Users, ...createOrg2Users]);
};

export default setupUserstore;
