import React, { useState, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { Ellipsis, LoginStyles } from '@userclouds/sharedui';
import API from '../API';
import requestParams from '../models/StartResetPasswordPageRequest';
import { mungePageParameters } from '../models/PageParametersResponse';

const StartResetPassword: React.FC = () => {
  const [params, setParams] = useState<Record<string, string>>();
  const [disabled, setDisabled] = useState<boolean>(false);
  const [statusText, setStatusText] = useState<string>('');
  const [isError, setIsError] = useState<boolean>(false);
  const [email, setEmail] = useState<string>('');

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

  const onEmailSubmit = async () => {
    setStatusText(params.resetPasswordStartStatusText);
    setIsError(false);
    setDisabled(true);
    const maybeError = await API.startResetPassword(sessionID, email);
    if (maybeError) {
      setStatusText(
        `${params.resetPasswordFailStatusText}: ${maybeError.message}`
      );
      setIsError(true);
    } else {
      setStatusText(`${params.resetPasswordSuccessStatusText} ${email}...`);
      setIsError(false);
    }
    setDisabled(false);
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
          await onEmailSubmit();
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
          <label htmlFor="form_email">{params.emailLabel}</label>
          <input
            type="email"
            id="form_email"
            name="form_email"
            value={email}
            required
            onChange={(e) => {
              setEmail(e.target.value.trim());
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

export default StartResetPassword;
