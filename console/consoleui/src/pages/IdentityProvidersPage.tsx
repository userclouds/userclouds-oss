import React, { useEffect } from 'react';
import { connect } from 'react-redux';
import { v4 as uuidv4 } from 'uuid';

import {
  Button,
  Card,
  InputReadOnly,
  Label,
  Select,
  InlineNotification,
  Table,
  TableRow,
  TableRowHead,
  TableBody,
  TableCell,
  TableHead,
  Text,
  ToolTip,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import { modifyPlexConfig } from '../actions/authn';

import { AppDispatch, RootState } from '../store';
import TenantPlexConfig, {
  addProvider,
  UpdatePlexConfigReason,
} from '../models/TenantPlexConfig';
import Provider, { ProviderType } from '../models/Provider';
import Tenant, { SelectedTenant } from '../models/Tenant';

import { fetchPlexConfig, savePlexConfig } from '../thunks/authn';
import Link from '../controls/Link';
import PageCommon from './PageCommon.module.css';
import styles from './IdentityProvidersPage.module.css';

const LoginProviders = ({
  modifiedProviders,
  savedProviders,
  query,
}: {
  modifiedProviders: Provider[];
  savedProviders: Provider[];
  query: URLSearchParams;
}) => {
  const cleanQuery = makeCleanPageLink(query);
  return (
    <Table
      spacing="packed"
      id="identityProviders"
      className={styles.providerstable}
    >
      <TableHead>
        <TableRow>
          <TableRowHead>Identity Provider</TableRowHead>
          <TableRowHead>Type</TableRowHead>
        </TableRow>
      </TableHead>
      <TableBody>
        {modifiedProviders.map((provider) => (
          <TableRow key={provider.id}>
            <TableCell>
              {savedProviders.find((p: Provider) => p.id === provider.id) ? (
                <Link
                  href={`/identityproviders/plex_provider/${provider.id}${cleanQuery}`}
                >
                  {provider.name}
                </Link>
              ) : (
                provider.name
              )}
            </TableCell>
            <TableCell>
              {provider.type === ProviderType.uc
                ? 'UserClouds'
                : provider.type === ProviderType.auth0
                  ? 'Auth0'
                  : 'Cognito'}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
};

const onAddProviderClick =
  (tenant: Tenant, modifiedConfig: TenantPlexConfig) =>
  async (dispatch: AppDispatch) => {
    const newProviderID = uuidv4();
    dispatch(
      modifyPlexConfig(
        addProvider(modifiedConfig, {
          id: newProviderID,
          name: 'New Plex Provider',
          type: ProviderType.uc,
          uc: {
            idp_url: tenant.tenant_url ? tenant.tenant_url : 'abc',
            apps: [],
          },
        })
      )
    );
  };

const IdentityProviders = ({
  tenant,
  plexConfig,
  modifiedConfig,
  isDirty,
  fetchError,
  isSaving,
  saveSuccess,
  saveError,
  query,
  dispatch,
}: {
  tenant: SelectedTenant | undefined;
  plexConfig: TenantPlexConfig | undefined;
  modifiedConfig: TenantPlexConfig | undefined;
  isDirty: boolean;
  fetchError: string;
  isSaving: boolean;
  saveSuccess: UpdatePlexConfigReason | undefined;
  saveError: string;
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
              Connect UserClouds to a pre-existing Identity Provider like Auth0
              or Cognito. UserClouds can automatically migrate from your
              existing Auth0 User Database or run two platforms simultaneously
              to minimize downtime.
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

        {tenant?.is_admin && modifiedConfig && (
          <Button
            theme="primary"
            size="small"
            className={PageCommon.listviewtablecontrolsButton}
            onClick={() => {
              dispatch(onAddProviderClick(tenant, modifiedConfig));
            }}
          >
            Create Provider
          </Button>
        )}
      </div>
      <Card listview>
        {fetchError || !plexConfig || !modifiedConfig ? (
          <Text>{fetchError || 'Loading...'}</Text>
        ) : (
          <>
            <LoginProviders
              modifiedProviders={
                modifiedConfig.tenant_config.plex_map.providers
              }
              savedProviders={plexConfig.tenant_config.plex_map.providers}
              query={query}
            />

            <Label>
              Active Provider:
              <br />
              {readOnly ? (
                <InputReadOnly>
                  {
                    modifiedConfig.tenant_config.plex_map.providers.find(
                      (p) =>
                        p.id ===
                        modifiedConfig.tenant_config.plex_map.policy
                          .active_provider_id
                    )?.name
                  }
                </InputReadOnly>
              ) : (
                <Select
                  name="active_provider"
                  defaultValue={
                    modifiedConfig.tenant_config.plex_map.policy
                      .active_provider_id
                  }
                  onChange={(e: React.ChangeEvent) => {
                    modifiedConfig.tenant_config.plex_map.policy.active_provider_id =
                      (e.target as HTMLSelectElement).value;
                    dispatch(modifyPlexConfig(modifiedConfig));
                  }}
                >
                  {modifiedConfig.tenant_config.plex_map.providers.map(
                    (provider) => (
                      <option key={`select-${provider.id}`} value={provider.id}>
                        {provider.name}
                      </option>
                    )
                  )}
                </Select>
              )}
            </Label>
            {saveError && (
              <InlineNotification theme="alert">{saveError}</InlineNotification>
            )}
            {saveSuccess === UpdatePlexConfigReason.AddProvider && (
              <InlineNotification theme="success">
                {saveSuccess}
              </InlineNotification>
            )}
            {tenant?.is_admin && (
              <Button
                disabled={!isDirty}
                isLoading={isSaving}
                onClick={() => {
                  dispatch(
                    savePlexConfig(
                      tenant.id,
                      modifiedConfig,
                      UpdatePlexConfigReason.AddProvider
                    )
                  );
                }}
              >
                Save providers
              </Button>
            )}
          </>
        )}
      </Card>
    </>
  );
};
const ConnectedIdentityProviders = connect((state: RootState) => {
  return {
    tenant: state.selectedTenant,
    plexConfig: state.tenantPlexConfig,
    modifiedConfig: state.modifiedPlexConfig,
    isDirty: state.plexConfigIsDirty,
    fetchError: state.fetchPlexConfigError,
    isSaving: state.savingPlexConfig,
    saveSuccess: state.savePlexConfigSuccess,
    saveError: state.savePlexConfigError,
    query: state.query,
  };
})(IdentityProviders);

const IdentityProvidersPage = ({
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

  return <ConnectedIdentityProviders />;
};

export default connect((state: RootState) => {
  return {
    tenantID: state.selectedTenantID,
    query: state.query,
  };
})(IdentityProvidersPage);
