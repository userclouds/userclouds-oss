import React, { useState, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { Ellipsis, LoginContents, LoginStyles } from '@userclouds/sharedui';
import API from '../API';
import requestParams from '../models/LoginPageRequest';
import { mungePageParameters } from '../models/PageParametersResponse';

const prepareLoginPageSourceOverride = (
  pageSource: string,
  sessionID: string
) => {
  return pageSource.replace('SESSION_ID_PLACEHOLDER', sessionID);
};

const Login: React.FC = () => {
  const [params, setParams] = useState<Record<string, string> | undefined>();
  const [busy, setBusy] = useState<boolean>(false);
  const [isError, setIsError] = useState<boolean>(false);
  const [password, setPassword] = useState<string>('');
  const [statusText, setStatusText] = useState<string>('');
  const [username, setUsername] = useState<string>('');

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
          if (pageParameters.page_source_override) {
            setParams({
              pageSourceOverride: prepareLoginPageSourceOverride(
                pageParameters.page_source_override,
                sessionID
              ),
            });
          } else {
            setParams(mungePageParameters(pageParameters));
          }
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
  const onCreateAccount = (e: React.MouseEvent) => {
    e.preventDefault();
    API.navigateToCreateAccount(sessionID);
  };
  const onPasswordlessLogin = (e: React.MouseEvent) => {
    e.preventDefault();
    API.navigateToPasswordlessLogin(sessionID);
  };
  const onForgotPassword = (e: React.MouseEvent) => {
    e.preventDefault();
    API.navigateToPasswordReset(sessionID);
  };

  const onSocialLogin = (provider: string) => {
    setBusy(true);
    setStatusText(params.socialRedirectStatusText);
    setIsError(false);
    API.startSocialLogin(sessionID, provider);
  };
  const onUsernamePasswordLogin = async () => {
    setStatusText(params.loginStartStatusText);
    setIsError(false);
    setBusy(true);
    const maybeError = await API.usernamePasswordLogin(
      sessionID,
      username,
      password
    );
    if (maybeError) {
      // on success it redirects and returns empty str, so if we get here it must be a failure
      setStatusText(`${params.loginFailStatusText}: ${maybeError.message}`);
      setIsError(true);
      setBusy(false);
    } else {
      setStatusText(params.loginSuccessStatusText);
      setIsError(false);
      setBusy(false);
    }
  };

  return (
    <LoginContents
      params={params}
      username={username}
      password={password}
      statusText={statusText}
      busy={busy}
      hasError={isError}
      submitHandler={onUsernamePasswordLogin}
      usernameChangeHandler={setUsername}
      passwordChangeHandler={setPassword}
      socialSubmitHandler={onSocialLogin}
      onCreateAccount={onCreateAccount}
      onPasswordlessLogin={onPasswordlessLogin}
      onForgotPassword={onForgotPassword}
    />
  );
};

export default Login;
