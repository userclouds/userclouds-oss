import {
  AccessPolicy,
  AccessPolicyTemplate,
  Accessor,
  Column,
  Mutator,
  Purpose,
  Transformer,
  User,
} from './models';
import { defaultLimit, paginationStart } from '../uc/pagination';
import BaseClient, { APIError, APIErrorResponse } from '../uc/baseclient';

class Client extends BaseClient {
  // Access Policy functions
  async createAccessPolicy(
    access_policy: AccessPolicy,
    ifNotExists = false
  ): Promise<AccessPolicy> {
    return this.makeRequest<AccessPolicy>(
      '/tokenizer/policies/access',
      'POST',
      undefined,
      JSON.stringify({ access_policy })
    ).catch((error) => {
      if (error instanceof APIError && error.code === 409 && ifNotExists) {
        const apiErrorResponse = APIErrorResponse.fromJSON(error.body);
        if (apiErrorResponse.identical) {
          const ret = Object.assign(access_policy, {
            id: apiErrorResponse.id,
          });
          return ret;
        }
      }
      throw error;
    });
  }

  async getAccessPolicy(accessPolicyId: string): Promise<AccessPolicy> {
    return this.makeRequest<AccessPolicy>(
      `/tokenizer/policies/access/${accessPolicyId}`
    );
  }

  async listAccessPolicies(
    startingAfter: string = paginationStart,
    limit = 100
  ): Promise<[AccessPolicy[], boolean]> {
    return this.makePaginatedRequest<AccessPolicy>(
      '/tokenizer/policies/access',
      {},
      startingAfter,
      limit
    );
  }

  async updateAccessPolicy(access_policy: AccessPolicy): Promise<AccessPolicy> {
    return this.makeRequest<AccessPolicy>(
      `/tokenizer/policies/access/${access_policy.id}`,
      'PUT',
      undefined,
      JSON.stringify({ access_policy })
    );
  }

  async deleteAccessPolicy(accessPolicyId: string): Promise<void> {
    return this.makeRequest<void>(
      `/tokenizer/policies/access/${accessPolicyId}`,
      'DELETE'
    );
  }

  // Access Policy Template functions
  async createAccessPolicyTemplate(
    access_policy_template: AccessPolicyTemplate,
    ifNotExists = false
  ): Promise<AccessPolicyTemplate> {
    return this.makeRequest<AccessPolicyTemplate>(
      '/tokenizer/policies/accesstemplate',
      'POST',
      undefined,
      JSON.stringify({ access_policy_template })
    ).catch((error) => {
      if (error instanceof APIError && error.code === 409 && ifNotExists) {
        const apiErrorResponse = APIErrorResponse.fromJSON(error.body);
        if (apiErrorResponse.identical) {
          const ret = Object.assign(access_policy_template, {
            id: apiErrorResponse.id,
          });
          return ret;
        }
      }
      throw error;
    });
  }

  async getAccessPolicyTemplate(
    accessPolicyTemplateId: string
  ): Promise<AccessPolicyTemplate> {
    return this.makeRequest<AccessPolicyTemplate>(
      `/tokenizer/policies/access/${accessPolicyTemplateId}`
    );
  }

  async listAccessPolicyTemplates(
    startingAfter: string = paginationStart,
    limit = 100
  ): Promise<[AccessPolicyTemplate[], boolean]> {
    return this.makePaginatedRequest<AccessPolicyTemplate>(
      '/tokenizer/policies/accesstemplate',
      {},
      startingAfter,
      limit
    );
  }

  async updateAccessPolicyTemplate(
    access_policy_template: AccessPolicyTemplate
  ): Promise<AccessPolicyTemplate> {
    return this.makeRequest<AccessPolicyTemplate>(
      `/tokenizer/policies/accesstemplate/${access_policy_template.id}`,
      'PUT',
      undefined,
      JSON.stringify({ access_policy_template })
    );
  }

  async deleteAccessPolicyTemplate(
    accessPolicyTemplateId: string
  ): Promise<void> {
    return this.makeRequest<void>(
      `/tokenizer/policies/accesstemplate/${accessPolicyTemplateId}`,
      'DELETE'
    );
  }

  // Accessor functions
  async createAccessor(
    accessor: Accessor,
    ifNotExists = false
  ): Promise<Accessor> {
    return this.makeRequest<Accessor>(
      `/userstore/config/accessors`,
      'POST',
      undefined,
      JSON.stringify({ accessor })
    ).catch((error) => {
      if (error instanceof APIError && error.code === 409 && ifNotExists) {
        const apiErrorResponse = APIErrorResponse.fromJSON(error.body);
        if (apiErrorResponse.identical) {
          const ret = Object.assign(accessor, {
            id: apiErrorResponse.id,
          });
          return ret;
        }
      }
      throw error;
    });
  }

  async getAccessor(accessorId: string): Promise<Accessor> {
    return this.makeRequest<Accessor>(
      `/userstore/config/accessors/${accessorId}`
    );
  }

  async listAccessors(): Promise<Accessor[]> {
    return this.makeRequest<Accessor[]>(`/userstore/config/accessors`).then(
      (json) => {
        if ('data' in json) {
          return json.data as Accessor[];
        }
        return [];
      }
    );
  }

  async updateAccessor(accessor: Accessor): Promise<Accessor> {
    return this.makeRequest<Accessor>(
      `/userstore/config/accessors/${accessor.id}`,
      'PUT',
      undefined,
      JSON.stringify({ accessor })
    );
  }

  async deleteAccessor(accessorId: string): Promise<void> {
    return this.makeRequest<void>(
      `/userstore/config/accessors/${accessorId}`,
      'DELETE'
    );
  }

  // Column functions
  async createColumn(column: Column, ifNotExists = false): Promise<Column> {
    return this.makeRequest<Column>(
      `/userstore/config/columns`,
      'POST',
      undefined,
      JSON.stringify({ column })
    ).catch((error) => {
      if (error instanceof APIError && error.code === 409 && ifNotExists) {
        const apiErrorResponse = APIErrorResponse.fromJSON(error.body);
        if (apiErrorResponse.identical) {
          const ret = Object.assign(column, {
            id: apiErrorResponse.id,
          });
          return ret;
        }
      }
      throw error;
    });
  }

  async getColumn(columnId: string): Promise<Column> {
    return this.makeRequest<Column>(`/userstore/config/columns/${columnId}`);
  }

  async listColumns(
    startingAfter: string = paginationStart,
    limit = defaultLimit
  ): Promise<[Column[], boolean]> {
    return this.makePaginatedRequest<Column>(
      `/userstore/config/columns`,
      {},
      startingAfter,
      limit
    );
  }

  async updateColumn(column: Column): Promise<Column> {
    return this.makeRequest<Column>(
      `/userstore/config/columns/${column.id}`,
      'PUT',
      undefined,
      JSON.stringify({ column })
    );
  }

  async deleteColumn(columnId: string): Promise<void> {
    return this.makeRequest<void>(
      `/userstore/config/columns/${columnId}`,
      'DELETE'
    );
  }

  // Mutator functions
  async createMutator(mutator: Mutator, ifNotExists = false): Promise<Mutator> {
    return this.makeRequest<Mutator>(
      `/userstore/config/mutators`,
      'POST',
      undefined,
      JSON.stringify({ mutator })
    ).catch((error) => {
      if (error instanceof APIError && error.code === 409 && ifNotExists) {
        const apiErrorResponse = APIErrorResponse.fromJSON(error.body);
        if (apiErrorResponse.identical) {
          const ret = Object.assign(mutator, {
            id: apiErrorResponse.id,
          });
          return ret;
        }
      }
      throw error;
    });
  }

  async getMutator(mutatorId: string): Promise<Mutator> {
    return this.makeRequest<Mutator>(`/userstore/config/mutators/${mutatorId}`);
  }

  async listMutators(): Promise<Mutator[]> {
    return this.makeRequest<Mutator[]>(`/userstore/config/mutators`).then(
      (json) => {
        if ('data' in json) {
          return json.data as Mutator[];
        }
        return [];
      }
    );
  }

  async updateMutator(mutator: Mutator): Promise<Mutator> {
    return this.makeRequest<Mutator>(
      `/userstore/config/mutators/${mutator.id}`,
      'PUT',
      undefined,
      JSON.stringify({ mutator })
    );
  }

  async deleteMutator(mutatorId: string): Promise<void> {
    return this.makeRequest<void>(
      `/userstore/config/mutators/${mutatorId}`,
      'DELETE'
    );
  }

  // Purpose functions
  async createPurpose(purpose: Purpose, ifNotExists = false): Promise<Purpose> {
    return this.makeRequest<Purpose>(
      `/userstore/config/purposes`,
      'POST',
      undefined,
      JSON.stringify({ purpose })
    ).catch((error) => {
      if (error instanceof APIError && error.code === 409 && ifNotExists) {
        const apiErrorResponse = APIErrorResponse.fromJSON(error.body);
        if (apiErrorResponse.identical) {
          const ret = Object.assign(purpose, {
            id: apiErrorResponse.id,
          });
          return ret;
        }
      }
      throw error;
    });
  }

  async getPurpose(purposeId: string): Promise<Purpose> {
    return this.makeRequest<Purpose>(`/userstore/config/purposes/${purposeId}`);
  }

  async listPurposes(
    startingAfter: string = paginationStart,
    limit = defaultLimit
  ): Promise<[Purpose[], boolean]> {
    return this.makePaginatedRequest<Purpose>(
      `/userstore/config/purposes`,
      {},
      startingAfter,
      limit
    );
  }

  async updatePurpose(purpose: Purpose): Promise<Purpose> {
    return this.makeRequest<Purpose>(
      `/userstore/config/purposes/${purpose.id}`,
      'PUT',
      undefined,
      JSON.stringify({ purpose })
    );
  }

  async deletePurpose(purposeId: string): Promise<void> {
    return this.makeRequest<void>(
      `/userstore/config/purposes/${purposeId}`,
      'DELETE'
    );
  }

  // Transformer functions
  async createTransformer(
    transformer: Transformer,
    ifNotExists = false
  ): Promise<Transformer> {
    return this.makeRequest<Transformer>(
      `/tokenizer/policies/transformation`,
      'POST',
      undefined,
      JSON.stringify({ transformer })
    ).catch((error) => {
      if (error instanceof APIError && error.code === 409 && ifNotExists) {
        const apiErrorResponse = APIErrorResponse.fromJSON(error.body);
        if (apiErrorResponse.identical) {
          const ret = Object.assign(transformer, {
            id: apiErrorResponse.id,
          });
          return ret;
        }
      }
      throw error;
    });
  }

  async getTransformer(transformerId: string): Promise<Transformer> {
    return this.makeRequest<Transformer>(
      `/tokenizer/policies/transformation/${transformerId}`
    );
  }

  async listTransformers(
    startingAfter: string = paginationStart,
    limit = defaultLimit
  ): Promise<[Transformer[], boolean]> {
    return this.makePaginatedRequest<Transformer>(
      `/tokenizer/policies/transformation`,
      {},
      startingAfter,
      limit
    );
  }

  async updateTransformer(transformer: Transformer): Promise<Transformer> {
    return this.makeRequest<Transformer>(
      `/tokenizer/policies/transformation/${transformer.id}`,
      'PUT',
      undefined,
      JSON.stringify({ transformer })
    );
  }

  async deleteTransformer(transformerId: string): Promise<void> {
    return this.makeRequest<void>(
      `/tokenizer/policies/transformation/${transformerId}`,
      'DELETE'
    );
  }

  // User functions
  async createUser(
    profile: object = {},
    id: string | null = null,
    organization_id: string | null = null,
    region: string | null = null
  ): Promise<string> {
    const body: { [key: string]: string | object } = { profile };
    if (id) {
      body.id = id;
    }
    if (organization_id) {
      body.organization_id = organization_id;
    }
    if (region) {
      body.region = region;
    }
    return this.makeRequest<string>(
      `/authn/users`,
      'POST',
      undefined,
      JSON.stringify(body)
    );
  }

  async createUserWithPassword(
    username: string,
    password: string,
    profile: object = {},
    id: string | null = null,
    organization_id: string | null = null,
    region: string | null = null
  ): Promise<string> {
    const body: { [key: string]: string | object } = {
      username,
      password,
      authn_type: 'password',
    };
    if (profile) {
      body.profile = profile;
    }
    if (id) {
      body.id = id;
    }
    if (organization_id) {
      body.organization_id = organization_id;
    }
    if (region) {
      body.region = region;
    }
    return this.makeRequest<string>(
      `/authn/users`,
      'POST',
      undefined,
      JSON.stringify(body)
    );
  }

  async createUserWithMutator(
    mutator_id: string,
    context: object,
    row_data: object,
    id: string | null = null,
    organization_id: string | null = null,
    region: string | null = null
  ): Promise<string> {
    const body: { [key: string]: string | object } = {
      mutator_id,
      context,
      row_data,
    };
    if (id) {
      body.id = id;
    }
    if (organization_id) {
      body.organization_id = organization_id;
    }
    if (region) {
      body.region = region;
    }
    return this.makeRequest<string>(
      `/userstore/api/mutators`,
      'POST',
      undefined,
      JSON.stringify(body)
    );
  }

  // Admin-only -- should use accessors and mutators instead
  async listUsers(
    organizationID = '',
    startingAfter: string = paginationStart,
    limit = defaultLimit
  ): Promise<[User[], boolean]> {
    const params: { [key: string]: string } = organizationID
      ? { organization_id: organizationID }
      : {};
    return this.makePaginatedRequest<User>(
      '/authn/users',
      params,
      startingAfter,
      limit
    );
  }

  // Admin-only -- should use accessors and mutators instead
  async getUser(userId: string): Promise<User> {
    return this.makeRequest<User>(`/authn/users/${userId}`);
  }

  // Admin-only -- should use accessors and mutators instead
  async updateUser(userId: string, profile: object): Promise<User> {
    return this.makeRequest<User>(
      `/authn/users/${userId}`,
      'PUT',
      undefined,
      JSON.stringify({ profile })
    );
  }

  async deleteUser(userId: string): Promise<void> {
    return this.makeRequest<void>(`/authn/users/${userId}`, 'DELETE');
  }

  async executeAccessor(
    accessor_id: string,
    context: object,
    selector_values: string[]
  ): Promise<{ data: string[] }> {
    return this.makeRequest<{ data: string[] }>(
      `/userstore/api/accessors`,
      'POST',
      undefined,
      JSON.stringify({ accessor_id, context, selector_values })
    );
  }

  async executeMutator(
    mutator_id: string,
    context: object,
    selector_values: string[],
    row_data: object
  ): Promise<string> {
    return this.makeRequest<string>(
      `/userstore/api/mutators`,
      'POST',
      undefined,
      JSON.stringify({ mutator_id, context, selector_values, row_data })
    );
  }
}

export default Client;
