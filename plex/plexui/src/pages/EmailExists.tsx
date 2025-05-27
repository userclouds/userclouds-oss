import { useState, useEffect } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Ellipsis, LoginStyles, SocialSignin } from '@userclouds/sharedui';
import API from '../API';
import requestParams from '../models/EmailExistsPageRequest';
import { mungePageParameters } from '../models/PageParametersResponse';

const EmailExistsPage = () => {
  const [params, setParams] = useState<Record<string, string>>();
  const [isError, setIsError] = useState<boolean>(false);
  const [statusText, setStatusText] = useState<string>('');
  const [busy, setBusy] = useState<boolean>(false);
  const [password, setPassword] = useState<string>('');
  const location = useLocation();
  const queryParams = new URLSearchParams(location.search);
  const sessionID = queryParams.get('session_id') || '';
  const existingUserLoginProviders = queryParams.get('authns') || '';
  const email = queryParams.get('email') || '';
  const needDivider =
    existingUserLoginProviders.search('password') !== -1 &&
    existingUserLoginProviders.search(',') !== -1;
  const navigate = useNavigate();

  useEffect(() => {
    if (sessionID) {
      API.fetchPageParameters(
        sessionID,
        requestParams.pageType,
        requestParams.parameterNames
      ).then(
        (pageParameters) => {
          setParams({
            ...mungePageParameters(pageParameters),
            authenticationMethods: existingUserLoginProviders,
          });
        },
        (error) => {
          setIsError(true);
          setStatusText(error.message);
        }
      );
    }
  }, [sessionID, existingUserLoginProviders]);

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

  const submitHandler = async () => {
    setStatusText(params.loginStartStatusText);
    setIsError(false);
    setBusy(true);

    await API.grantOrDenyAuthnAddPermission(sessionID, true);

    const maybeError = await API.usernamePasswordLogin(
      sessionID,
      email,
      password
    );

    if (maybeError) {
      setStatusText(`${params.loginFailStatusText}: ${maybeError.message}`);
      setIsError(true);
      setBusy(false);
    } else {
      setStatusText(params.loginSuccessStatusText);
      setIsError(false);
      setBusy(false);
    }
  };

  const onSocialLogin = async (provider: string) => {
    setBusy(true);
    setStatusText(params.socialRedirectStatusText);
    setIsError(false);

    await API.grantOrDenyAuthnAddPermission(sessionID, true);
    await API.startSocialLogin(sessionID, provider, email);
  };

  const onDeclineAddAuthn = async () => {
    setBusy(true);
    setStatusText('');
    setIsError(false);

    await API.grantOrDenyAuthnAddPermission(sessionID, false);
    navigate({
      pathname: '/login',
      search: `?session_id=${sessionID}`,
    });
  };

  return (
    <main
      className={LoginStyles.login}
      style={{
        background: `${params.pageBackgroundColor}`,
        color: `${params.pageTextColor}`,
      }}
      aria-busy={false}
    >
      <style>
        {`
              .${LoginStyles.loginLink},
              .${LoginStyles.heading},
              .${LoginStyles.subheading} {
                color: ${params.pageTextColor};
              }
              .${LoginStyles.loginLink}:hover,
              .${LoginStyles.loginLink}:focus {
                color: ${params.pageTextColor};
              }
              .${LoginStyles.socialField} {
                border-color: ${params.actionButtonBorderColor};
                color: ${params.pageTextColor};
              }
              .${LoginStyles.divider}::before,
              .${LoginStyles.divider}::after {
                border-color: ${params.pageTextColor};
              }
              .${LoginStyles.footer} a[href],
              .${LoginStyles.footer} a[href]:hover,
              .${LoginStyles.footer} a[href]:focus,
              .${LoginStyles.footer} a[href]:visited,
              .${LoginStyles.footer} a[href]:active {
                color: ${params.pageTextColor};
              }
              .${LoginStyles.socialText} {
                padding: 0;
              }
            `}
      </style>
      <form
        className={LoginStyles.loginForm}
        style={{
          borderColor: `${params.actionButtonBorderColor}`,
        }}
        onSubmit={(e) => {
          e.preventDefault();
          if (submitHandler) {
            submitHandler();
          }
        }}
      >
        <img
          src={params.logoImageFile}
          alt=""
          className={LoginStyles.logoImage}
        />
        <fieldset disabled={busy}>
          <h1 className={LoginStyles.heading}>{params.headingText}</h1>
          {params.subheadingText ? (
            <h2 className={LoginStyles.subheading}>
              {params.subheadingText.split('\n').map((s) => (
                <p>{s}</p>
              ))}
            </h2>
          ) : (
            ''
          )}
          <div
            className={`${LoginStyles.statusText} ${
              isError ? LoginStyles.statusError : LoginStyles.statusInfo
            }`}
          >
            {statusText}
          </div>
          <SocialSignin
            params={params}
            socialSubmitHandler={onSocialLogin}
            forMerge
          />

          {needDivider ? (
            <div className={LoginStyles.divider}>
              <span className={LoginStyles.dividerText}>OR</span>
            </div>
          ) : (
            ''
          )}

          {existingUserLoginProviders.search('password') !== -1 ? (
            <>
              <label htmlFor="form_username">{params.userNameLabel}</label>
              <input
                type="text"
                id="form_username"
                value={email}
                required
                readOnly
              />
              <label htmlFor="form_password">{params.passwordLabel}</label>
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
                style={{
                  background: params.actionButtonFillColor,
                  border: `2px solid ${
                    params.actionButtonBorderColor || 'transparent'
                  }`,
                  color: params.actionButtonTextColor,
                }}
                value={params.actionSubmitButtonText}
              />
            </>
          ) : (
            ''
          )}

          <button
            className={LoginStyles.loginLink}
            onClick={onDeclineAddAuthn}
            onKeyPress={onDeclineAddAuthn}
          >
            {params.emailExistsCreateAccountText}
          </button>
          <button
            className={LoginStyles.loginLink}
            onClick={onDeclineAddAuthn}
            onKeyPress={onDeclineAddAuthn}
          >
            {params.emailExistsLoginText}
          </button>

          <p
            className={LoginStyles.footer}
            dangerouslySetInnerHTML={{ __html: params.footerHTML }} // eslint-disable-line react/no-danger
          />
        </fieldset>
      </form>
    </main>
  );
};

export default EmailExistsPage;
