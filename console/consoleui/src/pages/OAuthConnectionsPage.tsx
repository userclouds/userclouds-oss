import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
  IconFacebookBlackAndWhite,
  IconGoogleBlackAndWhite,
  IconLinkedInBlackAndWhite,
  IconMicrosoftInBlackAndWhite,
  IconLogin,
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

import { AppDispatch, RootState } from '../store';
import TenantPlexConfig from '../models/TenantPlexConfig';
import { SelectedTenant } from '../models/Tenant';
import { OIDCProviderType } from '../models/OIDCProvider';
import { fetchPlexConfig } from '../thunks/authn';
import Link from '../controls/Link';
import PageCommon from './PageCommon.module.css';
import styles from './OAuthConnectionsPage.module.css';

const getProviderIcon = (providerType: string) => {
  if (providerType === OIDCProviderType.Custom) {
    return <IconLogin />;
  }
  if (providerType === OIDCProviderType.Facebook) {
    return <IconFacebookBlackAndWhite />;
  }
  if (providerType === OIDCProviderType.Google) {
    return <IconGoogleBlackAndWhite />;
  }
  if (providerType === OIDCProviderType.LinkedIn) {
    return <IconLinkedInBlackAndWhite />;
  }
  if (providerType === OIDCProviderType.Microsoft) {
    return <IconMicrosoftInBlackAndWhite />;
  }
};

const OAuthConnections = ({
  tenant,
  plexConfig,
  modifiedConfig,
  fetchError,
  query,
}: {
  tenant: SelectedTenant | undefined;
  plexConfig: TenantPlexConfig | undefined;
  modifiedConfig: TenantPlexConfig | undefined;
  fetchError: string;
  query: URLSearchParams;
}) => {
  if (fetchError || !plexConfig || !modifiedConfig) {
    return <Text>{fetchError || 'Loading...'}</Text>;
  }

  const readOnly = !tenant?.is_admin;

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              Configure Social and other 3rd party OIDC Identity Providers for
              Plex.
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
          >
            <Link
              href={
                '/oauthconnections/oidc_provider/create' +
                makeCleanPageLink(query)
              }
              applyStyles={false}
            >
              Create Provider
            </Link>
          </Button>
        )}
      </div>
      <Card
        lockedMessage={readOnly ? 'You do not have edit access' : ''}
        listview
      >
        <Table
          spacing="packed"
          id="oauthConnections"
          className={styles.oathconnectionstable}
        >
          <TableHead>
            <TableRow>
              <TableRowHead key="type_headertype">Type</TableRowHead>
              <TableRowHead key="type_headerdesc">Provider Name</TableRowHead>
              <TableRowHead key="type_headerurl">Provider URL</TableRowHead>
              <TableRowHead key="type_headerconfigured">
                Configured
              </TableRowHead>
            </TableRow>
          </TableHead>
          <TableBody>
            {modifiedConfig.tenant_config.oidc_providers.providers.map(
              (provider) => (
                <TableRow key={provider.name}>
                  <TableCell>{getProviderIcon(provider.type)}</TableCell>
                  <TableCell>
                    {!readOnly ? (
                      <Link
                        key={provider.name + provider.issuer_url}
                        href={
                          `/oauthconnections/oidc_provider/${provider.name}` +
                          makeCleanPageLink(query)
                        }
                      >
                        {provider.name}
                      </Link>
                    ) : (
                      provider.name
                    )}
                  </TableCell>
                  <TableCell>{provider.issuer_url}</TableCell>
                  <TableCell>
                    {provider.client_id && provider.client_secret
                      ? 'Yes'
                      : 'No'}
                  </TableCell>
                </TableRow>
              )
            )}
          </TableBody>
        </Table>
      </Card>
    </>
  );
};
const ConnectedOAuthConnections = connect((state: RootState) => {
  return {
    tenant: state.selectedTenant,
    plexConfig: state.tenantPlexConfig,
    modifiedConfig: state.modifiedPlexConfig,
    fetchError: state.fetchPlexConfigError,
    query: state.query,
  };
})(OAuthConnections);

const OAuthConnectionsPage = ({
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

  return <ConnectedOAuthConnections />;
};

export default connect((state: RootState) => {
  return {
    tenantID: state.selectedTenantID,
    query: state.query,
  };
})(OAuthConnectionsPage);
