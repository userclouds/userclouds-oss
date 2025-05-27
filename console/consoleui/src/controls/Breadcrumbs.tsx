import { connect } from 'react-redux';

import { RootState } from '../store';

import Link from './Link';
import { makeCleanPageLink } from '../AppNavigation';
import {
  ACCESSORS_CREATE_PATH,
  ACCESSORS_DETAILS_VERSION_PATH,
  ACCESSORS_PATH,
  ACCESSPOLICIES_CREATE_PATH,
  ACCESSPOLICIES_DETAILS_VERSION_PATH,
  ACCESSPOLICIES_PATH,
  AUDITLOG_PATH,
  COLUMNS_CREATE_PATH,
  COLUMNS_DETAILS_PATH,
  COLUMNS_PATH,
  COMMCHANNELS_PATH,
  DATAACCESSLOG_PATH,
  DATAACCESSLOG_DETAILS_PATH,
  DATASOURCES_PATH,
  DATASOURCE_CREATE_PATH,
  DATASOURCE_DETAILS_PATH,
  DATASOURCEELEMENT_DETAILS_PATH,
  DATASOURCESCHEMAS_PATH,
  DATA_TYPES_PATH,
  DATA_TYPE_CREATE_PATH,
  DATA_TYPE_DETAILS_PATH,
  EDGES_CREATE_PATH,
  EDGES_DETAILS_PATH,
  EDGES_PATH,
  EDGETYPES_CREATE_PATH,
  EDGETYPES_DETAILS_PATH,
  EDGETYPES_PATH,
  GLOBAL_IAM_PATH,
  HOME_PATH,
  IAM_PATH,
  IDENTITYPROVIDERS_PATH,
  IDENTITYPROVIDERS_PLEX_PROVIDER_DETAILS_PATH,
  LOGINAPPS_PATH,
  LOGINAPPS_PLEX_APP_DETAILS_PATH,
  LOGINAPPS_PLEX_EMPLOYEE_APP_PATH,
  MUTATORS_CREATE_PATH,
  MUTATORS_DETAILS_VERSION_PATH,
  MUTATORS_PATH,
  OAUTHCONNECTIONS_OIDC_PROVIDER_CREATE_PATH,
  OAUTHCONNECTIONS_OIDC_PROVIDER_NAME_PATH,
  OAUTHCONNECTIONS_PATH,
  OBJECTS_CREATE_PATH,
  OBJECTS_DETAILS_PATH,
  OBJECTS_PATH,
  OBJECTTYPES_CREATE_PATH,
  OBJECTTYPES_DETAILS_PATH,
  OBJECTTYPES_PATH,
  ORGANIZATIONS_CREATE_PATH,
  ORGANIZATIONS_ORG_PATH,
  ORGANIZATIONS_PATH,
  POLICYTEMPLATES_CREATE_PATH,
  POLICYTEMPLATES_DETAILS_VERSION_PATH,
  POLICYTEMPLATES_PATH,
  PURPOSES_CREATE_PATH,
  PURPOSES_DETAILS_PATH,
  PURPOSES_PATH,
  STATUS_PATH,
  SYSTEMLOG_PATH,
  SYSTEMLOG_RUN_DETAILS_PATH,
  TENANTS_CREATE_PATH,
  TENANTS_DETAILS_PATH,
  TRANSFORMERS_CREATE_PATH,
  TRANSFORMERS_PATH,
  TRANSFORMERS_POLICY_DETAILS_VERSION_PATH,
  USERS_DETAILS_PATH,
  USERS_PATH,
  OBJECT_STORES_PATH,
  OBJECT_STORE_DETAILS_PATH,
  EVENTS_LOG_PATH,
  RESOURCE_LIST_PATH,
} from '../routing';
import style from './Breadcrumbs.module.css';

const Breadcrumbs = ({
  location,
  query,
  routeParams,
  pattern,
}: {
  location: URL;
  query: URLSearchParams;
  routeParams: Record<string, string>;
  pattern: string;
}) => {
  const cleanQuery = makeCleanPageLink(query);
  const { pathname } = location;
  let bcLinks: JSX.Element[] = [];
  switch (pattern) {
    case HOME_PATH:
      bcLinks = [
        <Link
          applyStyles={false}
          key="Home"
          title="Home"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Home
        </Link>,
      ];
      break;
    case GLOBAL_IAM_PATH:
      bcLinks = [
        <span key="Manage">Manage</span>,
        <Link
          applyStyles={false}
          key="[dev] Global IAM"
          title="[dev] Global IAM"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          [dev] Global IAM
        </Link>,
      ];
      break;
    case ORGANIZATIONS_PATH:
      bcLinks = [
        <span key="Manage">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="Organizations"
          title="Manage organizations"
          href={`${pathname}${cleanQuery}`}
          className={style.currentLink}
        >
          Organizations
        </Link>,
      ];

      break;
    case ORGANIZATIONS_ORG_PATH:
      bcLinks = [
        <span key="Manage">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="Organizations"
          title="Manage organizations"
          href={`/organizations${cleanQuery}`}
        >
          Organizations
        </Link>,
        <Link
          applyStyles={false}
          key="OrgDetails"
          title="Organization details"
          href={`/organizations/${routeParams.orgID}${cleanQuery}`}
          className={style.currentLink}
        >
          Organization details
        </Link>,
      ];
      break;
    case ORGANIZATIONS_CREATE_PATH:
      bcLinks = [
        <span key="Manage">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="Organizations"
          title="Manage organizations"
          href={`/organizations${cleanQuery}`}
        >
          Organizations
        </Link>,
        <Link
          applyStyles={false}
          key="CreateOrg"
          title="Create organization"
          href={`/organizations/create${cleanQuery}`}
          className={style.currentLink}
        >
          Create organization
        </Link>,
      ];
      break;
    case TENANTS_CREATE_PATH:
      bcLinks = [
        <Link
          applyStyles={false}
          key="Create Tenant"
          title="Create Tenant"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Create Tenant
        </Link>,
      ];
      break;
    case TENANTS_DETAILS_PATH:
      bcLinks = [
        <Link
          applyStyles={false}
          key="Tenant Details"
          title="Tenant Details"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Tenant Details
        </Link>,
      ];
      break;
    case STATUS_PATH:
      bcLinks = [
        <span key="Monitor">Monitor</span>,
        <Link
          applyStyles={false}
          key="Status"
          title="Status"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Status
        </Link>,
      ];
      break;
    case EVENTS_LOG_PATH:
      bcLinks = [
        <span key="Monitor">Monitor</span>,
        <Link
          applyStyles={false}
          key="Events Log"
          title="Events Log"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Events Log
        </Link>,
      ];
      break;
    case RESOURCE_LIST_PATH:
      bcLinks = [
        <span key="Monitor">Monitor</span>,
        <Link
          applyStyles={false}
          key="Resources"
          title="Resources"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Resources
        </Link>,
      ];
      break;
    case COLUMNS_PATH:
      bcLinks = [
        <span key="User Data Storage">User Data Storage</span>,
        <Link
          applyStyles={false}
          key="columns"
          title="Columns"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Columns
        </Link>,
      ];
      break;
    case COLUMNS_CREATE_PATH:
      bcLinks = [
        <span key="User Data Storage">User Data Storage</span>,
        <Link
          applyStyles={false}
          key="columns"
          title="Columns"
          href={'/columns' + cleanQuery}
        >
          Columns
        </Link>,
        <Link
          applyStyles={false}
          key="columns"
          title="Columns"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Create Column
        </Link>,
      ];
      break;
    case COLUMNS_DETAILS_PATH:
      bcLinks = [
        <span key="User Data Storage">User Data Storage</span>,
        <Link
          applyStyles={false}
          key="columns"
          title="Columns"
          href={'/columns' + cleanQuery}
        >
          Columns
        </Link>,
        <Link
          applyStyles={false}
          key="columns"
          title="Columns"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Column Detail
        </Link>,
      ];
      break;
    case OBJECT_STORES_PATH:
      bcLinks = [
        <span key="User Data Storage">User Data Storage</span>,
        <Link
          applyStyles={false}
          key="objectstores"
          title="Object Stores"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Object Stores
        </Link>,
      ];
      break;
    case OBJECT_STORE_DETAILS_PATH:
      bcLinks = [
        <span key="User Data Storage">User Data Storage</span>,
        <Link
          applyStyles={false}
          key="objectstores"
          title="Object Stores"
          href={'/object_stores' + cleanQuery}
        >
          Object Stores
        </Link>,
        <Link
          applyStyles={false}
          key="objectstores"
          title="Object Stores"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Object Store Detail
        </Link>,
      ];
      break;
    case DATA_TYPES_PATH:
      bcLinks = [
        <span key="User Data Storage">User Data Storage</span>,
        <Link
          applyStyles={false}
          key="datatypes"
          title="DataTypes"
          href={'/datatypes' + cleanQuery}
          className={style.currentLink}
        >
          Data Types
        </Link>,
      ];
      break;
    case DATA_TYPE_CREATE_PATH:
      bcLinks = [
        <span key="User Data Storage">User Data Storage</span>,
        <Link
          applyStyles={false}
          key="datatypes"
          title="DataTypes"
          href={'/datatypes' + cleanQuery}
        >
          Data Types
        </Link>,
        <Link
          applyStyles={false}
          key="datatypes"
          title="DataTypes"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Create Data Type
        </Link>,
      ];
      break;
    case DATA_TYPE_DETAILS_PATH:
      bcLinks = [
        <span key="User Data Storage">User Data Storage</span>,
        <Link
          applyStyles={false}
          key="datatypes"
          title="DataTypes"
          href={'/datatypes' + cleanQuery}
        >
          Data Types
        </Link>,
        <Link
          applyStyles={false}
          key="datatypes"
          title="DataTypes"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Data Type Detail
        </Link>,
      ];
      break;
    case ACCESSORS_PATH:
      bcLinks = [
        <span key="Access Methods">Access Methods</span>,
        <Link
          applyStyles={false}
          key="accessors"
          title="Accessors"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Accessors
        </Link>,
      ];
      break;
    case ACCESSORS_CREATE_PATH:
      bcLinks = [
        <span key="Access Methods">Access Methods</span>,
        <Link
          applyStyles={false}
          key="accessors"
          title="Accessors"
          href={'/accessors' + cleanQuery}
        >
          Accessors
        </Link>,
        <Link
          applyStyles={false}
          key="Create Accessor"
          title="Create Accessor details"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Create Accessor
        </Link>,
      ];
      break;
    case ACCESSORS_DETAILS_VERSION_PATH:
      bcLinks = [
        <span key="Access Methods">Access Methods</span>,
        <Link
          applyStyles={false}
          key="accessors"
          title="Accessors"
          href={'/accessors' + cleanQuery}
        >
          Accessors
        </Link>,
        <Link
          applyStyles={false}
          key="Accessor details"
          title="Accessor details"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Accessor Details
        </Link>,
      ];
      break;
    case MUTATORS_PATH:
      bcLinks = [
        <span key="Access Methods">Access Methods</span>,
        <Link
          applyStyles={false}
          key="Mutators"
          title="Mutators"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Mutators
        </Link>,
      ];
      break;
    case MUTATORS_DETAILS_VERSION_PATH:
      bcLinks = [
        <span key="Access Methods">Access Methods</span>,
        <Link
          applyStyles={false}
          key="Mutators"
          title="Mutators"
          href={'/mutators' + cleanQuery}
        >
          Mutators
        </Link>,
        <Link
          applyStyles={false}
          key="Mutator details"
          title="Mutator details"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Mutator Details
        </Link>,
      ];
      break;
    case MUTATORS_CREATE_PATH:
      bcLinks = [
        <span key="Access Methods">Access Methods</span>,
        <Link
          applyStyles={false}
          key="mutators"
          title="Mutators"
          href={'/mutators' + cleanQuery}
        >
          Mutators
        </Link>,
        <Link
          applyStyles={false}
          key="Create Mutator"
          title="Create Mutator details"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Create Mutator
        </Link>,
      ];
      break;
    case PURPOSES_PATH:
      bcLinks = [
        <span key="Access Rules">Access Rules</span>,
        <Link
          applyStyles={false}
          key="Purposes"
          title="Purposes"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Purposes
        </Link>,
      ];
      break;
    case PURPOSES_DETAILS_PATH:
      bcLinks = [
        <span key="Access Rules">Access Rules</span>,
        <Link
          applyStyles={false}
          key="Purposes"
          title="Purposes"
          href={'/purposes' + cleanQuery}
        >
          Purposes
        </Link>,
        <Link
          applyStyles={false}
          key="Purpose Details"
          title="Purpose Details"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Purpose Details
        </Link>,
      ];
      break;
    case PURPOSES_CREATE_PATH:
      bcLinks = [
        <span key="Access Rules">Access Rules</span>,
        <Link
          applyStyles={false}
          key="Purposes"
          title="Purposes"
          href={'/purposes' + cleanQuery}
        >
          Purposes
        </Link>,
        <Link
          applyStyles={false}
          key="Purpose Details"
          title={`${'Create Purpose'}`}
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          {`${'Create Purpose'}`}
        </Link>,
      ];
      break;
    case TRANSFORMERS_PATH:
      bcLinks = [
        <span key="User Data Masking">User Data Masking</span>,
        <Link
          applyStyles={false}
          key="Transformers"
          title="Transformers"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Transformers
        </Link>,
      ];
      break;
    case TRANSFORMERS_CREATE_PATH:
      bcLinks = [
        <span key="User Data Masking">User Data Masking</span>,
        <Link
          applyStyles={false}
          key="Transformers"
          title="Transformers"
          href={'/transformers' + cleanQuery}
        >
          Transformers
        </Link>,
        <Link
          applyStyles={false}
          key="Transformer"
          title="Transformer"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Create Transformer
        </Link>,
      ];
      break;
    case TRANSFORMERS_POLICY_DETAILS_VERSION_PATH:
      bcLinks = [
        <span key="User Data Masking">User Data Masking</span>,
        <Link
          applyStyles={false}
          key="Transformers"
          title="Transformers"
          href={'/transformers' + cleanQuery}
        >
          Transformers
        </Link>,
        <Link
          applyStyles={false}
          key="Transformer"
          title="Transformer"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Transformer Detail
        </Link>,
      ];
      break;
    case ACCESSPOLICIES_PATH:
      bcLinks = [
        <span key="Access Rules">Access Rules</span>,
        <Link
          applyStyles={false}
          key="Policies"
          title="Policies"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Access Policies
        </Link>,
      ];
      break;
    case ACCESSPOLICIES_CREATE_PATH:
      bcLinks = [
        <span key="Access Rules">Access Rules</span>,
        <Link
          applyStyles={false}
          key="Policies"
          title="Policies"
          href={'/accesspolicies' + cleanQuery}
        >
          Access Policies
        </Link>,
        <Link
          applyStyles={false}
          key="Access Policy"
          title="Access Policy"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Create Access Policy
        </Link>,
      ];
      break;
    case ACCESSPOLICIES_DETAILS_VERSION_PATH:
      bcLinks = [
        <span key="Access Rules">Access Rules</span>,
        <Link
          applyStyles={false}
          key="Policies"
          title="Policies"
          href={'/accesspolicies' + cleanQuery}
        >
          Access Policies
        </Link>,
        <Link
          applyStyles={false}
          key="Access Policy"
          title="Access Policy"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Access Policy Detail
        </Link>,
      ];
      break;
    case POLICYTEMPLATES_PATH:
      bcLinks = [
        <span key="Access Rules">Access Rules</span>,
        <Link
          applyStyles={false}
          key="templates"
          title="Policies Templates"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Policy Templates
        </Link>,
      ];
      break;
    case POLICYTEMPLATES_CREATE_PATH:
      bcLinks = [
        <span key="Access Rules">Access Rules</span>,
        <Link
          applyStyles={false}
          key="templates"
          title="Policies Templates"
          href={'/policytemplates' + cleanQuery}
        >
          Policy Templates
        </Link>,
        <Link
          applyStyles={false}
          key="Create Policy Template"
          title="Create Policy Template"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Create Policy Template
        </Link>,
      ];
      break;
    case POLICYTEMPLATES_DETAILS_VERSION_PATH:
      bcLinks = [
        <span key="Access Rules">Access Rules</span>,
        <Link
          applyStyles={false}
          key="templates"
          title="Policies Templates"
          href={'/policytemplates' + cleanQuery}
        >
          Policy Templates
        </Link>,
        <Link
          applyStyles={false}
          key="Policy Template Details"
          title="Policy Template Details"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Policy Template Details
        </Link>,
      ];
      break;
    case OBJECTS_PATH:
      bcLinks = [
        <span key="Access Permisions">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="objects"
          title="Objects"
          href={pathname + makeCleanPageLink(query)}
          className={style.currentLink}
        >
          Objects
        </Link>,
      ];
      break;
    case OBJECTS_DETAILS_PATH:
      bcLinks = [
        <span key="Access Permisions">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="Objects"
          title="Objects"
          href={'/objects' + makeCleanPageLink(query)}
        >
          Objects
        </Link>,
        <Link
          applyStyles={false}
          key="Object Detail"
          title="Object Detail"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Object Detail
        </Link>,
      ];
      break;
    case OBJECTS_CREATE_PATH:
      bcLinks = [
        <span key="Access Permisions">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="Objects"
          title="Objects"
          href={'/objects' + makeCleanPageLink(query)}
        >
          Objects
        </Link>,
        <Link
          applyStyles={false}
          key="Create Object"
          title="Create Object"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Create Object
        </Link>,
      ];
      break;
    case EDGES_PATH:
      bcLinks = [
        <span key="Access Permisions">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="Edges"
          title="Edges"
          href={pathname + makeCleanPageLink(query)}
          className={style.currentLink}
        >
          Edges
        </Link>,
      ];
      break;
    case EDGES_DETAILS_PATH:
      bcLinks = [
        <span key="Access Permisions">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="Edges"
          title="Edges"
          href={'/edges' + makeCleanPageLink(query)}
        >
          Edges
        </Link>,
        <Link
          applyStyles={false}
          key="Edge Detail"
          title="Edge Detail"
          href={pathname + makeCleanPageLink(query)}
          className={style.currentLink}
        >
          Edge Detail
        </Link>,
      ];
      break;
    case EDGES_CREATE_PATH:
      bcLinks = [
        <span key="Access Permisions">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="Edges"
          title="Edges"
          href={'/edges' + makeCleanPageLink(query)}
        >
          Edges
        </Link>,
        <Link
          applyStyles={false}
          key="Create Edge"
          title="Create Edge"
          href={pathname + makeCleanPageLink(query)}
          className={style.currentLink}
        >
          Create Edge
        </Link>,
      ];
      break;
    case EDGETYPES_PATH:
      bcLinks = [
        <span key="Access Permisions">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="Edge Types"
          title="Edge Types"
          href={pathname + makeCleanPageLink(query)}
          className={style.currentLink}
        >
          Edge Types
        </Link>,
      ];
      break;
    case EDGETYPES_DETAILS_PATH:
      bcLinks = [
        <span key="Access Permisions">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="Edge Types"
          title="Edge Types"
          href={'/edgetypes' + makeCleanPageLink(query)}
        >
          Edge Types
        </Link>,
        <Link
          applyStyles={false}
          key="Edge Type Details"
          title="Edge Type Details"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Edge Type Details
        </Link>,
      ];
      break;
    case EDGETYPES_CREATE_PATH:
      bcLinks = [
        <span key="Access Permisions">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="Edge Types"
          title="Edge Types"
          href={'/edgetypes' + makeCleanPageLink(query)}
        >
          Edge Types
        </Link>,
        <Link
          applyStyles={false}
          key="Create Edge Type"
          title="Create Edge Type"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Create Edge Type
        </Link>,
      ];
      break;
    case OBJECTTYPES_PATH:
      bcLinks = [
        <span key="Access Permisions">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="Object Types"
          title="Object Types"
          href={pathname + makeCleanPageLink(query)}
          className={style.currentLink}
        >
          Object Types
        </Link>,
      ];
      break;
    case OBJECTTYPES_DETAILS_PATH:
      bcLinks = [
        <span key="Access Permisions">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="Object types"
          title="Object Types"
          href={'/objecttypes' + makeCleanPageLink(query)}
        >
          Object Types
        </Link>,
        <Link
          applyStyles={false}
          key="Object Type Detail"
          title="Object Type Detail"
          href={pathname + makeCleanPageLink(query)}
          className={style.currentLink}
        >
          Object Type Detail
        </Link>,
      ];
      break;
    case OBJECTTYPES_CREATE_PATH:
      bcLinks = [
        <span key="Access Permisions">Access Permissions</span>,
        <Link
          applyStyles={false}
          key="Object types"
          title="Object Types"
          href={'/objecttypes' + makeCleanPageLink(query)}
        >
          Object Types
        </Link>,
        <Link
          applyStyles={false}
          key="Create Object Type"
          title="Create Object Type"
          href={pathname + makeCleanPageLink(query)}
          className={style.currentLink}
        >
          Create Object Type
        </Link>,
      ];
      break;
    case LOGINAPPS_PATH:
      bcLinks = [
        <span key="User Authentication">User Authentication</span>,
        <Link
          applyStyles={false}
          key="Login Apps"
          title="Login Apps"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Login Apps
        </Link>,
      ];
      break;
    case IDENTITYPROVIDERS_PATH:
      bcLinks = [
        <span key="User Authentication">User Authentication</span>,
        <Link
          applyStyles={false}
          key="Identity Providers"
          title="Identity Providers"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Identity Providers
        </Link>,
      ];
      break;
    case OAUTHCONNECTIONS_PATH:
      bcLinks = [
        <span key="User Authentication">User Authentication</span>,
        <Link
          applyStyles={false}
          key="OAuth Connections"
          title="OAuth Connections"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          OAuth Connections
        </Link>,
      ];
      break;
    case COMMCHANNELS_PATH:
      bcLinks = [
        <span key="User Authentication">User Authentication</span>,
        <Link
          applyStyles={false}
          key="Comms Channel"
          title="Comm Channels"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Comm Channels
        </Link>,
      ];
      break;
    case IDENTITYPROVIDERS_PLEX_PROVIDER_DETAILS_PATH:
      bcLinks = [
        <span key="User Authentication">User Authentication</span>,
        <Link
          applyStyles={false}
          key="Identity Providers"
          title="Identity Providers"
          href={'/identityproviders' + cleanQuery}
        >
          Identity Providers
        </Link>,
        <Link
          applyStyles={false}
          key="Identity Platform"
          title="Identity Platform"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Identity Platform
        </Link>,
      ];
      break;
    case OAUTHCONNECTIONS_OIDC_PROVIDER_CREATE_PATH:
      bcLinks = [
        <span key="User Authentication">User Authentication</span>,
        <Link
          applyStyles={false}
          key="OAuth Connections"
          title="OAuth Connections"
          href={'/oauthconnections' + cleanQuery}
        >
          OAuth Connections
        </Link>,
        <Link
          applyStyles={false}
          key="OIDC"
          title="OIDC Connection"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          OIDC
        </Link>,
      ];
      break;
    case OAUTHCONNECTIONS_OIDC_PROVIDER_NAME_PATH:
      bcLinks = [
        <span key="User Authentication">User Authentication</span>,
        <Link
          applyStyles={false}
          key="OAuth Connections"
          title="OAuth Connections"
          href={'/oauthconnections' + cleanQuery}
        >
          OAuth Connections
        </Link>,
        <Link
          applyStyles={false}
          key="OIDC"
          title="OIDC Connection"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          OIDC
        </Link>,
      ];
      break;

    case LOGINAPPS_PLEX_APP_DETAILS_PATH:
      bcLinks = [
        <span key="User Authentication">User Authentication</span>,
        <Link
          applyStyles={false}
          key="Login Apps"
          title="Login Apps"
          href={'/loginapps' + cleanQuery}
        >
          Login Apps
        </Link>,
        <Link
          applyStyles={false}
          key="Login Application"
          title="Login Application"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Plex Application
        </Link>,
      ];
      break;
    case LOGINAPPS_PLEX_EMPLOYEE_APP_PATH:
      bcLinks = [
        <span key="User Authentication">User Authentication</span>,
        <Link
          applyStyles={false}
          key="Login Apps"
          title="Login Apps"
          href={'/loginapps' + cleanQuery}
        >
          Login Apps
        </Link>,
        <Link
          applyStyles={false}
          key="Plex Employee Application"
          title="Plex Employee Application"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Plex Employee Application
        </Link>,
      ];
      break;
    case USERS_PATH:
      bcLinks = [
        <span key="User Data Storage">User Data Storage</span>,
        <Link
          applyStyles={false}
          key="Users"
          title="Users"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Users
        </Link>,
      ];
      break;
    case USERS_DETAILS_PATH:
      bcLinks = [
        <span key="User Data Storage">User Data Storage</span>,
        <Link
          applyStyles={false}
          key="Users"
          title="Users"
          href={'/users' + cleanQuery}
        >
          Users
        </Link>,
        <Link
          applyStyles={false}
          key="User Details"
          title="User Details"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          User Details
        </Link>,
      ];
      break;
    case IAM_PATH:
      bcLinks = [
        <Link
          applyStyles={false}
          key="Team"
          title="Team"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Manage Team
        </Link>,
      ];
      break;
    case AUDITLOG_PATH:
      bcLinks = [
        <span key="monitor">Monitoring</span>,
        <Link
          applyStyles={false}
          key="Audit Log"
          title="Audit Log"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Audit Log
        </Link>,
      ];
      break;
    case DATAACCESSLOG_PATH:
      bcLinks = [
        <span key="monitor">Monitoring</span>,
        <Link
          applyStyles={false}
          key="Data Access Log"
          title="Data Access Log"
          href={pathname + cleanQuery}
          className={style.currentLink}
        >
          Data Access Log
        </Link>,
      ];
      break;
    case DATAACCESSLOG_DETAILS_PATH:
      bcLinks = [
        <span key="monitor">Monitoring</span>,
        <Link
          applyStyles={false}
          key="Data Access Log"
          title="Data Access Log"
          href={DATAACCESSLOG_PATH + cleanQuery}
          className={style.currentLink}
        >
          Data Access Log
        </Link>,
      ];
      break;

    case SYSTEMLOG_PATH:
      bcLinks = [
        <span key="monitor">Monitoring</span>,
        <Link
          applyStyles={false}
          key="Sytem Log"
          title="System Log"
          href={pathname + makeCleanPageLink(query)}
          className={style.currentLink}
        >
          System Log
        </Link>,
      ];
      break;
    case SYSTEMLOG_RUN_DETAILS_PATH:
      bcLinks = [
        <span key="monitor">Monitoring</span>,
        <Link
          applyStyles={false}
          key="System Log"
          title="System Log"
          href={'/systemlog' + makeCleanPageLink(query)}
        >
          System Log
        </Link>,
        <Link
          applyStyles={false}
          key="System Event Detail"
          title="System Event Detail"
          href={pathname + makeCleanPageLink(query)}
          className={style.currentLink}
        >
          System Event Detail
        </Link>,
      ];
      break;
    case DATASOURCES_PATH:
      bcLinks = [
        <span key="user-data-mapping">User Data Mapping</span>,
        <Link
          applyStyles={false}
          key="Data Sources"
          title="Data Sources"
          href={pathname + cleanQuery}
        >
          Data Sources
        </Link>,
      ];
      break;
    case DATASOURCE_CREATE_PATH:
      bcLinks = [
        <span key="user-data-mapping">User Data Mapping</span>,
        <Link
          applyStyles={false}
          key="Data Sources"
          title="Data Sources"
          href={DATASOURCES_PATH + cleanQuery}
        >
          Data Sources
        </Link>,
        <Link
          applyStyles={false}
          key="Add Data Source"
          title="Add Data Source"
          href={pathname + cleanQuery}
        >
          Add Data Source
        </Link>,
      ];
      break;
    case DATASOURCE_DETAILS_PATH:
      bcLinks = [
        <span key="user-data-mapping">User Data Mapping</span>,
        <Link
          applyStyles={false}
          key="Data Sources"
          title="Data Sources"
          href={DATASOURCES_PATH + cleanQuery}
        >
          Data Sources
        </Link>,
        <Link
          applyStyles={false}
          key="Data Source Details"
          title="Data Source Details"
          href={pathname + cleanQuery}
        >
          Data Source Details
        </Link>,
      ];
      break;
    case DATASOURCESCHEMAS_PATH:
      bcLinks = [
        <span key="user-data-mapping">User Data Mapping</span>,
        <Link
          applyStyles={false}
          key="Data Source Schemas"
          title="Data Source Schemas"
          href={pathname + cleanQuery}
        >
          Data Source Schemas
        </Link>,
      ];
      break;
    case DATASOURCEELEMENT_DETAILS_PATH:
      bcLinks = [
        <span key="user-data-mapping">User Data Mapping</span>,
        <Link
          applyStyles={false}
          key="Data Source Schemas"
          title="Data Source Schemas"
          href={DATASOURCESCHEMAS_PATH + cleanQuery}
        >
          Data Source Schemas
        </Link>,
        <Link
          applyStyles={false}
          key="Data Source Element Details"
          title="Data Source Element Details"
          href={pathname + cleanQuery}
        >
          Data Source Element Details
        </Link>,
      ];
      break;

    default:
      bcLinks = [];
      break;
  }

  return (
    <ul className={style.root}>
      {bcLinks.map((link, index) => (
        <li className={style.crumb} key={'bc' + String(index)}>
          <label className={style.dropdownLabel}>
            /&nbsp;
            {link}
            &nbsp;
          </label>
        </li>
      ))}
    </ul>
  );
};

export default connect((state: RootState) => ({
  location: state.location,
  pattern: state.routePattern,
  routeParams: state.routeParams,
  query: state.query,
}))(Breadcrumbs);
