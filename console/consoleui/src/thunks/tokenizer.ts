import { APIError, JSONValue } from '@userclouds/sharedui';
import { v4 as uuidv4 } from 'uuid';
import { AppDispatch, RootState } from '../store';
import {
  deleteTenantAccessPolicy,
  fetchAccessPolicies as fetchAPs,
  fetchTenantAccessPolicy,
  fetchAccessPolicyByVersion,
  fetchAccessPolicyTemplates,
  createTenantAccessPolicyTemplate,
  deletePolicyTemplate,
  updateTenantAccessPolicy,
  fetchTenantTransformers,
  deleteTenantTransformer,
  fetchTenantUserPolicyPermissions,
  updateAccessPolicyTemplate,
  fetchAccessPolicyTemplateByVersion,
  fetchAccessPolicyTemplate,
  fetchTenantTransformer,
  testRunTenantTransformer,
  createTenantAccessPolicy,
  createTenantTransformer,
  updateTenantTransformer,
  testRunAccessPolicy,
  testRunAccessPolicyTemplate,
} from '../API/tokenizer';
import {
  createTenantPolicySecret,
  deleteTenantPolicySecret,
  listTenantPolicySecrets,
} from '../API/policysecret';
import {
  modifyAccessPolicy,
  createPolicyTemplateError,
  createPolicyTemplateRequest,
  createPolicyTemplateSuccess,
  updateAccessPolicyError,
  updateAccessPolicyRequest,
  updateAccessPolicySuccess,
  getPolicyTemplatesRequest,
  getPolicyTemplatesSuccess,
  getPolicyTemplatesError,
  getAccessPolicyRequest,
  getAccessPolicySuccess,
  getAccessPolicyError,
  getGlobalAccessorPolicyRequest,
  getGlobalAccessorPolicySuccess,
  getGlobalAccessorPolicyError,
  getAccessPoliciesRequest,
  getAccessPoliciesSuccess,
  getAccessPoliciesError,
  getAllAccessPoliciesRequest,
  getAllAccessPoliciesSuccess,
  getAllAccessPoliciesError,
  getTransformersRequest,
  getTransformersSuccess,
  getTransformersError,
  deleteSingleAccessPolicyRequest,
  deleteSingleAccessPolicySuccess,
  deleteSingleAccessPolicyError,
  deleteAccessPoliciesSingleSuccess,
  deleteAccessPoliciesSingleError,
  bulkDeleteAccessPoliciesRequest,
  bulkDeleteAccessPoliciesSuccess,
  bulkDeleteAccessPoliciesFailure,
  getUserPolicyPermissionsRequest,
  getUserPolicyPermissionsSuccess,
  getUserPolicyPermissionsError,
  deleteSingleTransformerRequest,
  deleteSingleTransformerSuccess,
  deleteSingleTransformerError,
  deleteTransformersSingleSuccess,
  deleteTransformersSingleError,
  bulkDeleteTransformersRequest,
  bulkDeleteTransformersSuccess,
  bulkDeleteTransformersFailure,
  deleteSinglePolicyTemplateRequest,
  deleteSinglePolicyTemplateSuccess,
  deleteSinglePolicyTemplateError,
  deletePolicyTemplatesSingleSuccess,
  deletePolicyTemplatesSingleError,
  bulkDeletePolicyTemplatesRequest,
  bulkDeletePolicyTemplatesSuccess,
  bulkDeletePolicyTemplatesFailure,
  updatePolicyTemplateRequest,
  getPolicyTemplateRequest,
  getPolicyTemplateSuccess,
  getPolicyTemplateError,
  updatePolicyTemplateSuccess,
  updatePolicyTemplateError,
  getTransformerRequest,
  getTransformerSuccess,
  getTransformerError,
  getGlobalMutatorPolicyRequest,
  getGlobalMutatorPolicySuccess,
  getGlobalMutatorPolicyError,
  testPolicyRequest,
  testPolicyError,
  testPolicySuccess,
  testTransformerRequest,
  testTransformerSuccess,
  testTransformerError,
  getTransformerPermissionsRequest,
  getTransformerPermissionsSuccess,
  getTransformerPermissionsError,
  getAccessPolicyPermissionsRequest,
  getAccessPolicyPermissionsSuccess,
  getAccessPolicyPermissionsError,
  createAccessPolicyRequest,
  createAccessPolicySuccess,
  createAccessPolicyError,
  createTransformerRequest,
  createTransformerSuccess,
  createTransformerError,
  updateTransformerRequest,
  updateTransformerSuccess,
  updateTransformerError,
  getTokenAccessPolicyRequest,
  getTokenAccessPolicySuccess,
  getTokenAccessPolicyError,
  fetchTenantPolicySecretsRequest,
  fetchTenantPolicySecretsSuccess,
  fetchTenantPolicySecretsError,
  savePolicySecretRequest,
  savePolicySecretError,
} from '../actions/tokenizer';
import PaginatedResult from '../models/PaginatedResult';
import AccessPolicy, {
  AccessPolicyTemplate,
  ACCESS_POLICIES_PREFIX,
  ACCESS_POLICY_TEMPLATE_PREFIX,
  GLOBAL_ACCESSOR_POLICY_ID,
  GLOBAL_MUTATOR_POLICY_ID,
} from '../models/AccessPolicy';
import PermissionsOnObject from '../models/authz/Permissions';
import { PolicySecret } from '../models/PolicySecret';
import Transformer, {
  TRANSFORMERS_PREFIX,
  TransformerTestResult,
} from '../models/Transformer';
import AccessPolicyContext, {
  validateAccessPolicyContext,
} from '../models/AccessPolicyContext';
import { getParamsAsObject } from '../controls/PaginationHelper';
import { postAlertToast, postSuccessToast } from './notifications';
import { redirect } from '../routing';
import { PAGINATION_API_VERSION } from '../API';

const ACCESS_POLICIES_PAGE_SIZE = '50';
const POLICY_TEMPLATES_PAGE_SIZE = '50';
const TRANSFORMERS_PAGE_SIZE = '50';
const SECRETS_PAGE_SIZE = '50';

export const fetchTenantAccessPolicyByVersion =
  (tenantID: string, policyID: string, version: string) =>
  (dispatch: AppDispatch) => {
    dispatch(getAccessPolicyRequest());
    fetchAccessPolicyByVersion(
      tenantID,
      policyID,
      parseInt(version as string, 10)
    ).then(
      (policy: AccessPolicy) => {
        dispatch(getAccessPolicySuccess(policy));
      },
      (error: APIError) => {
        dispatch(getAccessPolicyError(error));
      }
    );
  };

export const fetchAccessPolicy =
  (tenantID: string, policyID: string) => (dispatch: AppDispatch) => {
    dispatch(getAccessPolicyRequest());
    fetchTenantAccessPolicy(tenantID, policyID).then(
      (policy: AccessPolicy) => {
        dispatch(getAccessPolicySuccess(policy));
      },
      (error: APIError) => {
        dispatch(getAccessPolicyError(error));
      }
    );
  };

export const fetchGlobalAccessorPolicy =
  (tenantID: string) => (dispatch: AppDispatch) => {
    dispatch(getGlobalAccessorPolicyRequest());
    fetchTenantAccessPolicy(tenantID, GLOBAL_ACCESSOR_POLICY_ID).then(
      (policy: AccessPolicy) => {
        dispatch(getGlobalAccessorPolicySuccess(policy));
      },
      (error: APIError) => {
        dispatch(getGlobalAccessorPolicyError(error));
      }
    );
  };

export const fetchGlobalMutatorPolicy =
  (tenantID: string) => (dispatch: AppDispatch) => {
    dispatch(getGlobalMutatorPolicyRequest());
    fetchTenantAccessPolicy(tenantID, GLOBAL_MUTATOR_POLICY_ID).then(
      (policy: AccessPolicy) => {
        dispatch(getGlobalMutatorPolicySuccess(policy));
      },
      (error: APIError) => {
        dispatch(getGlobalMutatorPolicyError(error));
      }
    );
  };

export const fetchTokenAccessPolicy =
  (tenantID: string, policyID: string) => (dispatch: AppDispatch) => {
    dispatch(getTokenAccessPolicyRequest());
    fetchTenantAccessPolicy(tenantID, policyID).then(
      (policy: AccessPolicy) => {
        dispatch(getTokenAccessPolicySuccess(policy));
      },
      (error: APIError) => {
        dispatch(getTokenAccessPolicyError(error));
      }
    );
  };

export const fetchAccessPolicies =
  (
    tenantID: string,
    queryParams: URLSearchParams,
    includeAutogenerated: boolean,
    versioned?: boolean,
    limit?: number
  ) =>
  (dispatch: AppDispatch) => {
    const paramsAsObject = getParamsAsObject(
      ACCESS_POLICIES_PREFIX,
      queryParams
    );

    if (limit) {
      paramsAsObject.limit = String(limit);
    } else if (!paramsAsObject.limit) {
      paramsAsObject.limit = ACCESS_POLICIES_PAGE_SIZE;
    }
    if (versioned) {
      paramsAsObject.versioned = 'true';
    }
    if (!paramsAsObject.sort_order) {
      paramsAsObject.sort_order = 'ascending';
    }
    if (!paramsAsObject.sort_key) {
      paramsAsObject.sort_key = 'name,id';
    }
    if (!paramsAsObject.version) {
      paramsAsObject.version = PAGINATION_API_VERSION;
    }
    dispatch(getAccessPoliciesRequest());
    fetchAPs(tenantID, paramsAsObject, includeAutogenerated).then(
      (policies: PaginatedResult<AccessPolicy>) => {
        dispatch(getAccessPoliciesSuccess(policies));
      },
      (error: APIError) => {
        dispatch(getAccessPoliciesError(error));
      }
    );
  };

export const fetchAllAccessPolicies =
  (tenantID: string) => async (dispatch: AppDispatch) => {
    dispatch(getAllAccessPoliciesRequest());
    const queryParams = { limit: '1500' };
    fetchAPs(tenantID, queryParams, false).then(
      (policies: PaginatedResult<AccessPolicy>) => {
        dispatch(getAllAccessPoliciesSuccess(policies));
      },
      (error: APIError) => {
        dispatch(getAllAccessPoliciesError(error));
      }
    );
  };

export const fetchPolicyTemplates =
  (
    tenantID: string,
    queryParams: URLSearchParams,
    versioned?: boolean,
    limit?: number
  ) =>
  (dispatch: AppDispatch) => {
    const paramsAsObject = getParamsAsObject(
      ACCESS_POLICY_TEMPLATE_PREFIX,
      queryParams
    );

    if (limit) {
      paramsAsObject.limit = String(limit);
    } else if (!paramsAsObject.limit) {
      paramsAsObject.limit = POLICY_TEMPLATES_PAGE_SIZE;
    }
    if (versioned) {
      paramsAsObject.versioned = 'true';
    }
    if (!paramsAsObject.sort_order) {
      paramsAsObject.sort_order = 'ascending';
    }
    if (!paramsAsObject.sort_key) {
      paramsAsObject.sort_key = 'name,id';
    }
    if (!paramsAsObject.version) {
      paramsAsObject.version = PAGINATION_API_VERSION;
    }
    dispatch(getPolicyTemplatesRequest());
    fetchAccessPolicyTemplates(tenantID, paramsAsObject).then(
      (templates: PaginatedResult<AccessPolicyTemplate>) => {
        dispatch(getPolicyTemplatesSuccess(templates));
      },
      (error: APIError) => {
        dispatch(getPolicyTemplatesError(error));
      }
    );
  };

export const fetchTransformer =
  (
    tenantID: string,
    transformerID: string,
    version: number | undefined = undefined
  ) =>
  (dispatch: AppDispatch) => {
    dispatch(getTransformerRequest());
    fetchTenantTransformer(tenantID, transformerID, version).then(
      (transformer: Transformer) => {
        dispatch(getTransformerSuccess(transformer));
      },
      (error: APIError) => {
        dispatch(getTransformerError(error));
      }
    );
  };

export const fetchTransformers =
  (tenantID: string, queryParams: URLSearchParams, limit?: number) =>
  (dispatch: AppDispatch) => {
    const paramsAsObject = getParamsAsObject(TRANSFORMERS_PREFIX, queryParams);

    if (limit) {
      paramsAsObject.limit = String(limit);
    } else if (!paramsAsObject.limit) {
      paramsAsObject.limit = TRANSFORMERS_PAGE_SIZE;
    }
    if (!paramsAsObject.sort_order) {
      paramsAsObject.sort_order = 'ascending';
    }
    if (!paramsAsObject.sort_key) {
      paramsAsObject.sort_key = 'name,id';
    }
    if (!paramsAsObject.version) {
      paramsAsObject.version = PAGINATION_API_VERSION;
    }
    dispatch(getTransformersRequest());
    fetchTenantTransformers(tenantID, paramsAsObject).then(
      (transformers: PaginatedResult<Transformer>) => {
        dispatch(getTransformersSuccess(transformers));
      },
      (error: APIError) => {
        dispatch(getTransformersError(error));
      }
    );
  };

export const createTransformer =
  (companyID: string, tenantID: string, policy: Transformer) =>
  async (dispatch: AppDispatch) => {
    dispatch(createTransformerRequest());
    const payload: Transformer = { ...policy, id: uuidv4() };
    createTenantTransformer(tenantID, payload).then(
      (transformer: Transformer) => {
        dispatch(createTransformerSuccess(transformer));
        // we're creating a new one
        dispatch(postSuccessToast('Successfully created transformer'));
        redirect(
          `/transformers/${transformer.id}/${transformer.version}?company_id=${companyID}&tenant_id=${tenantID}`
        );
      },
      (error: APIError) => {
        dispatch(createTransformerError(error));
      }
    );
  };

export const updateTransformer =
  (companyID: string, tenantID: string, policy: Transformer) =>
  async (dispatch: AppDispatch) => {
    dispatch(updateTransformerRequest());
    updateTenantTransformer(tenantID, policy).then(
      (transformer: Transformer) => {
        dispatch(updateTransformerSuccess(transformer));
        dispatch(postSuccessToast('Successfully updated transformer'));
        redirect(
          `/transformers/${transformer.id}/${transformer.version}?company_id=${companyID}&tenant_id=${tenantID}&updated=true`
        );
      },
      (error: APIError) => {
        dispatch(updateTransformerError(error));
      }
    );
  };

export const deleteSingleTransformer =
  (tenantID: string, policy: Transformer, query?: URLSearchParams) =>
  (dispatch: AppDispatch): Promise<void> => {
    dispatch(deleteSingleTransformerRequest());
    return new Promise((resolve) => {
      deleteTenantTransformer(tenantID, policy.id).then(
        () => {
          dispatch(deleteSingleTransformerSuccess());
          if (query) {
            dispatch(fetchTransformers(tenantID, query));
          } else {
            dispatch(fetchTransformers(tenantID, new URLSearchParams()));
          }
          dispatch(postSuccessToast('Successfully deleted transformer'));
          resolve();
        },
        (error: APIError) => {
          dispatch(deleteSingleTransformerError(error));
          dispatch(postAlertToast('Error deleting transformer: ' + error));
        }
      );
    });
  };

export const deleteTransformerBulk =
  (tenantID: string, transformerID: string) =>
  (dispatch: AppDispatch): Promise<boolean> => {
    return new Promise((resolve) => {
      return deleteTenantTransformer(tenantID, transformerID).then(
        () => {
          dispatch(deleteTransformersSingleSuccess(transformerID));
          resolve(true);
        },
        (error: APIError) => {
          dispatch(deleteTransformersSingleError(transformerID, error));
          resolve(false);
        }
      );
    });
  };

export const bulkDeleteTransformers =
  (selectedTenantID: string, transformerIDs: string[]) =>
  (dispatch: AppDispatch, getState: () => RootState) => {
    const { query } = getState();

    dispatch(bulkDeleteTransformersRequest());
    const reqs: Array<Promise<boolean>> = [];
    transformerIDs.forEach((id) => {
      reqs.push(dispatch(deleteTransformerBulk(selectedTenantID, id)));
    });
    Promise.all(reqs).then((values: boolean[]) => {
      if (values.every((val) => val === true)) {
        dispatch(fetchTransformers(selectedTenantID, query));
        dispatch(bulkDeleteTransformersSuccess());
      } else {
        if (!values.every((val) => val === false)) {
          // if all the reqs failed, there's no need to re-fetch
          dispatch(fetchTransformers(selectedTenantID, query));
        }
        dispatch(bulkDeleteTransformersFailure());
      }
    });
  };

export const runTransformerTest =
  (transformer: Transformer, testData: string, selectedTenantID: string) =>
  (dispatch: AppDispatch) => {
    dispatch(testTransformerRequest());
    // TODO: this is janky
    transformer.parameters = transformer.parameters.replace(/^\/\/.*$/gm, '');
    if (transformer.parameters !== '') {
      try {
        JSON.parse(transformer.parameters);
      } catch {
        dispatch(
          testTransformerError(
            new APIError('Parameters must be valid JSON', 400, undefined)
          )
        );
        return;
      }
    }
    testRunTenantTransformer(selectedTenantID, transformer, testData).then(
      (response: TransformerTestResult) => {
        dispatch(testTransformerSuccess(response));
      },
      (error: APIError) => {
        dispatch(testTransformerError(error));
      }
    );
  };

export const deleteSingleAccessPolicy =
  (tenantID: string, policy: AccessPolicy, queryParams?: URLSearchParams) =>
  (dispatch: AppDispatch, getState: () => RootState): Promise<void> => {
    const { accessPoliciesIncludeAutogenerated } = getState();

    dispatch(deleteSingleAccessPolicyRequest());
    return new Promise((resolve, reject) => {
      deleteTenantAccessPolicy(tenantID, policy.id, policy.version).then(
        () => {
          dispatch(deleteSingleAccessPolicySuccess());
          if (queryParams) {
            dispatch(
              fetchAccessPolicies(
                tenantID,
                queryParams,
                accessPoliciesIncludeAutogenerated
              )
            );
          } else {
            dispatch(
              fetchAccessPolicies(
                tenantID,
                new URLSearchParams(),
                accessPoliciesIncludeAutogenerated
              )
            );
          }
          dispatch(postSuccessToast('Successfully deleted policy'));
          resolve();
        },
        (error: APIError) => {
          dispatch(deleteSingleAccessPolicyError(error));
          dispatch(postAlertToast('Error deleting policy: ' + error));
          reject(error);
        }
      );
    });
  };

export const deleteAccessPolicyBulk =
  (tenantID: string, accessPolicyID: string) =>
  (dispatch: AppDispatch, getState: () => RootState): Promise<boolean> => {
    const { accessPolicies } = getState();
    const matchingPolicy = accessPolicies?.data.find(
      (policy: AccessPolicy) => policy.id === accessPolicyID
    );
    return new Promise((resolve, reject) => {
      if (!matchingPolicy) {
        return reject();
      }
      return deleteTenantAccessPolicy(
        tenantID,
        matchingPolicy.id,
        matchingPolicy.version
      ).then(
        () => {
          dispatch(deleteAccessPoliciesSingleSuccess(accessPolicyID));
          resolve(true);
        },
        (error: APIError) => {
          dispatch(deleteAccessPoliciesSingleError(accessPolicyID, error));
          resolve(false);
        }
      );
    });
  };

export const bulkDeleteAccessPolicies =
  (selectedTenantID: string, accessPolicyIDs: string[]) =>
  (dispatch: AppDispatch, getState: () => RootState) => {
    const { accessPoliciesIncludeAutogenerated, query } = getState();

    dispatch(bulkDeleteAccessPoliciesRequest());
    const reqs: Array<Promise<boolean>> = [];
    accessPolicyIDs.forEach((id) => {
      reqs.push(dispatch(deleteAccessPolicyBulk(selectedTenantID, id)));
    });
    Promise.all(reqs).then((values: boolean[]) => {
      if (values.every((val) => val === true)) {
        dispatch(
          fetchAccessPolicies(
            selectedTenantID,
            query,
            accessPoliciesIncludeAutogenerated
          )
        );
        dispatch(bulkDeleteAccessPoliciesSuccess());
      } else {
        if (!values.every((val) => val === false)) {
          dispatch(
            fetchAccessPolicies(
              selectedTenantID,
              query,
              accessPoliciesIncludeAutogenerated
            )
          );
        }
        dispatch(bulkDeleteAccessPoliciesFailure());
      }
    });
  };

export const fetchUserPolicyPermissions =
  (tenantID: string) => async (dispatch: AppDispatch) => {
    dispatch(getUserPolicyPermissionsRequest());
    fetchTenantUserPolicyPermissions(tenantID).then(
      (permissions: PermissionsOnObject) => {
        dispatch(getUserPolicyPermissionsSuccess(permissions));
      },
      (error: APIError) => {
        dispatch(getUserPolicyPermissionsError(error));
      }
    );
  };

export const fetchUserPermissionsForAccessPolicy =
  (tenantID: string, policyID: string) => async (dispatch: AppDispatch) => {
    dispatch(getAccessPolicyPermissionsRequest());
    fetchTenantUserPolicyPermissions(tenantID, policyID).then(
      (permissions: PermissionsOnObject) => {
        dispatch(getAccessPolicyPermissionsSuccess(permissions));
      },
      (error: APIError) => {
        dispatch(getAccessPolicyPermissionsError(error));
      }
    );
  };

export const fetchUserPermissionsForTransformer =
  (tenantID: string, policyID: string) => async (dispatch: AppDispatch) => {
    dispatch(getTransformerPermissionsRequest());
    fetchTenantUserPolicyPermissions(tenantID, policyID).then(
      (permissions: PermissionsOnObject) => {
        dispatch(getTransformerPermissionsSuccess(permissions));
      },
      (error: APIError) => {
        dispatch(getTransformerPermissionsError(error));
      }
    );
  };

export const createAccessPolicy =
  (tenantID: string, companyID: string, policy: AccessPolicy) =>
  async (dispatch: AppDispatch) => {
    dispatch(createAccessPolicyRequest());
    if (!policy.id) {
      policy.id = uuidv4();
    }
    try {
      policy.required_context = JSON.parse(
        policy.required_context_stringified!
      );
    } catch {
      dispatch(
        updateAccessPolicyError(
          new APIError('Required context must be valid JSON', 400, undefined)
        )
      );
      return;
    }
    createTenantAccessPolicy(tenantID, policy).then(
      (newPolicy: AccessPolicy) => {
        dispatch(createAccessPolicySuccess(newPolicy));
        dispatch(postSuccessToast('Successfully created access policy'));
        redirect(
          `/accesspolicies/${newPolicy.id}/0?company_id=${companyID}&tenant_id=${tenantID}`
        );
      },
      (error) => {
        dispatch(createAccessPolicyError(error));
      }
    );
  };

export const updateAccessPolicy =
  (tenantID: string, companyID: string, policy: AccessPolicy) =>
  async (dispatch: AppDispatch) => {
    dispatch(updateAccessPolicyRequest());
    try {
      policy.required_context = JSON.parse(
        policy.required_context_stringified!
      );
    } catch {
      dispatch(
        updateAccessPolicyError(
          new APIError('Required context must be valid JSON', 400, undefined)
        )
      );
      return;
    }
    updateTenantAccessPolicy(tenantID, policy).then(
      (updatedPolicy: AccessPolicy) => {
        dispatch(updateAccessPolicySuccess(updatedPolicy));
        dispatch(postSuccessToast('Successfully updated access policy'));
        redirect(
          `/accesspolicies/${updatedPolicy.id}/${updatedPolicy.version}?company_id=${companyID}&tenant_id=${tenantID}&updated=true`
        );
      },
      (error: APIError) => {
        dispatch(updateAccessPolicyError(error));
      }
    );
  };

export const runAccessPolicyTest =
  (policy: AccessPolicy, testContext: string, selectedTenantID: string) =>
  (dispatch: AppDispatch) => {
    dispatch(testPolicyRequest());

    // TODO: this is janky
    const c = testContext.replace(/^\/\/.*$/gm, '');

    let contextJSON: JSONValue;
    try {
      contextJSON = JSON.parse(c);
    } catch {
      dispatch(
        testPolicyError(
          new APIError('Parameters must be valid JSON', 400, undefined)
        )
      );
      return;
    }

    const validationError = validateAccessPolicyContext(contextJSON);
    if (validationError) {
      dispatch(testPolicyError(new APIError(validationError, 400, undefined)));
      return;
    }

    testRunAccessPolicy(
      selectedTenantID,
      policy,
      contextJSON as AccessPolicyContext
    ).then((response) => {
      if (response instanceof APIError) {
        dispatch(testPolicyError(response));
      } else {
        dispatch(testPolicySuccess(response));
      }
    });
  };

export const runAccessPolicyTemplateTest =
  (
    template: AccessPolicyTemplate,
    testContext: string,
    testParams: string,
    selectedTenantID: string
  ) =>
  (dispatch: AppDispatch) => {
    dispatch(testPolicyRequest());

    // TODO: this is janky
    const c = testContext.replace(/^\/\/.*$/gm, '');

    let contextJSON: JSONValue;
    try {
      contextJSON = JSON.parse(c);
    } catch {
      dispatch(
        testPolicyError(
          new APIError('Parameters must be valid JSON', 400, undefined)
        )
      );
      return;
    }

    const validationError = validateAccessPolicyContext(contextJSON);
    if (validationError) {
      dispatch(testPolicyError(new APIError(validationError, 400, undefined)));
      return;
    }

    testRunAccessPolicyTemplate(
      selectedTenantID,
      template,
      contextJSON as AccessPolicyContext,
      testParams
    ).then((response) => {
      if (response instanceof APIError) {
        dispatch(testPolicyError(response));
      } else {
        dispatch(testPolicySuccess(response));
      }
    });
  };

export const createAccessPolicyTemplate =
  (queryString: string) =>
  (tenantID: string, template: AccessPolicyTemplate) =>
  (dispatch: AppDispatch) => {
    dispatch(createPolicyTemplateRequest());
    createTenantAccessPolicyTemplate(tenantID, template).then(
      (newTemplate: AccessPolicyTemplate) => {
        dispatch(createPolicyTemplateSuccess(newTemplate));
        dispatch(
          postSuccessToast(
            `Successfully created policy template '${newTemplate.name}'`
          )
        );
        redirect(
          `/policytemplates/${newTemplate.id}/${newTemplate.version}${queryString}`
        );
      },
      (error: APIError) => {
        dispatch(createPolicyTemplateError(error));
      }
    );
  };

export const savePolicyTemplate =
  (queryString: string) =>
  (tenantID: string, template: AccessPolicyTemplate) =>
  (dispatch: AppDispatch) => {
    dispatch(updatePolicyTemplateRequest());
    updateAccessPolicyTemplate(tenantID, template).then(
      (updatedTemplate: AccessPolicyTemplate) => {
        dispatch(updatePolicyTemplateSuccess(updatedTemplate));
        redirect(
          `/policytemplates/${updatedTemplate.id}/${updatedTemplate.version}${queryString}&updated=true`
        );
      },
      (error: APIError) => {
        dispatch(updatePolicyTemplateError(error));
      }
    );
  };

export const fetchPolicyTemplate =
  (tenantID: string, templateID: string, version: string) =>
  (dispatch: AppDispatch) => {
    dispatch(getPolicyTemplateRequest());
    const fn = version
      ? fetchAccessPolicyTemplateByVersion(tenantID, templateID, version)
      : fetchAccessPolicyTemplate(tenantID, templateID);
    fn.then(
      (newTemplate: AccessPolicyTemplate) => {
        dispatch(getPolicyTemplateSuccess(newTemplate));
      },
      (error: APIError) => {
        dispatch(getPolicyTemplateError(error));
      }
    );
  };

export const createAccessPolicyTemplateForAccessPolicy =
  (closeHandler: Function) =>
  (
    tenantID: string,
    template: AccessPolicyTemplate,
    searchParams?: URLSearchParams,
    accessPolicy?: AccessPolicy
  ) =>
  (dispatch: AppDispatch) => {
    dispatch(createPolicyTemplateRequest());
    createTenantAccessPolicyTemplate(tenantID, template).then(
      (newTemplate: AccessPolicyTemplate) => {
        if (accessPolicy) {
          dispatch(
            modifyAccessPolicy({
              components: [
                ...accessPolicy.components,
                { template: newTemplate, template_parameters: '{}' },
              ],
            })
          );
        }
        dispatch(createPolicyTemplateSuccess(newTemplate));
        dispatch(postSuccessToast('Successfully created policy template'));
        if (searchParams) {
          dispatch(fetchPolicyTemplates(tenantID, searchParams));
        }
        closeHandler();
      },
      (error: APIError) => {
        dispatch(createPolicyTemplateError(error));
        dispatch(
          modifyAccessPolicy({
            id: uuidv4(),
          })
        );
      }
    );
  };

export const deleteSinglePolicyTemplate =
  (tenantID: string, template: AccessPolicyTemplate, query?: URLSearchParams) =>
  (dispatch: AppDispatch) => {
    dispatch(deleteSinglePolicyTemplateRequest(template));
    deletePolicyTemplate(tenantID, template.id, template.version).then(
      () => {
        dispatch(deleteSinglePolicyTemplateSuccess(template));
        dispatch(postSuccessToast('Successfully deleted template'));
        if (query) {
          dispatch(fetchPolicyTemplates(tenantID, query));
        } else {
          dispatch(fetchPolicyTemplates(tenantID, new URLSearchParams()));
        }
      },
      (error) => {
        dispatch(deleteSinglePolicyTemplateError(error));
        dispatch(postAlertToast('Error deleting template: ' + error));
      }
    );
  };

export const deletePolicyTemplateBulk =
  (tenantID: string, policyTemplateID: string) =>
  (dispatch: AppDispatch, getState: () => RootState): Promise<boolean> => {
    const { policyTemplates } = getState();
    const matchingTemplate = policyTemplates?.data.find(
      (template: AccessPolicyTemplate) => template.id === policyTemplateID
    );
    return new Promise((resolve, reject) => {
      if (!matchingTemplate) {
        return reject();
      }
      return deletePolicyTemplate(
        tenantID,
        policyTemplateID,
        matchingTemplate.version
      ).then(
        () => {
          dispatch(deletePolicyTemplatesSingleSuccess(policyTemplateID));
          resolve(true);
        },
        (error: APIError) => {
          dispatch(deletePolicyTemplatesSingleError(policyTemplateID, error));
          resolve(false);
        }
      );
    });
  };

export const bulkDeletePolicyTemplates =
  (selectedTenantID: string, policyTemplateIDs: string[]) =>
  (dispatch: AppDispatch, getState: () => RootState) => {
    const { query } = getState();

    dispatch(bulkDeletePolicyTemplatesRequest());
    const reqs: Array<Promise<boolean>> = [];
    policyTemplateIDs.forEach((id) => {
      reqs.push(dispatch(deletePolicyTemplateBulk(selectedTenantID, id)));
    });
    Promise.all(reqs).then((values: boolean[]) => {
      if (values.every((val) => val === true)) {
        dispatch(fetchPolicyTemplates(selectedTenantID, query));
        dispatch(bulkDeletePolicyTemplatesSuccess());
      } else {
        if (!values.every((val) => val === false)) {
          // if all the reqs failed, there's no need to re-fetch
          dispatch(fetchPolicyTemplates(selectedTenantID, query));
        }
        dispatch(bulkDeletePolicyTemplatesFailure());
      }
    });
  };

export const fetchPolicySecrets =
  (tenantID: string, params: URLSearchParams) => (dispatch: AppDispatch) => {
    const paramsAsObject = [
      'secret_starting_after',
      'secret_ending_before',
      'secret_limit',
      'secret_filter',
    ].reduce((acc: Record<string, string>, paramName: string) => {
      if (params.has(paramName)) {
        acc[paramName.substring(7)] = params.get(paramName) as string;
      }
      return acc;
    }, {});
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = SECRETS_PAGE_SIZE;
    }
    if (!paramsAsObject.sort_order) {
      paramsAsObject.sort_order = 'ascending';
    }
    if (!paramsAsObject.sort_key) {
      paramsAsObject.sort_key = 'name,id';
    }
    if (!paramsAsObject.version) {
      paramsAsObject.version = PAGINATION_API_VERSION;
    }
    dispatch(fetchTenantPolicySecretsRequest());
    return listTenantPolicySecrets(tenantID, paramsAsObject).then(
      (policySecrets: PaginatedResult<PolicySecret>) => {
        dispatch(fetchTenantPolicySecretsSuccess(policySecrets));
      },
      (error: APIError) => {
        dispatch(fetchTenantPolicySecretsError(error));
      }
    );
  };

export const deletePolicySecret =
  (tenantID: string, policySecretID: string) =>
  async (dispatch: AppDispatch) => {
    deleteTenantPolicySecret(tenantID, policySecretID).then(
      () => {
        dispatch(postSuccessToast('Successfully deleted secret'));
        dispatch(fetchPolicySecrets(tenantID, new URLSearchParams()));
      },
      (error: APIError) => {
        dispatch(postAlertToast('Unable to delete secret: ' + error));
      }
    );
  };

export const bulkDeletePolicySecrets =
  (tenantID: string, policySecretIDs: string[]) =>
  async (dispatch: AppDispatch, getState: () => RootState) => {
    const { query } = getState();
    return Promise.all(
      policySecretIDs.map((id: string) =>
        deleteTenantPolicySecret(tenantID, id)
      )
    ).then(
      () => {
        dispatch(postSuccessToast('Successfully deleted secrets'));
        dispatch(fetchPolicySecrets(tenantID, query));
      },
      (error: APIError) => {
        dispatch(postAlertToast('Unable to delete secrets: ' + error));
        dispatch(fetchPolicySecrets(tenantID, query));
      }
    );
  };

export const saveNewPolicySecret =
  (tenantID: string, secret: PolicySecret, onSuccess: Function) =>
  async (dispatch: AppDispatch) => {
    dispatch(savePolicySecretRequest());
    createTenantPolicySecret(tenantID, secret).then(
      () => {
        dispatch(postSuccessToast('Successfully created secret'));
        dispatch(fetchPolicySecrets(tenantID, new URLSearchParams()));
        onSuccess();
      },
      (error: APIError) => {
        dispatch(savePolicySecretError(error));
      }
    );
  };
