import React, { useEffect } from 'react';
import { connect } from 'react-redux';
import {
  Accordion,
  AccordionItem,
  Button,
  ButtonGroup,
  Card,
  Checkbox,
  GlobalStyles,
  Heading,
  HiddenTextInput,
  InputReadOnly,
  InlineNotification,
  Label,
  Radio,
  Select,
  Text,
  TextInput,
  TextShortener,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import { redirect } from '../routing';
import {
  updatePlexConfigRequest,
  updatePlexConfigSuccess,
  updatePlexConfigError,
  modifyPlexApp,
  clonePlexAppSettings,
  selectPlexApp,
} from '../actions/authn';
import { AppDispatch, RootState } from '../store';
import Provider, { ProviderApp, ProviderType } from '../models/Provider';
import LoginApp from '../models/LoginApp';
import { SAMLEntityDescriptor } from '../models/SAMLEntityDescriptor';
import TenantPlexConfig, {
  UpdatePlexConfigReason,
} from '../models/TenantPlexConfig';
import NavigationBlocker from '../NavigationBlocker';
import {
  deleteLoginAppFromTenant,
  enableSAMLIDPForLoginApp,
} from '../API/authn';
import {
  fetchPlexConfig,
  savePlexConfig,
  fetchPageParams,
  fetchEmailMessageElements,
  fetchSMSMessageElements,
} from '../thunks/authn';
import { postSuccessToast } from '../thunks/notifications';
import EmailTemplateEditor from '../PlexAppEmailSettings';
import SMSTemplateEditor from '../PlexAppSMSSettings';
import LoginSettingsEditor from '../PlexAppLoginSettings';
import PlexAppPageStyles from './PlexAppPage.module.css';
import { SAMLIDP } from '../models/SAMLIDP';
import { SelectedTenant } from '../models/Tenant';
import { NilUuid } from '../models/Uuids';
import Link from '../controls/Link';
import StringList from '../controls/StringList';
import PageCommon from './PageCommon.module.css';

// Merges the local React state copy of the Plex App back into the global
// plex config for the tenant (shared across views) and pushes back to the server.
const saveTenant =
  () => async (dispatch: AppDispatch, getState: () => RootState) => {
    const { selectedTenantID, modifiedPlexApp, modifiedPlexConfig } =
      getState();

    if (!selectedTenantID || !modifiedPlexConfig || !modifiedPlexApp) {
      return;
    }
    // TODO: this should happen in the reducer
    const modifiedConfigData = {
      ...modifiedPlexConfig,
      tenant_config: {
        ...modifiedPlexConfig.tenant_config,
        plex_map: {
          ...modifiedPlexConfig.tenant_config.plex_map,
          apps: modifiedPlexConfig.tenant_config.plex_map.apps.map(
            (app: any) => {
              if (app.id === modifiedPlexApp.id) {
                app = modifiedPlexApp;
              }
              return app;
            }
          ),
        },
      },
    };

    dispatch(
      savePlexConfig(
        selectedTenantID,
        modifiedConfigData,
        UpdatePlexConfigReason.ModifyApp
      )
    );
  };

const deleteApp =
  () => async (dispatch: AppDispatch, getState: () => RootState) => {
    const {
      selectedCompanyID,
      selectedTenantID,
      selectedPlexApp: plexApp,
    } = getState();
    if (!plexApp || !selectedTenantID) {
      return;
    }
    dispatch(updatePlexConfigRequest);
    deleteLoginAppFromTenant(selectedTenantID, plexApp.id).then(
      (tenantPlex: TenantPlexConfig) => {
        dispatch(updatePlexConfigSuccess(tenantPlex));
        dispatch(postSuccessToast(`Successfully deleted app ${plexApp.name}`));

        // Navigate to login apps or authn page afterward
        redirect(
          `/loginapps?company_id=${selectedCompanyID}&tenant_id=${selectedTenantID}`,
          false
        );
      },
      (error: APIError) => {
        dispatch(updatePlexConfigError(error));
      }
    );
  };

const enableSAML =
  () => async (dispatch: AppDispatch, getState: () => RootState) => {
    const {
      selectedTenantID,
      selectedPlexApp: plexApp,
      modifiedPlexApp,
    } = getState();
    if (!plexApp || !selectedTenantID || !modifiedPlexApp) {
      return;
    }

    enableSAMLIDPForLoginApp(selectedTenantID, plexApp.id).then(
      (samlIDP: SAMLIDP) => {
        modifiedPlexApp.saml_idp = samlIDP;
        dispatch(modifyPlexApp(modifiedPlexApp));
      }
    );
  };

const toggleProviderSelection =
  (checked: boolean, appID: string, apps: ProviderApp[]) =>
  (dispatch: AppDispatch, getState: () => RootState) => {
    const { modifiedPlexApp } = getState();
    if (!modifiedPlexApp) {
      return;
    }

    // TODO: this routine should really be done in the reducer
    if (checked) {
      // Add this app to the provider app IDs list, uncheck any other
      // apps from the same provider
      modifiedPlexApp.provider_app_ids =
        modifiedPlexApp.provider_app_ids.filter(
          (id) => !apps.find((innerApp: ProviderApp) => innerApp.id === id)
        );
      modifiedPlexApp.provider_app_ids.push(appID);
    } else {
      // Remove this provider app ID from the list
      modifiedPlexApp.provider_app_ids =
        modifiedPlexApp.provider_app_ids.filter((id) => id !== appID);
    }
    dispatch(modifyPlexApp(modifiedPlexApp));
  };

const ProviderApps = ({
  provider,
  modifiedPlexApp,
  dispatch,
}: {
  provider: Provider;
  modifiedPlexApp: LoginApp | undefined;
  dispatch: AppDispatch;
}) => {
  let apps: ProviderApp[] = [];
  if (provider.type === ProviderType.auth0) {
    apps = provider.auth0?.apps || [];
  } else if (provider.type === ProviderType.uc) {
    apps = provider.uc?.apps || [];
  } else if (provider.type === ProviderType.cognito) {
    apps = provider.cognito?.apps || [];
  }
  return apps.length ? (
    <>
      {apps.map((app: ProviderApp) => (
        <Checkbox
          key={app.id}
          checked={modifiedPlexApp?.provider_app_ids.includes(app.id)}
          onChange={(e: React.ChangeEvent) => {
            dispatch(
              toggleProviderSelection(
                (e.target as HTMLInputElement).checked,
                app.id,
                apps
              )
            );
          }}
        >
          {app.name}
        </Checkbox>
      ))}
    </>
  ) : (
    <Text>No apps for this provider.</Text>
  );
};
const ConnectedProviderApps = connect((state: RootState) => ({
  modifiedPlexApp: state.modifiedPlexApp,
}))(ProviderApps);

const Providers = ({ providers }: { providers: Provider[] }) => {
  return (
    <>
      {providers.map((provider) => (
        <div key={`section_${provider.id}`} className={GlobalStyles['mt-6']}>
          <strong>{provider.name}</strong>
          <ConnectedProviderApps provider={provider} />
        </div>
      ))}
    </>
  );
};

// TODO: this list should be auto-generated from the constant server-side?
type GrantType = {
  id: string;
  name: string;
};

const grantTypes = [
  { id: 'authorization_code', name: 'Authorization Code' },
  { id: 'client_credentials', name: 'Client Credentials' },
  { id: 'refresh_token', name: 'Refresh Token' },
  { id: 'password', name: 'Resource Owner Password' },
];

const toggleGrantTypeSelection =
  (checked: boolean, gtID: string) =>
  (dispatch: AppDispatch, getState: () => RootState) => {
    const { modifiedPlexApp } = getState();
    if (!modifiedPlexApp) {
      return;
    }

    // TODO: this routine should really be done in the reducer
    if (checked) {
      modifiedPlexApp.grant_types.push(gtID);
    } else {
      // Remove this grant type from the list
      modifiedPlexApp.grant_types = modifiedPlexApp.grant_types.filter(
        (id) => id !== gtID
      );
    }
    dispatch(modifyPlexApp(modifiedPlexApp));
  };

const GrantTypeRow = ({
  grantType,
  modifiedPlexApp,
  dispatch,
}: {
  grantType: GrantType;
  modifiedPlexApp: LoginApp | undefined;
  dispatch: AppDispatch;
}) => {
  return (
    <Checkbox
      key={grantType.id}
      checked={modifiedPlexApp?.grant_types?.includes(grantType.id)}
      onChange={(e: React.ChangeEvent) => {
        dispatch(
          toggleGrantTypeSelection(
            (e.target as HTMLInputElement).checked,
            grantType.id
          )
        );
      }}
    >
      {grantType.name}
    </Checkbox>
  );
};
const ConnectedGrantType = connect((state: RootState) => ({
  modifiedPlexApp: state.modifiedPlexApp,
}))(GrantTypeRow);

const GrantTypes = () => {
  return (
    <>
      {grantTypes.map((grantType) => (
        <ConnectedGrantType grantType={grantType} key={grantType.id} />
      ))}
    </>
  );
};

const TrustedSPs = ({
  modifiedPlexApp,
  dispatch,
}: {
  modifiedPlexApp: LoginApp | undefined;
  dispatch: AppDispatch;
}) => {
  if (!modifiedPlexApp?.saml_idp) {
    return <></>;
  }

  return (
    <>
      <Label htmlFor="trustedServiceProviders">Trusted Service Providers</Label>
      {modifiedPlexApp.saml_idp.trusted_service_providers &&
        modifiedPlexApp.saml_idp.trusted_service_providers.length > 0 && (
          <StringList
            id="trustedServiceProviders"
            strings={
              modifiedPlexApp.saml_idp.trusted_service_providers?.map(
                (sp) => sp.entity_id
              ) || []
            }
            onValueChange={(index: number, value: string) => {
              if (!modifiedPlexApp?.saml_idp?.trusted_service_providers) {
                return;
              }

              modifiedPlexApp.saml_idp.trusted_service_providers[
                index
              ].entity_id = value;
              dispatch(modifyPlexApp(modifiedPlexApp));
            }}
            onDeleteRow={(val: string) => {
              if (!modifiedPlexApp?.saml_idp?.trusted_service_providers) {
                return;
              }
              modifiedPlexApp.saml_idp.trusted_service_providers =
                modifiedPlexApp.saml_idp?.trusted_service_providers.filter(
                  (sp) => sp.entity_id !== val
                );
              dispatch(modifyPlexApp(modifiedPlexApp));
            }}
          />
        )}
      <Button
        className={PlexAppPageStyles.singleButton}
        theme="secondary"
        onClick={() => {
          if (!modifiedPlexApp?.saml_idp) {
            return;
          }
          if (!modifiedPlexApp.saml_idp.trusted_service_providers) {
            modifiedPlexApp.saml_idp.trusted_service_providers = [];
          }
          modifiedPlexApp.saml_idp?.trusted_service_providers.push(
            {} as SAMLEntityDescriptor
          );
          dispatch(modifyPlexApp(modifiedPlexApp));
        }}
      >
        Add Trusted Service Provider
      </Button>
    </>
  );
};

const ConnectedTrustedSPs = connect((state: RootState) => ({
  modifiedPlexApp: state.modifiedPlexApp,
}))(TrustedSPs);

const SAMLIDPBlock = ({
  modifiedPlexApp,
  dispatch,
}: {
  modifiedPlexApp: LoginApp | undefined;
  dispatch: AppDispatch;
}) => {
  return (
    <>
      {modifiedPlexApp?.saml_idp === undefined ? (
        <Button
          className={PlexAppPageStyles.singleButton}
          theme="secondary"
          onClick={() => {
            dispatch(enableSAML());
          }}
        >
          Enable
        </Button>
      ) : (
        <>
          <div className={PlexAppPageStyles.propertiesRow}>
            <Label>
              Entity ID
              <InputReadOnly>
                {modifiedPlexApp.saml_idp.metadata_url}
              </InputReadOnly>
            </Label>
          </div>
          <div className={PlexAppPageStyles.propertiesRow}>
            <Label>
              SSO URL
              <InputReadOnly>{modifiedPlexApp.saml_idp.sso_url}</InputReadOnly>
            </Label>
          </div>
          <div className={PlexAppPageStyles.propertiesRow}>
            <Label>
              Certificate
              <InputReadOnly monospace className={PlexAppPageStyles.publicKey}>
                {modifiedPlexApp.saml_idp.certificate}
              </InputReadOnly>
            </Label>
          </div>
          <ConnectedTrustedSPs />
        </>
      )}
    </>
  );
};

const ConnectedSAMLIDP = connect((state: RootState) => ({
  modifiedPlexApp: state.modifiedPlexApp,
}))(SAMLIDPBlock);

const AdvancedSettings = ({
  selectedTenant,
  plexConfig,
  selectedApp,
  isSaving,
  fetchingPageParams,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  plexConfig: TenantPlexConfig | undefined;
  selectedApp: LoginApp | undefined;
  isSaving: boolean;
  fetchingPageParams: boolean;
  dispatch: AppDispatch;
}) => {
  const cloneableApps = plexConfig?.tenant_config.plex_map.apps.filter(
    (app: LoginApp) => app.id !== selectedApp?.id
  );
  return (
    <Card title="Advanced" isClosed>
      <Heading headingLevel={2} elementName="h2">
        Clone login settings
      </Heading>
      <Text>
        Copy login screen, email configurations, and SMS configurations from
        another app into this one. This will not take effect until you re-save
        your login application. This is a one-time import: if you later change
        settings for the imported application, this application will not be
        affected.
      </Text>
      {plexConfig ? (
        <form
          onSubmit={(e: React.FormEvent) => {
            e.preventDefault();
            const data = new FormData(e.currentTarget as HTMLFormElement);
            const appID = data.get('clone_settings_app_select') as string;
            dispatch(clonePlexAppSettings(appID));

            setTimeout(() => {
              dispatch(
                postSuccessToast('Successfully cloned login app settings')
              );
              document.getElementById('pageContent')?.scrollTo({
                top: 0,
                left: 0,
                behavior: 'smooth',
              });
            }, 1);
          }}
        >
          <Label>
            Application
            <br />
            {cloneableApps?.length ? (
              <ButtonGroup>
                <Select
                  name="clone_settings_app_select"
                  onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
                    const appID: string = e.target.value;

                    dispatch(
                      fetchPageParams(selectedTenant?.id as string, appID)
                    );
                  }}
                >
                  {cloneableApps.map((app: LoginApp) => (
                    <option value={app.id} key={app.id}>
                      {app.name}
                    </option>
                  ))}
                </Select>
                <Button
                  type="submit"
                  theme="secondary"
                  disabled={fetchingPageParams}
                  isLoading={fetchingPageParams}
                  className={`${GlobalStyles['ml-2']} ${GlobalStyles['mb-1']}`}
                >
                  Clone settings
                </Button>
              </ButtonGroup>
            ) : (
              <InputReadOnly>
                {
                  'There are currently no apps from which to clone settings. You may add one '
                }
                <Link title="Login apps page" href="/loginapps">
                  here
                </Link>
              </InputReadOnly>
            )}
          </Label>
        </form>
      ) : (
        <Text>Loading ...</Text>
      )}
      {selectedTenant?.is_admin && (
        <>
          <Heading headingLevel={2} elementName="h2">
            Delete app
          </Heading>
          <Text>This action cannot be undone.</Text>
          <Button
            theme="dangerous"
            isLoading={isSaving}
            disabled={isSaving}
            onClick={() => {
              if (window.confirm(`Are you sure you want to delete this app?`)) {
                dispatch(deleteApp());
              }
            }}
          >
            Delete App
          </Button>
        </>
      )}
    </Card>
  );
};
const ConnectedAdvancedSettings = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  plexConfig: state.tenantPlexConfig,
  selectedApp: state.selectedPlexApp,
  isSaving: state.savingPlexConfig,
  fetchingPageParams: state.fetchingPageParameters,
}))(AdvancedSettings);

const PlexApp = ({
  selectedCompanyID,
  selectedTenantID,
  selectedTenant,
  plexConfig,
  plexApp,
  modifiedPlexApp,
  isDirty,
  fetchError,
  isSaving,
  saveSuccess,
  saveError,
  appToClone,
  location,
  routeParams,
  dispatch,
}: {
  selectedCompanyID: string | undefined;
  selectedTenantID: string;
  selectedTenant: SelectedTenant | undefined;
  plexConfig: TenantPlexConfig | undefined;
  plexApp: LoginApp | undefined;
  modifiedPlexApp: LoginApp | undefined;
  isDirty: boolean;
  fetchError: string;
  isSaving: boolean;
  saveSuccess: UpdatePlexConfigReason | undefined;
  saveError: string;
  appToClone: string;
  location: URL;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const { plexAppID } = routeParams;

  useEffect(() => {
    if (plexConfig) {
      const matchingApp = plexConfig.tenant_config.plex_map.apps.find(
        (app: LoginApp) => app.id === plexAppID
      );
      if (!matchingApp) {
        // App not found in tenant, go  back to AuthNPage or loginapps page
        redirect(
          `/loginapps?company_id=${selectedCompanyID}&tenant_id=${selectedTenantID}`,
          true
        );
      } else if (!plexApp || plexApp.id !== plexAppID) {
        dispatch(selectPlexApp(plexAppID as string));
      }
    } else {
      dispatch(fetchPlexConfig(selectedTenantID));
    }
  }, [
    plexConfig,
    plexAppID,
    plexApp,
    selectedTenantID,
    selectedCompanyID,
    location,
    dispatch,
  ]);
  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchEmailMessageElements(selectedTenantID));
      dispatch(fetchSMSMessageElements(selectedTenantID));
    }
  }, [selectedTenantID, dispatch]);

  if (fetchError || !plexConfig) {
    return <div>{fetchError || 'Loading...'}</div>;
  }

  if (!plexApp || !modifiedPlexApp) {
    return <div>{`Plex app ${plexAppID} not found`}</div>;
  }

  let importedFromProvider = <></>;
  if (!['', 'unknown', NilUuid].includes(plexApp.synced_from_provider)) {
    const p = plexConfig.tenant_config.plex_map.providers.find(
      (provider: Provider) => provider.id === plexApp.synced_from_provider
    );
    importedFromProvider = (
      // this div is required to keep this on one line
      <div className={PlexAppPageStyles.propertiesRow}>
        <Label>
          Imported from
          <InputReadOnly>{p?.name || 'unknown'}</InputReadOnly>
        </Label>
      </div>
    );
  }

  return (
    <>
      <NavigationBlocker showPrompt={isDirty} />

      <Card title="General settings">
        <Accordion className={PlexAppPageStyles.accordion}>
          <fieldset
            className={`${GlobalStyles['min-w-[42rem]']} ${GlobalStyles['max-w-[full]']}`}
          >
            <AccordionItem title="Application Settings">
              <Text element="h4">Configure OAuth2/OIDC settings.</Text>
              <div className={PlexAppPageStyles.propertiesRow}>
                <Label htmlFor="id">
                  ID
                  <TextShortener text={modifiedPlexApp.id} length={6} id="id" />
                </Label>
              </div>
              {selectedTenant && (
                <div className={PlexAppPageStyles.propertiesRow}>
                  <Label>
                    URL
                    <InputReadOnly>
                      <a
                        href={selectedTenant.tenant_url}
                        target="new"
                        rel="external"
                        title="Access your login app"
                        className={PageCommon.link}
                      >
                        {selectedTenant.tenant_url}
                      </a>
                    </InputReadOnly>
                  </Label>
                </div>
              )}
              {importedFromProvider}
              <div className={PlexAppPageStyles.propertiesRow}>
                <Label>
                  Name
                  <TextInput
                    id="name"
                    name="name"
                    value={modifiedPlexApp.name}
                    onChange={(e: React.ChangeEvent) => {
                      modifiedPlexApp.name = (
                        e.target as HTMLInputElement
                      ).value;
                      dispatch(modifyPlexApp(modifiedPlexApp));
                    }}
                  />
                </Label>
              </div>
              <div className={PlexAppPageStyles.propertiesRow}>
                {/* TODO: there's probably a better style to use here? */}
                <Label>
                  Description
                  <TextInput
                    id="description"
                    name="description"
                    value={modifiedPlexApp.description}
                    onChange={(e: React.ChangeEvent) => {
                      modifiedPlexApp.description = (
                        e.target as HTMLInputElement
                      ).value;
                      dispatch(modifyPlexApp(modifiedPlexApp));
                    }}
                  />
                </Label>
              </div>
              <div className={PlexAppPageStyles.propertiesRow}>
                <Label>
                  Token URL
                  <InputReadOnly id="tenant_url">
                    {
                      selectedTenant?.tenant_url /* TODO don't hardcode this URL path? */
                    }
                    /oauth/token
                  </InputReadOnly>
                </Label>
              </div>
              <div className={PlexAppPageStyles.propertiesRow}>
                <Label>
                  Client ID
                  <TextInput
                    id="client_id"
                    name="client_id"
                    value={modifiedPlexApp.client_id}
                    onChange={(e: React.ChangeEvent) => {
                      modifiedPlexApp.client_id = (
                        e.target as HTMLInputElement
                      ).value;
                      dispatch(modifyPlexApp(modifiedPlexApp));
                    }}
                  />
                </Label>
              </div>
              <div className={PlexAppPageStyles.propertiesRow}>
                <Label>
                  Client Secret
                  <HiddenTextInput
                    id="client_secret"
                    name="client_secret"
                    value={modifiedPlexApp.client_secret}
                    onChange={(e: React.ChangeEvent) => {
                      modifiedPlexApp.client_secret = (
                        e.target as HTMLInputElement
                      ).value;
                      dispatch(modifyPlexApp(modifiedPlexApp));
                    }}
                  />
                </Label>
              </div>
              <div className={PlexAppPageStyles.propertiesRow}>
                <Label>
                  Restricted Access
                  <Radio
                    id="allowAllLogin"
                    value="allowAllLogin"
                    name="restrictedAccess"
                    onChange={() => {
                      modifiedPlexApp.restricted_access = false;
                      dispatch(modifyPlexApp(modifiedPlexApp));
                    }}
                    checked={!modifiedPlexApp?.restricted_access}
                  >
                    Allow all users in tenant to log in to this application
                  </Radio>
                  <Radio
                    id="restrictLogin"
                    value="restrictLogin"
                    name="restrictedAccess"
                    onChange={() => {
                      if (
                        window.confirm(
                          `Are you sure you want to restrict access? Making this change will lock out all existing users without the can_login permission on this application.`
                        )
                      ) {
                        modifiedPlexApp.restricted_access = true;
                        dispatch(modifyPlexApp(modifiedPlexApp));
                      }
                    }}
                    checked={modifiedPlexApp?.restricted_access}
                  >
                    Restrict login access to only users with can_login
                    permission on this application
                  </Radio>
                </Label>
              </div>
            </AccordionItem>
          </fieldset>

          <fieldset
            className={`${GlobalStyles['min-w-[42rem]']} ${GlobalStyles['max-w-4xl']}`}
          >
            <AccordionItem title="Allowed Redirect URLs">
              <Text element="h4">
                Specify exact URLs allowed as OIDC/OAuth2 callback URLs.
              </Text>
              <StringList
                id="allowedRedirectURIs"
                inputType="url"
                strings={
                  modifiedPlexApp.allowed_redirect_uris
                    ? modifiedPlexApp.allowed_redirect_uris
                    : []
                }
                onValueChange={(index: number, value: string) => {
                  modifiedPlexApp.allowed_redirect_uris[index] = value;
                  dispatch(modifyPlexApp(modifiedPlexApp));
                }}
                onDeleteRow={(val: string) => {
                  modifiedPlexApp.allowed_redirect_uris =
                    modifiedPlexApp.allowed_redirect_uris.filter(
                      (redirectURI) => redirectURI !== val
                    );
                  dispatch(modifyPlexApp(modifiedPlexApp));
                }}
              />
              <Button
                className={PlexAppPageStyles.singleButton}
                theme="secondary"
                onClick={() => {
                  modifiedPlexApp.allowed_redirect_uris.push('');
                  dispatch(modifyPlexApp(modifiedPlexApp));
                }}
              >
                Add Redirect URL
              </Button>
            </AccordionItem>
          </fieldset>

          <fieldset
            className={`${GlobalStyles['min-w-[42rem]']} ${GlobalStyles['max-w-4xl']}`}
          >
            <AccordionItem title="Allowed Logout URLs">
              <Text element="h4">
                Specify exact URLs allowed as Auth logout URLs.
              </Text>
              <StringList
                id="allowedLogoutURLs"
                inputType="url"
                strings={
                  modifiedPlexApp.allowed_logout_uris
                    ? modifiedPlexApp.allowed_logout_uris
                    : []
                }
                onValueChange={(index: number, value: string) => {
                  modifiedPlexApp.allowed_logout_uris[index] = value;
                  dispatch(modifyPlexApp(modifiedPlexApp));
                }}
                onDeleteRow={(val: string) => {
                  modifiedPlexApp.allowed_logout_uris =
                    modifiedPlexApp.allowed_logout_uris.filter(
                      (logoutURI) => logoutURI !== val
                    );
                  dispatch(modifyPlexApp(modifiedPlexApp));
                }}
              />
              <Button
                className={PlexAppPageStyles.singleButton}
                theme="secondary"
                onClick={() => {
                  modifiedPlexApp.allowed_logout_uris.push('');
                  dispatch(modifyPlexApp(modifiedPlexApp));
                }}
              >
                Add Logout URL
              </Button>
            </AccordionItem>
          </fieldset>

          <fieldset
            className={`${GlobalStyles['min-w-[42rem]']} ${GlobalStyles['max-w-4xl']}`}
          >
            <AccordionItem title="Underlying Identity Provider Apps">
              <Text element="h4">
                Choose which underlying Identity Providers' applications
                (OAuth2/OIDC clients) map to this Plex App.
              </Text>
              <Providers
                providers={plexConfig.tenant_config.plex_map.providers}
              />
            </AccordionItem>
          </fieldset>

          <fieldset
            className={`${GlobalStyles['min-w-[42rem]']} ${GlobalStyles['max-w-4xl']}`}
          >
            <AccordionItem title="OAuth Grant Types">
              <Text element="h4">
                Choose which OAuth Grant Types this Application supports
              </Text>
              <GrantTypes />
            </AccordionItem>
          </fieldset>

          <fieldset
            className={`${GlobalStyles['min-w-[42rem]']} ${GlobalStyles['max-w-4xl']}`}
          >
            <AccordionItem title="SAML IDP">
              <Text element="h4">
                Allow this Login App to function as a SAML IDP (in addition to
                an OAuth IDP).
              </Text>
              <ConnectedSAMLIDP />
            </AccordionItem>
          </fieldset>

          {saveError && (
            <InlineNotification theme="alert">{saveError}</InlineNotification>
          )}
          {saveSuccess && (
            <InlineNotification theme="success">
              {saveSuccess}
            </InlineNotification>
          )}
          <ButtonGroup className={PlexAppPageStyles.buttonGroup}>
            <Button
              theme="primary"
              isLoading={isSaving}
              onClick={() => {
                dispatch(saveTenant());
              }}
              disabled={!isDirty || isSaving}
            >
              Save
            </Button>
          </ButtonGroup>
        </Accordion>
      </Card>
      {plexAppID ? (
        <>
          <LoginSettingsEditor tenantID={selectedTenantID} appID={plexAppID} />
          <EmailTemplateEditor
            tenantID={selectedTenantID}
            plexAppID={appToClone || plexAppID}
          />
          <SMSTemplateEditor
            tenantID={selectedTenantID}
            plexAppID={appToClone || plexAppID}
          />
          <ConnectedAdvancedSettings />
        </>
      ) : (
        <p>Something went wrong</p>
      )}
    </>
  );
};
const ConnectedPlexApp = connect((state: RootState) => {
  return {
    selectedCompanyID: state.selectedCompanyID,
    selectedTenantID: state.selectedTenantID || '',
    selectedTenant: state.selectedTenant,
    plexConfig: state.tenantPlexConfig,
    plexApp: state.selectedPlexApp,
    modifiedPlexApp: state.modifiedPlexApp,
    isDirty: state.plexConfigIsDirty,
    fetchError: state.fetchPlexConfigError,
    isSaving: state.savingPlexConfig,
    saveSuccess: state.savePlexConfigSuccess,
    saveError: state.savePlexConfigError,
    appToClone: state.appToClone,
    location: state.location,
    routeParams: state.routeParams,
  };
})(PlexApp);

export default ConnectedPlexApp;
