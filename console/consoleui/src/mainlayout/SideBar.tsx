import { useState } from 'react';
import { connect } from 'react-redux';
import clsx from 'clsx';

import {
  IconAccessMethods,
  IconAccessPermissions,
  IconAccessRules,
  IconMenuCheck,
  IconMenuHome,
  IconMonitoring,
  IconSettingsGear,
  IconUserAuthentication,
  IconUserDataMapping,
  IconUserDataMasking,
  IconUserDataStorage,
  SideBarStyles as styles,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState } from '../store';
import { SelectedTenant } from '../models/Tenant';
import Link from '../controls/Link';
import SidebarStyles from './Sidebar.module.css';
import {
  HOME_PATH,
  DATASOURCES_PATH,
  DATASOURCESCHEMAS_PATH,
  COLUMNS_PATH,
  USERS_PATH,
  DATA_TYPES_PATH,
  OBJECT_STORES_PATH,
  ACCESSORS_PATH,
  MUTATORS_PATH,
  PURPOSES_PATH,
  ACCESSPOLICIES_PATH,
  POLICYTEMPLATES_PATH,
  POLICY_SECRETS_PATH,
  ORGANIZATIONS_PATH,
  OBJECTTYPES_PATH,
  EDGETYPES_PATH,
  OBJECTS_PATH,
  EDGES_PATH,
  TRANSFORMERS_PATH,
  LOGINAPPS_PATH,
  OAUTHCONNECTIONS_PATH,
  IDENTITYPROVIDERS_PATH,
  COMMCHANNELS_PATH,
  AUDITLOG_PATH,
  DATAACCESSLOG_PATH,
  SYSTEMLOG_PATH,
  EVENTS_LOG_PATH,
  RESOURCE_LIST_PATH,
  STATUS_PATH,
  TENANTS_DETAILS_PATH,
} from '../routing';

const pathStartsWithAnyOf = (path: string, items: string[]) => {
  return items.some((item) => {
    if (path.indexOf(item) === 0) {
      return true;
    }
    return false;
  });
};

type NavMenuStructure = {
  userDataStorageIsOpen: boolean;
  accessMethodsIsOpen: boolean;
  accessRulesIsOpen: boolean;
  accessPermissionsIsOpen: boolean;
  userDataMaskingIsOpen: boolean;
  userAuthenticationIsOpen: boolean;
  monitoringIsOpen: boolean;
  datamappingIsOpen: boolean;
};

const closedMenuStructure: NavMenuStructure = {
  userDataStorageIsOpen: false,
  accessMethodsIsOpen: false,
  accessRulesIsOpen: false,
  accessPermissionsIsOpen: false,
  userDataMaskingIsOpen: false,
  userAuthenticationIsOpen: false,
  monitoringIsOpen: false,
  datamappingIsOpen: false,
};

const SideBar = ({
  isOpen,
  selectedTenantID,
  selectedTenant,
  location,
}: {
  isOpen: boolean;
  selectedTenantID: string | undefined;
  selectedTenant: SelectedTenant | undefined;
  location: URL;
}) => {
  const { pathname, search } = location;

  const [menuState, setMenuState] = useState<NavMenuStructure>({
    userDataStorageIsOpen: false,
    accessMethodsIsOpen: false,
    accessRulesIsOpen: false,
    accessPermissionsIsOpen: false,
    userDataMaskingIsOpen: false,
    userAuthenticationIsOpen: false,
    monitoringIsOpen: false,
    datamappingIsOpen: false,
  });

  const classes = clsx({
    [SidebarStyles.root]: true,
    [styles.isOpen]: isOpen,
  });

  const queryString = makeCleanPageLink(new URLSearchParams(search));

  return (
    <>
      <nav className={classes} id="mainNav">
        <ol>
          <li key="home-page">
            <Link
              href={HOME_PATH + queryString}
              className={clsx({
                [SidebarStyles.sideBarLink]: true,
                [SidebarStyles.isActive]: pathname === HOME_PATH,
              })}
            >
              <IconMenuHome size="medium" />
              Tenant Home
            </Link>
          </li>
          <li key="data-mapping">
            <button
              className={clsx({
                [SidebarStyles.sideBarLink]: true,
                [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                  DATASOURCES_PATH,
                ]),
              })}
              onClick={() => {
                setMenuState({
                  ...closedMenuStructure,
                  datamappingIsOpen: !menuState.datamappingIsOpen,
                });
              }}
            >
              <IconUserDataMapping size="medium" />
              User Data Mapping
            </button>
          </li>
          {(pathStartsWithAnyOf(pathname, [DATASOURCES_PATH]) ||
            menuState.datamappingIsOpen) && (
            <>
              <li
                key="data-sources"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={DATASOURCES_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]:
                      pathStartsWithAnyOf(pathname, [DATASOURCES_PATH + '/']) ||
                      pathname === DATASOURCES_PATH,
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [DATASOURCES_PATH + '/']) ||
                      pathname === DATASOURCES_PATH
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Data Sources
                </Link>
              </li>
              <li
                key="data-source-schemas"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={DATASOURCESCHEMAS_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      DATASOURCESCHEMAS_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [DATASOURCESCHEMAS_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Data Source Schemas
                </Link>
              </li>
            </>
          )}
          <li key="user-data-storage">
            <button
              className={clsx({
                [SidebarStyles.sideBarLink]: true,
                [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                  COLUMNS_PATH,
                  USERS_PATH,
                ]),
              })}
              onClick={() => {
                setMenuState({
                  ...closedMenuStructure,
                  userDataStorageIsOpen: !menuState.userDataStorageIsOpen,
                });
              }}
            >
              <IconUserDataStorage size="medium" />
              User Data Storage
            </button>
          </li>
          {(pathStartsWithAnyOf(pathname, [
            COLUMNS_PATH,
            USERS_PATH,
            DATA_TYPES_PATH,
            OBJECT_STORES_PATH,
          ]) ||
            menuState.userDataStorageIsOpen) && (
            <>
              <li
                key="user-data-storage-column"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={COLUMNS_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      COLUMNS_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathname.indexOf(COLUMNS_PATH) === 0
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Columns
                </Link>
              </li>
              <li
                key="user-data-storage-users"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={USERS_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      USERS_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathname.indexOf(USERS_PATH) === 0
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Users
                </Link>
              </li>
              <li
                key="user-data-storage-datatypes"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={DATA_TYPES_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      DATA_TYPES_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathname.indexOf(DATA_TYPES_PATH) === 0
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Data Types
                </Link>
              </li>
              <li
                key="user-data-storage-object-stores"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={OBJECT_STORES_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      OBJECT_STORES_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathname.indexOf(OBJECT_STORES_PATH) === 0
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Object Stores
                </Link>
              </li>
            </>
          )}
          <li key="access-methods">
            <button
              className={clsx({
                [SidebarStyles.sideBarLink]: true,
                [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                  ACCESSORS_PATH,
                  MUTATORS_PATH,
                ]),
              })}
              onClick={() => {
                setMenuState({
                  ...closedMenuStructure,
                  accessMethodsIsOpen: !menuState.accessMethodsIsOpen,
                });
              }}
            >
              <IconAccessMethods size="medium" />
              Access Methods
            </button>
          </li>
          {(pathStartsWithAnyOf(pathname, [ACCESSORS_PATH, MUTATORS_PATH]) ||
            menuState.accessMethodsIsOpen) && (
            <>
              <li
                key="access-methods-accessors"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={ACCESSORS_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      ACCESSORS_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [ACCESSORS_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Accessors
                </Link>
              </li>
              <li
                key="access-methods-mutators"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={MUTATORS_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      MUTATORS_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [MUTATORS_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Mutators
                </Link>
              </li>
            </>
          )}
          <li key="access-rules">
            <button
              className={clsx({
                [SidebarStyles.sideBarLink]: true,
                [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                  PURPOSES_PATH,
                  ACCESSPOLICIES_PATH,
                  POLICYTEMPLATES_PATH,
                  POLICY_SECRETS_PATH,
                ]),
              })}
              onClick={() => {
                setMenuState({
                  ...closedMenuStructure,
                  accessRulesIsOpen: !menuState.accessRulesIsOpen,
                });
              }}
            >
              <IconAccessRules size="medium" />
              Access Rules
            </button>
          </li>
          {(pathStartsWithAnyOf(pathname, [
            PURPOSES_PATH,
            ACCESSPOLICIES_PATH,
            POLICYTEMPLATES_PATH,
            POLICY_SECRETS_PATH,
          ]) ||
            menuState.accessRulesIsOpen) && (
            <>
              <li
                key="access-rules-purposes"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={PURPOSES_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      PURPOSES_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [PURPOSES_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Purposes
                </Link>
              </li>
              <li
                key="access-rules-access-policies"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={ACCESSPOLICIES_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      ACCESSPOLICIES_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [ACCESSPOLICIES_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Access Policies
                </Link>
              </li>
              <li
                key="access-rules-policy-templates"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={POLICYTEMPLATES_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      POLICYTEMPLATES_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [POLICYTEMPLATES_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Policy Templates
                </Link>
              </li>
              <li
                key="access-rules-secrets"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={POLICY_SECRETS_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      POLICY_SECRETS_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [POLICY_SECRETS_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Secrets
                </Link>
              </li>
            </>
          )}
          <li key="access-permissions">
            <button
              className={clsx({
                [SidebarStyles.sideBarLink]: true,
                [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                  ORGANIZATIONS_PATH,
                  OBJECTTYPES_PATH,
                  EDGETYPES_PATH,
                  OBJECTS_PATH,
                  EDGES_PATH,
                ]),
              })}
              onClick={() => {
                setMenuState({
                  ...closedMenuStructure,
                  accessPermissionsIsOpen: !menuState.accessPermissionsIsOpen,
                });
              }}
            >
              <IconAccessPermissions size="medium" />
              Access Permissions
            </button>
          </li>
          {(pathStartsWithAnyOf(pathname, [
            ORGANIZATIONS_PATH,
            OBJECTTYPES_PATH,
            EDGETYPES_PATH,
            OBJECTS_PATH,
            EDGES_PATH,
          ]) ||
            menuState.accessPermissionsIsOpen) && (
            <>
              {selectedTenant?.use_organizations && (
                <li
                  key="access-permissions-organizations"
                  className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
                >
                  <Link
                    href={ORGANIZATIONS_PATH + queryString}
                    className={clsx({
                      [SidebarStyles.sideBarLink]: true,
                      [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                        ORGANIZATIONS_PATH,
                      ]),
                    })}
                  >
                    <span
                      className={
                        pathStartsWithAnyOf(pathname, [ORGANIZATIONS_PATH])
                          ? SidebarStyles.icon
                          : SidebarStyles.invisible
                      }
                    >
                      <IconMenuCheck size="medium" />
                    </span>
                    Organizations
                  </Link>
                </li>
              )}
              <li
                key="access-permissions-object-types"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={OBJECTTYPES_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      OBJECTTYPES_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [OBJECTTYPES_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Object Types
                </Link>
              </li>
              <li
                key="access-permissions-edge-types"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={EDGETYPES_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      EDGETYPES_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [EDGETYPES_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Edge Types
                </Link>
              </li>
              <li
                key="access-permissions-objects"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={OBJECTS_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      OBJECTS_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [OBJECTS_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Objects
                </Link>
              </li>
              <li
                key="access-permissions-edges"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={EDGES_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      EDGES_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [EDGES_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Edges
                </Link>
              </li>
            </>
          )}
          <li key="user-data-masking">
            <button
              className={clsx({
                [SidebarStyles.sideBarLink]: true,
                [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                  TRANSFORMERS_PATH,
                ]),
              })}
              onClick={() => {
                setMenuState({
                  ...closedMenuStructure,
                  userDataMaskingIsOpen: !menuState.userDataMaskingIsOpen,
                });
              }}
            >
              <IconUserDataMasking size="medium" />
              User Data Masking
            </button>
          </li>
          {(pathStartsWithAnyOf(pathname, [TRANSFORMERS_PATH]) ||
            menuState.userDataMaskingIsOpen) && (
            <li
              key="user-data-masking-transformers"
              className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
            >
              <Link
                href={TRANSFORMERS_PATH + queryString}
                className={clsx({
                  [SidebarStyles.sideBarLink]: true,
                  [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                    TRANSFORMERS_PATH,
                  ]),
                })}
              >
                <span
                  className={
                    pathStartsWithAnyOf(pathname, [TRANSFORMERS_PATH])
                      ? SidebarStyles.icon
                      : SidebarStyles.invisible
                  }
                >
                  <IconMenuCheck size="medium" />
                </span>
                Transformers
              </Link>
            </li>
          )}
          <li key="user-authentication">
            <button
              className={clsx({
                [SidebarStyles.sideBarLink]: true,
                [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                  LOGINAPPS_PATH,
                  COMMCHANNELS_PATH,
                  OAUTHCONNECTIONS_PATH,
                  IDENTITYPROVIDERS_PATH,
                ]),
              })}
              onClick={() => {
                setMenuState({
                  ...closedMenuStructure,
                  userAuthenticationIsOpen: !menuState.userAuthenticationIsOpen,
                });
              }}
            >
              <IconUserAuthentication size="medium" />
              User Authentication
            </button>
          </li>
          {(pathStartsWithAnyOf(pathname, [
            LOGINAPPS_PATH,
            COMMCHANNELS_PATH,
            OAUTHCONNECTIONS_PATH,
            IDENTITYPROVIDERS_PATH,
          ]) ||
            menuState.userAuthenticationIsOpen) && (
            <>
              <li
                key="user-authentication-login-apps"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={LOGINAPPS_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      LOGINAPPS_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [LOGINAPPS_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Login Apps
                </Link>
              </li>
              <li
                key="user-authentication-oauth-connections"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={OAUTHCONNECTIONS_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      OAUTHCONNECTIONS_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [OAUTHCONNECTIONS_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  OAuth Connections
                </Link>
              </li>
              <li
                key="user-authentication-identity-providers"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={IDENTITYPROVIDERS_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      IDENTITYPROVIDERS_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [IDENTITYPROVIDERS_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Identity Providers
                </Link>
              </li>
              <li
                key="user-authentication-comms-channels"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={COMMCHANNELS_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      COMMCHANNELS_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [COMMCHANNELS_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Comms Channels
                </Link>
              </li>
            </>
          )}
          <li key="monitoring">
            <button
              className={clsx({
                [SidebarStyles.sideBarLink]: true,
                [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                  AUDITLOG_PATH,
                  DATAACCESSLOG_PATH,
                  SYSTEMLOG_PATH,
                  EVENTS_LOG_PATH,
                  RESOURCE_LIST_PATH,
                  STATUS_PATH,
                ]),
              })}
              onClick={() => {
                setMenuState({
                  ...closedMenuStructure,
                  monitoringIsOpen: !menuState.monitoringIsOpen,
                });
              }}
            >
              <IconMonitoring size="medium" />
              Monitoring
            </button>
          </li>
          {(pathStartsWithAnyOf(pathname, [
            AUDITLOG_PATH,
            DATAACCESSLOG_PATH,
            SYSTEMLOG_PATH,
            EVENTS_LOG_PATH,
            RESOURCE_LIST_PATH,
            STATUS_PATH,
          ]) ||
            menuState.monitoringIsOpen) && (
            <>
              <li
                key="monitoring-auditlog"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={AUDITLOG_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      AUDITLOG_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [AUDITLOG_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Audit Log
                </Link>
              </li>
              <li
                key="monitoring-dataaccesslog"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={DATAACCESSLOG_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      DATAACCESSLOG_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [DATAACCESSLOG_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Data Access Log
                </Link>
              </li>
              <li
                key="monitoring-system-log"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={SYSTEMLOG_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      SYSTEMLOG_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [SYSTEMLOG_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  System Log
                </Link>
              </li>
              <li
                key="monitoring-events-log"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={EVENTS_LOG_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      EVENTS_LOG_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [EVENTS_LOG_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Events Log
                </Link>
              </li>
              <li
                key="monitoring-resource-list"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={RESOURCE_LIST_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      RESOURCE_LIST_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [RESOURCE_LIST_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Resources
                </Link>
              </li>
              <li
                key="monitoring-status"
                className={clsx({ [SidebarStyles.sideBarSubLink]: true })}
              >
                <Link
                  href={STATUS_PATH + queryString}
                  className={clsx({
                    [SidebarStyles.sideBarLink]: true,
                    [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                      STATUS_PATH,
                    ]),
                  })}
                >
                  <span
                    className={
                      pathStartsWithAnyOf(pathname, [STATUS_PATH])
                        ? SidebarStyles.icon
                        : SidebarStyles.invisible
                    }
                  >
                    <IconMenuCheck size="medium" />
                  </span>
                  Status
                </Link>
              </li>
            </>
          )}
          <li key="settings">
            <Link
              className={clsx({
                [SidebarStyles.sideBarLink]: true,
                [SidebarStyles.isActive]: pathStartsWithAnyOf(pathname, [
                  TENANTS_DETAILS_PATH,
                ]),
              })}
              href={`${TENANTS_DETAILS_PATH.replace(':tenantID', selectedTenantID || 'current')}${queryString}`}
            >
              <IconSettingsGear size="medium" />
              Tenant Settings
            </Link>
          </li>
        </ol>
      </nav>
    </>
  );
};

export default connect((state: RootState) => {
  return {
    selectedTenantID: state.selectedTenantID,
    selectedTenant: state.selectedTenant,
    location: state.location,
    featureFlags: state.featureFlags,
  };
})(SideBar);
