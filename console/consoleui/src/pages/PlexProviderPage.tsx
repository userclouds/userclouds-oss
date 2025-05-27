import React, { useEffect, useState } from 'react';
import { connect } from 'react-redux';
import { v4 as uuidv4 } from 'uuid';

import {
  Button,
  ButtonGroup,
  Card,
  CardFooter,
  CardRow,
  EmptyState,
  GlobalStyles,
  HiddenTextInput,
  IconButton,
  IconDeleteBin,
  IconShieldKeyhole,
  InputReadOnly,
  InputReadOnlyHidden,
  TextInput,
  Label,
  Heading,
  Select,
  InlineNotification,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  Text,
  TextShortener,
} from '@userclouds/ui-component-lib';
import { APIError, makeAPIError } from '@userclouds/sharedui';

import { redirect } from '../routing';
import { AppDispatch, RootState } from '../store';
import { saveTenantPlexConfig } from '../API/authn';
import { fetchPlexConfig, savePlexConfig } from '../thunks/authn';
import {
  selectPlexProvider,
  modifyPlexProvider,
  updatePlexConfigRequest,
  updatePlexConfigSuccess,
  updatePlexConfigError,
  toggleAuth0AppsEditMode,
  toggleCognitoAppsEditMode,
  toggleUCAppsEditMode,
} from '../actions/authn';
import TenantPlexConfig, {
  findPlexAppReferencingProvider,
  findPlexAppReferencingProviderApp,
  replaceProvider,
  UpdatePlexConfigReason,
} from '../models/TenantPlexConfig';
import Provider, { ProviderType } from '../models/Provider';
import { SelectedTenant } from '../models/Tenant';
import NavigationBlocker from '../NavigationBlocker';
import PageCommon from './PageCommon.module.css';
import Styles from './PlexProviderPage.module.css';
import { blankAuth0Provider } from '../models/Auth0Provider';
import { blankUCProvider } from '../models/UCProvider';
import { blankCognitoProvider } from '../models/CognitoProvider';
import DeleteWithConfirmationButton from '../controls/DeleteWithConfirmationButton';

const onCreateAuth0AppClick =
  (provider: Provider) => (dispatch: AppDispatch) => {
    if (!provider.auth0) {
      return;
    }
    const newAppID = uuidv4();

    provider.auth0.apps.push({
      id: newAppID,
      name: 'New Auth0 App',
      client_id: 'fill-in-client-id',
      client_secret: 'fill-in-client-secret',
    });
    dispatch(modifyPlexProvider(provider));
  };
const onDeleteAuth0AppClick =
  (appID: string, provider: Provider, plexConfig: TenantPlexConfig) =>
  (dispatch: AppDispatch) => {
    if (!provider.auth0) {
      return;
    }
    const plexAppReferencingProviderApp = findPlexAppReferencingProviderApp(
      plexConfig.tenant_config.plex_map.apps,
      appID
    );
    if (plexAppReferencingProviderApp) {
      dispatch(
        updatePlexConfigError(
          makeAPIError(
            undefined,
            `Cannot delete provider app, since it is referenced by Plex App '${plexAppReferencingProviderApp.name}'.`
          )
        )
      );
      return;
    }

    provider.auth0.apps = provider.auth0.apps.filter((app) => app.id !== appID);
    dispatch(modifyPlexProvider(provider));
  };

const saveProvider =
  (
    tenantID: string,
    plexConfig: TenantPlexConfig | undefined,
    provider: Provider,
    reason: UpdatePlexConfigReason
  ) =>
  (dispatch: AppDispatch) => {
    if (!plexConfig || !provider) {
      return;
    }
    // Copy our local Plex Provider state back into the plex config object
    // TODO: reducer for the call to replaceProvider?
    dispatch(
      savePlexConfig(tenantID, replaceProvider(plexConfig, provider), reason)
    );
  };

const Auth0Apps = ({
  selectedTenant,
  provider,
  plexConfig,
  editMode,
  saveSuccess,
  saveError,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  provider: Provider | undefined;
  plexConfig: TenantPlexConfig | undefined;
  editMode: boolean;
  saveSuccess: UpdatePlexConfigReason | undefined;
  saveError: string;
  dispatch: AppDispatch;
}) => {
  if (!plexConfig || !provider || provider.type !== ProviderType.auth0) {
    return <></>;
  }

  return (
    <Card
      title="Auth0 Apps"
      description="Configure underlying Auth0 applications so they can be mapped to Plex applications."
      isDirty={editMode}
    >
      {provider.auth0?.apps && provider.auth0.apps.length ? (
        <>
          <Table spacing="packed" className={Styles.providerauth0apps}>
            <TableHead>
              <TableRow>
                <TableRowHead key="app_name_header">Name</TableRowHead>
                <TableRowHead key="app_id_header">ID</TableRowHead>
                <TableRowHead key="app_client_id_header">
                  Client ID
                </TableRowHead>
                <TableRowHead key="app_client_secret_header">
                  Client Secret
                </TableRowHead>
                <TableRowHead key="delete_app_header">&nbsp;</TableRowHead>
              </TableRow>
            </TableHead>
            <TableBody>
              {provider.auth0.apps.map((app) => (
                <TableRow key={app.id}>
                  <TableCell>
                    {editMode ? (
                      <TextInput
                        id="name"
                        name="name"
                        value={app.name}
                        onChange={(e: React.ChangeEvent) => {
                          const val = (e.target as HTMLInputElement).value;
                          app.name = val;
                          dispatch(modifyPlexProvider(provider));
                        }}
                      />
                    ) : (
                      <InputReadOnly>{app.name}</InputReadOnly>
                    )}
                  </TableCell>
                  <TableCell className={PageCommon.uuidtablecell}>
                    <InputReadOnly>{app.id}</InputReadOnly>
                  </TableCell>
                  <TableCell className={PageCommon.uuidtablecell}>
                    {editMode ? (
                      <TextInput
                        id="client_id"
                        name="client_id"
                        value={app.client_id}
                        onChange={(e: React.ChangeEvent) => {
                          const val = (e.target as HTMLInputElement).value;
                          app.client_id = val;
                          dispatch(modifyPlexProvider(provider));
                        }}
                      />
                    ) : (
                      <InputReadOnly>{app.client_id}</InputReadOnly>
                    )}
                  </TableCell>

                  <TableCell className={PageCommon.uuidtablecell}>
                    {editMode ? (
                      <HiddenTextInput
                        id="client_secret"
                        value={app.client_secret}
                        onChange={(e: React.ChangeEvent) => {
                          const val = (e.target as HTMLInputElement).value;
                          app.client_secret = val;
                          dispatch(modifyPlexProvider(provider));
                        }}
                      />
                    ) : (
                      <InputReadOnlyHidden value={app.client_secret} />
                    )}
                  </TableCell>
                  <TableCell>
                    {editMode && (
                      <IconButton
                        icon={<IconDeleteBin />}
                        onClick={() =>
                          dispatch(
                            onDeleteAuth0AppClick(app.id, provider, plexConfig)
                          )
                        }
                        title="Delete App"
                        aria-label="Delete App"
                      />
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {editMode && (
            <Button
              theme="secondary"
              onClick={() => {
                dispatch(onCreateAuth0AppClick(provider));
              }}
            >
              Create App
            </Button>
          )}
          {saveError && (
            <InlineNotification theme="alert">{saveError}</InlineNotification>
          )}
          {saveSuccess === UpdatePlexConfigReason.ModifyProviderApps && (
            <InlineNotification theme="success">
              {saveSuccess}
            </InlineNotification>
          )}
          <CardFooter>
            {selectedTenant?.is_admin && (
              <>
                {!editMode ? (
                  <Button
                    theme="secondary"
                    onClick={() => dispatch(toggleAuth0AppsEditMode(true))}
                  >
                    Edit Apps
                  </Button>
                ) : (
                  <ButtonGroup>
                    <Button
                      theme="primary"
                      onClick={() => {
                        dispatch(
                          saveProvider(
                            selectedTenant.id,
                            plexConfig,
                            provider,
                            UpdatePlexConfigReason.ModifyProviderApps
                          )
                        );
                      }}
                    >
                      Save
                    </Button>
                    <Button
                      theme="secondary"
                      onClick={() => {
                        dispatch(toggleAuth0AppsEditMode(false));
                      }}
                    >
                      Cancel
                    </Button>
                  </ButtonGroup>
                )}
              </>
            )}
          </CardFooter>
        </>
      ) : (
        <CardRow>
          <EmptyState
            title="No apps yet"
            image={<IconShieldKeyhole size="large" />}
          >
            {selectedTenant?.is_admin && (
              <Button
                theme="secondary"
                onClick={() => {
                  dispatch(onCreateAuth0AppClick(provider));
                }}
              >
                Create App
              </Button>
            )}
          </EmptyState>
        </CardRow>
      )}
    </Card>
  );
};
const ConnectedAuth0Apps = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  provider: state.modifiedPlexProvider,
  plexConfig: state.tenantPlexConfig,
  editMode: state.auth0AppsEditMode,
  saveSuccess: state.savePlexConfigSuccess,
  saveError: state.savePlexConfigError,
}))(Auth0Apps);

const onCreateCognitoAppClick =
  (provider: Provider) => (dispatch: AppDispatch) => {
    if (!provider.cognito) {
      return;
    }
    const newAppID = uuidv4();

    provider.cognito.apps.push({
      id: newAppID,
      name: 'New Cognito App',
      client_id: 'fill-in-client-id',
      client_secret: 'fill-in-client-secret',
    });
    dispatch(modifyPlexProvider(provider));
  };
const onDeleteCognitoAppClick =
  (appID: string, provider: Provider, plexConfig: TenantPlexConfig) =>
  (dispatch: AppDispatch) => {
    if (!provider.cognito) {
      return;
    }
    const plexAppReferencingProviderApp = findPlexAppReferencingProviderApp(
      plexConfig.tenant_config.plex_map.apps,
      appID
    );
    if (plexAppReferencingProviderApp) {
      dispatch(
        updatePlexConfigError(
          makeAPIError(
            undefined,
            `Cannot delete provider app, since it is referenced by Plex App '${plexAppReferencingProviderApp.name}'.`
          )
        )
      );
      return;
    }

    provider.cognito.apps = provider.cognito.apps.filter(
      (app) => app.id !== appID
    );
    dispatch(modifyPlexProvider(provider));
  };

const CognitoApps = ({
  selectedTenant,
  provider,
  plexConfig,
  editMode,
  saveSuccess,
  saveError,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  provider: Provider | undefined;
  plexConfig: TenantPlexConfig | undefined;
  editMode: boolean;
  saveSuccess: UpdatePlexConfigReason | undefined;
  saveError: string;
  dispatch: AppDispatch;
}) => {
  if (!plexConfig || !provider || provider.type !== ProviderType.cognito) {
    return <></>;
  }

  return (
    <Card
      title="Cognito Apps"
      description="Configure underlying Cognito applications so they can be mapped to Plex applications."
      isDirty={editMode}
    >
      {provider.cognito?.apps && provider.cognito.apps.length ? (
        <>
          <Table className={Styles.providercognitoapps}>
            <TableHead>
              <TableRow>
                <TableRowHead key="app_name_header">Name</TableRowHead>
                <TableRowHead key="app_id_header">ID</TableRowHead>
                <TableRowHead key="app_client_id_header">
                  Client ID
                </TableRowHead>
                <TableRowHead key="app_client_secret_header">
                  Client Secret
                </TableRowHead>
                <TableRowHead key="delete_app_header">&nbsp;</TableRowHead>
              </TableRow>
            </TableHead>
            <TableBody>
              {provider.cognito.apps.map((app) => (
                <TableRow key={app.id}>
                  <TableCell>
                    {editMode ? (
                      <TextInput
                        id="name"
                        name="name"
                        value={app.name}
                        onChange={(e: React.ChangeEvent) => {
                          const val = (e.target as HTMLInputElement).value;
                          app.name = val;
                          dispatch(modifyPlexProvider(provider));
                        }}
                      />
                    ) : (
                      <InputReadOnly>{app.name}</InputReadOnly>
                    )}
                  </TableCell>
                  <TableCell className={PageCommon.uuidtablecell}>
                    <InputReadOnly>{app.id}</InputReadOnly>
                  </TableCell>
                  <TableCell className={PageCommon.uuidtablecell}>
                    {editMode ? (
                      <TextInput
                        id="client_id"
                        name="client_id"
                        value={app.client_id}
                        onChange={(e: React.ChangeEvent) => {
                          const val = (e.target as HTMLInputElement).value;
                          app.client_id = val;
                          dispatch(modifyPlexProvider(provider));
                        }}
                      />
                    ) : (
                      <InputReadOnly>{app.client_id}</InputReadOnly>
                    )}
                  </TableCell>

                  <TableCell className={PageCommon.uuidtablecell}>
                    {editMode ? (
                      <HiddenTextInput
                        id="client_secret"
                        value={app.client_secret}
                        onChange={(e: React.ChangeEvent) => {
                          const val = (e.target as HTMLInputElement).value;
                          app.client_secret = val;
                          dispatch(modifyPlexProvider(provider));
                        }}
                      />
                    ) : (
                      <InputReadOnlyHidden value={app.client_secret} />
                    )}
                  </TableCell>
                  <TableCell>
                    {editMode && (
                      <IconButton
                        icon={<IconDeleteBin />}
                        onClick={() =>
                          dispatch(
                            onDeleteCognitoAppClick(
                              app.id,
                              provider,
                              plexConfig
                            )
                          )
                        }
                        title="Delete App"
                        aria-label="Delete App"
                      />
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {editMode && (
            <Button
              theme="secondary"
              onClick={() => {
                dispatch(onCreateCognitoAppClick(provider));
              }}
            >
              Create App
            </Button>
          )}
          {saveError && (
            <InlineNotification theme="alert">{saveError}</InlineNotification>
          )}
          {saveSuccess === UpdatePlexConfigReason.ModifyProviderApps && (
            <InlineNotification theme="success">
              {saveSuccess}
            </InlineNotification>
          )}
          <CardFooter>
            {selectedTenant?.is_admin && (
              <>
                {!editMode ? (
                  <Button
                    theme="secondary"
                    onClick={() => dispatch(toggleCognitoAppsEditMode(true))}
                  >
                    Edit Apps
                  </Button>
                ) : (
                  <ButtonGroup>
                    <Button
                      theme="primary"
                      onClick={() => {
                        dispatch(
                          saveProvider(
                            selectedTenant.id,
                            plexConfig,
                            provider,
                            UpdatePlexConfigReason.ModifyProviderApps
                          )
                        );
                      }}
                    >
                      Save
                    </Button>
                    <Button
                      theme="secondary"
                      onClick={() => {
                        dispatch(toggleCognitoAppsEditMode(false));
                      }}
                    >
                      Cancel
                    </Button>
                  </ButtonGroup>
                )}
              </>
            )}
          </CardFooter>
        </>
      ) : (
        <CardRow>
          <EmptyState
            title="No apps yet"
            image={<IconShieldKeyhole size="large" />}
          >
            {selectedTenant?.is_admin && (
              <Button
                theme="secondary"
                onClick={() => {
                  dispatch(onCreateCognitoAppClick(provider));
                }}
              >
                Create App
              </Button>
            )}
          </EmptyState>
        </CardRow>
      )}
    </Card>
  );
};
const ConnectedCognitoApps = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  provider: state.modifiedPlexProvider,
  plexConfig: state.tenantPlexConfig,
  editMode: state.cognitoAppsEditMode,
  saveSuccess: state.savePlexConfigSuccess,
  saveError: state.savePlexConfigError,
}))(CognitoApps);

const onCreateUCAppClick =
  (provider: Provider) => async (dispatch: AppDispatch) => {
    if (!provider.uc) {
      return;
    }

    const newAppID = uuidv4();

    provider.uc.apps.push(
      // Client ID & secret are not used for UC apps
      // TODO: get rid of UCApp altogether?
      {
        id: newAppID,
        name: 'New UserClouds App',
        client_id: '',
        client_secret: '',
      }
    );
    dispatch(modifyPlexProvider(provider));
  };

const onDeleteUCAppClick =
  (appID: string, provider: Provider, plexConfig: TenantPlexConfig) =>
  async (dispatch: AppDispatch) => {
    if (!provider.uc) {
      return;
    }

    const plexAppReferencingProviderApp = findPlexAppReferencingProviderApp(
      plexConfig.tenant_config.plex_map.apps,
      appID
    );
    if (plexAppReferencingProviderApp) {
      dispatch(
        updatePlexConfigError(
          makeAPIError(
            undefined,
            `Cannot delete provider app, since it is referenced by Plex App '${plexAppReferencingProviderApp.name}'.`
          )
        )
      );
      return;
    }

    provider.uc.apps = provider.uc.apps.filter((app) => app.id !== appID);
    dispatch(modifyPlexProvider(provider));
  };

const UCApps = ({
  selectedTenant,
  provider,
  plexConfig,
  editMode,
  saveSuccess,
  saveError,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  provider: Provider | undefined;
  plexConfig: TenantPlexConfig | undefined;
  editMode: boolean;
  saveSuccess: UpdatePlexConfigReason | undefined;
  saveError: string;
  dispatch: AppDispatch;
}) => {
  if (!plexConfig || !provider || provider.type !== ProviderType.uc) {
    return <></>;
  }
  return (
    <Card
      title="UserClouds Apps"
      description="Configure underlying UserClouds applications so they can be mapped to Plex applications."
    >
      {provider.uc?.apps && provider.uc.apps.length ? (
        <>
          <Table className={Styles.providerucapps}>
            <TableHead>
              <TableRow>
                <TableRowHead key="app_name_header">Name</TableRowHead>
                <TableRowHead key="app_id_header">ID</TableRowHead>
                <TableRowHead key="blank_header">&nbsp;</TableRowHead>
              </TableRow>
            </TableHead>
            <TableBody>
              {provider.uc.apps.map((app) => (
                <TableRow key={app.id}>
                  <TableCell>
                    {editMode ? (
                      <TextInput
                        id="app_name"
                        name="app_name"
                        value={app.name}
                        onChange={(e: React.ChangeEvent) => {
                          const val = (e.target as HTMLInputElement).value;
                          // TODO we're relying on a stable reference.
                          // We should modify provider directly
                          app.name = val;
                          dispatch(modifyPlexProvider(provider));
                        }}
                      />
                    ) : (
                      <InputReadOnly>{app.name}</InputReadOnly>
                    )}
                  </TableCell>
                  <TableCell className={PageCommon.uuidtablecell}>
                    <TextShortener text={app.id} length={6} />
                  </TableCell>
                  {editMode && (
                    <TableCell>
                      <DeleteWithConfirmationButton
                        id="deleteUCAppButton"
                        message="Are you sure you want to delete this app? This action is irreversible."
                        onConfirmDelete={() => {
                          dispatch(
                            onDeleteUCAppClick(app.id, provider, plexConfig)
                          );
                        }}
                        title="Delete App"
                      />
                    </TableCell>
                  )}
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {editMode && (
            <Button
              theme="secondary"
              onClick={() => {
                dispatch(onCreateUCAppClick(provider));
              }}
            >
              Create App
            </Button>
          )}
          {saveError && (
            <InlineNotification theme="alert">{saveError}</InlineNotification>
          )}
          {saveSuccess === UpdatePlexConfigReason.ModifyProviderApps && (
            <InlineNotification theme="success">
              {saveSuccess}
            </InlineNotification>
          )}
          <CardFooter>
            {selectedTenant?.is_admin && (
              <ButtonGroup>
                {!editMode ? (
                  <Button
                    theme="secondary"
                    onClick={() => dispatch(toggleUCAppsEditMode(true))}
                  >
                    Edit Apps
                  </Button>
                ) : (
                  <ButtonGroup>
                    <Button
                      theme="primary"
                      onClick={() => {
                        dispatch(
                          saveProvider(
                            selectedTenant.id,
                            plexConfig,
                            provider,
                            UpdatePlexConfigReason.ModifyProviderApps
                          )
                        );
                      }}
                    >
                      Save
                    </Button>
                    <Button
                      theme="secondary"
                      onClick={() => {
                        dispatch(toggleUCAppsEditMode(false));
                        dispatch(fetchPlexConfig(selectedTenant.id));
                      }}
                    >
                      Cancel
                    </Button>
                  </ButtonGroup>
                )}
              </ButtonGroup>
            )}
          </CardFooter>
        </>
      ) : (
        <CardRow>
          <EmptyState
            title="No apps yet"
            image={<IconShieldKeyhole size="large" />}
          >
            {selectedTenant?.is_admin && (
              <Button
                theme="secondary"
                onClick={() => {
                  dispatch(onCreateUCAppClick(provider));
                }}
              >
                Create App
              </Button>
            )}
          </EmptyState>
        </CardRow>
      )}
    </Card>
  );
};
const ConnectedUCApps = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  provider: state.modifiedPlexProvider,
  plexConfig: state.tenantPlexConfig,
  editMode: state.ucAppsEditMode,
  saveSuccess: state.savePlexConfigSuccess,
  saveError: state.savePlexConfigError,
}))(UCApps);

const ImportApps = ({
  tenantID,
  provider,
}: {
  tenantID: string | undefined;
  provider: Provider | undefined;
}) => {
  const [isImporting, setIsImporting] = useState<boolean>(true);
  return provider?.type === ProviderType.auth0 ? (
    <Card
      title="Import from Auth0"
      description="Automatically import your applications / clients from your Auth0 tenant"
    >
      <Button
        theme="secondary"
        onClick={() => {
          // TODO: this needs to give some better feedback, etc?
          setIsImporting(false);
          fetch(
            '/api/tenants/' + tenantID + '/plexconfig/providers/actions/import',
            { method: 'POST' }
          );
          setTimeout(() => window.location.reload(), 10000);
        }}
      >
        Import From Auth0
      </Button>
      {isImporting ? (
        <></>
      ) : (
        <Label id="importing" htmlFor="importing">
          Importing...
        </Label>
      )}
    </Card>
  ) : (
    <></>
  );
};
const ConnectedImportApps = connect((state: RootState) => ({
  tenantID: state.selectedTenantID,
  provider: state.selectedPlexProvider,
}))(ImportApps);

const deleteProvider =
  (
    companyID: string,
    tenantID: string,
    plexConfig: TenantPlexConfig | undefined,
    plexProvider: Provider | undefined
  ) =>
  async (dispatch: AppDispatch) => {
    if (!plexConfig || !plexProvider) {
      return;
    }
    if (
      plexConfig.tenant_config.plex_map.policy.active_provider_id ===
      plexProvider.id
    ) {
      dispatch(
        updatePlexConfigError(
          makeAPIError(undefined, 'Cannot delete active provider.')
        )
      );
      return;
    }

    const plexAppReferencingProvider = findPlexAppReferencingProvider(
      plexConfig.tenant_config.plex_map.apps,
      plexProvider
    );
    if (plexAppReferencingProvider) {
      dispatch(
        updatePlexConfigError(
          makeAPIError(
            undefined,
            `Cannot delete provider, since it has an app referenced by Plex App '${plexAppReferencingProvider.name}'.`
          )
        )
      );
      return;
    }

    plexConfig.tenant_config.plex_map.providers =
      plexConfig.tenant_config.plex_map.providers.filter(
        (provider) => provider.id !== plexProvider.id
      );

    dispatch(updatePlexConfigRequest());
    saveTenantPlexConfig(tenantID, plexConfig).then(
      (cfg: TenantPlexConfig) => {
        dispatch(updatePlexConfigSuccess(cfg));
      },
      (error: APIError) => {
        dispatch(updatePlexConfigError(error));

        // Navigate to identity providers or  authn page afterward
        redirect(
          `/identityproviders?company_id=${companyID}&tenant_id=${tenantID}`
        );
      }
    );
  };

const PlexProvider = ({
  selectedCompanyID,
  selectedTenant,
  plexConfig,
  selectedPlexProvider,
  modifiedPlexProvider,
  isDirty,
  fetchError,
  isSaving,
  saveSuccess,
  saveError,
  location,
  routeParams,
  dispatch,
}: {
  selectedCompanyID: string | undefined;
  selectedTenant: SelectedTenant | undefined;
  plexConfig: TenantPlexConfig | undefined;
  selectedPlexProvider: Provider | undefined;
  modifiedPlexProvider: Provider | undefined;
  isDirty: boolean;
  fetchError: string;
  isSaving: boolean;
  saveSuccess: UpdatePlexConfigReason | undefined;
  saveError: string;
  location: URL;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const { plexProviderID } = routeParams;

  useEffect(() => {
    if (plexConfig) {
      const matchingProvider = plexConfig.tenant_config.plex_map.providers.find(
        (provider: Provider) => provider.id === plexProviderID
      );
      if (!matchingProvider) {
        // App not found in tenant, go  back to identity providers page or AuthNPage
        redirect(
          `/identityproviders?company_id=${selectedCompanyID}&tenant_id=${selectedTenant?.id}`
        );
      } else if (
        !selectedPlexProvider ||
        selectedPlexProvider.id !== plexProviderID
      ) {
        dispatch(selectPlexProvider(plexProviderID as string));
      }
    } else {
      dispatch(fetchPlexConfig(selectedTenant?.id || ''));
    }
  }, [
    plexConfig,
    plexProviderID,
    selectedPlexProvider,
    modifiedPlexProvider,
    selectedTenant,
    selectedCompanyID,
    location,
    dispatch,
  ]);

  if (fetchError || !plexConfig) {
    return <Text>{fetchError || 'Loading...'}</Text>;
  }

  if (!modifiedPlexProvider) {
    return <div>{`Plex provider ${plexProviderID} not found`}</div>;
  }

  const providerTypes = [
    {
      key: 'auth0',
      label: 'Auth0',
      payload: ProviderType.auth0,
    },
    {
      key: 'uc',
      label: 'UserClouds',
      payload: ProviderType.uc,
    },
    {
      key: 'cognito',
      label: 'Cognito',
      payload: ProviderType.cognito,
    },
  ];

  const readOnly = !selectedTenant?.is_admin;

  return (
    <>
      <NavigationBlocker showPrompt={isDirty} />
      <Card
        title="General Settings"
        description="View and edit this Identity Platform’s name, company and tenant URL."
      >
        <fieldset>
          <Heading size="3" headingLevel="3">
            General Settings
          </Heading>
          <Label className={Styles.propertiesElement} htmlFor="id">
            ID
            <TextShortener text={modifiedPlexProvider.id} length={6} />
          </Label>
          <Label className={GlobalStyles['mt-3']}>
            Type
            <br />
            {readOnly ? (
              <InputReadOnly>
                {
                  providerTypes.find(
                    (o) => o.key === modifiedPlexProvider?.type
                  )?.label
                }
              </InputReadOnly>
            ) : (
              <Select
                name="provider_type"
                value={modifiedPlexProvider.type}
                onChange={(e: React.ChangeEvent) => {
                  if (
                    window.confirm(
                      'Changing provider type means losing the settings associated with your previous provider type. Are you sure you want to continue?'
                    )
                  ) {
                    const val = (e.target as HTMLSelectElement).value;
                    modifiedPlexProvider.type =
                      ProviderType[val as keyof typeof ProviderType];
                    // TODO (sgarrity 2/24): is there a cleaner way to handle this optional stuff?
                    modifiedPlexProvider.auth0 = undefined;
                    modifiedPlexProvider.uc = undefined;
                    modifiedPlexProvider.cognito = undefined;
                    if (modifiedPlexProvider.type === ProviderType.auth0) {
                      modifiedPlexProvider.auth0 = blankAuth0Provider();
                    } else if (modifiedPlexProvider.type === ProviderType.uc) {
                      modifiedPlexProvider.uc = blankUCProvider();
                    } else if (
                      modifiedPlexProvider.type === ProviderType.cognito
                    ) {
                      modifiedPlexProvider.cognito = blankCognitoProvider();
                    }
                    dispatch(modifyPlexProvider(modifiedPlexProvider));
                  }
                }}
              >
                {providerTypes.map((type) => (
                  <option key={type.key} value={type.payload}>
                    {type.label}
                  </option>
                ))}
              </Select>
            )}
          </Label>
          <Label className={Styles.propertiesElement}>
            Name
            <TextInput
              id="name"
              name="name"
              readOnly={readOnly}
              value={modifiedPlexProvider.name}
              onChange={(e: React.ChangeEvent) => {
                const val = (e.target as HTMLInputElement).value;
                modifiedPlexProvider!.name = val;
                dispatch(modifyPlexProvider(modifiedPlexProvider));
              }}
            />
          </Label>
        </fieldset>
        <fieldset>
          {modifiedPlexProvider.type === ProviderType.auth0 &&
            modifiedPlexProvider.auth0 && (
              <>
                <Heading size="3" headingLevel="3">
                  Auth0 Provider Settings
                </Heading>
                <Label className={GlobalStyles['mt-3']}>
                  Auth0 Domain
                  <TextInput
                    id="domain"
                    name="domain"
                    readOnly={readOnly}
                    value={modifiedPlexProvider.auth0.domain}
                    onChange={(e: React.ChangeEvent) => {
                      const val = (e.target as HTMLInputElement).value;
                      if (modifiedPlexProvider.auth0) {
                        modifiedPlexProvider.auth0.domain = val;
                        dispatch(modifyPlexProvider(modifiedPlexProvider));
                      }
                    }}
                  />
                </Label>
                {/* TODO: better UI if we want to support this in the future
                <Checkbox
                  checked={modifiedPlexProvider.auth0.redirect}
                  onChange={(e: React.ChangeEvent) => {
                    const checked = (e.target as HTMLInputElement).checked;
                    if (modifiedPlexProvider.auth0) {
                      modifiedPlexProvider.auth0.redirect = checked;
                      dispatch({
                        type: actions.MODIFY_PLEX_PROVIDER,
                        data: modifiedPlexProvider,
                      });
                    }
                  }}
                >
                  Redirect users to Auth0 Login UI (instead of Plex UI + Auth0
                  API)
                </Checkbox> */}
                <Text element="h3">Management Client</Text>
                <Label className={GlobalStyles['mt-3']}>
                  Client ID
                  <TextInput
                    id="mgmt_client_id"
                    name="mgmt_client_id"
                    readOnly={readOnly}
                    value={modifiedPlexProvider.auth0.management.client_id}
                    onChange={(e: React.ChangeEvent) => {
                      const val = (e.target as HTMLInputElement).value;
                      if (modifiedPlexProvider.auth0) {
                        modifiedPlexProvider.auth0.management.client_id = val;
                        dispatch(modifyPlexProvider(modifiedPlexProvider));
                      }
                    }}
                  />
                </Label>
                <Label className={GlobalStyles['mt-3']}>
                  Client Secret
                  <HiddenTextInput
                    id="mgmt_client_secret"
                    name="mgmt_client_secret"
                    readOnly={readOnly}
                    value={modifiedPlexProvider.auth0.management.client_secret}
                    onChange={(e: React.ChangeEvent) => {
                      const val = (e.target as HTMLInputElement).value;
                      if (modifiedPlexProvider.auth0) {
                        modifiedPlexProvider.auth0.management.client_secret =
                          val;
                        dispatch(modifyPlexProvider(modifiedPlexProvider));
                      }
                    }}
                  />
                </Label>
                <Label className={GlobalStyles['mt-3']}>
                  Audience
                  <TextInput
                    id="mgmt_audience"
                    name="mgmt_audience"
                    readOnly={readOnly}
                    value={modifiedPlexProvider.auth0.management.audience}
                    onChange={(e: React.ChangeEvent) => {
                      const val = (e.target as HTMLInputElement).value;
                      if (modifiedPlexProvider.auth0) {
                        modifiedPlexProvider.auth0.management.audience = val;
                        dispatch(modifyPlexProvider(modifiedPlexProvider));
                      }
                    }}
                  />
                </Label>
              </>
            )}
          {modifiedPlexProvider.type === ProviderType.uc &&
            modifiedPlexProvider.uc && (
              <>
                <Heading
                  className={Styles.propertiesElement}
                  size="3"
                  headingLevel="3"
                >
                  UserClouds Provider Settings
                </Heading>
                <Label className={GlobalStyles['mt-3']}>
                  IDP URL
                  <TextInput
                    id="idp_url"
                    name="idp_url"
                    readOnly={readOnly}
                    value={modifiedPlexProvider.uc.idp_url}
                    onChange={(e: React.ChangeEvent) => {
                      const val = (e.target as HTMLInputElement).value;
                      if (modifiedPlexProvider.uc) {
                        modifiedPlexProvider.uc.idp_url = val;
                      }
                      dispatch(modifyPlexProvider(modifiedPlexProvider));
                    }}
                  />
                </Label>
              </>
            )}
          {modifiedPlexProvider.type === ProviderType.cognito &&
            modifiedPlexProvider.cognito && (
              <>
                <Heading
                  className={Styles.propertiesElement}
                  size="3"
                  headingLevel="3"
                >
                  Cognito Provider Settings
                </Heading>
                <Label className={GlobalStyles['mt-3']}>
                  Access Key
                  <TextInput
                    id="mgmt_client_id"
                    name="mgmt_client_id"
                    readOnly={readOnly}
                    value={modifiedPlexProvider.cognito?.aws_config.access_key}
                    onChange={(e: React.ChangeEvent) => {
                      const val = (e.target as HTMLInputElement).value;
                      if (modifiedPlexProvider.cognito) {
                        modifiedPlexProvider.cognito.aws_config.access_key =
                          val;
                        dispatch(modifyPlexProvider(modifiedPlexProvider));
                      }
                    }}
                  />
                </Label>
                <Label className={GlobalStyles['mt-3']}>
                  Secret Key
                  <HiddenTextInput
                    id="mgmt_client_secret"
                    name="mgmt_client_secret"
                    readOnly={readOnly}
                    value={modifiedPlexProvider.cognito?.aws_config.secret_key}
                    onChange={(e: React.ChangeEvent) => {
                      const val = (e.target as HTMLInputElement).value;
                      if (modifiedPlexProvider.cognito) {
                        modifiedPlexProvider.cognito.aws_config.secret_key =
                          val;
                        dispatch(modifyPlexProvider(modifiedPlexProvider));
                      }
                    }}
                  />
                </Label>
                <Label className={GlobalStyles['mt-3']}>
                  AWS Region
                  <TextInput
                    id="idp_url"
                    name="idp_url"
                    readOnly={readOnly}
                    value={modifiedPlexProvider.cognito.aws_config.region}
                    onChange={(e: React.ChangeEvent) => {
                      const val = (e.target as HTMLInputElement).value;
                      if (modifiedPlexProvider.cognito) {
                        modifiedPlexProvider.cognito.aws_config.region = val;
                      }
                      dispatch(modifyPlexProvider(modifiedPlexProvider));
                    }}
                  />
                </Label>
                <Label className={GlobalStyles['mt-3']}>
                  User Pool ID
                  <TextInput
                    id="idp_url"
                    name="idp_url"
                    readOnly={readOnly}
                    value={modifiedPlexProvider.cognito.user_pool_id}
                    onChange={(e: React.ChangeEvent) => {
                      const val = (e.target as HTMLInputElement).value;
                      if (modifiedPlexProvider.cognito) {
                        modifiedPlexProvider.cognito.user_pool_id = val;
                      }
                      dispatch(modifyPlexProvider(modifiedPlexProvider));
                    }}
                  />
                </Label>
              </>
            )}
        </fieldset>
        {saveError && (
          <InlineNotification theme="alert">{saveError}</InlineNotification>
        )}
        {saveSuccess === UpdatePlexConfigReason.ModifyProvider && (
          <InlineNotification theme="success">{saveSuccess}</InlineNotification>
        )}
        <CardFooter>
          {selectedTenant?.is_admin && (
            <ButtonGroup>
              <Button
                onClick={() => {
                  dispatch(
                    saveProvider(
                      selectedTenant.id,
                      plexConfig,
                      modifiedPlexProvider,
                      UpdatePlexConfigReason.ModifyProvider
                    )
                  );
                }}
                disabled={!isDirty}
                isLoading={isSaving}
              >
                Save Provider
              </Button>
              <Button
                onClick={() => {
                  if (
                    window.confirm(
                      `Are you sure you want to delete this Provider?`
                    )
                  ) {
                    dispatch(
                      deleteProvider(
                        selectedCompanyID as string,
                        selectedTenant?.id || '',
                        plexConfig,
                        modifiedPlexProvider
                      )
                    );
                  }
                }}
                theme="dangerous"
              >
                Delete Provider
              </Button>
            </ButtonGroup>
          )}
        </CardFooter>
      </Card>

      <ConnectedAuth0Apps />
      <ConnectedCognitoApps />
      <ConnectedUCApps />
      <ConnectedImportApps />
    </>
  );
};

const ConnectedPlexProvider = connect((state: RootState) => {
  return {
    selectedCompanyID: state.selectedCompanyID,
    selectedTenant: state.selectedTenant,
    plexConfig: state.tenantPlexConfig,
    selectedPlexProvider: state.selectedPlexProvider,
    modifiedPlexProvider: state.modifiedPlexProvider,
    isDirty: state.plexConfigIsDirty,
    fetchError: state.fetchPlexConfigError,
    isSaving: state.savingPlexConfig,
    saveSuccess: state.savePlexConfigSuccess,
    saveError: state.savePlexConfigError,
    location: state.location,
    routeParams: state.routeParams,
  };
})(PlexProvider);

export default ConnectedPlexProvider;
