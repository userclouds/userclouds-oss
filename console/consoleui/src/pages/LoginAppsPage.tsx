import { useEffect } from 'react';
import { connect } from 'react-redux';
import { v4 as uuidv4 } from 'uuid';

import {
  Button,
  Card,
  Table,
  TableRow,
  TableRowHead,
  TableBody,
  TableCell,
  TableHead,
  Text,
  ToolTip,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import { makeCleanPageLink } from '../AppNavigation';
import {
  updatePlexConfigRequest,
  updatePlexConfigSuccess,
  updatePlexConfigError,
} from '../actions/authn';

import { AppDispatch, RootState } from '../store';
import TenantPlexConfig, {
  UpdatePlexConfigReason,
} from '../models/TenantPlexConfig';
import LoginApp from '../models/LoginApp';
import { SelectedTenant } from '../models/Tenant';

import { addLoginAppToTenant } from '../API/authn';
import { fetchPlexConfig } from '../thunks/authn';
import { RandomBase64, RandomHex } from '../util/Rand';
import Link from '../controls/Link';
import PageCommon from './PageCommon.module.css';
import styles from './LoginAppsPage.module.css';
import { redirect } from '../routing';

const PlexApps = ({
  apps,
  employeeApp,
  selectedTenant,
  query,
}: {
  apps: LoginApp[];
  employeeApp: LoginApp;
  selectedTenant: SelectedTenant | undefined;
  query: URLSearchParams;
}) => {
  const cleanQuery = makeCleanPageLink(query);
  return (
    <Table spacing="packed" id="loginApps" className={styles.loginappstable}>
      <TableHead>
        <TableRow>
          <TableRowHead>Name</TableRowHead>
          <TableRowHead>Grant types</TableRowHead>
        </TableRow>
      </TableHead>
      <TableBody>
        {apps.map((app) => (
          <TableRow key={app.id}>
            <TableCell>
              {selectedTenant?.is_admin ? (
                <Link href={`/loginapps/${app.id}${cleanQuery}`}>
                  {app.name}
                </Link>
              ) : (
                <Text>{app.name}</Text>
              )}
            </TableCell>
            <TableCell>
              {app && app.grant_types && app.grant_types.join(', ')}
            </TableCell>
          </TableRow>
        ))}
        {employeeApp.id && (
          <TableRow key={employeeApp.id}>
            <TableCell>
              {selectedTenant?.is_admin ? (
                <Link href={`/loginapps/plex_employee_app${cleanQuery}`}>
                  Employee App
                </Link>
              ) : (
                <Text>Employee App</Text>
              )}
            </TableCell>
            <TableCell>
              {employeeApp.grant_types && employeeApp.grant_types.join(', ')}
            </TableCell>
          </TableRow>
        )}
      </TableBody>
    </Table>
  );
};

const onCreateAppClick =
  (tenantID: string) => async (dispatch: AppDispatch) => {
    const newAppID = uuidv4();
    const newName = 'New Plex App';
    const newClientID = RandomHex(32);
    const newClientSecret = RandomBase64(64);

    dispatch(updatePlexConfigRequest());

    addLoginAppToTenant(
      tenantID,
      newAppID,
      newName,
      newClientID,
      newClientSecret
    ).then(
      (tenantPlex: TenantPlexConfig) => {
        dispatch(
          updatePlexConfigSuccess(tenantPlex, UpdatePlexConfigReason.AddApp)
        );

        redirect(`/loginapps/${newAppID}`);
      },
      (error: APIError) => {
        dispatch(updatePlexConfigError(error));
      }
    );
  };

const LoginApps = ({
  tenant,
  plexConfig,
  modifiedConfig,
  fetchError,
  query,
  dispatch,
}: {
  tenant: SelectedTenant | undefined;
  plexConfig: TenantPlexConfig | undefined;
  modifiedConfig: TenantPlexConfig | undefined;
  fetchError: string;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  const readOnly = !tenant?.is_admin;

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              Create different login experiences, user-facing emails and
              security requirements for different user groups. A UserClouds
              application corresponds to a single OAuth2 client application.
              <a
                href="https://docs.userclouds.com/docs/introduction-1"
                title="UserClouds documentation for key concepts in authentication"
                target="new"
                className={PageCommon.link}
              >
                Learn more here.
              </a>
            </>
          </ToolTip>
        </div>

        {tenant?.is_admin && (
          <Button
            theme="primary"
            size="small"
            className={PageCommon.listviewtablecontrolsButton}
            onClick={() => {
              dispatch(onCreateAppClick(tenant?.id || ''));
            }}
          >
            Create App
          </Button>
        )}
      </div>
      <Card
        lockedMessage={readOnly ? 'You do not have edit access' : ''}
        listview
      >
        {fetchError || !plexConfig || !modifiedConfig ? (
          <Text>{fetchError || 'Loading...'}</Text>
        ) : (
          <PlexApps
            apps={modifiedConfig.tenant_config.plex_map.apps}
            employeeApp={modifiedConfig.tenant_config.plex_map.employee_app}
            selectedTenant={tenant}
            query={query}
          />
        )}
      </Card>
    </>
  );
};
const ConnectedLoginApps = connect((state: RootState) => {
  return {
    tenant: state.selectedTenant,
    plexConfig: state.tenantPlexConfig,
    modifiedConfig: state.modifiedPlexConfig,
    fetchError: state.fetchPlexConfigError,
    query: state.query,
  };
})(LoginApps);

const LoginAppsPage = ({
  tenantID,
  query,
  dispatch,
}: {
  tenantID: string | undefined;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (tenantID) {
      dispatch(fetchPlexConfig(tenantID));
    }
  }, [tenantID, query, dispatch]);

  return <ConnectedLoginApps />;
};

export default connect((state: RootState) => {
  return {
    tenantID: state.selectedTenantID,
    query: state.query,
  };
})(LoginAppsPage);
