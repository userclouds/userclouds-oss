import React, { useState, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { Ellipsis, LoginStyles } from '@userclouds/sharedui';
import API from '../API';
import requestParams from '../models/PasswordlessLoginPageRequest';
import { mungePageParameters } from '../models/PageParametersResponse';

const PasswordlessLogin: React.FC = () => {
  const [params, setParams] = useState<Record<string, string>>();
  const [disabled, setDisabled] = useState<boolean>(false);
  const [statusText, setStatusText] = useState<string>('');
  const [isError, setIsError] = useState<boolean>(false);
  const [email, setEmail] = useState<string>('');
  const [code, setCode] = useState<string>('');
  const [emailSent, setEmailSent] = useState<boolean>(false);

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
    setStatusText(params.passwordlessSendEmailStartStatusText);
    setIsError(false);
    setDisabled(true);
    const maybeError = await API.startPasswordlessLogin(sessionID, email);
    if (maybeError) {
      setStatusText(`Login failed: ${maybeError.message}`);
      setIsError(true);
    } else {
      setStatusText(
        `${params.passwordlessSendEmailSuccessStatusText} ${email}...`
      );
      setIsError(false);
      setEmailSent(true);
    }
    setDisabled(false);
  };

  const onCodeSubmit = async () => {
    setStatusText(params.loginStartStatusText);
    setIsError(false);
    setDisabled(true);
    const maybeError = await API.finishPasswordlessLogin(
      sessionID,
      email,
      code
    );
    if (maybeError) {
      // on success it redirects and returns empty str, so if we get here it must be a failure
      setStatusText(`${params.loginFailStatusText}: ${maybeError.message}`);
      setIsError(true);
      setDisabled(false);
    } else {
      setStatusText(params.loginSuccessStatusText);
      setIsError(false);
      setEmailSent(true);
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
          if (emailSent) {
            await onCodeSubmit();
          } else {
            await onEmailSubmit();
          }
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
            required={!emailSent}
            readOnly={emailSent}
            onChange={(e) => {
              setEmail(e.target.value.trim());
            }}
          />
          {emailSent && (
            <>
              <label htmlFor="form_otp_code">{params.otpCodeLabel}</label>
              <input
                type="text"
                id="form_otp_code"
                name="form_otp_code"
                value={code}
                required
                onChange={(e) => {
                  setCode(e.target.value.trim());
                }}
              />
            </>
          )}
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

export default PasswordlessLogin;
