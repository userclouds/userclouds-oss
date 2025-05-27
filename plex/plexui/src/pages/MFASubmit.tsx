import React, { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Ellipsis, LoginStyles } from '@userclouds/sharedui';
import {
  Button,
  InlineNotification,
  Label,
  Text,
} from '@userclouds/ui-component-lib';
import API from '../API';
import requestParams from '../models/MFASubmitPageRequest';
import { mungePageParameters } from '../models/PageParametersResponse';
import { MFASubmitSettings } from '../models/MFASubmitSettings';
import { MFAPurpose } from '../models/MFAPurpose';
import Styles from './MFA.module.css';

const MFASubmit: React.FC = () => {
  const [params, setParams] = useState<Record<string, string>>();
  const [disabled, setDisabled] = useState<boolean>(false);
  const [statusText, setStatusText] = useState<string>('');
  const [isError, setIsError] = useState<boolean>(false);
  const [code, setCode] = useState<string>('');
  const [settings, setSettings] = useState<MFASubmitSettings>();
  const [canChangeChannel, setCanChangeChannel] = useState<boolean>();

  const [queryParams] = useSearchParams();

  const missingSessionIDMsg = `Missing required 'session_id' parameter`;
  const sessionID = queryParams.get('session_id');

  useEffect(() => {
    if (sessionID) {
      API.fetchPageParameters(
        sessionID,
        requestParams.pageType,
        requestParams.parameterNames
      ).then(
        (pageParameters) => {
          setParams(mungePageParameters(pageParameters));
        },
        () => {
          setIsError(true);
        }
      );
      API.fetchMFASubmitSettings(sessionID).then(
        (mfaSubmitSettings) => {
          setSettings(mfaSubmitSettings);
          setCanChangeChannel(mfaSubmitSettings.can_change_channel);
        },
        () => {
          setIsError(true);
        }
      );
    }
  }, [sessionID]);

  if (!sessionID) {
    return <div>{missingSessionIDMsg}</div>;
  }
  if (!params || !settings) {
    return (
      <main className={LoginStyles.login} aria-busy>
        <form className={LoginStyles.loginForm}>
          <p>Loading ...</p>
          <Ellipsis id="loadingIndicator" />
        </form>
      </main>
    );
  }

  const onCodeSubmit = async (configure = false) => {
    setStatusText(params.loginStartStatusText);
    setIsError(false);
    setDisabled(true);
    const maybeError = await API.mfaSubmit(sessionID, code, configure);
    if (maybeError) {
      // on success it redirects and returns empty str, so if we get here it must be a failure
      setStatusText(`${params.loginFailStatusText}: ${maybeError.message}`);
      setIsError(true);
      setDisabled(false);
    } else {
      setStatusText(params.loginSuccessStatusText);
      setIsError(false);
    }
  };

  let statusTypeClass = LoginStyles.statusInfo;
  if (isError) {
    statusTypeClass = LoginStyles.statusError;
  }

  return (
    <main
      className={LoginStyles.login}
      aria-busy={false}
      style={{
        background: `${params.pageBackgroundColor}`,
        color: `${params.pageTextColor}`,
      }}
    >
      <img
        src={params.logoImageFile}
        alt=""
        className={LoginStyles.logoImage}
      />

      <div className={Styles.container}>
        <h1
          className={Styles.heading}
          style={{
            color: `${params.pageTextColor}`,
          }}
        >
          {params.headingText}
        </h1>
        {statusText && (
          <Text className={`${LoginStyles.statusText} ${statusTypeClass}`}>
            {statusText}
          </Text>
        )}
        {settings.challenge_description && (
          <Text htmlFor="form_mfa">{settings.challenge_description}</Text>
        )}
        {settings.challenge_status && (
          <InlineNotification theme="alert">
            {settings.challenge_status}
          </InlineNotification>
        )}
        {settings.registration_qr_code && (
          <div className={Styles.fitContainer}>
            <img
              className={Styles.qrCode}
              src={settings.registration_qr_code}
              alt={settings.registration_link}
            />
          </div>
        )}

        <form
          className={Styles.form}
          onSubmit={(e) => {
            e.preventDefault();
            onCodeSubmit();
          }}
        >
          {settings.can_submit_code && (
            <Label className={Styles.formElement}>
              Input Code
              <input
                className={Styles.textInput}
                name="form_mfa"
                value={code}
                required
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  setCode(e.target.value.trim());
                }}
              />
            </Label>
          )}
          {settings.challenge_block_expiration && (
            <>
              <Label>
                You can try again:
                <Text>
                  {new Date(settings.challenge_block_expiration).toString()}
                </Text>
              </Label>
            </>
          )}

          {!disabled && (
            <>
              {settings.mfa_purpose === MFAPurpose.Login &&
                settings.can_submit_code && (
                  <>
                    <button
                      type="submit"
                      className={Styles.submitButton}
                      style={{
                        color: `${params.actionButtonTextColor}`,
                        borderColor: `${params.actionButtonBorderColor}`,
                        backgroundColor: `${params.actionButtonFillColor}`,
                      }}
                    >
                      Log In
                    </button>

                    <button
                      onClick={() => onCodeSubmit(true)}
                      className={Styles.submitButtonSecondary}
                      style={{
                        borderColor: `${params.pageTextColor}`,
                      }}
                    >
                      Log In and Manage MFA Methods
                    </button>
                  </>
                )}

              {settings.mfa_purpose !== MFAPurpose.Login &&
                settings.can_submit_code && (
                  <>
                    <button
                      type="submit"
                      className={Styles.submitButton}
                      style={{
                        color: `${params.actionButtonTextColor}`,
                        borderColor: `${params.actionButtonBorderColor}`,
                        backgroundColor: `${params.actionButtonFillColor}`,
                      }}
                    >
                      Verify Code
                    </button>

                    {settings.mfa_purpose === MFAPurpose.LoginSetup && (
                      <button
                        onClick={() => onCodeSubmit(true)}
                        className={Styles.submitButtonSecondary}
                        style={{
                          borderColor: `${params.pageTextColor}`,
                        }}
                      >
                        Verify Code and Manage MFA Methods
                      </button>
                    )}
                  </>
                )}
              {settings.mfa_purpose === MFAPurpose.Configure && (
                <button
                  onClick={() => API.navigateToConfigureMFA(sessionID)}
                  className={Styles.submitButtonSecondary}
                  style={{
                    borderColor: `${params.pageTextColor}`,
                  }}
                >
                  Return To Configuration Menu
                </button>
              )}
              <div className={Styles.formElement}>
                {settings.can_reissue_challenge && (
                  <Button
                    theme="outline"
                    size="small"
                    onClick={() => {
                      API.getNewChallenge(sessionID, settings.channel_id);
                    }}
                  >
                    Re-Send Code
                  </Button>
                )}
                {canChangeChannel && (
                  <Button
                    className={Styles.signInButton}
                    theme="outline"
                    size="small"
                    onClick={() => {
                      API.navigateToChooseMFAChannel(sessionID);
                    }}
                  >
                    {settings.mfa_purpose === MFAPurpose.Configure
                      ? 'Configure a Different Method'
                      : 'Use a Different Method'}
                  </Button>
                )}
              </div>
            </>
          )}
        </form>
      </div>
    </main>
  );
};

export default MFASubmit;
