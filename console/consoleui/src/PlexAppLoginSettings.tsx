import React, { useState, useEffect } from 'react';
import { connect } from 'react-redux';
import {
  Accordion,
  AccordionItem,
  Button,
  ButtonGroup,
  Card,
  Checkbox,
  FormNote,
  GlobalStyles,
  Heading,
  InlineNotification,
  Label,
  Radio,
  Text,
  TextArea,
  TextInput,
} from '@userclouds/ui-component-lib';
import {
  APIError,
  LoginContents,
  CreateUserContents,
} from '@userclouds/sharedui';

import { AppDispatch, RootState } from './store';
import { uploadImageForApp, saveAppPageParameters } from './API/authn';
import { fetchPageParams } from './thunks/authn';
import {
  modifyPageParameters,
  resetPageParameters,
  updatePageParametersRequest,
  updatePageParametersSuccess,
  updatePageParametersError,
} from './actions/authn';
import {
  PageParametersResponse,
  updatePageParameters,
  pageParametersForPreview,
  pageParametersForSave,
  toggleArrayParam,
  arrayParamAsSet,
} from './models/PageParameters';
import { oidcAuthSettingsFromPageParams } from './models/OIDCAuthSettings';
import Styles from './pages/PlexAppPage.module.css';

const uploadFile = (tenantID: string, appID: string, file: File) => {
  return uploadImageForApp(tenantID, appID, file).then((data) => {
    return data.image_url;
  });
};

const supportedImageFormats = ['gif', 'png', 'jpg'];
// Note: this logic will break if we have less than 3 image formats in supportedImageFormats
const supportedImageFormatsDisplay =
  supportedImageFormats
    .map((f) => '.' + f)
    .slice(0, -1)
    .join(', ') +
  ', and .' +
  supportedImageFormats.at(-1);
const saveParams =
  (params: PageParametersResponse) => (dispatch: AppDispatch) => {
    dispatch(updatePageParametersRequest());
    saveAppPageParameters(
      params.tenant_id,
      params.app_id,
      pageParametersForSave(params)
    ).then(
      (savedParams: PageParametersResponse) => {
        dispatch(updatePageParametersSuccess(savedParams));
      },
      (error: APIError) => {
        dispatch(updatePageParametersError(error));
      }
    );
  };

type HEX = `#${string}` | 'transparent';
type PreviewPage = 'login' | 'createUser';
const LoginSettingsEditor = ({
  tenantID,
  appID,
  tenantPageParameters,
  modifiedPageParameters,
  isFetching,
  fetchError,
  isSaving,
  saveSuccess,
  saveError,
  dispatch,
}: {
  tenantID: string;
  appID: string;
  tenantPageParameters: Record<string, PageParametersResponse>;
  modifiedPageParameters: PageParametersResponse | undefined;
  isFetching: boolean;
  fetchError: string;
  isSaving: boolean;
  saveSuccess: boolean;
  saveError: string;
  dispatch: AppDispatch;
}): JSX.Element => {
  const [imageUploadError, setImageUploadError] = useState<Error | undefined>();
  const [previewPage, setPreviewPage] = useState<PreviewPage>(
    window.location.hash === '#previewCreateUser' ? 'createUser' : 'login'
  );
  const pageParameters = tenantPageParameters[appID];
  const isDirty =
    JSON.stringify(pageParameters) !== JSON.stringify(modifiedPageParameters);
  const isBusy = isFetching || isSaving;
  useEffect(() => {
    if (tenantID && appID) {
      dispatch(fetchPageParams(tenantID, appID));
    }
  }, [tenantID, appID, dispatch]);

  return (
    <Card
      title="Login settings"
      className={Styles.loginSettingsCard}
      description="Customize this application’s login methods, security settings and user interface. To set up new connections, go back to the Authentication page."
    >
      <form
        id="loginSettingsForm"
        className={Styles.loginSettingsForm}
        aria-busy={isBusy ? 'true' : 'false'}
        onSubmit={(e: React.FormEvent) => {
          e.preventDefault();

          if (modifiedPageParameters) {
            dispatch(saveParams(modifiedPageParameters));
          }
        }}
        onReset={() => {
          dispatch(resetPageParameters(pageParameters));
        }}
      >
        <Accordion title="Login Settings">
          <AccordionItem title="Authentication methods">
            <fieldset className={Styles.accordionItem}>
              <Text element="h4">
                Configure permitted authentication methods. You can enable
                grayed-out methods in Tenant Settings
              </Text>
              <Checkbox
                name="authenticationMethods"
                disabled={
                  !modifiedPageParameters ||
                  !arrayParamAsSet(
                    modifiedPageParameters,
                    'every_page',
                    'enabledAuthenticationMethods'
                  ).has('password')
                }
                value="password"
                onChange={(e: React.ChangeEvent) => {
                  // TODO: we want to enforce that at least one of these is checked.
                  // It would be good to prevent unchecking the last checked box
                  // with e.preventDefault, but our Checkbox component doesn't allow
                  // access to the event object.
                  if (modifiedPageParameters) {
                    dispatch(
                      modifyPageParameters(
                        updatePageParameters(
                          modifiedPageParameters,
                          'every_page',
                          'authenticationMethods',
                          toggleArrayParam(
                            modifiedPageParameters.page_type_parameters
                              .every_page.authenticationMethods.current_value,
                            'password',
                            (e.currentTarget as HTMLInputElement).checked
                          )
                        )
                      )
                    );
                  }
                }}
                checked={
                  modifiedPageParameters
                    ? arrayParamAsSet(
                        modifiedPageParameters,
                        'every_page',
                        'authenticationMethods'
                      ).has('password')
                    : false
                }
              >
                Username and password
              </Checkbox>
              <Checkbox
                label=""
                name="authenticationMethods"
                disabled={
                  !modifiedPageParameters ||
                  !arrayParamAsSet(
                    modifiedPageParameters,
                    'every_page',
                    'enabledAuthenticationMethods'
                  ).has('passwordless')
                }
                value="passwordless"
                onChange={(e: React.ChangeEvent) => {
                  if (modifiedPageParameters) {
                    dispatch(
                      modifyPageParameters(
                        updatePageParameters(
                          modifiedPageParameters,
                          'every_page',
                          'authenticationMethods',
                          toggleArrayParam(
                            modifiedPageParameters.page_type_parameters
                              .every_page.authenticationMethods.current_value,
                            'passwordless',
                            (e.currentTarget as HTMLInputElement).checked
                          )
                        )
                      )
                    );
                  }
                }}
                checked={
                  modifiedPageParameters
                    ? arrayParamAsSet(
                        modifiedPageParameters,
                        'every_page',
                        'authenticationMethods'
                      ).has('passwordless')
                    : false
                }
              >
                Email - passwordless
              </Checkbox>
              {oidcAuthSettingsFromPageParams(modifiedPageParameters).map(
                (method) => (
                  <Checkbox
                    key={method.name}
                    name="authenticationMethods"
                    disabled={
                      !modifiedPageParameters ||
                      !arrayParamAsSet(
                        modifiedPageParameters,
                        'every_page',
                        'enabledAuthenticationMethods'
                      ).has(method.name)
                    }
                    value={method.name}
                    onChange={(e: React.ChangeEvent) => {
                      if (modifiedPageParameters) {
                        dispatch(
                          modifyPageParameters(
                            updatePageParameters(
                              modifiedPageParameters,
                              'every_page',
                              'authenticationMethods',
                              toggleArrayParam(
                                modifiedPageParameters.page_type_parameters
                                  .every_page.authenticationMethods
                                  .current_value,
                                method.name,
                                (e.currentTarget as HTMLInputElement).checked
                              )
                            )
                          )
                        );
                      }
                    }}
                    checked={
                      modifiedPageParameters
                        ? arrayParamAsSet(
                            modifiedPageParameters,
                            'every_page',
                            'authenticationMethods'
                          ).has(method.name)
                        : false
                    }
                  >
                    {method.description}
                  </Checkbox>
                )
              )}
            </fieldset>
          </AccordionItem>
          <AccordionItem title="Authentication settings">
            <fieldset className={Styles.accordionItem}>
              <Text element="h4">Multi-factor authentication</Text>
              <Checkbox
                name="requireMFA"
                onChange={(e: React.ChangeEvent) => {
                  if (modifiedPageParameters) {
                    dispatch(
                      modifyPageParameters(
                        updatePageParameters(
                          modifiedPageParameters,
                          'every_page',
                          'mfaRequired',
                          (
                            e.currentTarget as HTMLInputElement
                          ).checked.toString()
                        )
                      )
                    );
                  }
                }}
                checked={
                  modifiedPageParameters
                    ? modifiedPageParameters.page_type_parameters.every_page
                        .mfaRequired.current_value === 'true'
                    : false
                }
              >
                Require MFA for all non-social logins
              </Checkbox>

              <Text element="h4">MFA methods</Text>
              <Checkbox
                name="mfaMethods"
                disabled={
                  !modifiedPageParameters ||
                  !arrayParamAsSet(
                    modifiedPageParameters,
                    'every_page',
                    'enabledMFAMethods'
                  ).has('email')
                }
                value="email"
                onChange={(e: React.ChangeEvent) => {
                  if (modifiedPageParameters) {
                    dispatch(
                      modifyPageParameters(
                        updatePageParameters(
                          modifiedPageParameters,
                          'every_page',
                          'mfaMethods',
                          toggleArrayParam(
                            modifiedPageParameters.page_type_parameters
                              .every_page.mfaMethods.current_value,
                            'email',
                            (e.currentTarget as HTMLInputElement).checked
                          )
                        )
                      )
                    );
                  }
                }}
                checked={
                  modifiedPageParameters
                    ? arrayParamAsSet(
                        modifiedPageParameters,
                        'every_page',
                        'mfaMethods'
                      ).has('email')
                    : false
                }
              >
                Email
              </Checkbox>
              <Checkbox
                name="mfaMethods"
                disabled={
                  !modifiedPageParameters ||
                  !arrayParamAsSet(
                    modifiedPageParameters,
                    'every_page',
                    'enabledMFAMethods'
                  ).has('sms')
                }
                value="sms"
                onChange={(e: React.ChangeEvent) => {
                  if (modifiedPageParameters) {
                    dispatch(
                      modifyPageParameters(
                        updatePageParameters(
                          modifiedPageParameters,
                          'every_page',
                          'mfaMethods',
                          toggleArrayParam(
                            modifiedPageParameters.page_type_parameters
                              .every_page.mfaMethods.current_value,
                            'sms',
                            (e.currentTarget as HTMLInputElement).checked
                          )
                        )
                      )
                    );
                  }
                }}
                checked={
                  modifiedPageParameters
                    ? arrayParamAsSet(
                        modifiedPageParameters,
                        'every_page',
                        'mfaMethods'
                      ).has('sms')
                    : false
                }
              >
                SMS
              </Checkbox>
              <Checkbox
                name="mfaMethods"
                disabled={
                  !modifiedPageParameters ||
                  !arrayParamAsSet(
                    modifiedPageParameters,
                    'every_page',
                    'enabledMFAMethods'
                  ).has('authenticator')
                }
                value="authenticator"
                onChange={(e: React.ChangeEvent) => {
                  if (modifiedPageParameters) {
                    dispatch(
                      modifyPageParameters(
                        updatePageParameters(
                          modifiedPageParameters,
                          'every_page',
                          'mfaMethods',
                          toggleArrayParam(
                            modifiedPageParameters.page_type_parameters
                              .every_page.mfaMethods.current_value,
                            'authenticator',
                            (e.currentTarget as HTMLInputElement).checked
                          )
                        )
                      )
                    );
                  }
                }}
                checked={
                  modifiedPageParameters
                    ? arrayParamAsSet(
                        modifiedPageParameters,
                        'every_page',
                        'mfaMethods'
                      ).has('authenticator')
                    : false
                }
              >
                Authenticator app
              </Checkbox>
              <Checkbox
                name="mfaMethods"
                disabled={
                  !modifiedPageParameters ||
                  !arrayParamAsSet(
                    modifiedPageParameters,
                    'every_page',
                    'enabledMFAMethods'
                  ).has('recovery_code')
                }
                value="recovery_code"
                onChange={(e: React.ChangeEvent) => {
                  if (modifiedPageParameters) {
                    dispatch(
                      modifyPageParameters(
                        updatePageParameters(
                          modifiedPageParameters,
                          'every_page',
                          'mfaMethods',
                          toggleArrayParam(
                            modifiedPageParameters.page_type_parameters
                              .every_page.mfaMethods.current_value,
                            'recovery_code',
                            (e.currentTarget as HTMLInputElement).checked
                          )
                        )
                      )
                    );
                  }
                }}
                checked={
                  modifiedPageParameters
                    ? arrayParamAsSet(
                        modifiedPageParameters,
                        'every_page',
                        'mfaMethods'
                      ).has('recovery_code')
                    : false
                }
              >
                Recovery code
              </Checkbox>

              <Text element="h4" className={GlobalStyles['mt-3']}>
                Page order
              </Text>
              <Radio
                id="emailFirst"
                name="pageOrderSocialFirst"
                value="emailFirst"
                onChange={(e: React.ChangeEvent) => {
                  if (modifiedPageParameters) {
                    dispatch(
                      modifyPageParameters(
                        updatePageParameters(
                          modifiedPageParameters,
                          'every_page',
                          'pageOrderSocialFirst',
                          (e.currentTarget as HTMLInputElement).checked
                            ? 'false'
                            : 'true'
                        )
                      )
                    );
                  }
                }}
                checked={
                  modifiedPageParameters
                    ? modifiedPageParameters.page_type_parameters.every_page
                        .pageOrderSocialFirst.current_value === 'false'
                    : true
                }
              >
                Email first
              </Radio>
              <Radio
                id="socialFirst"
                name="pageOrderSocialFirst"
                value="socialFirst"
                onChange={(e: React.ChangeEvent) => {
                  if (modifiedPageParameters) {
                    dispatch(
                      modifyPageParameters(
                        updatePageParameters(
                          modifiedPageParameters,
                          'every_page',
                          'pageOrderSocialFirst',
                          (e.currentTarget as HTMLInputElement).checked
                            ? 'true'
                            : 'false'
                        )
                      )
                    );
                  }
                }}
                checked={
                  modifiedPageParameters
                    ? modifiedPageParameters.page_type_parameters.every_page
                        .pageOrderSocialFirst.current_value === 'true'
                    : false
                }
              >
                Social first
              </Radio>
            </fieldset>
          </AccordionItem>
          <AccordionItem title="Logo & colors">
            <fieldset className={Styles.accordionItem}>
              <button
                type="button"
                onClick={() => {
                  (
                    document.getElementById('logoImage') as HTMLImageElement
                  ).click();
                }}
                aria-label="Change Logo"
              >
                <img
                  src={
                    modifiedPageParameters
                      ? modifiedPageParameters.page_type_parameters.every_page
                          .logoImageFile.current_value
                      : ''
                  }
                  alt=""
                  className={Styles.logoImagePreview}
                  title={
                    modifiedPageParameters &&
                    modifiedPageParameters.page_type_parameters.every_page
                      .logoImageFile.current_value
                      ? 'Click to change image'
                      : 'Click to add an image'
                  }
                />
              </button>
              <input
                type="file"
                id="logoImage"
                name="logoImage"
                accept={supportedImageFormats
                  .map((fm) => 'image/' + fm)
                  .join(' ')}
                className={Styles.logoImageSelector}
                onChange={(e) => {
                  if (e.target && e.target.files) {
                    const file: File = e.target.files[0];
                    uploadFile(tenantID, appID, file).then(
                      (url) => {
                        setImageUploadError(undefined);
                        if (modifiedPageParameters) {
                          dispatch(
                            modifyPageParameters(
                              updatePageParameters(
                                modifiedPageParameters,
                                'every_page',
                                'logoImageFile',
                                url
                              )
                            )
                          );
                        }
                      },
                      (error: APIError) => {
                        setImageUploadError(error);
                      }
                    );
                  }
                }}
              />
              <input
                type="hidden"
                name="logoImageFile"
                value={
                  modifiedPageParameters
                    ? modifiedPageParameters.page_type_parameters.every_page
                        .logoImageFile.current_value
                    : ''
                }
              />
              {imageUploadError ? (
                <InlineNotification theme="alert">
                  {imageUploadError.message}
                </InlineNotification>
              ) : (
                ''
              )}
              <FormNote>
                Accepted image types: {supportedImageFormatsDisplay}. Image
                should be less than 80 pixels tall and 200 pixels wide. This
                image will appear up on your authentication screens.
              </FormNote>

              <Label className={GlobalStyles['mt-3']}>
                Login button fill
                <TextInput
                  id="actionButtonFill"
                  name="actionButtonFill"
                  value={
                    modifiedPageParameters
                      ? modifiedPageParameters.page_type_parameters.every_page
                          .actionButtonFillColor.current_value
                      : '#1090ff'
                  }
                  pattern="(#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})|transparent)"
                  onChange={(e: React.ChangeEvent) => {
                    const val = (e.target as HTMLInputElement).value;
                    if (modifiedPageParameters) {
                      dispatch(
                        modifyPageParameters(
                          updatePageParameters(
                            modifiedPageParameters,
                            'every_page',
                            'actionButtonFillColor',
                            val.toLowerCase() as HEX
                          )
                        )
                      );
                    }
                  }}
                  innerLeft={
                    <input
                      type="color"
                      name="actionButtonFillColor"
                      id="actionButtonFillColor"
                      className={Styles.colorInput}
                      value={
                        modifiedPageParameters
                          ? modifiedPageParameters.page_type_parameters
                              .every_page.actionButtonFillColor.current_value
                          : '#1090ff'
                      }
                      onChange={(e) => {
                        if (modifiedPageParameters) {
                          dispatch(
                            modifyPageParameters(
                              updatePageParameters(
                                modifiedPageParameters,
                                'every_page',
                                'actionButtonFillColor',
                                e.target.value as HEX
                              )
                            )
                          );
                        }
                      }}
                    />
                  }
                />
              </Label>

              <Label className={GlobalStyles['mt-3']}>
                Login button text
                <TextInput
                  id="actionButtonText"
                  name="actionButtonText"
                  value={
                    modifiedPageParameters
                      ? modifiedPageParameters.page_type_parameters.every_page
                          .actionButtonTextColor.current_value
                      : '#ffffff'
                  }
                  pattern="(#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})|transparent)"
                  onChange={(e: React.ChangeEvent) => {
                    const val = (e.target as HTMLInputElement).value;
                    if (modifiedPageParameters) {
                      dispatch(
                        modifyPageParameters(
                          updatePageParameters(
                            modifiedPageParameters,
                            'every_page',
                            'actionButtonTextColor',
                            val.toLowerCase() as HEX
                          )
                        )
                      );
                    }
                  }}
                  innerLeft={
                    <input
                      type="color"
                      name="actionButtonTextColor"
                      id="actionButtonTextColor"
                      className={Styles.colorInput}
                      value={
                        modifiedPageParameters
                          ? modifiedPageParameters.page_type_parameters
                              .every_page.actionButtonTextColor.current_value
                          : '#1090ff'
                      }
                      onChange={(e) => {
                        if (modifiedPageParameters) {
                          dispatch(
                            modifyPageParameters(
                              updatePageParameters(
                                modifiedPageParameters,
                                'every_page',
                                'actionButtonTextColor',
                                e.target.value as HEX
                              )
                            )
                          );
                        }
                      }}
                    />
                  }
                />
              </Label>

              <Label className={GlobalStyles['mt-3']}>
                Login button border
                <TextInput
                  id="actionButtonBorder"
                  name="actionButtonBorder"
                  value={
                    modifiedPageParameters
                      ? modifiedPageParameters.page_type_parameters.every_page
                          .actionButtonBorderColor.current_value
                      : '#1090ff'
                  }
                  pattern="(#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})|transparent)"
                  onChange={(e: React.ChangeEvent) => {
                    const val = (e.target as HTMLInputElement).value;
                    if (modifiedPageParameters) {
                      dispatch(
                        modifyPageParameters(
                          updatePageParameters(
                            modifiedPageParameters,
                            'every_page',
                            'actionButtonBorderColor',
                            val.toLowerCase() as HEX
                          )
                        )
                      );
                    }
                  }}
                  innerLeft={
                    <input
                      type="color"
                      name="actionButtonBorderColor"
                      id="actionButtonBorderColor"
                      className={Styles.colorInput}
                      value={
                        modifiedPageParameters
                          ? modifiedPageParameters.page_type_parameters
                              .every_page.actionButtonBorderColor.current_value
                          : '#1090ff'
                      }
                      onChange={(e) => {
                        if (modifiedPageParameters) {
                          dispatch(
                            modifyPageParameters(
                              updatePageParameters(
                                modifiedPageParameters,
                                'every_page',
                                'actionButtonBorderColor',
                                e.target.value as HEX
                              )
                            )
                          );
                        }
                      }}
                    />
                  }
                />
              </Label>

              <Label className={GlobalStyles['mt-3']}>
                Background
                <TextInput
                  id="pageBackground"
                  name="pageBackground"
                  label=""
                  value={
                    modifiedPageParameters
                      ? modifiedPageParameters.page_type_parameters.every_page
                          .pageBackgroundColor.current_value
                      : '#ffffff'
                  }
                  pattern="(#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})|transparent)"
                  onChange={(e: React.ChangeEvent) => {
                    const val = (e.target as HTMLInputElement).value;
                    if (modifiedPageParameters) {
                      dispatch(
                        modifyPageParameters(
                          updatePageParameters(
                            modifiedPageParameters,
                            'every_page',
                            'pageBackgroundColor',
                            val.toLowerCase() as HEX
                          )
                        )
                      );
                    }
                  }}
                  innerLeft={
                    <input
                      type="color"
                      name="pageBackgroundColor"
                      id="pageBackgroundColor"
                      className={Styles.colorInput}
                      value={
                        modifiedPageParameters
                          ? modifiedPageParameters.page_type_parameters
                              .every_page.pageBackgroundColor.current_value
                          : '#ffffff'
                      }
                      onChange={(e) => {
                        if (modifiedPageParameters) {
                          dispatch(
                            modifyPageParameters(
                              updatePageParameters(
                                modifiedPageParameters,
                                'every_page',
                                'pageBackgroundColor',
                                e.target.value as HEX
                              )
                            )
                          );
                        }
                      }}
                    />
                  }
                />
              </Label>

              <Label className={GlobalStyles['mt-3']}>
                Text Color
                <TextInput
                  id="pageText"
                  name="pageText"
                  label=""
                  value={
                    modifiedPageParameters
                      ? modifiedPageParameters.page_type_parameters.every_page
                          .pageTextColor.current_value
                      : '#ffffff'
                  }
                  pattern="(#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})|transparent)"
                  onChange={(e: React.ChangeEvent) => {
                    const val = (e.target as HTMLInputElement).value;
                    if (modifiedPageParameters) {
                      dispatch(
                        modifyPageParameters(
                          updatePageParameters(
                            modifiedPageParameters,
                            'every_page',
                            'pageTextColor',
                            val.toLowerCase() as HEX
                          )
                        )
                      );
                    }
                  }}
                  innerLeft={
                    <input
                      type="color"
                      name="pageTextColor"
                      id="pageTextColor"
                      className={Styles.colorInput}
                      value={
                        modifiedPageParameters
                          ? modifiedPageParameters.page_type_parameters
                              .every_page.pageTextColor.current_value
                          : '#ffffff'
                      }
                      onChange={(e) => {
                        if (modifiedPageParameters) {
                          dispatch(
                            modifyPageParameters(
                              updatePageParameters(
                                modifiedPageParameters,
                                'every_page',
                                'pageTextColor',
                                e.target.value as HEX
                              )
                            )
                          );
                        }
                      }}
                    />
                  }
                />
              </Label>
            </fieldset>
          </AccordionItem>
          <AccordionItem title="Login page settings">
            <fieldset className={Styles.accordionItem}>
              <Label className={GlobalStyles['mt-3']}>
                Header copy
                <TextInput
                  id="loginPageHeadingCopy"
                  name="loginPageHeadingCopy"
                  value={
                    modifiedPageParameters
                      ? modifiedPageParameters.page_type_parameters
                          .plex_login_page.headingText.current_value
                      : ''
                  }
                  onChange={(e: React.ChangeEvent) => {
                    if (modifiedPageParameters) {
                      dispatch(
                        modifyPageParameters(
                          updatePageParameters(
                            modifiedPageParameters,
                            'plex_login_page',
                            'headingText',
                            (e.target as HTMLInputElement).value
                          )
                        )
                      );
                    }
                  }}
                />
              </Label>
              <Label className={GlobalStyles['mt-3']}>
                Subheading copy
                <TextInput
                  id="loginPageSubheadingCopy"
                  name="loginPageSubheadingCopy"
                  value={
                    modifiedPageParameters
                      ? modifiedPageParameters.page_type_parameters
                          .plex_login_page.subheadingText.current_value
                      : ''
                  }
                  onChange={(e: React.ChangeEvent) => {
                    if (modifiedPageParameters) {
                      dispatch(
                        modifyPageParameters(
                          updatePageParameters(
                            modifiedPageParameters,
                            'plex_login_page',
                            'subheadingText',
                            (e.target as HTMLInputElement).value
                          )
                        )
                      );
                    }
                  }}
                />
              </Label>
              <Label className={GlobalStyles['mt-3']}>
                Footer contents
                <TextArea
                  id="loginPageFooterContents"
                  name="loginPageFooterContents"
                  rows={6}
                  value={
                    modifiedPageParameters
                      ? modifiedPageParameters.page_type_parameters
                          .plex_login_page.footerHTML.current_value
                      : ''
                  }
                  onChange={(e: React.ChangeEvent) => {
                    if (modifiedPageParameters) {
                      dispatch(
                        modifyPageParameters(
                          updatePageParameters(
                            modifiedPageParameters,
                            'plex_login_page',
                            'footerHTML',
                            (e.target as HTMLTextAreaElement).value
                          )
                        )
                      );
                    }
                  }}
                />
              </Label>
              <FormNote>
                Add hyperlinks with HTML anchor tags &lt;a&gt; and &lt;/a&gt;.
                Permitted attributes: href, title, target & rel.
              </FormNote>
              {modifiedPageParameters?.page_type_parameters.plex_login_page
                .customLoginPagePreMainHTMLCSS !== undefined && (
                <Label className={GlobalStyles['mt-3']}>
                  Custom CSS Styles
                  <TextArea
                    id="loginPagePreMainHTMLCSS"
                    name="loginPagePreMainHTMLCSS"
                    rows={6}
                    value={
                      modifiedPageParameters
                        ? modifiedPageParameters.page_type_parameters
                            .plex_login_page.customLoginPagePreMainHTMLCSS
                            .current_value
                        : ''
                    }
                    onChange={(e: React.ChangeEvent) => {
                      if (modifiedPageParameters) {
                        dispatch(
                          modifyPageParameters(
                            updatePageParameters(
                              modifiedPageParameters,
                              'plex_login_page',
                              'customLoginPagePreMainHTMLCSS',
                              (e.target as HTMLTextAreaElement).value
                            )
                          )
                        );
                      }
                    }}
                  />
                </Label>
              )}
              {modifiedPageParameters?.page_type_parameters.plex_login_page
                .customLoginPagePreFormHTMLCSS !== undefined && (
                <Label className={GlobalStyles['mt-3']}>
                  Custom HTML before login form
                  <TextArea
                    id="loginPagePreFormHTMLCSS"
                    name="loginPagePreFormHTMLCSS"
                    rows={6}
                    value={
                      modifiedPageParameters
                        ? modifiedPageParameters.page_type_parameters
                            .plex_login_page.customLoginPagePreFormHTMLCSS
                            .current_value
                        : ''
                    }
                    onChange={(e: React.ChangeEvent) => {
                      if (modifiedPageParameters) {
                        dispatch(
                          modifyPageParameters(
                            updatePageParameters(
                              modifiedPageParameters,
                              'plex_login_page',
                              'customLoginPagePreFormHTMLCSS',
                              (e.target as HTMLTextAreaElement).value
                            )
                          )
                        );
                      }
                    }}
                  />
                </Label>
              )}
              {modifiedPageParameters?.page_type_parameters.plex_login_page
                .customLoginPagePreFieldsetHTMLCSS !== undefined && (
                <Label className={GlobalStyles['mt-3']}>
                  Custom HTML inside login form before fieldset
                  <TextArea
                    id="loginPagePreFieldsetHTMLCSS"
                    name="loginPagePreFieldsetHTMLCSS"
                    rows={6}
                    value={
                      modifiedPageParameters
                        ? modifiedPageParameters.page_type_parameters
                            .plex_login_page.customLoginPagePreFieldsetHTMLCSS
                            .current_value
                        : ''
                    }
                    onChange={(e: React.ChangeEvent) => {
                      if (modifiedPageParameters) {
                        dispatch(
                          modifyPageParameters(
                            updatePageParameters(
                              modifiedPageParameters,
                              'plex_login_page',
                              'customLoginPagePreFieldsetHTMLCSS',
                              (e.target as HTMLTextAreaElement).value
                            )
                          )
                        );
                      }
                    }}
                  />
                </Label>
              )}
              {modifiedPageParameters?.page_type_parameters.plex_login_page
                .customLoginPagePostFieldsetHTMLCSS !== undefined && (
                <Label className={GlobalStyles['mt-3']}>
                  Custom HTML inside login form after fieldset
                  <TextArea
                    id="loginPagePostFieldsetHTMLCSS"
                    name="loginPagePostFieldsetHTMLCSS"
                    rows={6}
                    value={
                      modifiedPageParameters
                        ? modifiedPageParameters.page_type_parameters
                            .plex_login_page.customLoginPagePostFieldsetHTMLCSS
                            .current_value
                        : ''
                    }
                    onChange={(e: React.ChangeEvent) => {
                      if (modifiedPageParameters) {
                        dispatch(
                          modifyPageParameters(
                            updatePageParameters(
                              modifiedPageParameters,
                              'plex_login_page',
                              'customLoginPagePostFieldsetHTMLCSS',
                              (e.target as HTMLTextAreaElement).value
                            )
                          )
                        );
                      }
                    }}
                  />
                </Label>
              )}
              {modifiedPageParameters?.page_type_parameters.plex_login_page
                .customLoginPagePostFormHTMLCSS !== undefined && (
                <Label className={GlobalStyles['mt-3']}>
                  Custom HTML after login form
                  <TextArea
                    id="loginPagePostFormHTMLCSS"
                    name="loginPagePostFormHTMLCSS"
                    rows={6}
                    value={
                      modifiedPageParameters
                        ? modifiedPageParameters.page_type_parameters
                            .plex_login_page.customLoginPagePostFormHTMLCSS
                            .current_value
                        : ''
                    }
                    onChange={(e: React.ChangeEvent) => {
                      if (modifiedPageParameters) {
                        dispatch(
                          modifyPageParameters(
                            updatePageParameters(
                              modifiedPageParameters,
                              'plex_login_page',
                              'customLoginPagePostFormHTMLCSS',
                              (e.target as HTMLTextAreaElement).value
                            )
                          )
                        );
                      }
                    }}
                  />
                </Label>
              )}
            </fieldset>
          </AccordionItem>
          <AccordionItem title="Create account page settings">
            <fieldset>
              <Heading
                size="3"
                headingLevel="3"
                className={GlobalStyles['mt-6']}
              />
              <Label className={GlobalStyles['mt-3']}>
                Additional data fields
                <Checkbox
                  name="requireName"
                  checked={
                    modifiedPageParameters
                      ? modifiedPageParameters.page_type_parameters
                          .plex_create_user_page.requireName.current_value ===
                        'true'
                      : false
                  }
                  onChange={(e: React.ChangeEvent) => {
                    if (modifiedPageParameters) {
                      dispatch(
                        modifyPageParameters(
                          updatePageParameters(
                            modifiedPageParameters,
                            'plex_create_user_page',
                            'requireName',
                            (e.currentTarget as HTMLInputElement).checked
                              ? 'true'
                              : 'false'
                          )
                        )
                      );
                    }
                  }}
                >
                  Name
                </Checkbox>
              </Label>
              <Label className={GlobalStyles['mt-3']}>
                Header copy
                <TextInput
                  id="createAccountPageHeadingCopy"
                  name="createAccountPageHeadingCopy"
                  value={
                    modifiedPageParameters
                      ? modifiedPageParameters.page_type_parameters
                          .plex_create_user_page.headingText.current_value
                      : ''
                  }
                  onChange={(e: React.ChangeEvent) => {
                    if (modifiedPageParameters) {
                      dispatch(
                        modifyPageParameters(
                          updatePageParameters(
                            modifiedPageParameters,
                            'plex_create_user_page',
                            'headingText',
                            (e.target as HTMLInputElement).value
                          )
                        )
                      );
                    }
                  }}
                />
              </Label>
              <Label className={GlobalStyles['mt-3']}>
                Subheading copy
                <TextInput
                  id="createAccountPageSubheadingCopy"
                  name="createAccountPageSubheadingCopy"
                  value={
                    modifiedPageParameters
                      ? modifiedPageParameters.page_type_parameters
                          .plex_create_user_page.subheadingText.current_value
                      : ''
                  }
                  onChange={(e: React.ChangeEvent) => {
                    if (modifiedPageParameters) {
                      dispatch(
                        modifyPageParameters(
                          updatePageParameters(
                            modifiedPageParameters,
                            'plex_create_user_page',
                            'subheadingText',
                            (e.target as HTMLInputElement).value
                          )
                        )
                      );
                    }
                  }}
                />
              </Label>
              <Label className={GlobalStyles['mt-3']}>
                Footer contents
                <TextArea
                  id="createAccountPageFooterContents"
                  name="createAccountPageFooterContents"
                  rows={6}
                  value={
                    modifiedPageParameters
                      ? modifiedPageParameters.page_type_parameters
                          .plex_create_user_page.footerHTML.current_value
                      : ''
                  }
                  onChange={(e: React.ChangeEvent) => {
                    if (modifiedPageParameters) {
                      dispatch(
                        modifyPageParameters(
                          updatePageParameters(
                            modifiedPageParameters,
                            'plex_create_user_page',
                            'footerHTML',
                            (e.target as HTMLTextAreaElement).value
                          )
                        )
                      );
                    }
                  }}
                />
              </Label>
              <FormNote>
                Add hyperlinks with HTML anchor tags &lt;a&gt; and &lt;/a&gt;.
                Permitted attributes: href, title, target & rel.
              </FormNote>
              {modifiedPageParameters?.page_type_parameters
                .plex_create_user_page.customCreateUserPagePreMainHTMLCSS !==
                undefined && (
                <Label className={GlobalStyles['mt-3']}>
                  Custom CSS Styles
                  <TextArea
                    id="createPagePreMainHTMLCSS"
                    name="createPagePreMainHTMLCSS"
                    rows={6}
                    value={
                      modifiedPageParameters
                        ? modifiedPageParameters.page_type_parameters
                            .plex_create_user_page
                            .customCreateUserPagePreMainHTMLCSS.current_value
                        : ''
                    }
                    onChange={(e: React.ChangeEvent) => {
                      if (modifiedPageParameters) {
                        dispatch(
                          modifyPageParameters(
                            updatePageParameters(
                              modifiedPageParameters,
                              'plex_create_user_page',
                              'customCreateUserPagePreMainHTMLCSS',
                              (e.target as HTMLTextAreaElement).value
                            )
                          )
                        );
                      }
                    }}
                  />
                </Label>
              )}
              {modifiedPageParameters?.page_type_parameters
                .plex_create_user_page.customCreateUserPagePreFormHTMLCSS !==
                undefined && (
                <Label className={GlobalStyles['mt-3']}>
                  Custom HTML before create user form
                  <TextArea
                    id="createPagePreFormHTMLCSS"
                    name="createPagePreFormHTMLCSS"
                    rows={6}
                    value={
                      modifiedPageParameters
                        ? modifiedPageParameters.page_type_parameters
                            .plex_create_user_page
                            .customCreateUserPagePreFormHTMLCSS.current_value
                        : ''
                    }
                    onChange={(e: React.ChangeEvent) => {
                      if (modifiedPageParameters) {
                        dispatch(
                          modifyPageParameters(
                            updatePageParameters(
                              modifiedPageParameters,
                              'plex_create_user_page',
                              'customCreateUserPagePreFormHTMLCSS',
                              (e.target as HTMLTextAreaElement).value
                            )
                          )
                        );
                      }
                    }}
                  />
                </Label>
              )}
              {modifiedPageParameters?.page_type_parameters
                .plex_create_user_page
                .customCreateUserPagePreFieldsetHTMLCSS !== undefined && (
                <Label className={GlobalStyles['mt-3']}>
                  Custom HTML inside create user form before fieldset
                  <TextArea
                    id="createPagePreFieldsetHTMLCSS"
                    name="createPagePreFieldsetHTMLCSS"
                    rows={6}
                    value={
                      modifiedPageParameters
                        ? modifiedPageParameters.page_type_parameters
                            .plex_create_user_page
                            .customCreateUserPagePreFieldsetHTMLCSS
                            .current_value
                        : ''
                    }
                    onChange={(e: React.ChangeEvent) => {
                      if (modifiedPageParameters) {
                        dispatch(
                          modifyPageParameters(
                            updatePageParameters(
                              modifiedPageParameters,
                              'plex_create_user_page',
                              'customCreateUserPagePreFieldsetHTMLCSS',
                              (e.target as HTMLTextAreaElement).value
                            )
                          )
                        );
                      }
                    }}
                  />
                </Label>
              )}
              {modifiedPageParameters?.page_type_parameters
                .plex_create_user_page
                .customCreateUserPagePostFieldsetHTMLCSS !== undefined && (
                <Label className={GlobalStyles['mt-3']}>
                  Custom HTML inside create user form after fieldset
                  <TextArea
                    id="createPagePostFieldsetHTMLCSS"
                    name="createPagePostFieldsetHTMLCSS"
                    rows={6}
                    value={
                      modifiedPageParameters
                        ? modifiedPageParameters.page_type_parameters
                            .plex_create_user_page
                            .customCreateUserPagePostFieldsetHTMLCSS
                            .current_value
                        : ''
                    }
                    onChange={(e: React.ChangeEvent) => {
                      if (modifiedPageParameters) {
                        dispatch(
                          modifyPageParameters(
                            updatePageParameters(
                              modifiedPageParameters,
                              'plex_create_user_page',
                              'customCreateUserPagePostFieldsetHTMLCSS',
                              (e.target as HTMLTextAreaElement).value
                            )
                          )
                        );
                      }
                    }}
                  />
                </Label>
              )}
              {modifiedPageParameters?.page_type_parameters
                .plex_create_user_page.customCreateUserPagePostFormHTMLCSS !==
                undefined && (
                <Label className={GlobalStyles['mt-3']}>
                  Custom HTML after create user form
                  <TextArea
                    id="createPagePostFormHTMLCSS"
                    name="createPagePostFormHTMLCSS"
                    rows={6}
                    value={
                      modifiedPageParameters
                        ? modifiedPageParameters.page_type_parameters
                            .plex_create_user_page
                            .customCreateUserPagePostFormHTMLCSS.current_value
                        : ''
                    }
                    onChange={(e: React.ChangeEvent) => {
                      if (modifiedPageParameters) {
                        dispatch(
                          modifyPageParameters(
                            updatePageParameters(
                              modifiedPageParameters,
                              'plex_create_user_page',
                              'customCreateUserPagePostFormHTMLCSS',
                              (e.target as HTMLTextAreaElement).value
                            )
                          )
                        );
                      }
                    }}
                  />
                </Label>
              )}
            </fieldset>
          </AccordionItem>
        </Accordion>
        {fetchError ? (
          <InlineNotification theme="alert">{fetchError}</InlineNotification>
        ) : (
          <></>
        )}
        {saveError ? (
          <InlineNotification theme="alert">{saveError}</InlineNotification>
        ) : (
          <></>
        )}
        {saveSuccess ? (
          <InlineNotification theme="success">
            Successfully saved login settings
          </InlineNotification>
        ) : (
          <></>
        )}
        <ButtonGroup className={GlobalStyles['mt-6']}>
          <Button type="submit" disabled={!isDirty} theme="primary">
            Save login settings
          </Button>
          <Button type="reset" disabled={!isDirty} theme="secondary">
            Cancel
          </Button>
        </ButtonGroup>
      </form>
      {modifiedPageParameters ? (
        <div id="loginPagePreview" className={Styles.loginPagePreview}>
          {previewPage === 'login' ? (
            <>
              <Button
                theme="ghost"
                onClick={() => setPreviewPage('createUser')}
              >
                See create user page
              </Button>
              <div className={Styles.loginPreviewFrame}>
                <LoginContents
                  params={pageParametersForPreview(
                    modifiedPageParameters,
                    'plex_login_page'
                  )}
                />
              </div>
            </>
          ) : (
            <>
              <Button theme="ghost" onClick={() => setPreviewPage('login')}>
                See login page
              </Button>
              <div className={Styles.loginPreviewFrame}>
                <CreateUserContents
                  params={pageParametersForPreview(
                    modifiedPageParameters,
                    'plex_create_user_page'
                  )}
                  disabled={false}
                />
              </div>
            </>
          )}
        </div>
      ) : null}
    </Card>
  );
};

export default connect((state: RootState) => ({
  tenantPageParameters: state.appPageParameters,
  modifiedPageParameters: state.modifiedPageParameters,
  isFetching: state.fetchingPageParameters,
  fetchError: state.pageParametersFetchError,
  isSaving: state.savingPageParameters,
  saveSuccess: state.pageParametersSaveSuccess,
  saveError: state.pageParametersSaveError,
}))(LoginSettingsEditor);
