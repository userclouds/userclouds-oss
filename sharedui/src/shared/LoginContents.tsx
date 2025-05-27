import React from 'react';
import InnerHTML from 'dangerously-set-html-content';
import Styles from './Login.module.css';
import {
  oidcAuthSettingsFromPageParams,
  getOIDCButtonImage,
} from '../OIDCAuthSettings';

const SocialSignin = ({
  params,
  socialSubmitHandler = undefined,
  forMerge,
}: {
  params: Record<string, string>;
  socialSubmitHandler?: (provider: string) => void;
  forMerge: boolean;
}) => {
  const authMethods = new Set(params.authenticationMethods.split(','));
  return (
    <ul className="socialLogin">
      {oidcAuthSettingsFromPageParams(params, forMerge).map(
        (method) =>
          authMethods.has(method.name) && (
            <li key={method.name} className={Styles.socialField}>
              <button
                key={method.buttonText}
                type="button"
                onClick={(e) => {
                  e.preventDefault();
                  if (socialSubmitHandler) {
                    socialSubmitHandler(method.name);
                  }
                }}
              >
                <img
                  src={getOIDCButtonImage(method)}
                  alt=""
                  className={Styles.logoImg}
                />
                <div className={Styles.socialText}>{method.buttonText}</div>
              </button>
            </li>
          )
      )}
    </ul>
  );
};

const StyledLoginContents = ({
  params,
  username = '',
  password = '',
  statusText = '',
  busy = false,
  hasError = false,
  submitHandler = () => null,
  usernameChangeHandler = () => null,
  passwordChangeHandler = () => null,
  socialSubmitHandler = () => null,
  onCreateAccount = () => null,
  onPasswordlessLogin = () => null,
  onForgotPassword = () => null,
}: {
  params: Record<string, string>;
  username?: string;
  password?: string;
  statusText?: string;
  busy?: boolean;
  hasError?: boolean;
  submitHandler?: () => void;
  usernameChangeHandler?: React.Dispatch<React.SetStateAction<string>>;
  passwordChangeHandler?: React.Dispatch<React.SetStateAction<string>>;
  socialSubmitHandler?: (provider: string) => void;
  onCreateAccount?: (e: React.MouseEvent) => void;
  onPasswordlessLogin?: (e: React.MouseEvent) => void;
  onForgotPassword?: (e: React.MouseEvent) => void;
}): JSX.Element => {
  const authMethods = new Set(params.authenticationMethods.split(','));
  const passwordEnabled: boolean = authMethods.has('password');
  authMethods.delete('password');
  const passwordlessEnabled: boolean = authMethods.has('passwordless');
  authMethods.delete('passwordless');
  const oidcEnabled: boolean = authMethods.size > 0;

  return (
    <main
      className={Styles.login}
      style={{
        background: `${params.pageBackgroundColor}`,
        color: `${params.pageTextColor}`,
      }}
      aria-busy={false}
    >
      <style>
        {`
          .${Styles.loginLink},
          .${Styles.heading},
          .${Styles.subheading} {
            color: ${params.pageTextColor};
          }
          .${Styles.loginLink}:hover,
          .${Styles.loginLink}:focus {
            color: ${params.pageTextColor};
          }
          .${Styles.socialField} {
            border-color: ${params.actionButtonBorderColor};
            color: ${params.pageTextColor};
          }
          .${Styles.divider}::before,
          .${Styles.divider}::after {
            border-color: ${params.pageTextColor};
          }
          .${Styles.footer} a[href],
          .${Styles.footer} a[href]:hover,
          .${Styles.footer} a[href]:focus,
          .${Styles.footer} a[href]:visited,
          .${Styles.footer} a[href]:active {
            color: ${params.pageTextColor};
          }
        `}
      </style>
      <form
        id="loginForm"
        className={Styles.loginForm}
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
        <img src={params.logoImageFile} alt="" className={Styles.logoImage} />
        <h1 className={Styles.heading}>{params.headingText}</h1>
        {params.subheadingText ? (
          <h2 className={Styles.subheading}>{params.subheadingText}</h2>
        ) : (
          ''
        )}
        <div
          className={`${Styles.statusText} ${
            hasError ? Styles.statusError : Styles.statusInfo
          }`}
        >
          {statusText}
        </div>
        {oidcEnabled && params.pageOrderSocialFirst === 'true' ? (
          <>
            <SocialSignin
              params={params}
              socialSubmitHandler={socialSubmitHandler}
              forMerge={false}
            />
            {passwordEnabled ? (
              <div className={Styles.divider}>
                <span className={Styles.dividerText}>OR</span>
              </div>
            ) : (
              ''
            )}
          </>
        ) : (
          ''
        )}
        <fieldset disabled={busy}>
          {passwordEnabled ? (
            <>
              <label htmlFor="form_username">{params.userNameLabel}</label>
              <input
                type="text"
                id="form_username"
                name="form_username"
                value={username || ''}
                required
                onChange={(e) => {
                  if (usernameChangeHandler) {
                    usernameChangeHandler(e.target.value.trim());
                  }
                }}
              />
              <label htmlFor="form_password">{params.passwordLabel}</label>
              <input
                type="password"
                id="form_password"
                name="form_password"
                value={password || ''}
                required
                onChange={(e) => {
                  if (passwordChangeHandler) {
                    passwordChangeHandler(e.target.value);
                  }
                }}
              />
            </>
          ) : (
            ''
          )}
          {passwordEnabled && params.passwordResetEnabled === 'true' && (
            <a
              href="/plexui/startresetpassword"
              className={Styles.loginLink}
              onClick={onForgotPassword}
            >
              {params.forgotPasswordText}
            </a>
          )}
          {passwordlessEnabled && (
            <a
              href="/plexui/passwordlesslogin"
              className={Styles.loginLink}
              onClick={onPasswordlessLogin}
            >
              {params.passwordlessLoginText}
            </a>
          )}
          {passwordEnabled ? (
            <button
              className={Styles.loginButton}
              type="submit"
              style={{
                background: params.actionButtonFillColor,
                border: `2px solid ${
                  params.actionButtonBorderColor || 'transparent'
                }`,
                color: params.actionButtonTextColor,
              }}
            >
              {params.actionSubmitButtonText}
            </button>
          ) : (
            ''
          )}
        </fieldset>
        {oidcEnabled && params.pageOrderSocialFirst === 'false' ? (
          <>
            {passwordEnabled ? (
              <div className={Styles.divider}>
                <span className={Styles.dividerText}>OR</span>
              </div>
            ) : (
              ''
            )}
            <SocialSignin
              params={params}
              socialSubmitHandler={socialSubmitHandler}
              forMerge={false}
            />
          </>
        ) : (
          ''
        )}
        {passwordEnabled && params.allowCreation === 'true' && (
          <a
            href="/plexui/createuser"
            className={Styles.loginLink}
            onClick={onCreateAccount}
          >
            {params.createAccountText}
          </a>
        )}
        <p
          className={Styles.footer}
          dangerouslySetInnerHTML={{ __html: params.footerHTML }} // eslint-disable-line react/no-danger
        />
      </form>
    </main>
  );
};

const UnstyledLoginContents = ({
  params,
  username = '',
  password = '',
  statusText = '',
  busy = false,
  hasError = false,
  submitHandler = () => null,
  usernameChangeHandler = () => null,
  passwordChangeHandler = () => null,
  socialSubmitHandler = () => null,
  onCreateAccount = () => null,
  onPasswordlessLogin = () => null,
  onForgotPassword = () => null,
}: {
  params: Record<string, string>;
  username?: string;
  password?: string;
  statusText?: string;
  busy?: boolean;
  hasError?: boolean;
  submitHandler?: () => void;
  usernameChangeHandler?: React.Dispatch<React.SetStateAction<string>>;
  passwordChangeHandler?: React.Dispatch<React.SetStateAction<string>>;
  socialSubmitHandler?: (provider: string) => void;
  onCreateAccount?: (e: React.MouseEvent) => void;
  onPasswordlessLogin?: (e: React.MouseEvent) => void;
  onForgotPassword?: (e: React.MouseEvent) => void;
}): JSX.Element => {
  const authMethods = new Set(params.authenticationMethods.split(','));
  const passwordEnabled: boolean = authMethods.has('password');
  authMethods.delete('password');
  const passwordlessEnabled: boolean = authMethods.has('passwordless');
  authMethods.delete('passwordless');
  const oidcEnabled: boolean = authMethods.size > 0;

  return (
    <>
      <div
        className="login_premain_div"
        // eslint-disable-next-line react/no-danger
        dangerouslySetInnerHTML={{
          __html: params.customLoginPagePreMainHTMLCSS,
        }}
      />
      <main className="login_main">
        <div
          className="login_preform_div"
          // eslint-disable-next-line react/no-danger
          dangerouslySetInnerHTML={{
            __html: params.customLoginPagePreFormHTMLCSS,
          }}
        />
        <form
          className="login_form"
          onSubmit={(e) => {
            e.preventDefault();
            if (submitHandler) {
              submitHandler();
            }
          }}
        >
          <div
            className="login_prefields_div"
            // eslint-disable-next-line react/no-danger
            dangerouslySetInnerHTML={{
              __html: params.customLoginPagePreFieldsetHTMLCSS,
            }}
          />
          <h1 className="login_heading">{params.headingText}</h1>
          {params.subheadingText ? (
            <h2 className="login_subheading">{params.subheadingText}</h2>
          ) : (
            ''
          )}
          <fieldset className="login_fieldset" disabled={busy}>
            <div className={`login_status ${hasError ? 'error' : 'info'}`}>
              {statusText}
            </div>
            {oidcEnabled && params.pageOrderSocialFirst === 'true' ? (
              <>
                <SocialSignin
                  params={params}
                  socialSubmitHandler={socialSubmitHandler}
                  forMerge={false}
                />
                {passwordEnabled ? (
                  <div className="login_divider">
                    <span className="login_divider_or">OR</span>
                  </div>
                ) : (
                  ''
                )}
              </>
            ) : (
              ''
            )}
            {passwordEnabled ? (
              <>
                <label className="login_username_label" htmlFor="form_username">
                  {params.userNameLabel}
                </label>
                <input
                  type="text"
                  className="login_username_input"
                  id="form_username"
                  value={username || ''}
                  required
                  onChange={(e) => {
                    if (usernameChangeHandler) {
                      usernameChangeHandler(e.target.value.trim());
                    }
                  }}
                />
                <label className="login_password_label" htmlFor="form_password">
                  {params.passwordLabel}
                </label>
                <input
                  type="password"
                  className="login_password_input"
                  id="form_password"
                  value={password || ''}
                  required
                  onChange={(e) => {
                    if (passwordChangeHandler) {
                      passwordChangeHandler(e.target.value);
                    }
                  }}
                />
              </>
            ) : (
              ''
            )}
            {passwordEnabled && params.passwordResetEnabled === 'true' && (
              <a
                href="/plexui/startresetpassword"
                className="login_forgot_password_link"
                onClick={onForgotPassword}
              >
                {params.forgotPasswordText}
              </a>
            )}
            {passwordlessEnabled && (
              <a
                href="/plexui/passwordlesslogin"
                className="login_passwordless_link"
                onClick={onPasswordlessLogin}
              >
                {params.passwordlessLoginText}
              </a>
            )}
            {passwordEnabled ? (
              <input
                className="login_button"
                type="submit"
                value={params.actionSubmitButtonText}
              />
            ) : (
              ''
            )}
            {oidcEnabled && params.pageOrderSocialFirst === 'false' ? (
              <>
                {passwordEnabled ? (
                  <div className="login_divider">
                    <span className="login_divider_or">OR</span>
                  </div>
                ) : (
                  ''
                )}
                <SocialSignin
                  params={params}
                  socialSubmitHandler={socialSubmitHandler}
                  forMerge={false}
                />
              </>
            ) : (
              ''
            )}
            {passwordEnabled && params.allowCreation === 'true' && (
              <a
                href="/plexui/createuser"
                className="login_create_account"
                onClick={onCreateAccount}
              >
                {params.createAccountText}
              </a>
            )}
            <p
              className="login_footer"
              // eslint-disable-next-line react/no-danger
              dangerouslySetInnerHTML={{ __html: params.footerHTML }}
            />
          </fieldset>
          <div
            className="login_postfields_div"
            // eslint-disable-next-line react/no-danger
            dangerouslySetInnerHTML={{
              __html: params.customLoginPagePostFieldsetHTMLCSS,
            }}
          />
        </form>
        <div
          className="login_postform_div"
          // eslint-disable-next-line react/no-danger
          dangerouslySetInnerHTML={{
            __html: params.customLoginPagePostFormHTMLCSS,
          }}
        />
      </main>
    </>
  );
};

const LoginContents = ({
  params,
  username = '',
  password = '',
  statusText = '',
  busy = false,
  hasError = false,
  submitHandler = () => null,
  usernameChangeHandler = () => null,
  passwordChangeHandler = () => null,
  socialSubmitHandler = () => null,
  onCreateAccount = () => null,
  onPasswordlessLogin = () => null,
  onForgotPassword = () => null,
}: {
  params: Record<string, string>;
  username?: string;
  password?: string;
  statusText?: string;
  busy?: boolean;
  hasError?: boolean;
  submitHandler?: () => void;
  usernameChangeHandler?: React.Dispatch<React.SetStateAction<string>>;
  passwordChangeHandler?: React.Dispatch<React.SetStateAction<string>>;
  socialSubmitHandler?: (provider: string) => void;
  onCreateAccount?: (e: React.MouseEvent) => void;
  onPasswordlessLogin?: (e: React.MouseEvent) => void;
  onForgotPassword?: (e: React.MouseEvent) => void;
}): JSX.Element => {
  // full page source override
  if (params.pageSourceOverride) {
    return <InnerHTML html={params.pageSourceOverride} />;
  }

  // custom html and css so use unstyled login contents
  if (
    params.customLoginPagePostFieldsetHTMLCSS ||
    params.customLoginPagePreFieldsetHTMLCSS ||
    params.customLoginPagePostFormHTMLCSS ||
    params.customLoginPagePreFormHTMLCSS ||
    params.customLoginPagePreMainHTMLCSS
  ) {
    return (
      <UnstyledLoginContents
        params={params}
        username={username}
        password={password}
        statusText={statusText}
        busy={busy}
        hasError={hasError}
        submitHandler={submitHandler}
        usernameChangeHandler={usernameChangeHandler}
        passwordChangeHandler={passwordChangeHandler}
        socialSubmitHandler={socialSubmitHandler}
        onCreateAccount={onCreateAccount}
        onPasswordlessLogin={onPasswordlessLogin}
        onForgotPassword={onForgotPassword}
      />
    );
  }

  // default login contents
  return (
    <StyledLoginContents
      params={params}
      username={username}
      password={password}
      statusText={statusText}
      busy={busy}
      hasError={hasError}
      submitHandler={submitHandler}
      usernameChangeHandler={usernameChangeHandler}
      passwordChangeHandler={passwordChangeHandler}
      socialSubmitHandler={socialSubmitHandler}
      onCreateAccount={onCreateAccount}
      onPasswordlessLogin={onPasswordlessLogin}
      onForgotPassword={onForgotPassword}
    />
  );
};

export { LoginContents, SocialSignin };
