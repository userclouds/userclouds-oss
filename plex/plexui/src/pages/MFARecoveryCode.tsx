import React, { useState, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { Ellipsis, LoginStyles } from '@userclouds/sharedui';
import {
  IconButton,
  IconCopy,
  InlineNotification,
  Text,
} from '@userclouds/ui-component-lib';
import API from '../API';
import requestParams from '../models/MFARecoveryCodePageRequest';
import { mungePageParameters } from '../models/PageParametersResponse';
import Styles from './MFA.module.css';

const MFARecoveryCode: React.FC = () => {
  const [params, setParams] = useState<Record<string, string>>();
  const [isError, setIsError] = useState<boolean>(false);

  const location = useLocation();
  const queryParams = new URLSearchParams(location.search);

  const missingRecoveryCodeMsg = `Missing required 'recovery_code' parameter`;
  const missingSessionIDMsg = `Missing required 'session_id' parameter`;
  const recoveryCode = queryParams.get('recovery_code');
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
    }
  }, [sessionID]);

  if (!sessionID) {
    return <div>{missingSessionIDMsg}</div>;
  }
  if (!recoveryCode) {
    return <div>{missingRecoveryCodeMsg}</div>;
  }
  if (!params) {
    return (
      <main className={LoginStyles.login} aria-busy>
        <form className={LoginStyles.loginForm}>
          <p>Loading ...</p>
          <Ellipsis id="loadingIndicator" />
        </form>
      </main>
    );
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

      {recoveryCode ? (
        <>
          <div className={Styles.container}>
            <h1 className={Styles.heading}>
              {params.headingText
                ? params.headingText
                : 'Download your recovery code'}
            </h1>
            <Text className={Styles.text}>
              Save your recovery code somewhere secure. It can be used to login
              to your account if you lose access to your other authentication
              methods.
            </Text>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                API.mfaConfirmRecoveryCode(sessionID);
              }}
              className={Styles.form}
            >
              <textarea
                rows={1}
                cols={52}
                disabled
                className={Styles.formElement}
              >
                {recoveryCode}
              </textarea>
              {window.isSecureContext && (
                <IconButton
                  icon={<IconCopy />}
                  onClick={() => {
                    navigator.clipboard.writeText(recoveryCode);
                  }}
                  title="Copy to Clipboard"
                  aria-label="Copy to Clipboard"
                />
              )}
              <button
                type="submit"
                className={Styles.formElement + ' ' + Styles.submitButton}
                style={{
                  color: `${params.actionButtonTextColor}`,
                  borderColor: `${params.actionButtonBorderColor}`,
                  backgroundColor: `${params.actionButtonFillColor}`,
                }}
              >
                I have saved my code
              </button>
            </form>
          </div>
        </>
      ) : (
        isError && (
          <InlineNotification theme="alert">
            Something went wrong
          </InlineNotification>
        )
      )}
    </main>
  );
};

export default MFARecoveryCode;
