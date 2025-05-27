import React, { useState, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { Ellipsis, LoginStyles } from '@userclouds/sharedui';
import API from '../API';
import requestParams from '../models/FinishResetPasswordPageRequest';
import { mungePageParameters } from '../models/PageParametersResponse';

const FinishResetPassword: React.FC = () => {
  const [params, setParams] = useState<Record<string, string>>();
  const [disabled, setDisabled] = useState<boolean>(false);
  const [statusText, setStatusText] = useState<string>('');
  const [isError, setIsError] = useState<boolean>(false);
  const [password, setPassword] = useState<string>('');

  const location = useLocation();
  const queryParams = new URLSearchParams(location.search);

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
    }
  }, [sessionID]);

  if (!sessionID) {
    return <div>{missingSessionIDMsg}</div>;
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

  const otpCode = queryParams.get('otp_code');
  if (!otpCode) {
    return <div>{params.missingOTPCodeText}</div>;
  }

  const onPasswordSubmit = async () => {
    setStatusText(params.resetPasswordStartStatusText);
    setIsError(false);
    setDisabled(true);
    const maybeError = await API.finishResetPassword(
      sessionID,
      otpCode,
      password
    );
    if (maybeError) {
      // on success it redirects and returns empty str, so if we get here it must be a failure
      setStatusText(
        `${params.resetPasswordFailStatusText}: ${maybeError.message}`
      );
      setIsError(true);
      setDisabled(false);
    } else {
      setStatusText(params.resetPasswordSuccessStatusText);
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
      <form
        className={LoginStyles.loginForm}
        style={{
          borderColor: `${params.actionButtonBorderColor}`,
        }}
        onSubmit={async (e) => {
          e.preventDefault();
          await onPasswordSubmit();
        }}
      >
        <fieldset disabled={disabled}>
          <h1
            className={LoginStyles.heading}
            style={{
              color: `${params.pageTextColor}`,
            }}
          >
            {params.headingText}
          </h1>
          <div className={`${LoginStyles.statusText} ${statusTypeClass}`}>
            {statusText}
          </div>
          <label htmlFor="form_username">{params.newPasswordLabel}</label>
          <input
            type="password"
            id="form_password"
            value={password}
            required
            onChange={(e) => {
              setPassword(e.target.value);
            }}
          />
          <input
            className={LoginStyles.loginButton}
            type="submit"
            value={params.actionSubmitButtonText}
            style={{
              background: params.actionButtonFillColor,
              border: `2px solid ${
                params.actionButtonBorderColor || 'transparent'
              }`,
              color: params.actionButtonTextColor,
            }}
          />
        </fieldset>
      </form>
    </main>
  );
};

export default FinishResetPassword;
