import { useEffect } from 'react';

import routeMatcher from './routeMatcher';
import { AppDispatch } from './store';
import { navigate } from './actions/routing';
import UCAdminOnlyRoute from './AdminOnlyRoute';
import TenantAdminOnlyRoute from './TenantAdminOnlyRoute';
import TenantSpecificRoute from './TenantSpecificRoute';
import OrganizationDetailsPage from './pages/OrganizationDetailsPage';
import HomePage from './pages/HomePage';
import UsersPage from './pages/UsersPage';
import UserDetailPage from './pages/UserDetailPage';
import IAMPage from './pages/IAMPage';
import GlobalIAMPage from './pages/GlobalIAMPage';
import OrgsPage from './pages/OrgsPage';
import CreateOrgPage from './pages/CreateOrgPage';
import StatusPage from './pages/StatusPage';
import EventsLogPage from './pages/EventsLogPage';
import ResourceListPage from './pages/ResourceListPage';
import CreateTenantPage from './pages/CreateTenantPage';
import TenantDetailsPage from './pages/TenantDetailsPage';
import CreateAccessorPage from './pages/CreateAccessorPage';
import AccessorDetailPage from './pages/AccessorDetailPage';
import MutatorDetailPage from './pages/MutatorDetailPage';
import CreateMutatorPage from './pages/CreateMutatorPage';
import PurposePage from './pages/PurposePage';
import AuthZPage from './pages/AuthZPage';
import EdgePage from './pages/EdgePage';
import ObjectPage from './pages/ObjectPage';
import ObjectTypePage from './pages/ObjectTypePage';
import EdgeTypePage from './pages/EdgeTypePage';
import PlexAppPage from './pages/PlexAppPage';
import PlexEmployeeAppPage from './pages/PlexEmployeeAppPage';
import PlexProviderPage from './pages/PlexProviderPage';
import AuditLogPage from './pages/AuditLogPage';
import SystemLogPage from './pages/SystemLogPage';
import AccessPolicyPage from './pages/AccessPolicyDetailPage';
import TransformerPage from './pages/TransformerDetailPage';
import CreatePolicyTemplatePage from './pages/CreatePolicyTemplatePage';
import PolicyTemplateDetailsPage from './pages/PolicyTemplateDetailsPage';
import SystemEventDetailPage from './pages/SystemEventDetailPage';
import AuthNEditOIDCPage from './pages/AuthNEditOIDCPage';
import ColumnPage from './pages/ColumnPage';
import ColumnsPage from './pages/ColumnsPage';
import AccessorsPage from './pages/AccessorsPage';
import MutatorsPage from './pages/MutatorsPage';
import PurposesPage from './pages/PurposesPage';
import TransformersPage from './pages/TransformersPage';
import AccessPoliciesPage from './pages/AccessPoliciesPage';
import PolicyTemplatesPage from './pages/PolicyTemplatesPage';
import ConnectedObjectsPage from './pages/ObjectsPage';
import ConnectedEdgesPage from './pages/EdgesPage';
import ConnectedEdgeTypesPage from './pages/EdgeTypesPage';
import LoginAppsPage from './pages/LoginAppsPage';
import IdentityProvidersPage from './pages/IdentityProvidersPage';
import OAuthConnectionsPage from './pages/OAuthConnectionsPage';
import CommsChannelsPage from './pages/CommsChannelsPage';
import ConnectedObjectTypesPage from './pages/ObjectTypesPage';
import DataTypesPage from './pages/DataTypesPage';
import DataSourcesPage from './pages/DataSourcesPage';
import DataSourcePage from './pages/DataSourcePage';
import DataSourceSchemasPage from './pages/DataSourceSchemasPage';
import DataSourceElementPage from './pages/DataSourceElementPage';
import DataAccessLogPage from './pages/DataAccessLogPage';
import DataAccessLogDetailsPage from './pages/DataAccessLogDetailsPage';
import CreateDataTypePage from './pages/CreateDataTypePage';
import DataTypeDetailPage from './pages/DataTypeDetailPage';
import ObjectStoresPage from './pages/ObjectStoresPage';
import ObjectStoreDetailsPage from './pages/ObjectStoreDetailsPage';
import SecretsPage from './pages/SecretsPage';
import { updateLastViewedTenantFromURL } from './util/localStorageTenant';

export const HOME_PATH = '/';
export const GLOBAL_IAM_PATH = '/globaliam';
export const ORGANIZATIONS_PATH = '/organizations';
export const ORGANIZATIONS_ORG_PATH = '/organizations/:orgID';
export const ORGANIZATIONS_CREATE_PATH = '/organizations/create';
export const TENANTS_CREATE_PATH = '/tenants/create';
export const TENANTS_DETAILS_PATH = '/tenants/:tenantID';
export const STATUS_PATH = '/status';
export const EVENTS_LOG_PATH = '/events';
export const RESOURCE_LIST_PATH = '/resources';
export const COLUMNS_PATH = '/columns';
export const DATA_TYPES_PATH = '/datatypes';
export const DATA_TYPE_CREATE_PATH = '/datatypes/create';
export const DATA_TYPE_DETAILS_PATH = '/datatypes/:datatypeID';
export const COLUMNS_CREATE_PATH = '/columns/create';
export const COLUMNS_DETAILS_PATH = '/columns/:columnID';
export const OBJECT_STORES_PATH = '/object_stores';
export const OBJECT_STORE_DETAILS_PATH = '/object_stores/:objectStoreID';
export const POLICY_SECRETS_PATH = '/secrets';
export const ACCESSORS_PATH = '/accessors';
export const ACCESSORS_CREATE_PATH = '/accessors/create';
export const ACCESSORS_DETAILS_VERSION_PATH = '/accessors/:accessorID/:version';
export const MUTATORS_PATH = '/mutators';
export const MUTATORS_DETAILS_VERSION_PATH = '/mutators/:mutatorID/:version';
export const MUTATORS_CREATE_PATH = '/mutators/create';
export const PURPOSES_PATH = '/purposes';
export const PURPOSES_DETAILS_PATH = '/purposes/:purposeID';
export const PURPOSES_CREATE_PATH = '/purposes/create';
export const TRANSFORMERS_PATH = '/transformers';
export const TRANSFORMERS_CREATE_PATH = '/transformers/create';
export const TRANSFORMERS_POLICY_DETAILS_VERSION_PATH =
  '/transformers/:transformerID/:version';
export const ACCESSPOLICIES_PATH = '/accesspolicies';
export const ACCESSPOLICIES_CREATE_PATH = '/accesspolicies/create';
export const ACCESSPOLICIES_DETAILS_VERSION_PATH =
  '/accesspolicies/:policyID/:version';
export const POLICYTEMPLATES_PATH = '/policytemplates';
export const POLICYTEMPLATES_CREATE_PATH = '/policytemplates/create';
export const POLICYTEMPLATES_DETAILS_VERSION_PATH =
  '/policytemplates/:templateID/:version';
export const OBJECTS_PATH = '/objects';
export const OBJECTS_DETAILS_PATH = '/objects/:objectID';
export const OBJECTS_CREATE_PATH = '/objects/create';
export const EDGES_PATH = '/edges';
export const EDGES_DETAILS_PATH = '/edges/:edgeID';
export const EDGES_CREATE_PATH = '/edges/create';
export const EDGETYPES_PATH = '/edgetypes';
export const EDGETYPES_DETAILS_PATH = '/edgetypes/:edgeTypeID';
export const EDGETYPES_CREATE_PATH = '/edgetypes/create';
export const OBJECTTYPES_PATH = '/objecttypes';
export const OBJECTTYPES_DETAILS_PATH = '/objecttypes/:objectTypeID';
export const OBJECTTYPES_CREATE_PATH = '/objecttypes/create';
export const LOGINAPPS_PATH = '/loginapps';
export const IDENTITYPROVIDERS_PATH = '/identityproviders';
export const OAUTHCONNECTIONS_PATH = '/oauthconnections';
export const COMMCHANNELS_PATH = '/commschannels';
export const TENANTS_USERSTORE_PATH = '/tenants/userstore';
export const TENANTS_USERSTORE_COLUMNS_CREATE_PATH =
  '/tenants/userstore/columns/create';
export const TENANTS_USERSTORE_COLUMNS_DETAILS_PATH =
  '/tenants/userstore/columns/:columnID';
export const TENANTS_USERSTORE_ACCESSORS_CREATE_PATH =
  '/tenants/userstore/accessors/create';
export const TENANTS_USERSTORE_ACCESSORS_DETAILS_VERSION_PATH =
  '/tenants/userstore/accessors/:accessorID/:version';
export const TENANTS_USERSTORE_MUTATORS_DETAILS_VERSION_PATH =
  '/tenants/userstore/mutators/:mutatorID/:version';
export const TENANTS_USERSTORE_MUTATORS_CREATE_PATH =
  '/tenants/userstore/mutators/create';
export const TENANTS_USERSTORE_PURPOSES_DETAILS_PATH =
  '/tenants/userstore/purposes/:purposeID';
export const TENANTS_USERSTORE_PURPOSES_CREATE_PATH =
  '/tenants/userstore/purposes/create';
export const AUTHZ_PATH = '/authz';
export const AUTHN_PATH = '/authn';
export const IDENTITYPROVIDERS_PLEX_PROVIDER_DETAILS_PATH =
  '/identityproviders/plex_provider/:plexProviderID';
export const OAUTHCONNECTIONS_OIDC_PROVIDER_CREATE_PATH =
  '/oauthconnections/oidc_provider/create';
export const OAUTHCONNECTIONS_OIDC_PROVIDER_NAME_PATH =
  '/oauthconnections/oidc_provider/:oidcProviderName';
export const LOGINAPPS_PLEX_APP_DETAILS_PATH = '/loginapps/:plexAppID';
export const LOGINAPPS_PLEX_EMPLOYEE_APP_PATH = '/loginapps/plex_employee_app';
export const USERS_PATH = '/users';
export const USERS_DETAILS_PATH = '/users/:userID';
export const IAM_PATH = '/iam';
export const AUDITLOG_PATH = '/auditlog';
export const DATAACCESSLOG_PATH = '/dataaccesslog';
export const DATAACCESSLOG_DETAILS_PATH = '/dataaccesslog/:entryID';
export const SYSTEMLOG_PATH = '/systemlog';
export const SYSTEMLOG_RUN_DETAILS_PATH = '/systemlog/:runID';
export const DATASOURCES_PATH = '/datasources';
export const DATASOURCE_CREATE_PATH = '/datasources/create';
export const DATASOURCE_DETAILS_PATH = '/datasources/:dataSourceID';
export const DATASOURCESCHEMAS_PATH = '/datasourceschemas';
export const DATASOURCEELEMENT_DETAILS_PATH = '/datasourceschemas/:elementID';

export const redirect = (href: string, replace: boolean = false) => {
  const scrollY = document.getElementById('pageContent')?.scrollTop;

  if (replace) {
    window.history.replaceState({ scrollY: 0 }, '', href);
  } else {
    window.history.replaceState({ scrollY }, '', window.location.href);
    window.history.pushState({ scrollY: 0 }, '', href);
  }

  updateLastViewedTenantFromURL(href);

  const popStateEvent = new PopStateEvent('popstate', { state: { scrollY } });
  dispatchEvent(popStateEvent);
};

export const handleRoute = (href: string) => (dispatch: AppDispatch) => {
  const location = new URL(href);
  const result = router.match(location);
  if (result) {
    const { handler, pattern, params } = result;
    dispatch(
      navigate(location, handler as unknown as Function, pattern, params)
    );
  }
};

export const startListening = (dispatch: AppDispatch) => {
  updateLastViewedTenantFromURL(window.location.href);

  dispatch(handleRoute(window.location.href));

  window.addEventListener('popstate', () => {
    updateLastViewedTenantFromURL(window.location.href);
    dispatch(handleRoute(window.location.href));
  });
};

export const Navigate = ({
  to,
  replace,
}: {
  to: string;
  replace?: boolean;
}) => {
  replace = replace || false;
  useEffect(() => redirect(to, replace), [to, replace]);

  return null;
};

const router = routeMatcher([
  {
    path: HOME_PATH,
    handler: () => <HomePage />,
  },
  {
    path: GLOBAL_IAM_PATH,
    handler: () => (
      <UCAdminOnlyRoute>
        <GlobalIAMPage />
      </UCAdminOnlyRoute>
    ),
  },
  {
    path: ORGANIZATIONS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <OrgsPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: ORGANIZATIONS_ORG_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <OrganizationDetailsPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: ORGANIZATIONS_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <CreateOrgPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: TENANTS_CREATE_PATH,
    handler: () => <CreateTenantPage />,
  },
  {
    path: TENANTS_DETAILS_PATH,
    handler: () => <TenantDetailsPage />,
  },
  {
    path: STATUS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <StatusPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: EVENTS_LOG_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <EventsLogPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: RESOURCE_LIST_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <ResourceListPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: OBJECT_STORES_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <ObjectStoresPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: OBJECT_STORE_DETAILS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <ObjectStoreDetailsPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: POLICY_SECRETS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <SecretsPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: COLUMNS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <ColumnsPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: DATA_TYPES_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <DataTypesPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: DATA_TYPE_CREATE_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <CreateDataTypePage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: DATA_TYPE_DETAILS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <DataTypeDetailPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: COLUMNS_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <ColumnPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: COLUMNS_DETAILS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <ColumnPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: ACCESSORS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <AccessorsPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: ACCESSORS_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <CreateAccessorPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: ACCESSORS_DETAILS_VERSION_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <AccessorDetailPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: MUTATORS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <MutatorsPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: MUTATORS_DETAILS_VERSION_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <MutatorDetailPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: MUTATORS_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <CreateMutatorPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: PURPOSES_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <PurposesPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: PURPOSES_DETAILS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <PurposePage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: PURPOSES_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <PurposePage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: TRANSFORMERS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <TransformersPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: TRANSFORMERS_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <TransformerPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: TRANSFORMERS_POLICY_DETAILS_VERSION_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <TransformerPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: ACCESSPOLICIES_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <AccessPoliciesPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: ACCESSPOLICIES_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <AccessPolicyPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: ACCESSPOLICIES_DETAILS_VERSION_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <AccessPolicyPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: POLICYTEMPLATES_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <PolicyTemplatesPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: POLICYTEMPLATES_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <CreatePolicyTemplatePage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: POLICYTEMPLATES_DETAILS_VERSION_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <PolicyTemplateDetailsPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: OBJECTS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <ConnectedObjectsPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: OBJECTS_DETAILS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <ObjectPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: OBJECTS_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <ObjectPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: EDGES_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <ConnectedEdgesPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: EDGES_DETAILS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <EdgePage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: EDGES_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <EdgePage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: EDGETYPES_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <ConnectedEdgeTypesPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: EDGETYPES_DETAILS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <EdgeTypePage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: EDGETYPES_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <EdgeTypePage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: OBJECTTYPES_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <ConnectedObjectTypesPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: OBJECTTYPES_DETAILS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <ObjectTypePage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: OBJECTTYPES_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <ObjectTypePage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: LOGINAPPS_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <LoginAppsPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: IDENTITYPROVIDERS_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <IdentityProvidersPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: OAUTHCONNECTIONS_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <OAuthConnectionsPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: COMMCHANNELS_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <CommsChannelsPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: TENANTS_USERSTORE_COLUMNS_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <ColumnPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: TENANTS_USERSTORE_COLUMNS_DETAILS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <ColumnPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: TENANTS_USERSTORE_ACCESSORS_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <CreateAccessorPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: TENANTS_USERSTORE_ACCESSORS_DETAILS_VERSION_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <AccessorDetailPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: TENANTS_USERSTORE_MUTATORS_DETAILS_VERSION_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <MutatorDetailPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: TENANTS_USERSTORE_MUTATORS_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <CreateMutatorPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: AUTHZ_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <AuthZPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: AUTHN_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <LoginAppsPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: IDENTITYPROVIDERS_PLEX_PROVIDER_DETAILS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <PlexProviderPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: OAUTHCONNECTIONS_OIDC_PROVIDER_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <AuthNEditOIDCPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: OAUTHCONNECTIONS_OIDC_PROVIDER_NAME_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <AuthNEditOIDCPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: LOGINAPPS_PLEX_APP_DETAILS_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <PlexAppPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: LOGINAPPS_PLEX_EMPLOYEE_APP_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <PlexEmployeeAppPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: USERS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <UsersPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: USERS_DETAILS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <UserDetailPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: IAM_PATH,
    handler: () => <IAMPage />,
  },
  {
    path: AUDITLOG_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <AuditLogPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: DATAACCESSLOG_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <DataAccessLogPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: DATAACCESSLOG_DETAILS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <DataAccessLogDetailsPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: SYSTEMLOG_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <SystemLogPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: SYSTEMLOG_RUN_DETAILS_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <SystemEventDetailPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: DATASOURCES_PATH,
    handler: () => (
      <TenantSpecificRoute>
        <DataSourcesPage />
      </TenantSpecificRoute>
    ),
  },
  {
    path: DATASOURCE_CREATE_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <DataSourcePage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: DATASOURCE_DETAILS_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <DataSourcePage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: DATASOURCESCHEMAS_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <DataSourceSchemasPage />
      </TenantAdminOnlyRoute>
    ),
  },
  {
    path: DATASOURCEELEMENT_DETAILS_PATH,
    handler: () => (
      <TenantAdminOnlyRoute>
        <DataSourceElementPage />
      </TenantAdminOnlyRoute>
    ),
  },
]);

export default router;
