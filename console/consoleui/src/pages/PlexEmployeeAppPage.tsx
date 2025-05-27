import React, { useEffect } from 'react';
import { connect } from 'react-redux';
import {
  Accordion,
  AccordionItem,
  Button,
  ButtonGroup,
  Card,
  GlobalStyles,
  InputReadOnly,
  InlineNotification,
  Label,
  Text,
  TextInput,
  TextShortener,
} from '@userclouds/ui-component-lib';

import { redirect } from '../routing';
import { fetchPlexConfig, savePlexConfig } from '../thunks/authn';
import { modifyPlexEmployeeApp } from '../actions/authn';
import { AppDispatch, RootState } from '../store';
import { SelectedTenant } from '../models/Tenant';
import TenantPlexConfig, {
  UpdatePlexConfigReason,
} from '../models/TenantPlexConfig';
import LoginApp from '../models/LoginApp';
import NavigationBlocker from '../NavigationBlocker';
import StringList from '../controls/StringList';
import PlexAppPageStyles from './PlexAppPage.module.css';

const PlexEmployeeApp = ({
  selectedCompanyID,
  selectedTenant,
  plexConfig,
  modifiedPlexConfig,
  modifiedPlexEmployeeApp,
  isDirty,
  fetchError,
  isSaving,
  saveSuccess,
  saveError,
  location,
  dispatch,
}: {
  selectedCompanyID: string | undefined;
  selectedTenant: SelectedTenant | undefined;
  plexConfig: TenantPlexConfig | undefined;
  modifiedPlexConfig: TenantPlexConfig | undefined;
  modifiedPlexEmployeeApp: LoginApp | undefined;
  isDirty: boolean;
  fetchError: string;
  isSaving: boolean;
  saveSuccess: UpdatePlexConfigReason | undefined;
  saveError: string;
  location: URL;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenant) {
      if (plexConfig) {
        if (!plexConfig.tenant_config.plex_map.employee_app.id) {
          // Employee App not configured in tenant, go back to loginapps page or  AuthNPage
          redirect(
            `/loginapps?company_id=${selectedCompanyID}&tenant_id=${selectedTenant.id}`,
            true
          );
        } else if (!modifiedPlexEmployeeApp) {
          dispatch(
            modifyPlexEmployeeApp({
              ...plexConfig.tenant_config.plex_map.employee_app,
            })
          );
        }
      } else {
        dispatch(fetchPlexConfig(selectedTenant.id));
      }
    }
  }, [
    plexConfig,
    modifiedPlexEmployeeApp,
    selectedTenant,
    selectedCompanyID,
    location,
    dispatch,
  ]);

  if (fetchError || !plexConfig) {
    return <Text>{fetchError || 'Loading...'}</Text>;
  }

  if (!modifiedPlexEmployeeApp) {
    return <Text>Plex Employee app not configured</Text>;
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
                <Label
                  className={PlexAppPageStyles.propertiesElement}
                  htmlFor="plex_employee_app_id"
                >
                  ID
                  <TextShortener
                    text={modifiedPlexEmployeeApp.id}
                    length={6}
                    id="plex_employee_app_id"
                  />
                </Label>
                <Label className={PlexAppPageStyles.propertiesElement}>
                  Name
                  <InputReadOnly>{modifiedPlexEmployeeApp.name}</InputReadOnly>
                </Label>
              </div>
              <div className={PlexAppPageStyles.propertiesRow}>
                <Label className={PlexAppPageStyles.propertiesElement}>
                  Client ID
                  <TextInput
                    id="client_id"
                    name="client_id"
                    value={modifiedPlexEmployeeApp.client_id}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      dispatch(
                        modifyPlexEmployeeApp({
                          ...modifiedPlexEmployeeApp,
                          client_id: e.target.value,
                        })
                      );
                    }}
                  />
                </Label>

                <Label className={PlexAppPageStyles.propertiesElement}>
                  Client Secret
                  <TextInput
                    id="client_secret"
                    name="client_secret"
                    value={modifiedPlexEmployeeApp.client_secret}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      dispatch(
                        modifyPlexEmployeeApp({
                          ...modifiedPlexEmployeeApp,
                          client_secret: e.target.value,
                        })
                      );
                    }}
                  />
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
                strings={modifiedPlexEmployeeApp.allowed_redirect_uris}
                onValueChange={(index: number, value: string) => {
                  const newArr = Array.from(
                    modifiedPlexEmployeeApp.allowed_redirect_uris
                  );
                  newArr[index] = value;
                  dispatch(
                    modifyPlexEmployeeApp({
                      ...modifiedPlexEmployeeApp,
                      allowed_redirect_uris: newArr,
                    })
                  );
                }}
                onDeleteRow={(val: string) => {
                  dispatch(
                    modifyPlexEmployeeApp({
                      ...modifiedPlexEmployeeApp,
                      allowed_redirect_uris: Array.from(
                        modifiedPlexEmployeeApp.allowed_redirect_uris
                      ).filter((redirectURI) => redirectURI !== val),
                    })
                  );
                }}
              />
              {selectedTenant?.is_admin && (
                <Button
                  className={PlexAppPageStyles.singleButton}
                  theme="secondary"
                  onClick={() => {
                    dispatch(
                      modifyPlexEmployeeApp({
                        ...modifiedPlexEmployeeApp,
                        allowed_redirect_uris:
                          modifiedPlexEmployeeApp.allowed_redirect_uris
                            ? Array.from(
                                modifiedPlexEmployeeApp.allowed_redirect_uris
                              ).concat([''])
                            : [''],
                      })
                    );
                  }}
                >
                  Add Redirect URL
                </Button>
              )}
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
                id="allowedLogoutURIs"
                inputType="url"
                strings={modifiedPlexEmployeeApp.allowed_logout_uris}
                onValueChange={(index: number, value: string) => {
                  const newArr = Array.from(
                    modifiedPlexEmployeeApp.allowed_logout_uris
                  );
                  newArr[index] = value;
                  dispatch(
                    modifyPlexEmployeeApp({
                      ...modifiedPlexEmployeeApp,
                      allowed_logout_uris: newArr,
                    })
                  );
                }}
                onDeleteRow={(val: string) => {
                  dispatch(
                    modifyPlexEmployeeApp({
                      ...modifiedPlexEmployeeApp,
                      allowed_logout_uris: Array.from(
                        modifiedPlexEmployeeApp.allowed_logout_uris
                      ).filter((logoutURI) => logoutURI !== val),
                    })
                  );
                }}
              />
              {selectedTenant?.is_admin && (
                <Button
                  className={PlexAppPageStyles.singleButton}
                  theme="secondary"
                  onClick={() => {
                    dispatch(
                      modifyPlexEmployeeApp({
                        ...modifiedPlexEmployeeApp,
                        allowed_logout_uris:
                          modifiedPlexEmployeeApp.allowed_logout_uris
                            ? Array.from(
                                modifiedPlexEmployeeApp.allowed_logout_uris
                              ).concat([''])
                            : [''],
                      })
                    );
                  }}
                >
                  Add Logout URL
                </Button>
              )}
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
          {selectedTenant?.is_admin && (
            <ButtonGroup className={PlexAppPageStyles.buttonGroup}>
              <Button
                theme="primary"
                isLoading={isSaving}
                onClick={() => {
                  if (modifiedPlexConfig) {
                    dispatch(
                      savePlexConfig(
                        selectedTenant.id,
                        modifiedPlexConfig,
                        UpdatePlexConfigReason.ModifyEmployeeApp
                      )
                    );
                  }
                }}
                disabled={!isDirty || isSaving}
              >
                Save
              </Button>
            </ButtonGroup>
          )}
        </Accordion>
      </Card>
    </>
  );
};
const ConnectedPlexEmployeeApp = connect((state: RootState) => {
  return {
    selectedCompanyID: state.selectedCompanyID,
    selectedTenant: state.selectedTenant,
    plexConfig: state.tenantPlexConfig,
    modifiedPlexConfig: state.modifiedPlexConfig,
    modifiedPlexEmployeeApp: state.modifiedPlexEmployeeApp,
    isDirty: state.plexConfigIsDirty,
    isFetching: state.fetchingPlexConfig,
    fetchError: state.fetchPlexConfigError,
    isSaving: state.savingPlexConfig,
    saveSuccess: state.savePlexConfigSuccess,
    saveError: state.savePlexConfigError,
    location: state.location,
  };
})(PlexEmployeeApp);

export default ConnectedPlexEmployeeApp;
