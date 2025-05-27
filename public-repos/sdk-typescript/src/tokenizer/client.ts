import {
  AccessPolicy,
  AccessPolicyTemplate,
  Transformer,
} from '../userstore/models';
import InspectTokenResponse from './models';
import ResourceID from '../userstore/models/resource_id';
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

  async createToken(
    data: string,
    transformerRID: ResourceID,
    accessPolicyRID: ResourceID
  ): Promise<void> {
    return this.makeRequest<void>(
      '/tokenizer/tokens',
      'POST',
      undefined,
      JSON.stringify({
        data,
        transformer_rid: transformerRID,
        access_policy_rid: accessPolicyRID,
      })
    );
  }

  async resolveTokens(
    tokens: string[],
    resolutionContext: Record<string, unknown>,
    purposes: ResourceID[]
  ): Promise<string[]> {
    return this.makeRequest<string[]>(
      '/tokenizer/tokens/actions/resolve',
      'POST',
      undefined,
      JSON.stringify({
        tokens,
        context: resolutionContext,
        purposes,
      })
    );
  }

  async lookupToken(
    token: string,
    transformerRID: ResourceID,
    accessPolicyRID: ResourceID
  ): Promise<string[]> {
    return this.makeRequest<string[]>(
      '/tokenizer/tokens/actions/lookup',
      'POST',
      undefined,
      JSON.stringify({
        data: token,
        transformer_rid: transformerRID,
        access_policy_rid: accessPolicyRID,
      })
    );
  }

  async lookupOrCreateTokens(
    data: string[],
    transformerRIDs: ResourceID[],
    accessPolicyRIDs: ResourceID[]
  ): Promise<string[]> {
    return this.makeRequest<string[]>(
      '/tokenizer/tokens/actions/lookuporcreate',
      'POST',
      undefined,
      JSON.stringify({
        data,
        transformer_rids: transformerRIDs,
        access_policy_rids: accessPolicyRIDs,
      })
    );
  }

  async deleteToken(token: string): Promise<void> {
    return this.makeRequest<void>(
      `/tokenizer/tokens/${token}`,
      'DELETE',
      undefined,
      undefined
    );
  }

  async inspectToken(token: string): Promise<InspectTokenResponse> {
    return this.makeRequest<InspectTokenResponse>(
      '/tokenizer/tokens/actions/inspect',
      'POST',
      undefined,
      JSON.stringify({
        token,
      })
    );
  }
}

export default Client;
