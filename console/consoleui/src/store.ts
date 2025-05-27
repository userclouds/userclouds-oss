import { AnyAction } from 'redux';
import { configureStore } from '@reduxjs/toolkit';
import thunk, { ThunkAction, ThunkDispatch } from 'redux-thunk';

import { JSONValue } from '@userclouds/sharedui';
import ServiceInfo from './ServiceInfo';
import { FeatureFlags, featureFlagsAreEnabled } from './models/FeatureFlag';
import PaginatedResult from './models/PaginatedResult';
import Company from './models/Company';
import Organization from './models/Organization';
import LoginApp from './models/LoginApp';
import Tenant, { SelectedTenant, TenantStateType } from './models/Tenant';
import TenantPlexConfig, {
  UpdatePlexConfigReason,
} from './models/TenantPlexConfig';
import { PageParametersResponse } from './models/PageParameters';
import { AppMessageElement, MessageType } from './models/MessageElements';
import Provider from './models/Provider';
import { Column } from './models/TenantUserStoreConfig';
import { ObjectType } from './models/authz/ObjectType';
import EdgeType from './models/authz/EdgeType';
import UCObject from './models/authz/Object';
import Edge from './models/authz/Edge';
import {
  AuthorizationRequest,
  CheckAttributePathRow,
} from './models/authz/CheckAttribute';
import DataSource, {
  blankDataSource,
  DataSourceElement,
  blankDataSourceElement,
} from './models/DataSource';
import Accessor, {
  blankAccessor,
  AccessorColumn,
  AccessorSavePayload,
  ExecuteAccessorResponse,
} from './models/Accessor';
import Mutator, {
  blankMutator,
  MutatorSavePayload,
  MutatorColumn,
} from './models/Mutator';
import Purpose from './models/Purpose';
import {
  ColumnRetentionDurationsResponse,
  PurposeRetentionDuration,
} from './models/ColumnRetentionDurations';
import { SqlshimDatabase } from './models/SqlshimDatabase';
import { ObjectStore } from './models/ObjectStore';
import { PolicySecret } from './models/PolicySecret';
import {
  MyProfile,
  UserProfileSerialized,
  UserEvent,
  UserBaseProfile,
} from './models/UserProfile';
import { UserInvite } from './models/UserInvite';
import { UserRoles } from './models/UserRoles';
import AccessPolicy, {
  blankPolicy as blankAccessPolicy,
  AccessPolicyTemplate,
  AccessPolicyTestResult,
  ComponentPolicy,
  AccessPolicyComponent,
  PolicySelectorResourceType,
} from './models/AccessPolicy';
import Transformer, {
  blankTransformer,
  TransformerTestResult,
} from './models/Transformer';
import PermissionsOnObject from './models/authz/Permissions';
import { AuditLogEntry, DataAccessLogEntry } from './models/AuditLogEntry';
import { SystemLogEntry, SystemLogEntryRecord } from './models/SystemLogEntry';
import Notification from './models/Notification';
import ActiveInstance from './chart/ActiveInstance';
import type { ChartRenderableData } from './models/Chart';
import LogRow from './chart/LogRow';
import rootReducer from './reducers/root';
import TenantURL from './models/TenantURL';
import { Filter } from './models/authz/SearchFilters';
import { DataAccessLogFilter } from './models/DataAccessLogFilter';
import { OIDCProvider } from './models/OIDCProvider';
import { TagModel } from './models/Tag';
import { blankDataType, DataType } from './models/DataType';
import type { CountMetric } from './models/Metrics';

// NB:ksj: we probably DON'T want to break up
// the definition of the store (neither this interface
// nor initialState), as keys in the store must be unique
// even if they're typically touched by different reducers.
export interface RootState {
  /** **************************************
   *                                      *
   * APP-WIDE PERSISTED STATE             *
   *                                      *
   *************************************** */
  // feature flags
  featureFlags: FeatureFlags | undefined;
  fetchingFeatureFlags: boolean;
  featureFlagsFetchError: string;

  // service info:
  serviceInfo: ServiceInfo | undefined;
  serviceInfoError: string | undefined;
  myProfile: MyProfile | undefined;

  // companies
  globalCompanies: Company[] | undefined;
  companies: Company[] | undefined;

  // tenants
  tenants: Tenant[] | undefined;
  tenantURLs: TenantURL[] | undefined;

  /** **************************************
   *                                      *
   * PAGE-SPECIFIC PERSISTED STATE        *
   *                                      *
   *************************************** */
  // organizations
  organizations: PaginatedResult<Organization> | undefined;
  loginAppsForOrg: LoginApp[] | undefined;

  // authn/plex
  tenantPlexConfig: TenantPlexConfig | undefined;
  tenantEmailMessageElements: AppMessageElement[] | undefined;
  tenantSMSMessageElements: AppMessageElement[] | undefined;
  appPageParameters: Record<string, PageParametersResponse>;

  // authz
  objectTypes: PaginatedResult<ObjectType> | undefined;
  edgeTypes: PaginatedResult<EdgeType> | undefined;
  displayEdges: PaginatedResult<Edge> | undefined;
  authzObjects: PaginatedResult<UCObject> | undefined;
  edgesForObject: Edge[] | undefined;
  objectsForEdgesForObject: UCObject[] | undefined;

  // userstore
  userStoreDisplayColumns: PaginatedResult<Column> | undefined;
  userStoreColumns: Column[] | undefined;
  columnRetentionDurations: ColumnRetentionDurationsResponse | undefined;
  accessorsForColumn: PaginatedResult<Accessor> | undefined;

  accessorMetrics: CountMetric[] | undefined;
  accessors: PaginatedResult<Accessor> | undefined;
  mutators: PaginatedResult<Mutator> | undefined;
  dataTypes: PaginatedResult<DataType> | undefined;

  // tags
  tags: TagModel[] | undefined;

  // purposes
  purposes: PaginatedResult<Purpose> | undefined;

  // customer end users
  tenantUsersOrganizations: PaginatedResult<Organization> | undefined;
  tenantUsers: PaginatedResult<UserBaseProfile> | undefined;

  // team (customer employee IAM)
  teamUserRoles: UserRoles[] | undefined;
  companyUserRoles: UserRoles[] | undefined;
  companyInvites: PaginatedResult<UserInvite> | undefined;

  // policies
  userPolicyPermissions: PermissionsOnObject | undefined;
  fetchingUserPolicyPermissions: boolean;
  userPolicyPermissionsFetchError: string;
  accessPolicyPermissions: PermissionsOnObject | undefined;
  transformerPermissions: PermissionsOnObject | undefined;
  accessPolicies: PaginatedResult<AccessPolicy> | undefined;
  allAccessPolicies: PaginatedResult<AccessPolicy> | undefined;
  policyTemplates: PaginatedResult<AccessPolicyTemplate> | undefined;
  componentPolicies: ComponentPolicy[] | undefined;
  transformers: PaginatedResult<Transformer> | undefined;

  // system log
  systemLogEntries: PaginatedResult<SystemLogEntry> | undefined;
  systemLogEntryRecords: PaginatedResult<SystemLogEntryRecord> | undefined;

  // audit log
  auditLogEntries: PaginatedResult<AuditLogEntry> | undefined;

  // data access log
  dataAccessLogEntries: PaginatedResult<DataAccessLogEntry> | undefined;
  dataAccessLogEntry: DataAccessLogEntry | undefined;

  // status page (monitoring)
  activeInstances: ActiveInstance[] | undefined;
  logEvents: LogRow[] | undefined;
  chartData: ChartRenderableData[][] | undefined;

  // JWT signing keys
  tenantPublicKey: string;

  /** **************************************
   *                                      *
   * APP-WIDE EPHEMERAL STATE             *
   *                                      *
   *************************************** */
  // notifications
  notifications: Notification[];

  // companies
  fetchingAllCompanies: boolean;
  companiesFetchError: string;
  creatingCompany: boolean;
  createCompanyError: string;
  selectedCompanyID: string | undefined;
  selectedCompany: Company | undefined;
  fetchingCompanies: boolean;
  companyFetchError: string;
  createCompanyDialogIsOpen: boolean;
  changeCompanyDialogIsOpen: boolean;

  // tenants
  selectedTenantID: string | undefined;
  selectedTenant: SelectedTenant | undefined;

  /** **************************************
   *                                      *
   * PAGE-SPECIFIC EPHEMERAL STATE        *
   *                                      *
   *************************************** */
  // routing
  location: URL;
  query: URLSearchParams;
  routeHandler: Function | undefined;
  routeParams: Record<string, string>;
  routePattern: string;

  // tenants
  creatingTenant: boolean;
  createTenantError: string;
  editingTenant: boolean;
  savingTenant: boolean;
  saveTenantSuccess: string;
  saveTenantError: string;
  fetchingSelectedTenant: boolean;
  deletingTenant: boolean;
  deleteTenantError: string;
  modifiedTenant: SelectedTenant | undefined;
  fetchingTenants: boolean;
  tenantFetchError: string;
  currentTenantURL: TenantURL | undefined;
  modifiedTenantURL: TenantURL | undefined;
  tenantURLIsDirty: boolean;
  tenantURLDialogIsOpen: boolean;
  tenantDatabaseDialogIsOpen: boolean;
  tenantIssuerDialogIsOpen: boolean;
  tenantProviderName: string;
  creatingIssuer: boolean;
  editingIssuerIndex: number;
  tenantURLsIsDirty: boolean;
  creatingURL: boolean;
  modifiedTenantUrls: TenantURL[];
  tenantURLsToCreate: TenantURL[];
  tenantURLsToUpdate: TenantURL[];
  tenantURLsToDelete: string[];
  fetchingTenantURLs: boolean;
  fetchingTenantURLsError: string;
  savingTenantURLs: boolean;
  editingTenantURL: boolean;
  editingTenantURLError: string;
  creatingNewTenantURL: boolean;
  createTenantDialogIsOpen: boolean;
  tenantCreationState: TenantStateType | null;

  // organizations
  selectedOrganization: Organization | undefined;
  modifiedOrganization: Organization | undefined;
  fetchingOrganizations: boolean;
  organizationsFetchError: string;
  editingOrganization: boolean;
  savingOrganization: boolean;
  createOrganizationError: string;
  updateOrganizationError: string;

  // authn / plex
  fetchingPlexConfig: boolean;
  fetchPlexConfigError: string;
  selectedPlexApp: LoginApp | undefined;
  selectedPlexProvider: Provider | undefined;
  oidcProvider: OIDCProvider | undefined;
  savingOIDCProvider: boolean;
  saveOIDCProviderSuccess: boolean;
  saveOIDCProviderError: string;
  modifiedPlexConfig: TenantPlexConfig | undefined;
  modifiedPlexApp: LoginApp | undefined;
  plexEmployeeApp: LoginApp | undefined;
  modifiedPlexEmployeeApp: LoginApp | undefined;
  modifiedPlexProvider: Provider | undefined;
  plexConfigIsDirty: boolean;
  savingPlexConfig: boolean;
  savePlexConfigSuccess: UpdatePlexConfigReason | undefined;
  savePlexConfigError: string;
  editingPlexConfig: boolean;
  auth0AppsEditMode: boolean;
  cognitoAppsEditMode: boolean;
  ucAppsEditMode: boolean;
  fetchingLoginApps: boolean;
  loginAppsFetchError: string;
  modifiedEmailMessageElements: AppMessageElement[] | undefined;
  emailMessageElementsAreDirty: boolean;
  fetchingEmailMessageElements: boolean;
  emailMessageElementsFetchError: string;
  savingEmailMessageElements: boolean;
  emailMessageElementsSaveSuccess: boolean;
  emailMessageElementsSaveError: string;
  selectedEmailMessageType: MessageType;
  modifiedSMSMessageElements: AppMessageElement[] | undefined;
  smsMessageElementsAreDirty: boolean;
  fetchingSMSMessageElements: boolean;
  smsMessageElementsFetchError: string;
  savingSMSMessageElements: boolean;
  smsMessageElementsSaveSuccess: boolean;
  smsMessageElementsSaveError: string;
  selectedSMSMessageType: MessageType;
  modifiedPageParameters: PageParametersResponse | undefined;
  fetchingPageParameters: boolean;
  pageParametersFetchError: string;
  savingPageParameters: boolean;
  pageParametersSaveSuccess: boolean;
  pageParametersSaveError: string;
  appToClone: string;

  // authz
  currentObjectSearchFilter: Filter;
  currentEdgeSearchFilter: Filter;
  edgeTypeSearchFilter: Filter;
  objectTypeSearchFilter: Filter;

  authorizationRequest: AuthorizationRequest | undefined;
  authorizationSuccess: string;
  authorizationFailure: string;
  authorizationPath: CheckAttributePathRow[];

  fetchObjectTypesError: string;
  fetchingObjectTypes: boolean;
  currentObjectType: ObjectType | undefined;
  objectTypeFetchError: string;
  deletingObjectType: undefined;
  objectTypeDeleteSuccess: string;
  objectTypeDeleteError: string;
  objectTypeEditMode: boolean;
  fetchObjectTypeError: string;
  savingObjectType: boolean;
  saveObjectTypeSuccess: string;
  saveObjectTypeError: string;
  objectTypeDeleteQueue: string[];
  savingObjectTypes: boolean;

  currentEdgeType: EdgeType | undefined;
  currentEdgeTypeValid: boolean;
  edgeTypeIsDirty: boolean;
  edgeTypeFetchError: string;
  deletingEdgeType: undefined;
  edgeTypeDeleteSuccess: string;
  edgeTypeDeleteError: string;
  edgeTypeEditMode: boolean;
  fetchEdgeTypeError: string;
  savingEdgeType: boolean;
  saveEdgeTypeSuccess: string;
  saveEdgeTypeError: string;
  fetchEdgeTypesError: string;
  fetchingEdgeTypes: boolean;
  edgeTypeDeleteQueue: string[];
  savingEdgeTypes: boolean;

  fetchingAuthzEdges: boolean;
  fetchEdgesError: string;
  editEdgesSuccess: string;
  editEdgesError: string;
  savingEdges: boolean;

  currentEdge: Edge | undefined;
  edgeIsDirty: boolean;
  deletingEdge: undefined;
  edgeDeleteSuccess: string;
  edgeDeleteError: string;
  edgeEditMode: boolean;
  fetchEdgeError: string;
  savingEdge: boolean;
  saveEdgeSuccess: string;
  saveEdgeError: string;
  edgeDeleteQueue: string[];

  fetchingAuthzObjects: boolean;
  fetchObjectsError: string;
  lastFetchedObjectTypeID: string | undefined;
  fetchObjectError: string;
  fetchEdgesForObjectError: string;
  editingObjects: boolean;
  objectDeleteQueue: string[];
  editObjectsSuccess: string;
  editObjectsError: string;
  savingObjects: boolean;
  currentObject: UCObject | undefined;
  objectIsDirty: boolean;
  objectFetchError: string;
  deletingObject: boolean;
  objectDeleteSuccess: string;
  objectDeleteError: string;
  objectEditMode: boolean;
  savingObject: boolean;
  saveObjectSuccess: string;
  saveObjectError: string;

  // datamapping
  dataSourceDetailsEditMode: boolean;
  modifiedDataSource: DataSource;
  selectedDataSource: DataSource | undefined;
  fetchDataSourceError: string;
  fetchingDataSources: boolean;
  dataSources: PaginatedResult<DataSource> | undefined;
  dataSourcesDeleteQueue: string[];
  dataSourcesSearchFilter: Filter;
  deletingDataSources: boolean;
  deleteDataSourceSuccess: string;
  deleteDataSourceError: string;
  bulkDeleteDataSourcesSuccess: string;
  bulkDeleteDataSourcesErrors: string[];
  savingDataSource: boolean;
  dataSourceSaveSuccess: string;
  dataSourceSaveError: string;

  dataSourceElementDetailsEditMode: boolean;
  modifiedDataSourceElement: DataSourceElement;
  selectedDataSourceElement: DataSourceElement | undefined;
  fetchDataSourceElementError: string;
  fetchingDataSourceElements: boolean;
  dataSourceElements: PaginatedResult<DataSourceElement> | undefined;
  dataSourceElementsSearchFilter: Filter;
  savingDataSourceElement: boolean;
  dataSourceElementSaveSuccess: string;
  dataSourceElementSaveError: string;

  // user store
  fetchingUserStoreConfig: boolean;
  fetchUserStoreConfigError: string;
  userStoreColumnsToAdd: Column[];
  userStoreColumnsToModify: Record<string, Column>;
  // we only need to store the ids, but it doesn't
  // hurt to put the whole column in
  userStoreColumnsToDelete: Record<string, Column>;
  userStoreConfigIsDirty: boolean;
  savingUserStoreConfig: boolean;
  saveUserStoreConfigSuccess: string;
  saveUserStoreConfigErrors: string[];
  userStoreEditMode: boolean;

  // user store column
  selectedColumn: Column | undefined;
  modifiedColumn: Column | undefined;
  columnIsDirty: boolean;
  fetchingColumn: boolean;
  fetchingColumnError: string;
  savingColumn: boolean;
  savingColumnSuccess: string;
  saveColumnError: string;
  columnEditMode: boolean;
  columnPurposesEditMode: boolean;
  modifiedRetentionDurations: PurposeRetentionDuration[];
  fetchingColumnRetentionDurations: boolean;
  columnDurationsFetchError: string;
  savingColumnRetentionDurations: boolean;
  retentionDurationsSaveSuccess: boolean;
  retentionDurationsSaveError: string;
  purposeSettingsAreDirty: boolean;
  columnSearchFilter: Filter;

  // user store dataType
  fetchingDataTypes: boolean;
  fetchDataTypesError: string;
  dataTypesDeleteQueue: string[];
  savingDataTypes: boolean;
  dataTypesEditMode: boolean;
  selectedDataType: DataType | undefined;
  dataTypeToCreate: DataType;
  modifiedDataType: DataType | undefined;
  dataTypeIsDirty: boolean;
  fetchingDataType: boolean;
  fetchingDataTypeError: string;
  savingDataType: boolean;
  savingDataTypeSuccess: string;
  saveDataTypeError: string;
  dataTypeEditMode: boolean;
  dataTypeSearchFilter: Filter;

  // user store: accessors
  fetchingAccessors: boolean;
  fetchAccessorsError: string;
  selectedAccessorID: string | undefined;
  selectedAccessor: Accessor | undefined;
  modifiedAccessor: Accessor | undefined;
  modifiedAccessorIsDirty: boolean;
  accessorToCreate: AccessorSavePayload;
  savingAccessor: boolean;
  saveAccessorSuccess: string;
  saveAccessorError: string;
  accessorListEditMode: boolean;
  accessorListIncludeAutogenerated: boolean;
  updatingAccessors: boolean;
  bulkUpdateAccessorsSuccess: string;
  bulkUpdateAccessorsErrors: string[];
  accessorsToDelete: Record<string, Accessor>;
  accessorDetailsEditMode: boolean;
  accessorColumnsEditMode: boolean;
  accessorSelectorEditMode: boolean;
  accessorColumnsToAdd: AccessorColumn[];
  // we only need to store the ids, but it doesn't
  // hurt to put the whole column in
  accessorColumnsToDelete: Record<string, AccessorColumn>;
  accessorAddColumnDropdownValue: string;
  accessorPoliciesEditMode: boolean;
  saveAccessorColumnsSuccess: string;
  saveAccessorColumnsError: string;
  selectedAccessPolicyForAccessor: string;
  selectedTokenAccessPolicyForAccessor: string;
  accessorSearchFilter: Filter;
  fetchingAccessorMetrics: boolean;
  fetchAccessorMetricsError: string;

  saveAccessorPoliciesSuccess: string;
  saveAccessorPoliciesError: string;
  createAccessorError: string;
  executeAccessorContext: string;
  executeAccessorSelectorValues: string;
  executeAccessorResponse: ExecuteAccessorResponse | undefined;
  executeAccessorError: string;
  executeAccessorStats:
    | {
        frequencies: Record<string, number>;
        uniqueness: Record<string, Record<string, Record<string, number>>>;
      }
    | undefined;

  // user store: mutators
  fetchingMutators: boolean;
  fetchMutatorsError: string;
  selectedMutatorID: string | undefined;
  selectedMutator: Mutator | undefined;
  modifiedMutator: Mutator | undefined;
  modifiedMutatorIsDirty: boolean;
  mutatorToCreate: MutatorSavePayload;
  savingMutator: boolean;
  savingMutatorSuccess: string;
  savingMutatorError: string;
  mutatorListEditMode: boolean;
  updatingMutators: boolean;
  bulkUpdateMutatorsSuccess: string;
  bulkUpdateMutatorsErrors: string[];
  mutatorsToDelete: Record<string, Mutator>;
  mutatorDetailsEditMode: boolean;
  mutatorColumnsEditMode: boolean;
  mutatorSelectorEditMode: boolean;
  mutatorColumnsToAdd: MutatorColumn[];
  // we only need to store the ids, but it doesn't
  // hurt to put the whole column in
  mutatorColumnsToDelete: Record<string, MutatorColumn>;
  mutatorAddColumnDropdownValue: string;
  mutatorPoliciesEditMode: boolean;
  saveMutatorColumnsSuccess: string;
  saveMutatorColumnsError: string;
  selectedAccessPolicyForMutator: string;
  saveMutatorPoliciesSuccess: string;
  saveMutatorPoliciesError: string;
  createMutatorError: string;
  mutatorSearchFilter: Filter;

  // tags
  fetchingTags: boolean;
  fetchTagsError: string;
  savingTags: boolean;
  savingTagsSuccess: string;
  savingTagsError: string;

  // purposes
  selectedPurpose: Purpose | undefined;
  modifiedPurpose: Purpose | undefined;
  fetchingPurposes: boolean;
  purposesFetchError: string;
  creatingPurpose: boolean;
  createPurposeError: string;
  savingPurpose: boolean;
  savePurposeError: string;
  deletingPurpose: boolean;
  deletePurposeError: string;
  purposeDetailsEditMode: boolean;
  purposesBulkEditMode: boolean;
  purposesDeleteQueue: string[];
  deletePurposesSuccess: string;
  deletePurposesErrors: string[];
  purposesBulkSaving: boolean;
  purposeSearchFilter: Filter;

  // sqlshim database
  sqlShimDatabases: PaginatedResult<SqlshimDatabase> | undefined;
  modifiedSqlShimDatabases: SqlshimDatabase[];
  savingSqlshimDatabase: boolean;
  saveSqlshimDatabaseSuccess: string;
  saveSqlshimDatabaseError: string;
  testingSqlshimDatabase: boolean;
  testSqlshimDatabaseSuccess: boolean;
  testSqlshimDatabaseError: string;
  modifiedSqlshimDatabase: SqlshimDatabase | undefined;
  currentSqlshimDatabase: SqlshimDatabase | undefined;
  fetchingSqlshimDatabase: boolean;
  databaseIsDirty: boolean;
  creatingDatabase: boolean;

  // object store
  savingObjectStore: boolean;
  saveObjectStoreError: string;
  modifiedObjectStore: ObjectStore | undefined;
  currentObjectStore: ObjectStore | undefined;
  fetchingObjectStore: boolean;
  editingObjectStore: boolean;
  objectStores: PaginatedResult<ObjectStore> | undefined;
  objectStoreDeleteQueue: string[];

  // policy secrets
  modifiedPolicySecret: PolicySecret | undefined;
  savePolicySecretError: string;
  fetchingPolicySecrets: boolean;
  fetchPolicySecretsError: string;
  policySecrets: PaginatedResult<PolicySecret> | undefined;
  policySecretDeleteQueue: string[];

  // customer end users
  tenantUsersSelectedOrganizationID: string | undefined;
  fetchingUsers: boolean;
  fetchUsersError: string;
  userDeleteQueue: string[];
  userBulkSaveErrors: string[];
  savingUsers: boolean;
  currentTenantUser: UserProfileSerialized | undefined;
  currentTenantUserProfileEdited: Record<string, JSONValue>;
  currentTenantUserError: string;
  currentTenantUserConsentedPurposes: Array<object> | undefined;
  currentTenantUserConsentedPurposesError: string;
  currentTenantUserEvents: Array<UserEvent> | undefined;
  currentTenantUserEventsError: string;
  userEditMode: boolean;
  saveTenantUserError: string;

  // team (customer employee IAM)
  userRolesEditMode: boolean;
  companyUserRolesEditMode: boolean;
  fetchingUserRoles: boolean;
  fetchingCompanyUserRoles: boolean;
  fetchUserRolesError: string;
  fetchCompanyUserRolesError: string;
  companyUserRolesDeleteQueue: string[];
  userRolesBulkSaveErrors: string[];
  companyUserRolesBulkSaveErrors: string[];
  modifiedUserRoles: UserRoles[];
  modifiedCompanyUserRoles: UserRoles[];
  savingUserRoles: boolean;
  savingCompanyUserRoles: boolean;
  fetchingInvites: boolean;
  fetchInvitesError: string;

  // policies
  currentAccessPolicy: AccessPolicy | undefined;
  modifiedAccessPolicy: AccessPolicy | undefined;
  currentTokenAccessPolicy: AccessPolicy | undefined;
  modifiedTokenAccessPolicy: AccessPolicy | undefined;
  accessPolicyIsDirty: boolean;
  tokenAccessPolicyIsDirty: boolean;
  accessPolicyTestContext: string;
  accessPolicyTestParams: string;
  fetchingAccessPolicies: boolean;
  fetchingAllAccessPolicies: boolean;
  accessPolicyFetchError: string | undefined;
  allAccessPoliciesFetchError: string | undefined;
  tokenAccessPolicyFetchError: string;
  deletingAccessPolicy: boolean;
  accessPoliciesIncludeAutogenerated: boolean;
  accessPolicyDeleteSuccess: boolean;
  accessPolicyDeleteError: string;
  accessPoliciesDeleteQueue: string[];
  deleteAccessPoliciesSuccess: string;
  deleteAccessPoliciesErrors: string[];
  accessPolicyEditMode: boolean;
  accessPolicySearchFilter: Filter;
  accessPolicyDetailsEditMode: boolean;
  globalAccessorPolicy: AccessPolicy | undefined;
  fetchingGlobalAccessorPolicy: boolean;
  globalAccessorPolicyFetchError: string;

  // access policy templates
  fetchingPolicyTemplates: boolean;
  policyTemplatesFetchError: string;
  deletingPolicyTemplate: boolean;
  policyTemplateDeleteSuccess: boolean;
  policyTemplateDeleteError: string | undefined;
  policyTemplatesDeleteQueue: string[];
  deletePolicyTemplatesSuccess: string;
  deletePolicyTemplatesErrors: string[];
  selectedPolicyTemplate: AccessPolicyTemplate | undefined;
  policyTemplateToCreate: AccessPolicyTemplate | undefined;
  policyTemplateToModify: AccessPolicyTemplate | undefined;
  policyTemplateEditMode: boolean;
  savingPolicyTemplate: boolean;
  policyTemplateSaveSuccess: boolean;
  policyTemplateSaveError: string;
  accessPolicyTemplateSearchFilter: Filter;

  paginatedPolicyChooserComponent: AccessPolicyComponent | undefined;
  paginatedPolicyChooserSelectedResource:
    | AccessPolicy
    | AccessPolicyTemplate
    | undefined;
  policySelectorResourceType: PolicySelectorResourceType;
  policyChooserIsOpen: boolean;
  policyTemplateDialogIsOpen: boolean;

  currentTransformer: Transformer | undefined;
  modifiedTransformer: Transformer | undefined;
  transformerIsDirty: boolean;
  transformerTestData: string;
  fetchingTransformers: boolean;
  transformerFetchError: string | undefined;
  deletingTransformer: boolean;
  transformerDeleteSuccess: boolean;
  transformerDeleteError: string;
  transformersDeleteQueue: string[];
  deleteTransformersSuccess: string;
  deleteTransformersErrors: string[];
  transformerEditMode: boolean;
  transformerSearchFilter: Filter;
  transformerDetailsEditMode: boolean;
  globalMutatorPolicy: AccessPolicy | undefined;
  fetchingGlobalMutatorPolicy: boolean;
  globalMutatorPolicyFetchError: string;

  // these are shared between AP, token AP
  testingPolicy: boolean;
  testingPolicyResult: AccessPolicyTestResult | undefined;
  testingPolicyError: string;

  testingTransformer: boolean;
  testingTransformerResult: TransformerTestResult | undefined;
  testingTransformerError: string;

  savingAccessPolicy: boolean;
  saveAccessPolicyError: string;

  savingTransformer: boolean;
  saveTransformerSuccess: string;
  saveTransformerError: string;

  savingTokenAccessPolicy: boolean;
  saveTokenAccessPolicySuccess: string;
  saveTokenAccessPolicyError: string;

  // system log
  fetchingSystemLogEntries: boolean;
  fetchSystemLogEntriesError: string;
  systemLogSearchFilter: Filter;
  systemLogEntry: SystemLogEntry | undefined;
  fetchingSingleSystemLogEntry: boolean;
  fetchSingleSystemLogEntryError: string;
  fetchingSystemLogEntry: boolean;
  fetchSystemLogEntryError: string;
  systemLogEntryDetailSearchFilter: Filter;

  // audit log
  fetchingAuditLogEntries: boolean;
  fetchAuditLogEntriesError: string;
  auditLogSearchFilter: Filter;

  // data access log
  fetchingDataAccessLogEntries: boolean;
  fetchDataAccessLogEntriesError: string;
  dataAccessLogFilter: DataAccessLogFilter;

  // status page
  fetchingActiveInstances: boolean;
  activeInstancesFetchError: string;
  fetchingLogEvents: boolean;
  logEventsFetchError: string;
  fetchingChartData: boolean;
  chartDataFetchError: string;
  chartDataService: string;
  chartDataTimePeriod: string;

  // JWT signing keys
  fetchingPublicKeys: boolean;
  rotatingTenantKeys: boolean;
  fetchTenantPublicKeysError: string;
  rotateTenantKeysError: string;
}

// TODO:ksj: this hack is for running unit tests.
// need something better
let startingHref: string;
let startingQuery: string;
try {
  startingHref = window.location.href;
  startingQuery = window.location.search;
} catch {
  startingHref = 'https://console.dev.userclouds.tools:3010/?';
  startingQuery = '?';
}
const initialGlobalPersistedState = () => ({
  serviceInfo: undefined,
  myProfile: undefined,
  companies: undefined,
  globalCompanies: undefined,
});

const initialAppState = () => ({
  // service info
  serviceInfoError: undefined,

  // routing
  location: new URL(startingHref),
  query: new URLSearchParams(startingQuery),
  routeHandler: undefined,
  routeParams: {},
  routePattern: '/',

  // notifications
  notifications: [],

  // feature flags
  featureFlags: featureFlagsAreEnabled() ? undefined : {},
  fetchingFeatureFlags: false,
  featureFlagsFetchError: '',

  // companies
  creatingCompany: false,
  createCompanyError: '',
  fetchingCompanies: true,
  companyFetchError: '',
  fetchingAllCompanies: false,
  companiesFetchError: '',
  createCompanyDialogIsOpen: false,
  changeCompanyDialogIsOpen: false,
});

export const initialCompanyPersistedState = {
  tenants: undefined,
  teamUserRoles: undefined,
  companyUserRoles: undefined,
  companyInvites: undefined,
};

export const initialCompanyAppState = () => {
  return {
    selectedCompanyID: undefined,
    selectedCompany: undefined,
    selectedTenantID: undefined,
    selectedTenant: undefined,
  };
};

export const initialCompanyPageState = {
  // team (customer employee IAM)
  userRolesEditMode: false,
  companyUserRolesEditMode: false,
  fetchingUserRoles: false,
  fetchingCompanyUserRoles: false,
  fetchUserRolesError: '',
  fetchCompanyUserRolesError: '',
  companyUserRolesDeleteQueue: [],
  userRolesBulkSaveErrors: [],
  companyUserRolesBulkSaveErrors: [],
  modifiedUserRoles: [],
  modifiedCompanyUserRoles: [],
  savingUserRoles: false,
  savingCompanyUserRoles: false,
  fetchingInvites: false,
  fetchInvitesError: '',

  // create and list tenants
  creatingTenant: false,
  createTenantError: '',
  tenantCreationState: null,
  editingTenant: false,
  savingTenant: false,
  saveTenantSuccess: '',
  saveTenantError: '',
  fetchingSelectedTenant: false,
  deletingTenant: false,
  deleteTenantError: '',
  modifiedTenant: undefined,
  fetchingTenants: false,
  tenantFetchError: '',
  currentTenantURL: undefined,
  modifiedTenantURL: undefined,
  tenantURLIsDirty: false,
  tenantURLDialogIsOpen: false,
  creatingURL: false,
  modifiedTenantUrls: [],
  tenantURLsToCreate: [],
  tenantURLsToUpdate: [],
  tenantURLsToDelete: [],
  tenantDatabaseDialogIsOpen: false,
  tenantIssuerDialogIsOpen: false,
  tenantProviderName: '',
  creatingIssuer: false,
  editingIssuerIndex: -1,
  createTenantDialogIsOpen: false,
};

export const initialTenantPersistedState = {
  // tenant
  tenantURLs: undefined,

  // organizations
  organizations: undefined,
  selectedOrganization: undefined,
  modifiedOrganization: undefined,
  loginAppsForOrg: undefined,

  // authn / plex
  tenantPlexConfig: undefined,
  tenantEmailMessageElements: undefined,
  tenantSMSMessageElements: undefined,
  appPageParameters: {},
  tenantPublicKey: '',

  // authz
  displayEdges: undefined,
  objectTypes: undefined,
  edgeTypes: undefined,
  authzObjects: undefined,
  edgesForObject: undefined,
  objectsForEdgesForObject: undefined,
  currentObjectType: undefined,
  currentEdgeType: undefined,
  currentEdge: undefined,
  currentObject: undefined,
  object: undefined,

  // user store
  userStoreDisplayColumns: undefined,
  userStoreColumns: undefined,
  columnRetentionDurations: undefined,
  accessorsForColumn: undefined,
  accessorMetrics: undefined,
  accessors: undefined,
  selectedAccessor: undefined,
  mutators: undefined,
  selectedMutator: undefined,
  selectedColumn: undefined,
  dataTypes: undefined,
  selectedDataType: undefined,

  // sqlshim database
  sqlShimDatabases: undefined,

  // tags
  tags: [],

  // purposes
  purposes: undefined,
  selectedPurpose: undefined,

  // end users
  tenantUsersOrganizations: undefined,
  tenantUsers: undefined,
  currentTenantUser: undefined,

  // policies
  userPolicyPermissions: undefined,
  fetchingUserPolicyPermissions: false,
  userPolicyPermissionsFetchError: '',
  accessPolicyPermissions: undefined,
  transformerPermissions: undefined,
  accessPolicies: undefined,
  allAccessPolicies: undefined,
  policyTemplates: undefined,
  componentPolicies: undefined,
  transformers: undefined,
  currentAccessPolicy: blankAccessPolicy(),
  modifiedAccessPolicy: blankAccessPolicy(),
  currentTokenAccessPolicy: blankAccessPolicy(),
  modifiedTokenAccessPolicy: blankAccessPolicy(),
  selectedPolicyTemplate: undefined,
  currentTransformer: blankTransformer(),
  modifiedTransformer: undefined,

  // monitoring
  systemLogEntries: undefined,
  auditLogEntries: undefined,
  dataAccessLogEntries: undefined,
  dataAccessLogEntry: undefined,
  activeInstances: undefined,
  logEvents: undefined,
  chartData: undefined,
  systemLogEntry: undefined,
};

export const initialTenantPageState = {
  // tenant
  tenantURLsIsDirty: false,
  fetchingTenantURLs: false,
  fetchingTenantURLsError: '',
  savingTenantURLs: false,
  editingTenantURL: false,
  editingTenantURLError: '',
  creatingNewTenantURL: false,

  // organizations
  fetchingOrganizations: false,
  organizationsFetchError: '',
  editingOrganization: false,
  savingOrganization: false,
  createOrganizationError: '',
  updateOrganizationError: '',

  // authn / plex
  fetchingPlexConfig: false,
  fetchPlexConfigError: '',
  modifiedPlexConfig: undefined,
  selectedPlexApp: undefined,
  modifiedPlexApp: undefined,
  plexEmployeeApp: undefined,
  modifiedPlexEmployeeApp: undefined,
  selectedPlexProvider: undefined,
  oidcProvider: undefined,
  savingOIDCProvider: false,
  saveOIDCProviderSuccess: false,
  saveOIDCProviderError: '',
  modifiedPlexProvider: undefined,
  plexConfigIsDirty: false,
  savingPlexConfig: false,
  savePlexConfigSuccess: undefined,
  savePlexConfigError: '',
  editingPlexConfig: false,
  auth0AppsEditMode: false,
  cognitoAppsEditMode: false,
  ucAppsEditMode: false,
  fetchingLoginApps: false,
  loginAppsFetchError: '',
  modifiedEmailMessageElements: undefined,
  emailMessageElementsAreDirty: false,
  fetchingEmailMessageElements: false,
  emailMessageElementsFetchError: '',
  savingEmailMessageElements: false,
  emailMessageElementsSaveSuccess: false,
  emailMessageElementsSaveError: '',
  selectedEmailMessageType: MessageType.EmailInviteNew,
  modifiedSMSMessageElements: undefined,
  smsMessageElementsAreDirty: false,
  fetchingSMSMessageElements: false,
  smsMessageElementsFetchError: '',
  savingSMSMessageElements: false,
  smsMessageElementsSaveSuccess: false,
  smsMessageElementsSaveError: '',
  selectedSMSMessageType: MessageType.SMSMFAChallenge,
  modifiedPageParameters: undefined,
  fetchingPageParameters: false,
  pageParametersFetchError: '',
  savingPageParameters: false,
  pageParametersSaveSuccess: false,
  pageParametersSaveError: '',
  appToClone: '',

  // authz
  currentObjectSearchFilter: { columnName: '', operator: '', value: '' },
  currentEdgeSearchFilter: { columnName: '', operator: '', value: '' },
  objectTypeSearchFilter: { columnName: '', operator: '', value: '' },
  edgeTypeSearchFilter: { columnName: '', operator: '', value: '' },
  authorizationRequest: undefined,
  authorizationSuccess: '',
  authorizationFailure: '',
  authorizationPath: [],
  fetchObjectTypesError: '',
  fetchingObjectTypes: false,
  currentObjectTypeValid: true,
  objectTypeFetchError: '',
  deletingObjectType: undefined,
  objectTypeDeleteSuccess: '',
  objectTypeDeleteError: '',
  objectTypeEditMode: false,
  fetchObjectTypeError: '',
  savingObjectType: false,
  saveObjectTypeSuccess: '',
  saveObjectTypeError: '',
  objectTypeDeleteQueue: [],
  savingObjectTypes: false,

  currentEdgeTypeValid: true,
  edgeTypeIsDirty: false,
  edgeTypeFetchError: '',
  deletingEdgeType: undefined,
  edgeTypeDeleteSuccess: '',
  edgeTypeDeleteError: '',
  edgeTypeEditMode: false,
  fetchEdgeTypeError: '',
  savingEdgeType: false,
  saveEdgeTypeSuccess: '',
  saveEdgeTypeError: '',
  fetchEdgeTypesError: '',
  fetchingEdgeTypes: false,
  edgeTypeDeleteQueue: [],
  savingEdgeTypes: false,

  fetchingAuthzEdges: false,
  fetchEdgesError: '',
  editEdgesSuccess: '',
  editEdgesError: '',
  savingEdges: false,

  edgeIsDirty: false,
  deletingEdge: undefined,
  edgeDeleteSuccess: '',
  edgeDeleteError: '',
  edgeEditMode: false,
  fetchEdgeError: '',
  savingEdge: false,
  saveEdgeSuccess: '',
  saveEdgeError: '',
  edgeDeleteQueue: [],

  fetchingAuthzObjects: false,
  fetchObjectsError: '',
  lastFetchedObjectTypeID: '',
  fetchObjectError: '',
  fetchEdgesForObjectError: '',
  editingObjects: false,
  objectDeleteQueue: [],
  editObjectsSuccess: '',
  editObjectsError: '',
  savingObjects: false,
  objectIsDirty: false,
  objectFetchError: '',
  deletingObject: false,
  objectDeleteSuccess: '',
  objectDeleteError: '',
  objectEditMode: false,
  savingObject: false,
  saveObjectSuccess: '',
  saveObjectError: '',

  // datamapping
  dataSourceDetailsEditMode: false,
  modifiedDataSource: blankDataSource(),
  selectedDataSource: undefined,
  fetchDataSourceError: '',
  fetchingDataSources: false,
  dataSources: undefined,
  dataSourcesDeleteQueue: [],
  dataSourcesSearchFilter: { columnName: '', operator: '', value: '' },
  deletingDataSources: false,
  deleteDataSourceSuccess: '',
  deleteDataSourceError: '',
  bulkDeleteDataSourcesSuccess: '',
  bulkDeleteDataSourcesErrors: [],
  savingDataSource: false,
  dataSourceSaveSuccess: '',
  dataSourceSaveError: '',
  dataSourceElementDetailsEditMode: false,
  modifiedDataSourceElement: blankDataSourceElement(),
  selectedDataSourceElement: undefined,
  fetchDataSourceElementError: '',
  fetchingDataSourceElements: false,
  dataSourceElements: undefined,
  dataSourceElementsSearchFilter: { columnName: '', operator: '', value: '' },
  savingDataSourceElement: false,
  dataSourceElementSaveSuccess: '',
  dataSourceElementSaveError: '',

  // user store
  fetchingUserStoreConfig: false,
  fetchUserStoreConfigError: '',
  userStoreColumnsToAdd: [],
  userStoreColumnsToModify: {},
  userStoreColumnsToDelete: {},
  userStoreConfigIsDirty: false,
  savingUserStoreConfig: false,
  saveUserStoreConfigSuccess: '',
  saveUserStoreConfigErrors: [],
  userStoreEditMode: false,

  // user store column
  modifiedColumn: undefined,
  columnIsDirty: false,
  fetchingColumn: false,
  fetchingColumnError: '',
  savingColumn: false,
  savingColumnSuccess: '',
  saveColumnError: '',
  columnEditMode: false,
  columnPurposesEditMode: false,
  modifiedRetentionDurations: [],
  fetchingColumnRetentionDurations: false,
  columnDurationsFetchError: '',
  savingColumnRetentionDurations: false,
  retentionDurationsSaveSuccess: false,
  retentionDurationsSaveError: '',
  purposeSettingsAreDirty: false,
  columnSearchFilter: { columnName: '', operator: '', value: '' },

  // user store dataType
  fetchingDataTypes: false,
  fetchDataTypesError: '',
  dataTypesDeleteQueue: [],
  savingDataTypes: false,
  dataTypesEditMode: false,
  selectedDataType: undefined,
  dataTypeToCreate: blankDataType(),
  modifiedDataType: undefined,
  dataTypeIsDirty: false,
  fetchingDataType: false,
  fetchingDataTypeError: '',
  savingDataType: false,
  savingDataTypeSuccess: '',
  saveDataTypeError: '',
  dataTypeEditMode: false,
  dataTypeSearchFilter: { columnName: '', operator: '', value: '' },

  // user store accessors
  fetchingAccessors: false,
  fetchAccessorsError: '',
  selectedAccessorID: undefined,
  modifiedAccessor: undefined,
  modifiedAccessorIsDirty: false,
  accessorToCreate: blankAccessor(),
  savingAccessor: false,
  saveAccessorSuccess: '',
  saveAccessorError: '',
  accessorListEditMode: false,
  accessorListIncludeAutogenerated: true,
  updatingAccessors: false,
  bulkUpdateAccessorsSuccess: '',
  bulkUpdateAccessorsErrors: [],
  accessorsToDelete: {},
  accessorDetailsEditMode: false,
  accessorColumnsEditMode: false,
  accessorSelectorEditMode: false,
  accessorColumnsToAdd: [],
  accessorColumnsToDelete: {},
  accessorAddColumnDropdownValue: '',
  accessorPoliciesEditMode: false,
  saveAccessorColumnsSuccess: '',
  saveAccessorColumnsError: '',
  selectedAccessPolicyForAccessor: '',
  selectedTokenAccessPolicyForAccessor: '',
  saveAccessorPoliciesSuccess: '',
  saveAccessorPoliciesError: '',
  createAccessorError: '',
  accessorSearchFilter: { columnName: '', operator: '', value: '' },
  executeAccessorContext: '{}',
  executeAccessorSelectorValues: '[""]',
  executeAccessorResponse: undefined,
  executeAccessorError: '',
  executeAccessorStats: undefined,
  fetchingAccessorMetrics: false,
  fetchAccessorMetricsError: '',

  // user store mutators
  fetchingMutators: false,
  fetchMutatorsError: '',
  selectedMutatorID: undefined,
  modifiedMutator: undefined,
  modifiedMutatorIsDirty: false,
  mutatorToCreate: blankMutator(),
  savingMutator: false,
  savingMutatorSuccess: '',
  savingMutatorError: '',
  mutatorListEditMode: false,
  updatingMutators: false,
  bulkUpdateMutatorsSuccess: '',
  bulkUpdateMutatorsErrors: [],
  mutatorsToDelete: {},
  mutatorDetailsEditMode: false,
  mutatorColumnsEditMode: false,
  mutatorSelectorEditMode: false,
  mutatorColumnsToAdd: [],
  mutatorColumnsToDelete: {},
  mutatorAddColumnDropdownValue: '',
  mutatorPoliciesEditMode: false,
  saveMutatorColumnsSuccess: '',
  saveMutatorColumnsError: '',
  selectedAccessPolicyForMutator: '',
  saveMutatorPoliciesSuccess: '',
  saveMutatorPoliciesError: '',
  createMutatorError: '',
  mutatorSearchFilter: { columnName: '', operator: '', value: '' },

  // Tags
  fetchingTags: false,
  fetchTagsError: '',
  savingTags: false,
  savingTagsSuccess: '',
  savingTagsError: '',

  // purposes
  modifiedPurpose: undefined,
  fetchingPurposes: false,
  purposesFetchError: '',
  creatingPurpose: false,
  createPurposeError: '',
  savingPurpose: false,
  savePurposeError: '',
  deletingPurpose: false,
  deletePurposeError: '',
  purposeDetailsEditMode: false,
  purposesBulkEditMode: false,
  purposesDeleteQueue: [],
  deletePurposesSuccess: '',
  deletePurposesErrors: [],
  purposesBulkSaving: false,
  purposeSearchFilter: { columnName: '', operator: '', value: '' },

  // sqlshim database
  modifiedSqlShimDatabases: [],
  savingSqlshimDatabase: false,
  saveSqlshimDatabaseSuccess: '',
  saveSqlshimDatabaseError: '',
  testingSqlshimDatabase: false,
  testSqlshimDatabaseSuccess: false,
  testSqlshimDatabaseError: '',
  modifiedSqlshimDatabase: undefined,
  currentSqlshimDatabase: undefined,
  fetchingSqlshimDatabase: false,
  databaseIsDirty: false,
  creatingDatabase: false,

  // object store
  savingObjectStore: false,
  saveObjectStoreError: '',
  modifiedObjectStore: undefined,
  currentObjectStore: undefined,
  fetchingObjectStore: false,
  editingObjectStore: false,
  objectStores: undefined,
  objectStoreDeleteQueue: [],

  // policy secrets
  modifiedPolicySecret: undefined,
  savePolicySecretError: '',
  fetchingPolicySecrets: false,
  fetchPolicySecretsError: '',
  policySecrets: undefined,
  policySecretDeleteQueue: [],

  // customer end users
  tenantUsersSelectedOrganizationID: undefined,
  fetchingUsers: false,
  fetchUsersError: '',
  userDeleteQueue: [],
  userBulkSaveErrors: [],
  savingUsers: false,
  currentTenantUserProfileEdited: {},
  currentTenantUserError: '',
  currentTenantUserConsentedPurposes: undefined,
  currentTenantUserConsentedPurposesError: '',
  currentTenantUserEvents: undefined,
  currentTenantUserEventsError: '',
  userEditMode: false,
  saveTenantUserError: '',

  // policies
  accessPolicyIsDirty: false,
  tokenAccessPolicyIsDirty: false,
  accessPolicyTestContext: `// Context changes on a per-resolution basis
{
  "server": {
    "ip_address": "127.0.0.1",
    "claims": {
      "sub": "bob"
    },
    "action": "resolve"
  },
  "client": {
    "purpose": "marketing"
  },
  "user": {
    "name": "Jane Doe",
    "email": "jane@doe.com",
    "phone": "+15703211564"
  }
}`,
  accessPolicyTestParams: '{}',
  fetchingAccessPolicies: false,
  fetchingAllAccessPolicies: false,
  accessPolicyFetchError: undefined,
  allAccessPoliciesFetchError: undefined,
  tokenAccessPolicyFetchError: '',
  deletingAccessPolicy: false,
  accessPoliciesIncludeAutogenerated: false,
  accessPolicyDeleteSuccess: false,
  accessPolicyDeleteError: '',
  accessPoliciesDeleteQueue: [],
  deleteAccessPoliciesSuccess: '',
  deleteAccessPoliciesErrors: [],
  accessPolicyEditMode: false,
  accessPolicySearchFilter: { columnName: '', operator: '', value: '' },
  accessPolicyDetailsEditMode: false,
  globalAccessorPolicy: undefined,
  fetchingGlobalAccessorPolicy: false,
  globalAccessorPolicyFetchError: '',

  // access policy templates
  fetchingPolicyTemplates: false,
  policyTemplatesFetchError: '',
  deletingPolicyTemplate: false,
  policyTemplateDeleteSuccess: false,
  policyTemplateDeleteError: undefined,
  policyTemplatesDeleteQueue: [],
  deletePolicyTemplatesSuccess: '',
  deletePolicyTemplatesErrors: [],
  policyTemplateToCreate: undefined,
  policyTemplateToModify: undefined,
  policyTemplateEditMode: false,
  savingPolicyTemplate: false,
  policyTemplateSaveSuccess: false,
  policyTemplateSaveError: '',
  accessPolicyTemplateSearchFilter: { columnName: '', operator: '', value: '' },

  paginatedPolicyChooserComponent: undefined,
  paginatedPolicyChooserSelectedResource: undefined,
  policySelectorResourceType: PolicySelectorResourceType.POLICY,
  policyChooserIsOpen: false,
  policyTemplateDialogIsOpen: false,

  transformerIsDirty: false,
  transformerTestData: 'secret data goes here',
  fetchingTransformers: false,
  transformerFetchError: undefined,
  deletingTransformer: false,
  transformerDeleteSuccess: false,
  transformerDeleteError: '',
  transformersDeleteQueue: [],
  deleteTransformersSuccess: '',
  deleteTransformersErrors: [],
  transformerEditMode: false,
  transformerSearchFilter: { columnName: '', operator: '', value: '' },
  transformerDetailsEditMode: false,
  globalMutatorPolicy: undefined,
  fetchingGlobalMutatorPolicy: false,
  globalMutatorPolicyFetchError: '',

  // these are shared by access policies, token access policies
  testingPolicy: false,
  testingPolicyResult: undefined,
  testingPolicyError: '',

  testingTransformer: false,
  testingTransformerResult: undefined,
  testingTransformerError: '',

  savingAccessPolicy: false,
  saveAccessPolicyError: '',

  savingTransformer: false,
  saveTransformerSuccess: '',
  saveTransformerError: '',

  savingTokenAccessPolicy: false,
  saveTokenAccessPolicySuccess: '',
  saveTokenAccessPolicyError: '',

  // system log
  fetchingSystemLogEntries: false,
  fetchSystemLogEntriesError: '',
  systemLogSearchFilter: { columnName: '', operator: '', value: '' },
  fetchingSingleSystemLogEntry: false,
  fetchSingleSystemLogEntryError: '',
  systemLogEntryRecords: undefined,
  fetchingSystemLogEntry: false,
  fetchSystemLogEntryError: '',
  systemLogEntryDetailSearchFilter: { columnName: '', operator: '', value: '' },

  // audit log
  fetchingAuditLogEntries: false,
  fetchAuditLogEntriesError: '',
  auditLogSearchFilter: { columnName: '', operator: '', value: '' },

  // data access log
  fetchingDataAccessLogEntries: false,
  fetchDataAccessLogEntriesError: '',
  dataAccessLogFilter: {
    column_id: '',
    accessor_id: '',
    actor_id: '',
    selector_value: '',
  },

  // status page
  fetchingActiveInstances: false,
  activeInstancesFetchError: '',
  fetchingLogEvents: false,
  logEventsFetchError: '',
  fetchingChartData: false,
  chartDataFetchError: '',
  chartDataService: 'plex',
  chartDataTimePeriod: 'day',

  // JWT signing keys
  fetchingPublicKeys: false,
  rotatingTenantKeys: false,
  fetchTenantPublicKeysError: '',
  rotateTenantKeysError: '',
};

export const initialState: RootState = {
  ...initialGlobalPersistedState(),
  ...initialAppState(),
  ...initialCompanyPersistedState,
  ...initialCompanyAppState(),
  ...initialCompanyPageState,
  ...initialTenantPersistedState,
  ...initialTenantPageState,
};

const store = configureStore({
  reducer: rootReducer,
  middleware: [thunk],
  preloadedState: initialState,
});

export type AppThunk<ReturnType = void> = ThunkAction<
  ReturnType,
  RootState,
  unknown,
  AnyAction
>;
export type AppDispatch = ThunkDispatch<RootState, unknown, AnyAction>;

export default store;
