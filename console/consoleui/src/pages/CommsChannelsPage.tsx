import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  GlobalStyles,
  InputReadOnly,
  Label,
  LoaderDots,
  InlineNotification,
  Text,
  TextInput,
  ToolTip,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import { modifyPlexConfig, modifyTelephonyProvider } from '../actions/authn';
import {
  getTenantKeysRequest,
  getTenantKeysSuccess,
  getTenantKeysError,
  rotateTenantKeysRequest,
  rotateTenantKeysSuccess,
  rotateTenantKeysError,
} from '../actions/keys';
import { AppDispatch, RootState } from '../store';
import TenantPlexConfig, {
  UpdatePlexConfigReason,
} from '../models/TenantPlexConfig';

import { SelectedTenant } from '../models/Tenant';
import { TwilioPropertyType } from '../models/TelephonyProvider';

import { fetchTenantPublicKeys, rotateTenantKeys } from '../API/TenantKeys';
import { fetchPlexConfig, savePlexConfig } from '../thunks/authn';
import Styles from './AuthNPage.module.css';
import PageCommon from './PageCommon.module.css';

const CommsChannels = ({
  tenant,
  plexConfig,
  modifiedConfig,
  isDirty,
  fetchError,
  isSaving,
  saveSuccess,
  saveError,
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
  dispatch: AppDispatch;
}) => {
  const readOnly = !tenant?.is_admin;

  return (
    <Card
      lockedMessage={readOnly ? 'You do not have edit access' : ''}
      listview
    >
      {fetchError || !plexConfig || !modifiedConfig ? (
        <Text>{fetchError || 'Loading telephony…'}</Text>
      ) : (
        <CardRow
          title="Telephony Provider Settings"
          tooltip={
            <>
              Configure telephony provider settings for sending SMS messages.
              Twilio is currently supported.
            </>
          }
          collapsible
        >
          <>
            <div />
            <div>
              <Label className={Styles.clientDataElement}>
                Twilio Account SID
                <br />
                <TextInput
                  id="telephony_provider_twilio_account_sid"
                  name="telephony_provider_twilio_account_sid"
                  readOnly={readOnly}
                  defaultValue={
                    modifiedConfig.tenant_config.plex_map.telephony_provider
                      .properties[TwilioPropertyType.AccountSID]
                  }
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    dispatch(
                      modifyTelephonyProvider({
                        [TwilioPropertyType.AccountSID]: e.target.value,
                      })
                    );
                  }}
                />
              </Label>
            </div>
            <div>
              <Label className={Styles.clientDataElement}>
                Twilio Standard API Key SID
                <br />
                <TextInput
                  id="telephony_provider_twilio_api_key_sid"
                  name="telephony_provider_twilio_api_key_sid"
                  readOnly={readOnly}
                  defaultValue={
                    modifiedConfig.tenant_config.plex_map.telephony_provider
                      .properties[TwilioPropertyType.APIKeySID]
                  }
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    dispatch(
                      modifyTelephonyProvider({
                        [TwilioPropertyType.APIKeySID]: e.target.value,
                      })
                    );
                  }}
                />
              </Label>
            </div>
            <div>
              <Label className={Styles.clientDataElement}>
                Twilio Standard API Secret
                <br />
                <TextInput
                  id="telephony_provider_twilio_api_secret"
                  name="telephony_provider_twilio_api_secret"
                  readOnly={readOnly}
                  defaultValue={
                    modifiedConfig.tenant_config.plex_map.telephony_provider
                      .properties[TwilioPropertyType.APISecret]
                  }
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    dispatch(
                      modifyTelephonyProvider({
                        [TwilioPropertyType.APISecret]: e.target.value,
                      })
                    );
                  }}
                />
              </Label>
            </div>
            {saveError && (
              <InlineNotification theme="alert">{saveError}</InlineNotification>
            )}
            {saveSuccess === UpdatePlexConfigReason.ModifyTelephonyProvider && (
              <InlineNotification theme="success">
                {saveSuccess}
              </InlineNotification>
            )}
            {tenant?.is_admin && (
              <ButtonGroup>
                <Button
                  disabled={!isDirty}
                  isLoading={isSaving}
                  onClick={() => {
                    dispatch(
                      savePlexConfig(
                        tenant.id,
                        modifiedConfig,
                        UpdatePlexConfigReason.ModifyTelephonyProvider
                      )
                    );
                  }}
                >
                  Save
                </Button>
              </ButtonGroup>
            )}
          </>
        </CardRow>
      )}

      {fetchError || !plexConfig || !modifiedConfig ? (
        <Text>{fetchError || 'Loading email settings...'}</Text>
      ) : (
        <>
          <CardRow
            title="Email Settings"
            tooltip="Configure an email server integration."
            collapsible
          >
            <>
              <div />
              <div className={PageCommon.carddetailsrow}>
                <Label>
                  Email Server Host (optional)
                  <br />
                  <TextInput
                    id="email_host"
                    name="email_host"
                    readOnly={readOnly}
                    defaultValue={
                      modifiedConfig.tenant_config.plex_map.email_host
                    }
                    onChange={(e: React.ChangeEvent) => {
                      modifiedConfig.tenant_config.plex_map.email_host = (
                        e.target as HTMLInputElement
                      ).value;
                      dispatch(modifyPlexConfig(modifiedConfig));
                    }}
                  />
                </Label>
                <Label>
                  Port
                  <br />
                  <TextInput
                    id="email_port"
                    name="email_port"
                    readOnly={readOnly}
                    defaultValue={
                      modifiedConfig.tenant_config.plex_map.email_port
                    }
                    type="number"
                    size="medium"
                    onChange={(e: React.ChangeEvent) => {
                      modifiedConfig.tenant_config.plex_map.email_port = (
                        e.target as HTMLInputElement
                      ).valueAsNumber;
                      dispatch(modifyPlexConfig(modifiedConfig));
                    }}
                  />
                </Label>
                <Label>
                  Username
                  <br />
                  <TextInput
                    id="email_username"
                    name="email_username"
                    readOnly={readOnly}
                    defaultValue={
                      modifiedConfig.tenant_config.plex_map.email_username
                    }
                    onChange={(e: React.ChangeEvent) => {
                      modifiedConfig.tenant_config.plex_map.email_username = (
                        e.target as HTMLInputElement
                      ).value;
                      dispatch(modifyPlexConfig(modifiedConfig));
                    }}
                  />
                </Label>
                <Label className={Styles.clientDataElement}>
                  Password
                  <br />
                  <TextInput
                    id="email_password"
                    name="email_password"
                    readOnly={readOnly}
                    defaultValue={
                      modifiedConfig.tenant_config.plex_map.email_password
                    }
                    type="password"
                    onChange={(e: React.ChangeEvent) => {
                      modifiedConfig.tenant_config.plex_map.email_password = (
                        e.target as HTMLInputElement
                      ).value;
                      dispatch(modifyPlexConfig(modifiedConfig));
                    }}
                  />
                </Label>

                {saveError && (
                  <InlineNotification theme="alert">
                    {saveError}
                  </InlineNotification>
                )}
                {saveSuccess === UpdatePlexConfigReason.ModifyEmailServer && (
                  <InlineNotification theme="success">
                    {saveSuccess}
                  </InlineNotification>
                )}
                {tenant?.is_admin && (
                  <ButtonGroup>
                    <Button
                      disabled={!isDirty}
                      isLoading={isSaving}
                      onClick={() => {
                        dispatch(
                          savePlexConfig(
                            tenant.id,
                            modifiedConfig,
                            UpdatePlexConfigReason.ModifyEmailServer
                          )
                        );
                      }}
                    >
                      Save email settings
                    </Button>
                  </ButtonGroup>
                )}
              </div>
            </>
          </CardRow>
          <ConnectedJWTKeys />
        </>
      )}
    </Card>
  );
};
const ConnectedCommsChannels = connect((state: RootState) => {
  return {
    tenant: state.selectedTenant,
    plexConfig: state.tenantPlexConfig,
    modifiedConfig: state.modifiedPlexConfig,
    isDirty: state.plexConfigIsDirty,
    fetchError: state.fetchPlexConfigError,
    isSaving: state.savingPlexConfig,
    saveSuccess: state.savePlexConfigSuccess,
    saveError: state.savePlexConfigError,
  };
})(CommsChannels);

const fetchKeys = (tenantID: string) => (dispatch: AppDispatch) => {
  dispatch(getTenantKeysRequest());
  fetchTenantPublicKeys(tenantID).then(
    (response: string[]) => {
      dispatch(getTenantKeysSuccess(response));
    },
    (error: APIError) => {
      dispatch(getTenantKeysError(error));
    }
  );
};

const rotateKeys = (tenantID: string) => (dispatch: AppDispatch) => {
  dispatch(rotateTenantKeysRequest());
  rotateTenantKeys(tenantID).then(
    () => {
      dispatch(rotateTenantKeysSuccess());
      dispatch(fetchKeys(tenantID));
    },
    (error: APIError) => {
      dispatch(rotateTenantKeysError(error));
    }
  );
};

const JWTKeys = ({
  selectedTenant,
  tenantPublicKey,
  fetchingKeys,
  rotatingKeys,
  fetchError,
  rotateError,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  tenantPublicKey: string;
  fetchingKeys: boolean;
  rotatingKeys: boolean;
  fetchError: string;
  rotateError: string;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenant) {
      // TODO: fetch keys only once card is open
      // need to expose the internal open/closed state of cards
      dispatch(fetchKeys(selectedTenant.id));
    }
  }, [selectedTenant, dispatch]);

  return (
    <>
      {selectedTenant && tenantPublicKey ? (
        <CardRow
          title="JWT Signing Keys"
          tooltip={
            <>
              Copy, download and rotate your public and private signing keys for
              this tenant.
            </>
          }
          collapsible
        >
          <div>
            <Label>
              Public Key
              <InputReadOnly monospace className={Styles.publicKey}>
                {tenantPublicKey}
              </InputReadOnly>
            </Label>
            {selectedTenant.is_admin && (
              <>
                {rotateError && (
                  <InlineNotification theme="alert">
                    {rotateError}
                  </InlineNotification>
                )}
                <ButtonGroup className={GlobalStyles['mt-6']}>
                  <Button
                    theme="primary"
                    isLoading={rotatingKeys}
                    onClick={() => {
                      dispatch(rotateKeys(selectedTenant.id));
                    }}
                  >
                    Rotate Keys
                  </Button>
                  <Button theme="outline" disabled={rotatingKeys}>
                    <a
                      href={`/api/tenants/${selectedTenant.id}/keys/private`}
                      title=""
                    >
                      Download Private Key
                    </a>
                  </Button>
                </ButtonGroup>
              </>
            )}
          </div>
        </CardRow>
      ) : fetchingKeys ? (
        <LoaderDots
          assistiveText="Loading JWT..."
          size="medium"
          theme="brand"
        />
      ) : (
        ''
      )}
      {fetchError && (
        <InlineNotification theme="alert">{fetchError}</InlineNotification>
      )}
    </>
  );
};
const ConnectedJWTKeys = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  tenantPublicKey: state.tenantPublicKey,
  fetchingKeys: state.fetchingPublicKeys,
  rotatingKeys: state.rotatingTenantKeys,
  fetchError: state.fetchTenantPublicKeysError,
  rotateError: state.rotateTenantKeysError,
}))(JWTKeys);

const CommsChannelsPage = ({
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
  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              Set up your communication channels in UserClouds.
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
      </div>
      <ConnectedCommsChannels />
    </>
  );
};

export default connect((state: RootState) => {
  return {
    tenantID: state.selectedTenantID,
    query: state.query,
  };
})(CommsChannelsPage);
