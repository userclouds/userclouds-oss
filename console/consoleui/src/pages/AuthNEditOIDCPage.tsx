import React, { useEffect, useState } from 'react';
import { connect } from 'react-redux';

import { APIError } from '@userclouds/sharedui';
import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  HiddenTextInput,
  IconButton,
  IconDeleteBin,
  InlineNotification,
  Label,
  Table,
  TableRow,
  TableBody,
  TableCell,
  TextInput,
  ToolTip,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import { redirect } from '../routing';
import { AppDispatch, RootState } from '../store';
import TenantPlexConfig, {
  replaceOIDCProvider,
  UpdatePlexConfigReason,
} from '../models/TenantPlexConfig';
import { OIDCProvider } from '../models/OIDCProvider';
import {
  changeCurrentOIDCProvider,
  getBlankOIDCProvider,
  updateOIDCProviderRequest,
  updateOIDCProviderSuccess,
  updateOIDCProviderError,
  togglePlexConfigEditMode,
} from '../actions/authn';
import { createOIDCProvider, deleteOIDCProvider } from '../API/authn';
import { fetchPlexConfig, savePlexConfig } from '../thunks/authn';
import { postSuccessToast } from '../thunks/notifications';
import Styles from './AuthNPage.module.css';
import PageCommon from './PageCommon.module.css';

import { PageTitle } from '../mainlayout/PageWrap';

type Scope = { scope: string; id: number };

const addAdditionalScopesToProvider = (
  oidcProvider: OIDCProvider,
  additionalScopes: Scope[]
) => {
  oidcProvider.additional_scopes = additionalScopes
    .map((s) => {
      return s.scope;
    })
    .filter((s) => {
      return s !== '' && s !== ' ';
    })
    .join(' ');
  return oidcProvider;
};

const onSubmitHandlerEdit =
  (
    tenantID: string,
    plexConfig: TenantPlexConfig | undefined,
    oidcProvider: OIDCProvider,
    oidcProviderName: string | undefined,
    additionalScopes: Scope[] | undefined
  ) =>
  async (dispatch: AppDispatch) => {
    if (plexConfig) {
      if (additionalScopes) {
        dispatch(
          saveTenant(
            tenantID,
            plexConfig,
            addAdditionalScopesToProvider(oidcProvider, additionalScopes),
            oidcProviderName
          )
        );
      } else {
        dispatch(
          saveTenant(tenantID, plexConfig, oidcProvider, oidcProviderName)
        );
      }
    }
  };

const onSubmitHandlerCreate =
  (
    tenantID: string | undefined,
    oidcProvider: OIDCProvider,
    additionalScopes: Scope[] | undefined,
    searchParams: URLSearchParams
  ) =>
  async (dispatch: AppDispatch) => {
    if (tenantID) {
      if (additionalScopes) {
        dispatch(
          createOIDC(
            tenantID,
            addAdditionalScopesToProvider(oidcProvider, additionalScopes),
            searchParams
          )
        );
      } else {
        dispatch(createOIDC(tenantID, oidcProvider, searchParams));
      }
    }
  };

const saveTenant =
  (
    tenantID: string,
    config: TenantPlexConfig,
    oidcProvider: OIDCProvider,
    oidcProviderName: string | undefined
  ) =>
  async (dispatch: AppDispatch) => {
    if (!config || !oidcProvider || !oidcProviderName) {
      return;
    }

    dispatch(
      savePlexConfig(
        tenantID,
        replaceOIDCProvider(config, oidcProvider, oidcProviderName),
        UpdatePlexConfigReason.ModifyOIDCProvider
      )
    );
  };

const createOIDC =
  (
    tenantID: string,
    oidcProvider: OIDCProvider,
    searchParams: URLSearchParams
  ) =>
  async (dispatch: AppDispatch) => {
    if (!tenantID || !oidcProvider) {
      return;
    }
    dispatch(updateOIDCProviderRequest());
    createOIDCProvider(tenantID, oidcProvider).then(
      () => {
        dispatch(updateOIDCProviderSuccess());
        dispatch(fetchPlexConfig(tenantID));
        const url =
          '/oauthconnections/oidc_provider/' +
          oidcProvider.name +
          makeCleanPageLink(searchParams);
        redirect(url);
      },
      (error: APIError) => {
        dispatch(updateOIDCProviderError(error));
      }
    );
  };
const deleteProvider =
  (
    tenantID: string,
    oidcProvider: OIDCProvider,
    searchParams: URLSearchParams
  ) =>
  async (dispatch: AppDispatch) => {
    if (!tenantID || !oidcProvider) {
      return;
    }
    if (window.confirm(`Are you sure you want to delete this OIDC Provider?`)) {
      dispatch(updateOIDCProviderRequest());
      deleteOIDCProvider(tenantID, oidcProvider.name).then(
        () => {
          dispatch(updateOIDCProviderSuccess());
          postSuccessToast('Successfully deleted provider');
          redirect('/oauthconnections' + makeCleanPageLink(searchParams));
        },
        (error: APIError) => {
          dispatch(updateOIDCProviderError(error));
        }
      );
    }
  };

const getScopesFromString = (scopes: string) => {
  return scopes.length > 0
    ? scopes.split(' ').map((str, i) => {
        return {
          scope: str,
          id: i,
        };
      })
    : undefined;
};

const deleteScopeByID = (id: number, scopes: Scope[] | undefined) => {
  return scopes
    ? scopes
        .filter((s) => s.id !== id)
        .map((s) => {
          return s;
        })
    : [];
};

const AuthNOIDC = ({
  tenantID,
  oidcProvider,
  plexConfig,
  isSavingPlexConfig,
  saveSuccessPlexConfig,
  saveErrorPlexConfig,
  editMode,
  isSavingOIDCProvider,
  saveSuccessOIDCProvider,
  saveErrorOIDCProvider,
  location,
  query,
  routeParams,
  dispatch,
}: {
  tenantID: string | undefined;
  oidcProvider: OIDCProvider | undefined;
  plexConfig: TenantPlexConfig | undefined;
  isSavingPlexConfig: boolean;
  saveSuccessPlexConfig: UpdatePlexConfigReason | undefined;
  saveErrorPlexConfig: string;
  editMode: boolean;
  isSavingOIDCProvider: boolean;
  saveSuccessOIDCProvider: boolean;
  saveErrorOIDCProvider: string;
  location: URL;
  query: URLSearchParams;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const disabled = new Map();
  disabled.set('openid', true);
  disabled.set('profile', true);
  disabled.set('email', true);
  const { pathname } = location;
  const createPage = pathname.indexOf('create') > -1;
  const { oidcProviderName } = routeParams;

  const [additionalScopes, setAdditionalScopes] = useState<Scope[]>();

  useEffect(() => {
    if (tenantID) {
      if (createPage) {
        if (!oidcProvider || !(oidcProvider.type === 'custom')) {
          dispatch(getBlankOIDCProvider());
        }
        if (!additionalScopes && oidcProvider?.additional_scopes) {
          setAdditionalScopes(
            getScopesFromString(oidcProvider.additional_scopes)
          );
        }
      } else if (plexConfig) {
        const myProvider =
          plexConfig.tenant_config.oidc_providers.providers.find(
            ({ name }) => name === oidcProviderName
          );
        if (
          !oidcProvider ||
          !(oidcProvider.name === oidcProviderName) ||
          !myProvider
        ) {
          if (myProvider) {
            dispatch(changeCurrentOIDCProvider(myProvider));
          }
        }
        if (!additionalScopes && oidcProvider?.additional_scopes) {
          setAdditionalScopes(
            getScopesFromString(oidcProvider.additional_scopes)
          );
        }
      }
    }
  }, [
    oidcProvider,
    plexConfig,
    tenantID,
    oidcProviderName,
    createPage,
    additionalScopes,
    dispatch,
  ]);

  if (!tenantID) {
    return (
      <InlineNotification theme="alert">
        Error fetching Tenant
      </InlineNotification>
    );
  }
  return oidcProvider ? (
    <form
      onSubmit={(e: React.FormEvent) => {
        e.preventDefault();
        if (createPage) {
          dispatch(
            onSubmitHandlerCreate(
              tenantID,
              oidcProvider,
              additionalScopes,
              query
            )
          );
        } else {
          dispatch(
            onSubmitHandlerEdit(
              tenantID,
              plexConfig,
              oidcProvider,
              oidcProviderName,
              additionalScopes
            )
          );
        }
      }}
      className={Styles.oidcForm}
    >
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle
          title="OIDC Connection"
          itemName={createPage ? 'New Connection' : oidcProvider.name}
        />

        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              {
                'Configure social and other 3rd party OIDC Identity Providers for Plex. '
              }
              <a
                href="https://docs.userclouds.com/docs/authentication-methods#1-configure-your-account-with-the-third-party"
                title="UserClouds documentation for key concepts about OIDC Identity Providers"
                target="new"
                className={PageCommon.link}
              >
                Learn more here.
              </a>
            </>
          </ToolTip>
        </div>

        {createPage ? (
          <ButtonGroup>
            <Button
              isLoading={isSavingPlexConfig || isSavingOIDCProvider}
              type="submit"
              size="small"
              theme="primary"
            >
              Create Connection
            </Button>
          </ButtonGroup>
        ) : (
          <ButtonGroup>
            {editMode ? (
              <>
                {oidcProvider.type === 'custom' && editMode && (
                  <Button
                    isLoading={isSavingPlexConfig || isSavingOIDCProvider}
                    theme="dangerous"
                    size="small"
                    onClick={() => {
                      if (tenantID) {
                        dispatch(deleteProvider(tenantID, oidcProvider, query));
                      }
                    }}
                  >
                    Delete
                  </Button>
                )}

                <Button
                  size="small"
                  theme="secondary"
                  onClick={() => {
                    dispatch(togglePlexConfigEditMode());
                    if (plexConfig) {
                      const myProvider =
                        plexConfig.tenant_config.oidc_providers.providers.find(
                          ({ name }) => name === oidcProviderName
                        );
                      if (myProvider) {
                        setAdditionalScopes(
                          getScopesFromString(oidcProvider.additional_scopes)
                        );
                        dispatch(changeCurrentOIDCProvider(myProvider));
                      }
                    }
                  }}
                >
                  Cancel
                </Button>
                <Button
                  isLoading={isSavingPlexConfig || isSavingOIDCProvider}
                  type="submit"
                  size="small"
                  theme="primary"
                >
                  Save Connection
                </Button>
              </>
            ) : (
              <Button
                onClick={() => {
                  dispatch(togglePlexConfigEditMode());
                }}
                size="small"
                theme="primary"
              >
                Edit Connection
              </Button>
            )}
          </ButtonGroup>
        )}
      </div>

      <Card detailview>
        {saveErrorPlexConfig && (
          <InlineNotification theme="alert">
            {saveErrorPlexConfig}
          </InlineNotification>
        )}
        {saveErrorOIDCProvider && (
          <InlineNotification theme="alert">
            {saveErrorOIDCProvider}
          </InlineNotification>
        )}
        {saveSuccessPlexConfig && (
          <InlineNotification theme="success">
            {saveSuccessPlexConfig}
          </InlineNotification>
        )}
        {saveSuccessOIDCProvider && (
          <InlineNotification theme="success">
            Successfully saved.
          </InlineNotification>
        )}

        <CardRow
          title="Basic Details"
          tooltip={<>Configure the basic details of this connection</>}
          collapsible
        >
          <Label htmlFor="Connection Provider">
            Connection Provider
            <br />
            <TextInput
              id="provider_type"
              name="provider_type"
              disabled
              value={oidcProvider.type}
            />
          </Label>
          <Label>
            Provider Name (must be unique)
            <br />
            <TextInput
              id="provider_name"
              name="provider_name"
              disabled={!createPage}
              value={oidcProvider.name}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                dispatch(
                  changeCurrentOIDCProvider({
                    ...oidcProvider,
                    name: e.target.value,
                  })
                );
              }}
            />
          </Label>
          <Label>
            Description
            <br />
            <TextInput
              id="provider_description"
              name="provider_description"
              value={oidcProvider.description}
              disabled={
                !createPage && (!editMode || oidcProvider.type !== 'custom')
              }
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                dispatch(
                  changeCurrentOIDCProvider({
                    ...oidcProvider,
                    description: e.target.value,
                  })
                );
              }}
            />
          </Label>
        </CardRow>
        <CardRow
          title="Configuration"
          tooltip={<>Configure the settings of this connection.</>}
          collapsible
        >
          <Label>
            Provider URL (must be unique)
            <br />
            <TextInput
              id="provider_url"
              name="provider_url"
              value={oidcProvider.issuer_url}
              disabled={
                !createPage && (!editMode || oidcProvider.type !== 'custom')
              }
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                dispatch(
                  changeCurrentOIDCProvider({
                    ...oidcProvider,
                    issuer_url: e.target.value,
                  })
                );
              }}
            />
          </Label>
          <Label>
            Client ID
            <br />
            <TextInput
              id="client_id"
              name="client_id"
              value={oidcProvider.client_id}
              disabled={!createPage && !editMode}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                dispatch(
                  changeCurrentOIDCProvider({
                    ...oidcProvider,
                    client_id: e.target.value,
                  })
                );
              }}
            />
          </Label>
          <Label>
            Client secret
            <br />
            <HiddenTextInput
              id="client_secret"
              name="client_secret"
              value={oidcProvider.client_secret}
              disabled={!createPage && !editMode}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                dispatch(
                  changeCurrentOIDCProvider({
                    ...oidcProvider,
                    client_secret: e.target.value,
                  })
                );
              }}
            />
          </Label>
          <Label element="h4">
            Scopes
            <br />
            <Table>
              <TableBody>
                {oidcProvider.default_scopes.split(' ').map((scope) => (
                  <TableRow key={scope}>
                    <TableCell key={scope}>
                      <TextInput
                        id={'defaultScopeName' + scope}
                        name={'defaultScopeName' + scope}
                        key={'defaultScopeName' + scope}
                        disabled
                        value={scope}
                      />
                    </TableCell>
                  </TableRow>
                ))}
                {additionalScopes !== undefined &&
                  additionalScopes.map((scope) => (
                    <TableRow key={scope.id}>
                      <TableCell>
                        <TextInput
                          id={`additionalScopeName` + scope.id}
                          name={`additionalScopeName` + scope.id}
                          disabled={!createPage && !editMode}
                          value={scope.scope}
                          onChange={(
                            e: React.ChangeEvent<HTMLInputElement>
                          ) => {
                            const newScope = e.target.value.replaceAll(
                              ' ',
                              '_'
                            );
                            setAdditionalScopes(
                              additionalScopes?.map((s) => {
                                if (s.id === scope.id) {
                                  return { id: s.id, scope: newScope };
                                }
                                return s;
                              })
                            );
                          }}
                        />
                      </TableCell>
                      <TableCell>
                        <IconButton
                          icon={<IconDeleteBin />}
                          disabled={!createPage && !editMode}
                          onClick={() => {
                            setAdditionalScopes(
                              deleteScopeByID(scope.id, additionalScopes)
                            );
                          }}
                          title="Delete Scope"
                          aria-label="Delete Scope"
                        />
                      </TableCell>
                    </TableRow>
                  ))}
              </TableBody>
            </Table>
          </Label>
          <Button
            className={Styles.addScopeButton}
            size="small"
            theme="outline"
            isLoading={isSavingPlexConfig || isSavingOIDCProvider}
            disabled={!createPage && !editMode}
            onClick={() => {
              if (additionalScopes) {
                const id = Math.max(...additionalScopes.map((o) => o.id), 0);
                additionalScopes.push({ scope: 'new_scope', id: id + 1 });
                setAdditionalScopes(
                  additionalScopes.map((s) => {
                    return s;
                  })
                );
              } else {
                setAdditionalScopes([{ scope: 'new_scope', id: 0 }]);
              }
            }}
          >
            Add Scope
          </Button>
        </CardRow>
      </Card>
    </form>
  ) : (
    <Card>
      <InlineNotification theme="alert">Provider Not Found</InlineNotification>
    </Card>
  );
};

const ConnectedAuthNOIDC = connect((state: RootState) => {
  return {
    tenantID: state.selectedTenantID,
    oidcProvider: state.oidcProvider,
    plexConfig: state.tenantPlexConfig,
    modifiedConfig: state.modifiedPlexConfig,
    isDirtyPlexConfig: state.plexConfigIsDirty,
    isSavingPlexConfig: state.savingPlexConfig,
    saveSuccessPlexConfig: state.savePlexConfigSuccess,
    saveErrorPlexConfig: state.savePlexConfigError,
    editMode: state.editingPlexConfig,
    isSavingOIDCProvider: state.savingOIDCProvider,
    saveSuccessOIDCProvider: state.saveOIDCProviderSuccess,
    saveErrorOIDCProvider: state.saveOIDCProviderError,
    location: state.location,
    query: state.query,
    routeParams: state.routeParams,
  };
})(AuthNOIDC);

const AuthNEditOIDCPage = ({
  tenantID,
  dispatch,
}: {
  tenantID: string | undefined;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (tenantID) {
      dispatch(fetchPlexConfig(tenantID));
    }
  }, [tenantID, dispatch]);

  return <ConnectedAuthNOIDC />;
};

export default connect((state: RootState) => {
  return {
    tenantID: state.selectedTenantID,
    oidcProvider: state.oidcProvider,
    plexConfig: state.tenantPlexConfig,
    modifiedConfig: state.modifiedPlexConfig,
  };
})(AuthNEditOIDCPage);
