import { APIError } from '@userclouds/sharedui';
import { AppDispatch } from '../store';
import PermissionsOnObject from '../models/authz/Permissions';
import { Filter } from '../models/authz/SearchFilters';
import PaginatedResult from '../models/PaginatedResult';
import AccessPolicy, {
  AccessPolicyComponent,
  AccessPolicyTemplate,
  AccessPolicyTestResult,
} from '../models/AccessPolicy';
import Transformer, { TransformerTestResult } from '../models/Transformer';
import { PolicySecret } from '../models/PolicySecret';

export const MODIFY_POLICY_TEMPLATE = 'MODIFY_POLICY_TEMPLATE';

export const CREATE_POLICY_TEMPLATE_REQUEST = 'CREATE_POLICY_TEMPLATE_REQUEST';
export const CREATE_POLICY_TEMPLATE_SUCCESS = 'CREATE_POLICY_TEMPLATE_SUCCESS';
export const CREATE_POLICY_TEMPLATE_ERROR = 'CREATE_POLICY_TEMPLATE_ERROR';

export const UPDATE_POLICY_TEMPLATE_REQUEST = 'UPDATE_POLICY_TEMPLATE_REQUEST';
export const UPDATE_POLICY_TEMPLATE_SUCCESS = 'UPDATE_POLICY_TEMPLATE_SUCCESS';
export const UPDATE_POLICY_TEMPLATE_ERROR = 'UPDATE_POLICY_TEMPLATE_ERROR';

export const TOGGLE_POLICY_TEMPLATES_EDIT_MODE =
  'TOGGLE_POLICY_TEMPLATES_EDIT_MODE';
export const GET_POLICY_TEMPLATES_REQUEST = 'GET_POLICY_TEMPLATES_REQUEST';
export const GET_POLICY_TEMPLATES_SUCCESS = 'GET_POLICY_TEMPLATES_SUCCESS';
export const GET_POLICY_TEMPLATES_ERROR = 'GET_POLICY_TEMPLATES_ERROR';

export const GET_POLICY_TEMPLATE_REQUEST = 'GET_POLICY_TEMPLATE_REQUEST';
export const GET_POLICY_TEMPLATE_SUCCESS = 'GET_POLICY_TEMPLATE_SUCCESS';
export const GET_POLICY_TEMPLATE_ERROR = 'GET_POLICY_TEMPLATE_ERROR';

export const TOGGLE_POLICY_TEMPLATE_FOR_DELETE =
  'TOGGLE_POLICY_TEMPLATE_FOR_DELETE';
export const DELETE_SINGLE_POLICY_TEMPLATE_REQUEST =
  'DELETE_SINGLE_POLICY_TEMPLATE_REQUEST';
export const DELETE_SINGLE_POLICY_TEMPLATE_SUCCESS =
  'DELETE_SINGLE_POLICY_TEMPLATE_SUCCESS';
export const DELETE_SINGLE_POLICY_TEMPLATE_ERROR =
  'DELETE_SINGLE_POLICY_TEMPLATE_ERROR';
export const DELETE_POLICY_TEMPLATES_SINGLE_SUCCESS =
  'DELETE_POLICY_TEMPLATES_SINGLE_SUCCESS';
export const DELETE_POLICY_TEMPLATES_SINGLE_ERROR =
  'DELETE_POLICY_TEMPLATES_SINGLE_ERROR';
export const BULK_DELETE_POLICY_TEMPLATES_REQUEST =
  'BULK_DELETE_POLICY_TEMPLATES_REQUEST';
export const BULK_DELETE_POLICY_TEMPLATES_SUCCESS =
  'BULK_DELETE_POLICY_TEMPLATES_SUCCESS';
export const BULK_DELETE_POLICY_TEMPLATES_FAILURE =
  'BULK_DELETE_POLICY_TEMPLATES_FAILURE';

export const CHANGE_ACCESS_POLICY_TEMPLATE_SEARCH_FILTER =
  'CHANGE_ACCESS_POLICY_TEMPLATE_SEARCH_FILTER';

export const MODIFY_ACCESS_POLICY = 'MODIFY_ACCESS_POLICY';
export const MODIFY_ACCESS_POLICY_THRESHOLDS =
  'MODIFY_ACCESS_POLICY_THRESHOLDS';
export const CHANGE_ACCESS_POLICY_TEST_CONTEXT =
  'CHANGE_ACCESS_POLICY_TEST_CONTEXT';
export const CHANGE_ACCESS_POLICY_TEST_PARAMS =
  'CHANGE_ACCESS_POLICY_TEST_PARAMS';

export const MODIFY_TOKEN_ACCESS_POLICY = 'MODIFY_TOKEN_ACCESS_POLICY';

export const CREATE_TOKEN_POLICY_REQUEST = 'CREATE_TOKEN_POLICY_REQUEST';
export const CREATE_TOKEN_POLICY_SUCCESS = 'CREATE_TOKEN_POLICY_SUCCESS';
export const CREATE_TOKEN_POLICY_ERROR = 'CREATE_TOKEN_POLICY_ERROR';

export const CHANGE_POLICY_COMPONENTS = 'CHANGE_POLICY_COMPONENTS';

export const GET_USER_POLICY_PERMISSIONS_REQUEST =
  'GET_USER_POLICY_PERMISSIONS_REQUEST';
export const GET_USER_POLICY_PERMISSIONS_SUCCESS =
  'GET_USER_POLICY_PERMISSIONS_SUCCESS';
export const GET_USER_POLICY_PERMISSIONS_ERROR =
  'GET_USER_POLICY_PERMISSIONS_ERROR';

export const GET_ACCESS_POLICY_PERMISSIONS_REQUEST =
  'GET_ACCESS_POLICY_PERMISSIONS_REQUEST';
export const GET_ACCESS_POLICY_PERMISSIONS_SUCCESS =
  'GET_ACCESS_POLICY_PERMISSIONS_SUCCESS';
export const GET_ACCESS_POLICY_PERMISSIONS_ERROR =
  'GET_ACCESS_POLICY_PERMISSIONS_ERROR';

export const GET_TRANSFORMER_PERMISSIONS_REQUEST =
  'GET_TRANSFORMER_PERMISSIONS_REQUEST';
export const GET_TRANSFORMER_PERMISSIONS_SUCCESS =
  'GET_TRANSFORMER_PERMISSIONS_SUCCESS';
export const GET_TRANSFORMER_PERMISSIONS_ERROR =
  'GET_TRANSFORMER_PERMISSIONS_ERROR';

export const GET_ACCESS_POLICIES_REQUEST = 'GET_ACCESS_POLICIES_REQUEST';
export const GET_ACCESS_POLICIES_SUCCESS = 'GET_ACCESS_POLICIES_SUCCESS';
export const GET_ACCESS_POLICIES_ERROR = 'GET_ACCESS_POLICIES_ERROR';

export const GET_ALL_ACCESS_POLICIES_REQUEST =
  'GET_ALL_ACCESS_POLICIES_REQUEST';
export const GET_ALL_ACCESS_POLICIES_SUCCESS =
  'GET_ALL_ACCESS_POLICIES_SUCCESS';
export const GET_ALL_ACCESS_POLICIES_ERROR = 'GET_ALL_ACCESS_POLICIES_ERROR';

export const GET_ACCESS_POLICY_REQUEST = 'GET_ACCESS_POLICY_REQUEST';
export const GET_ACCESS_POLICY_SUCCESS = 'GET_ACCESS_POLICY_SUCCESS';
export const GET_ACCESS_POLICY_ERROR = 'GET_ACCESS_POLICY_ERROR';

export const GET_GLOBAL_ACCESSOR_POLICY_REQUEST =
  'GET_GLOBAL_ACCESSOR_POLICY_REQUEST';
export const GET_GLOBAL_ACCESSOR_POLICY_SUCCESS =
  'GET_GLOBAL_ACCESSOR_POLICY_SUCCESS';
export const GET_GLOBAL_ACCESSOR_POLICY_ERROR =
  'GET_GLOBAL_ACCESSOR_POLICY_ERROR';
export const GET_GLOBAL_MUTATOR_POLICY_REQUEST =
  'GET_GLOBAL_MUTATOR_POLICY_REQUEST';
export const GET_GLOBAL_MUTATOR_POLICY_SUCCESS =
  'GET_GLOBAL_MUTATOR_POLICY_SUCCESS';
export const GET_GLOBAL_MUTATOR_POLICY_ERROR =
  'GET_GLOBAL_MUTATOR_POLICY_ERROR';

export const GET_TOKEN_ACCESS_POLICY_REQUEST =
  'GET_TOKEN_ACCESS_POLICY_REQUEST';
export const GET_TOKEN_ACCESS_POLICY_SUCCESS =
  'GET_TOKEN_ACCESS_POLICY_SUCCESS';
export const GET_TOKEN_ACCESS_POLICY_ERROR = 'GET_TOKEN_ACCESS_POLICY_ERROR';

export const TOGGLE_ACCESS_POLICY_FOR_DELETE =
  'TOGGLE_ACCESS_POLICY_FOR_DELETE';
export const CHANGE_ACCESS_POLICY_LIST_INCLUDE_AUTOGENERATED =
  'CHANGE_ACCESS_POLICY_LIST_INCLUDE_AUTOGENERATED';
export const DELETE_SINGLE_ACCESS_POLICY_REQUEST =
  'DELETE_SINGLE_ACCESS_POLICY_REQUEST';
export const DELETE_SINGLE_ACCESS_POLICY_SUCCESS =
  'DELETE_SINGLE_ACCESS_POLICY_SUCCESS';
export const DELETE_SINGLE_ACCESS_POLICY_ERROR =
  'DELETE_SINGLE_ACCESS_POLICY_ERROR';
export const DELETE_ACCESS_POLICIES_SINGLE_SUCCESS =
  'DELETE_ACCESS_POLICIES_SINGLE_SUCCESS';
export const DELETE_ACCESS_POLICIES_SINGLE_ERROR =
  'DELETE_ACCESS_POLICIES_SINGLE_ERROR';
export const BULK_DELETE_ACCESS_POLICIES_REQUEST =
  'BULK_DELETE_ACCESS_POLICIES_REQUEST';
export const BULK_DELETE_ACCESS_POLICIES_SUCCESS =
  'BULK_DELETE_ACCESS_POLICIES_SUCCESS';
export const BULK_DELETE_ACCESS_POLICIES_FAILURE =
  'BULK_DELETE_ACCESS_POLICIES_FAILURE';

export const TOGGLE_ACCESS_POLICY_EDIT_MODE = 'TOGGLE_ACCESS_POLICY_EDIT_MODE';
export const TOGGLE_ACCESS_POLICY_DETAILS_EDIT_MODE =
  'TOGGLE_ACCESS_POLICY_DETAILS_EDIT_MODE';

export const CHANGE_ACCESS_POLICY_SEARCH_FILTER =
  'CHANGE_ACCESS_POLICY_SEARCH_FILTER';

export const GET_TRANSFORMERS_REQUEST = 'GET_TRANSFORMERS_REQUEST';
export const GET_TRANSFORMERS_SUCCESS = 'GET_TRANSFORMERS_SUCCESS';
export const GET_TRANSFORMERS_ERROR = 'GET_TRANSFORMERS_ERROR';

export const GET_TRANSFORMER_REQUEST = 'GET_TRANSFORMER_REQUEST';
export const GET_TRANSFORMER_SUCCESS = 'GET_TRANSFORMER_SUCCESS';
export const GET_TRANSFORMER_ERROR = 'GET_TRANSFORMER_ERROR';

export const TOGGLE_TRANSFORMER_FOR_DELETE = 'TOGGLE_TRANSFORMER_FOR_DELETE';
export const DELETE_SINGLE_TRANSFORMER_REQUEST =
  'DELETE_SINGLE_TRANSFORMER_REQUEST';
export const DELETE_SINGLE_TRANSFORMER_SUCCESS =
  'DELETE_SINGLE_TRANSFORMER_SUCCESS';
export const DELETE_SINGLE_TRANSFORMER_ERROR =
  'DELETE_SINGLE_TRANSFORMER_ERROR';
export const DELETE_TRANSFORMERS_SINGLE_SUCCESS =
  'DELETE_TRANSFORMERS_SINGLE_SUCCESS';
export const DELETE_TRANSFORMERS_SINGLE_ERROR =
  'DELETE_TRANSFORMERS_SINGLE_ERROR';
export const BULK_DELETE_TRANSFORMERS_REQUEST =
  'BULK_DELETE_TRANSFORMERS_REQUEST';
export const BULK_DELETE_TRANSFORMERS_SUCCESS =
  'BULK_DELETE_TRANSFORMERS_SUCCESS';
export const BULK_DELETE_TRANSFORMERS_FAILURE =
  'BULK_DELETE_TRANSFORMERS_FAILURE';

export const TOGGLE_TRANSFORMER_EDIT_MODE = 'TOGGLE_TRANSFORMER_EDIT_MODE';
export const TOGGLE_TRANSFORMER_DETAILS_EDIT_MODE =
  'TOGGLE_TRANSFORMER_DETAILS_EDIT_MODE';

export const CHANGE_TRANSFORMER = 'CHANGE_TRANSFORMER';
export const CHANGE_TRANSFORMER_TEST_DATA = 'CHANGE_TRANSFORMER_TEST_DATA';

export const CHANGE_TRANSFORMER_SEARCH_FILTER =
  'CHANGE_TRANSFORMER_SEARCH_FILTER';

export const CREATE_TRANSFORMER_REQUEST = 'CREATE_TRANSFORMER_REQUEST';
export const CREATE_TRANSFORMER_SUCCESS = 'CREATE_TRANSFORMER_SUCCESS';
export const CREATE_TRANSFORMER_ERROR = 'CREATE_TRANSFORMER_ERROR';

export const UPDATE_TRANSFORMER_REQUEST = 'UPDATE_TRANSFORMER_REQUEST';
export const UPDATE_TRANSFORMER_SUCCESS = 'UPDATE_TRANSFORMER_SUCCESS';
export const UPDATE_TRANSFORMER_ERROR = 'UPDATE_TRANSFORMER_ERROR';

export const TEST_TRANSFORMER_REQUEST = 'TEST_TRANSFORMER_REQUEST';
export const TEST_TRANSFORMER_SUCCESS = 'TEST_TRANSFORMER_SUCCESS';
export const TEST_TRANSFORMER_ERROR = 'TEST_TRANSFORMER_ERROR';

export const CREATE_ACCESS_POLICY_REQUEST = 'CREATE_ACCESS_POLICY_REQUEST';
export const CREATE_ACCESS_POLICY_SUCCESS = 'CREATE_ACCESS_POLICY_SUCCESS';
export const CREATE_ACCESS_POLICY_ERROR = 'CREATE_ACCESS_POLICY_ERROR';

export const UPDATE_ACCESS_POLICY_REQUEST = 'UPDATE_ACCESS_POLICY_REQUEST';
export const UPDATE_ACCESS_POLICY_SUCCESS = 'UPDATE_ACCESS_POLICY_SUCCESS';
export const UPDATE_ACCESS_POLICY_ERROR = 'UPDATE_ACCESS_POLICY_ERROR';

export const TEST_POLICY_REQUEST = 'TEST_POLICY_REQUEST';
export const TEST_POLICY_SUCCESS = 'TEST_POLICY_SUCCESS';
export const TEST_POLICY_ERROR = 'TEST_POLICY_ERROR';

export const SET_POLICY_PAGINATED_SELECTOR_COMPONENT =
  'SET_POLICY_PAGINATED_SELECTOR_COMPONENT';

export const LAUNCH_POLICY_CHOOSER_FOR_ACCESS_POLICY =
  'LAUNCH_POLICY_CHOOSER_FOR_ACCESS_POLICY';
export const LAUNCH_POLICY_CHOOSER_FOR_POLICY_TEMPLATE =
  'LAUNCH_POLICY_CHOOSER_FOR_POLICY_TEMPLATE';
export const SELECT_POLICY_OR_TEMPLATE_FROM_CHOOSER =
  'SELECT_POLICY_OR_TEMPLATE_FROM_CHOOSER';
export const CLOSE_POLICY_CHOOSER = 'CLOSE_POLICY_CHOOSER';

export const LAUNCH_POLICY_TEMPLATE_DIALOG = 'LAUNCH_POLICY_TEMPLATE_DIALOG';
export const CLOSE_POLICY_TEMPLATE_DIALOG = 'CLOSE_POLICY_TEMPLATE_DIALOG';

export const TOGGLE_POLICY_SECRET_FOR_DELETE =
  'TOGGLE_POLICY_SECRET_FOR_DELETE';
export const TOGGLE_POLICY_SECRETS_DELETE_ALL =
  'TOGGLE_POLICY_SECRETS_DELETE_ALL';
export const FETCH_TENANT_POLICY_SECRETS_REQUEST =
  'FETCH_TENANT_POLICY_SECRETS_REQUEST';
export const FETCH_TENANT_POLICY_SECRETS_SUCCESS =
  'FETCH_TENANT_POLICY_SECRETS_SUCCESS';
export const FETCH_TENANT_POLICY_SECRETS_ERROR =
  'FETCH_TENANT_POLICY_SECRETS_ERROR';
export const INITIALIZE_TENANT_POLICY_SECRET =
  'INITIALIZE_TENANT_POLICY_SECRET';
export const MODIFY_POLICY_SECRET = 'MODIFY_POLICY_SECRET';
export const SAVE_POLICY_SECRET_REQUEST = 'SAVE_POLICY_SECRET_REQUEST';
export const SAVE_POLICY_SECRET_ERROR = 'SAVE_POLICY_SECRET_ERROR';

export const modifyPolicyTemplate = (
  changes: Record<string, string>,
  isNew?: boolean
) => ({
  type: MODIFY_POLICY_TEMPLATE,
  data: {
    changes,
    isNew,
  },
});

export const createPolicyTemplateRequest = () => ({
  type: CREATE_POLICY_TEMPLATE_REQUEST,
});

export const createPolicyTemplateSuccess = (
  template: AccessPolicyTemplate
) => ({
  type: CREATE_POLICY_TEMPLATE_SUCCESS,
  data: template,
});

export const createPolicyTemplateError = (error: APIError) => ({
  type: CREATE_POLICY_TEMPLATE_ERROR,
  data: error.message,
});

export const updatePolicyTemplateRequest = () => ({
  type: UPDATE_POLICY_TEMPLATE_REQUEST,
});

export const updatePolicyTemplateSuccess = (
  template: AccessPolicyTemplate
) => ({
  type: UPDATE_POLICY_TEMPLATE_SUCCESS,
  data: template,
});

export const updatePolicyTemplateError = (error: APIError) => ({
  type: UPDATE_POLICY_TEMPLATE_ERROR,
  data: error.message,
});

export const togglePolicyTemplatesEditMode = (editMode?: boolean) => ({
  type: TOGGLE_POLICY_TEMPLATES_EDIT_MODE,
  data: editMode,
});

export const getPolicyTemplatesRequest = () => ({
  type: GET_POLICY_TEMPLATES_REQUEST,
});

export const getPolicyTemplatesSuccess = (
  templates: PaginatedResult<AccessPolicyTemplate>
) => ({
  type: GET_POLICY_TEMPLATES_SUCCESS,
  data: templates,
});

export const getPolicyTemplatesError = (error: APIError) => ({
  type: GET_POLICY_TEMPLATES_ERROR,
  data: error.message,
});

export const getPolicyTemplateRequest = () => ({
  type: GET_POLICY_TEMPLATE_REQUEST,
});

export const getPolicyTemplateSuccess = (template: AccessPolicyTemplate) => ({
  type: GET_POLICY_TEMPLATE_SUCCESS,
  data: template,
});

export const getPolicyTemplateError = (error: APIError) => ({
  type: GET_POLICY_TEMPLATE_ERROR,
  data: error.message,
});

export const togglePolicyTemplateForDelete = (id: string) => ({
  type: TOGGLE_POLICY_TEMPLATE_FOR_DELETE,
  data: id,
});

export const deleteSinglePolicyTemplateRequest = (
  template: AccessPolicyTemplate
) => ({
  type: DELETE_SINGLE_POLICY_TEMPLATE_REQUEST,
  data: template,
});

export const deleteSinglePolicyTemplateSuccess = (
  template: AccessPolicyTemplate
) => ({
  type: DELETE_SINGLE_POLICY_TEMPLATE_SUCCESS,
  data: template,
});

export const deletePolicyTemplatesSingleSuccess = (
  policyTemplateID: string
) => ({
  type: DELETE_POLICY_TEMPLATES_SINGLE_SUCCESS,
  data: policyTemplateID,
});

export const deletePolicyTemplatesSingleError = (
  policyTemplateID: string,
  error: APIError
) => ({
  type: DELETE_POLICY_TEMPLATES_SINGLE_ERROR,
  data: error.message,
});

export const deleteSinglePolicyTemplateError = (error: APIError) => ({
  type: DELETE_SINGLE_POLICY_TEMPLATE_ERROR,
  data: error.message,
});

export const bulkDeletePolicyTemplatesRequest = () => ({
  type: BULK_DELETE_POLICY_TEMPLATES_REQUEST,
});

export const bulkDeletePolicyTemplatesSuccess = () => ({
  type: BULK_DELETE_POLICY_TEMPLATES_SUCCESS,
});

export const bulkDeletePolicyTemplatesFailure = () => ({
  type: BULK_DELETE_POLICY_TEMPLATES_FAILURE,
});

export const changeAccessPolicyTemplateSearchFilter = (filter: Filter) => ({
  type: CHANGE_ACCESS_POLICY_TEMPLATE_SEARCH_FILTER,
  data: filter,
});

export const modifyAccessPolicy = (changes: Record<string, any>) => ({
  type: MODIFY_ACCESS_POLICY,
  data: changes,
});

export const modifyAccessPolicyThresholds = (changes: Record<string, any>) => ({
  type: MODIFY_ACCESS_POLICY_THRESHOLDS,
  data: changes,
});

export const modifyTokenAccessPolicy = (changes: Record<string, any>) => ({
  type: MODIFY_TOKEN_ACCESS_POLICY,
  data: changes,
});

export const changePolicyComponents = (
  policies: AccessPolicy[],
  templates: AccessPolicyTemplate[]
) => ({
  type: CHANGE_POLICY_COMPONENTS,
  data: { policies, templates },
});

export const changeAccessPolicyTestContext = (value: string) => ({
  type: CHANGE_ACCESS_POLICY_TEST_CONTEXT,
  data: value,
});

export const changeAccessPolicyTestParams = (value: string) => ({
  type: CHANGE_ACCESS_POLICY_TEST_PARAMS,
  data: value,
});

export const getUserPolicyPermissionsRequest = () => ({
  type: GET_USER_POLICY_PERMISSIONS_REQUEST,
});

export const getUserPolicyPermissionsSuccess = (
  permissions: PermissionsOnObject
) => ({
  type: GET_USER_POLICY_PERMISSIONS_SUCCESS,
  data: permissions,
});

export const getUserPolicyPermissionsError = (error: APIError) => ({
  type: GET_USER_POLICY_PERMISSIONS_ERROR,
  data: error.message,
});

export const getAccessPolicyPermissionsRequest = () => ({
  type: GET_ACCESS_POLICY_PERMISSIONS_REQUEST,
});

export const getAccessPolicyPermissionsSuccess = (
  permissions: PermissionsOnObject
) => ({
  type: GET_ACCESS_POLICY_PERMISSIONS_SUCCESS,
  data: permissions,
});

export const getAccessPolicyPermissionsError = (error: APIError) => ({
  type: GET_ACCESS_POLICY_PERMISSIONS_ERROR,
  data: error.message,
});

export const getTransformerPermissionsRequest = () => ({
  type: GET_TRANSFORMER_PERMISSIONS_REQUEST,
});

export const getTransformerPermissionsSuccess = (
  permissions: PermissionsOnObject
) => ({
  type: GET_TRANSFORMER_PERMISSIONS_SUCCESS,
  data: permissions,
});

export const getTransformerPermissionsError = (error: APIError) => ({
  type: GET_TRANSFORMER_PERMISSIONS_ERROR,
  data: error.message,
});

export const getAccessPoliciesRequest = () => ({
  type: GET_ACCESS_POLICIES_REQUEST,
});

export const getAccessPoliciesSuccess = (
  accessPolicies: PaginatedResult<AccessPolicy>
) => ({
  type: GET_ACCESS_POLICIES_SUCCESS,
  data: accessPolicies,
});

export const getAccessPoliciesError = (error: APIError) => ({
  type: GET_ACCESS_POLICIES_ERROR,
  data: error.message,
});

export const getAllAccessPoliciesRequest = () => ({
  type: GET_ALL_ACCESS_POLICIES_REQUEST,
});

export const getAllAccessPoliciesSuccess = (
  accessPolicies: PaginatedResult<AccessPolicy>
) => ({
  type: GET_ALL_ACCESS_POLICIES_SUCCESS,
  data: accessPolicies,
});

export const getAllAccessPoliciesError = (error: APIError) => ({
  type: GET_ALL_ACCESS_POLICIES_ERROR,
  data: error.message,
});

export const getAccessPolicyRequest = () => ({
  type: GET_ACCESS_POLICY_REQUEST,
});

export const getAccessPolicySuccess = (policy: AccessPolicy) => ({
  type: GET_ACCESS_POLICY_SUCCESS,
  data: policy,
});

export const getAccessPolicyError = (error: APIError) => ({
  type: GET_ACCESS_POLICY_ERROR,
  data: error.message,
});

export const getGlobalAccessorPolicyRequest = () => ({
  type: GET_GLOBAL_ACCESSOR_POLICY_REQUEST,
});

export const getGlobalAccessorPolicySuccess = (policy: AccessPolicy) => ({
  type: GET_GLOBAL_ACCESSOR_POLICY_SUCCESS,
  data: policy,
});

export const getGlobalAccessorPolicyError = (error: APIError) => ({
  type: GET_GLOBAL_ACCESSOR_POLICY_ERROR,
  data: error.message,
});

export const getGlobalMutatorPolicyRequest = () => ({
  type: GET_GLOBAL_MUTATOR_POLICY_REQUEST,
});

export const getGlobalMutatorPolicySuccess = (policy: AccessPolicy) => ({
  type: GET_GLOBAL_MUTATOR_POLICY_SUCCESS,
  data: policy,
});

export const getGlobalMutatorPolicyError = (error: APIError) => ({
  type: GET_GLOBAL_MUTATOR_POLICY_ERROR,
  data: error.message,
});

export const getTokenAccessPolicyRequest = () => ({
  type: GET_TOKEN_ACCESS_POLICY_REQUEST,
});

export const getTokenAccessPolicySuccess = (policy: AccessPolicy) => ({
  type: GET_TOKEN_ACCESS_POLICY_SUCCESS,
  data: policy,
});

export const getTokenAccessPolicyError = (error: APIError) => ({
  type: GET_TOKEN_ACCESS_POLICY_ERROR,
  data: error.message,
});

export const toggleAccessPolicyForDelete = (id: string) => ({
  type: TOGGLE_ACCESS_POLICY_FOR_DELETE,
  data: id,
});

export const toggleAccessPolicyListIncludeAutogenerated = (
  include: boolean
) => ({
  type: CHANGE_ACCESS_POLICY_LIST_INCLUDE_AUTOGENERATED,
  data: include,
});

export const deleteSingleAccessPolicyRequest = () => ({
  type: DELETE_SINGLE_ACCESS_POLICY_REQUEST,
});

export const deleteSingleAccessPolicySuccess = () => ({
  type: DELETE_SINGLE_ACCESS_POLICY_SUCCESS,
});

export const deleteSingleAccessPolicyError = (error: APIError) => ({
  type: DELETE_SINGLE_ACCESS_POLICY_ERROR,
  data: error.message,
});

export const deleteAccessPoliciesSingleSuccess = (policyID: string) => ({
  type: DELETE_ACCESS_POLICIES_SINGLE_SUCCESS,
  data: policyID,
});

export const deleteAccessPoliciesSingleError = (
  policyID: string,
  error: APIError
) => ({
  type: DELETE_ACCESS_POLICIES_SINGLE_ERROR,
  data: error.message,
});

export const bulkDeleteAccessPoliciesRequest = () => ({
  type: BULK_DELETE_ACCESS_POLICIES_REQUEST,
});

export const bulkDeleteAccessPoliciesSuccess = () => ({
  type: BULK_DELETE_ACCESS_POLICIES_SUCCESS,
});

export const bulkDeleteAccessPoliciesFailure = () => ({
  type: BULK_DELETE_ACCESS_POLICIES_FAILURE,
});

export const toggleAccessPolicyEditMode = (editMode?: boolean) => ({
  type: TOGGLE_ACCESS_POLICY_EDIT_MODE,
  data: editMode,
});

export const toggleAccessPolicyDetailsEditMode = (editMode?: boolean) => ({
  type: TOGGLE_ACCESS_POLICY_DETAILS_EDIT_MODE,
  data: editMode,
});

export const changeAccessPolicySearchFilter = (filter: Filter) => ({
  type: CHANGE_ACCESS_POLICY_SEARCH_FILTER,
  data: filter,
});

export const getTransformersRequest = () => ({
  type: GET_TRANSFORMERS_REQUEST,
});

export const getTransformersSuccess = (
  transformers: PaginatedResult<Transformer>
) => ({
  type: GET_TRANSFORMERS_SUCCESS,
  data: transformers,
});

export const getTransformersError = (error: APIError) => ({
  type: GET_TRANSFORMERS_ERROR,
  data: error.message,
});

export const getTransformerRequest = () => ({
  type: GET_TRANSFORMER_REQUEST,
});

export const getTransformerSuccess = (transformer: Transformer) => ({
  type: GET_TRANSFORMER_SUCCESS,
  data: transformer,
});

export const getTransformerError = (error: APIError) => ({
  type: GET_TRANSFORMER_ERROR,
  data: error.message,
});

export const toggleTransformerForDelete = (id: string) => ({
  type: TOGGLE_TRANSFORMER_FOR_DELETE,
  data: id,
});

export const deleteSingleTransformerRequest = () => ({
  type: DELETE_SINGLE_TRANSFORMER_REQUEST,
});

export const deleteSingleTransformerSuccess = () => ({
  type: DELETE_SINGLE_TRANSFORMER_SUCCESS,
});

export const deleteSingleTransformerError = (error: APIError) => ({
  type: DELETE_SINGLE_TRANSFORMER_ERROR,
  data: error.message,
});

export const deleteTransformersSingleSuccess = (transformerID: string) => ({
  type: DELETE_TRANSFORMERS_SINGLE_SUCCESS,
  data: transformerID,
});

export const deleteTransformersSingleError = (
  transformerID: string,
  error: APIError
) => ({
  type: DELETE_TRANSFORMERS_SINGLE_ERROR,
  data: error.message,
});

export const bulkDeleteTransformersRequest = () => ({
  type: BULK_DELETE_TRANSFORMERS_REQUEST,
});

export const bulkDeleteTransformersSuccess = () => ({
  type: BULK_DELETE_TRANSFORMERS_SUCCESS,
});

export const bulkDeleteTransformersFailure = () => ({
  type: BULK_DELETE_TRANSFORMERS_FAILURE,
});

export const toggleTransformerEditMode = (editMode?: boolean) => ({
  type: TOGGLE_TRANSFORMER_EDIT_MODE,
  data: editMode,
});

export const toggleTransformerDetailsEditMode = (editMode?: boolean) => ({
  type: TOGGLE_TRANSFORMER_DETAILS_EDIT_MODE,
  data: editMode,
});

export const changeTransformer = (changes: Record<string, any>) => ({
  type: CHANGE_TRANSFORMER,
  data: changes,
});

export const changeTransformerTestData = (testData: string) => ({
  type: CHANGE_TRANSFORMER_TEST_DATA,
  data: testData,
});

export const changeTransformerSearchFilter = (filter: Filter) => ({
  type: CHANGE_TRANSFORMER_SEARCH_FILTER,
  data: filter,
});

export const createAccessPolicyRequest = () => ({
  type: CREATE_ACCESS_POLICY_REQUEST,
});

export const createAccessPolicySuccess = (policy: AccessPolicy) => ({
  type: CREATE_ACCESS_POLICY_SUCCESS,
  data: policy,
});

export const createAccessPolicyError = (error: APIError) => ({
  type: CREATE_ACCESS_POLICY_ERROR,
  data: error.message,
});

export const updateAccessPolicyRequest = () => ({
  type: UPDATE_ACCESS_POLICY_REQUEST,
});

export const updateAccessPolicySuccess = (policy: AccessPolicy) => ({
  type: UPDATE_ACCESS_POLICY_SUCCESS,
  data: policy,
});

export const updateAccessPolicyError = (error: APIError) => ({
  type: UPDATE_ACCESS_POLICY_ERROR,
  data: error.message,
});

export const createTransformerRequest = () => ({
  type: CREATE_TRANSFORMER_REQUEST,
});

export const createTransformerSuccess = (transformer: Transformer) => ({
  type: CREATE_TRANSFORMER_SUCCESS,
  data: transformer,
});

export const createTransformerError = (error: APIError) => ({
  type: CREATE_TRANSFORMER_ERROR,
  data: error.message,
});

export const updateTransformerRequest = () => ({
  type: UPDATE_TRANSFORMER_REQUEST,
});

export const updateTransformerSuccess = (transformer: Transformer) => ({
  type: UPDATE_TRANSFORMER_SUCCESS,
  data: transformer,
});

export const updateTransformerError = (error: APIError) => ({
  type: UPDATE_TRANSFORMER_ERROR,
  data: error.message,
});

export const testTransformerRequest = () => ({
  type: TEST_TRANSFORMER_REQUEST,
});

export const testTransformerSuccess = (result: TransformerTestResult) => ({
  type: TEST_TRANSFORMER_SUCCESS,
  data: result,
});

export const testTransformerError = (error: APIError) => ({
  type: TEST_TRANSFORMER_ERROR,
  data: error.message,
});

export const testPolicyRequest = () => ({
  type: TEST_POLICY_REQUEST,
});

export const testPolicySuccess = (result: AccessPolicyTestResult) => ({
  type: TEST_POLICY_SUCCESS,
  data: result,
});

export const testPolicyError = (error: APIError) => ({
  type: TEST_POLICY_ERROR,
  data: error.message,
});

export const setPaginatedPolicyChooserComponent = (
  component: AccessPolicyComponent
) => ({
  type: SET_POLICY_PAGINATED_SELECTOR_COMPONENT,
  data: component,
});

export const launchPolicyChooserForAccessPolicy = () => ({
  type: LAUNCH_POLICY_CHOOSER_FOR_ACCESS_POLICY,
});
export const launchPolicyChooserForPolicyTemplate = () => ({
  type: LAUNCH_POLICY_CHOOSER_FOR_POLICY_TEMPLATE,
});
export const selectPolicyOrTemplateFromChooser = (
  policyOrTemplate: AccessPolicy | AccessPolicyTemplate
) => ({
  type: SELECT_POLICY_OR_TEMPLATE_FROM_CHOOSER,
  data: policyOrTemplate,
});
export const closePolicyChooser = () => ({
  type: CLOSE_POLICY_CHOOSER,
});

export const launchPolicyTemplateDialog = () => ({
  type: LAUNCH_POLICY_TEMPLATE_DIALOG,
});
export const closePolicyTemplateDialog = () => ({
  type: CLOSE_POLICY_TEMPLATE_DIALOG,
});

export const togglePolicySecretForDelete = (policySecret: PolicySecret) => ({
  type: TOGGLE_POLICY_SECRET_FOR_DELETE,
  data: policySecret.id,
});

export const togglePolicySecretsDeleteAll = (checked: boolean) => ({
  type: TOGGLE_POLICY_SECRETS_DELETE_ALL,
  data: checked,
});

export const fetchTenantPolicySecretsRequest =
  () => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_TENANT_POLICY_SECRETS_REQUEST,
    });
  };

export const fetchTenantPolicySecretsSuccess =
  (policySecrets: PaginatedResult<PolicySecret>) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_TENANT_POLICY_SECRETS_SUCCESS,
      data: policySecrets,
    });
  };

export const fetchTenantPolicySecretsError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_TENANT_POLICY_SECRETS_ERROR,
      data: error.message,
    });
  };

export const initializeNewPolicySecret = () => (dispatch: AppDispatch) => {
  dispatch({ type: INITIALIZE_TENANT_POLICY_SECRET });
};

export const modifyPolicySecret = (changes: Record<string, string>) => ({
  type: MODIFY_POLICY_SECRET,
  data: changes,
});

export const savePolicySecretRequest = () => ({
  type: SAVE_POLICY_SECRET_REQUEST,
});

export const savePolicySecretError = (error: APIError) => ({
  type: SAVE_POLICY_SECRET_ERROR,
  data: error.message,
});
