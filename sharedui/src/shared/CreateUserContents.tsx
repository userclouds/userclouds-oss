import React from 'react';
import InnerHTML from 'dangerously-set-html-content';
import Styles from './Login.module.css';

const StyledCreateUserContents = ({
  params,
  disabled,
  isError,
  statusText,
  email = '',
  password = '',
  personalName = '',
  onCreate = () => null,
  setEmail = () => null,
  setPassword = () => null,
  setName = () => null,
}: {
  params: Record<string, string>;
  disabled: boolean;
  isError?: boolean;
  statusText?: string;
  email?: string;
  password?: string;
  personalName?: string;
  onCreate?: () => void;
  setEmail?: React.Dispatch<React.SetStateAction<string>>;
  setPassword?: React.Dispatch<React.SetStateAction<string>>;
  setName?: React.Dispatch<React.SetStateAction<string>>;
}): JSX.Element => {
  let statusTypeClass = Styles.statusInfo;
  if (isError) {
    statusTypeClass = Styles.statusError;
  }

  return (
    <main
      className={Styles.login}
      aria-busy={disabled}
      style={{
        background: `${params.pageBackgroundColor}`,
        color: `${params.pageTextColor}`,
      }}
    >
      <style>
        {`
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
        id="createUserForm"
        className={Styles.loginForm}
        style={{
          borderColor: `${params.actionButtonBorderColor}`,
        }}
        onSubmit={async (e) => {
          e.preventDefault();
          if (onCreate) {
            onCreate();
          }
        }}
      >
        <img src={params.logoImageFile} alt="" className={Styles.logoImage} />
        <fieldset disabled={disabled}>
          <h1
            className={Styles.heading}
            style={{
              color: `${params.pageTextColor}`,
            }}
          >
            {params.headingText}
          </h1>
          {params.subheadingText ? (
            <h2
              className={Styles.subheading}
              style={{
                color: `${params.pageTextColor}`,
              }}
            >
              {params.subheadingText}
            </h2>
          ) : (
            ''
          )}
          {statusText ? (
            <div className={`${Styles.statusText} ${statusTypeClass}`}>
              {statusText}
            </div>
          ) : (
            ''
          )}
          {params.requireName && (
            <>
              <label htmlFor="form_name">Full name</label>
              <input
                type="text"
                id="form_name"
                name="form_name"
                value={personalName || ''}
                required
                onChange={(e) => {
                  if (setName) {
                    setName(e.target.value);
                  }
                }}
              />
            </>
          )}
          <label htmlFor="form_email">{params.emailLabel}</label>
          <input
            type="email"
            id="form_email"
            name="form_email"
            value={email || ''}
            required
            onChange={(e) => {
              if (setEmail) {
                setEmail(e.target.value.trim());
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
              if (setPassword) {
                setPassword(e.target.value);
              }
            }}
          />
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
          <p
            className={Styles.footer}
            dangerouslySetInnerHTML={{ __html: params.footerHTML }} // eslint-disable-line react/no-danger
          />
        </fieldset>
      </form>
    </main>
  );
};

const UnstyledCreateUserContents = ({
  params,
  disabled,
  isError,
  statusText,
  email = '',
  password = '',
  personalName = '',
  onCreate = () => null,
  setEmail = () => null,
  setPassword = () => null,
  setName = () => null,
}: {
  params: Record<string, string>;
  disabled: boolean;
  isError?: boolean;
  statusText?: string;
  email?: string;
  password?: string;
  personalName?: string;
  onCreate?: () => void;
  setEmail?: React.Dispatch<React.SetStateAction<string>>;
  setPassword?: React.Dispatch<React.SetStateAction<string>>;
  setName?: React.Dispatch<React.SetStateAction<string>>;
}): JSX.Element => {
  let statusTypeClass = Styles.statusInfo;
  if (isError) {
    statusTypeClass = Styles.statusError;
  }

  return (
    <>
      <div
        className="create_premain_div"
        // eslint-disable-next-line react/no-danger
        dangerouslySetInnerHTML={{
          __html: params.customCreateUserPagePreMainHTMLCSS,
        }}
      />
      <main className="create_main">
        <div
          className="create_preform_div"
          // eslint-disable-next-line react/no-danger
          dangerouslySetInnerHTML={{
            __html: params.customCreateUserPagePreFormHTMLCSS,
          }}
        />
        <form
          className="create_form"
          onSubmit={async (e) => {
            e.preventDefault();
            if (onCreate) {
              onCreate();
            }
          }}
        >
          <div
            className="create_prefieldset_div"
            // eslint-disable-next-line react/no-danger
            dangerouslySetInnerHTML={{
              __html: params.customCreateUserPagePreFieldsetHTMLCSS,
            }}
          />
          <fieldset className="create_fieldset_div" disabled={disabled}>
            <h1 className="create_heading">{params.headingText}</h1>
            {params.subheadingText ? (
              <h2 className="create_subheading">{params.subheadingText}</h2>
            ) : (
              ''
            )}
            {statusText ? (
              <div className={`create_status ${statusTypeClass}`}>
                {statusText}
              </div>
            ) : (
              ''
            )}
            {params.requireName && (
              <>
                <label className="create_name_label" htmlFor="form_name">
                  Full name
                </label>
                <input
                  className="create_name_input"
                  type="text"
                  id="form_name"
                  name="form_name"
                  value={personalName || ''}
                  required
                  onChange={(e) => {
                    if (setName) {
                      setName(e.target.value);
                    }
                  }}
                />
              </>
            )}
            <label className="create_email_label" htmlFor="form_email">
              {params.emailLabel}
            </label>
            <input
              className="create_email_input"
              type="email"
              id="form_email"
              name="form_email"
              value={email || ''}
              required
              onChange={(e) => {
                if (setEmail) {
                  setEmail(e.target.value.trim());
                }
              }}
            />
            <label className="create_password_label" htmlFor="form_password">
              {params.passwordLabel}
            </label>
            <input
              className="create_password_input"
              type="password"
              id="form_password"
              name="form_password"
              value={password || ''}
              required
              onChange={(e) => {
                if (setPassword) {
                  setPassword(e.target.value);
                }
              }}
            />
            <input
              className="create_submit"
              type="submit"
              value={params.actionSubmitButtonText}
            />
            <p
              className="create_footer"
              dangerouslySetInnerHTML={{ __html: params.footerHTML }} // eslint-disable-line react/no-danger
            />
          </fieldset>
          <div
            className="create_postfieldset_div"
            // eslint-disable-next-line react/no-danger
            dangerouslySetInnerHTML={{
              __html: params.customCreateUserPagePostFieldsetHTMLCSS,
            }}
          />
        </form>
        <div
          className="create_postform_div"
          // eslint-disable-next-line react/no-danger
          dangerouslySetInnerHTML={{
            __html: params.customCreateUserPagePostFormHTMLCSS,
          }}
        />
      </main>
    </>
  );
};

const CreateUserContents = ({
  params,
  disabled,
  isError,
  statusText,
  email = '',
  password = '',
  personalName = '',
  onCreate = () => null,
  setEmail = () => null,
  setPassword = () => null,
  setName = () => null,
}: {
  params: Record<string, string>;
  disabled: boolean;
  isError?: boolean;
  statusText?: string;
  email?: string;
  password?: string;
  personalName?: string;
  onCreate?: () => void;
  setEmail?: React.Dispatch<React.SetStateAction<string>>;
  setPassword?: React.Dispatch<React.SetStateAction<string>>;
  setName?: React.Dispatch<React.SetStateAction<string>>;
}): JSX.Element => {
  // full page source override
  if (params.pageSourceOverride) {
    return <InnerHTML html={params.pageSourceOverride} />;
  }

  // custom html and css so use unstyled create user contents
  if (
    params.customCreateUserPagePostFieldsetHTMLCSS ||
    params.customCreateUserPagePreFieldsetHTMLCSS ||
    params.customCreateUserPagePostFormHTMLCSS ||
    params.customCreateUserPagePreFormHTMLCSS ||
    params.customCreateUserPagePreMainHTMLCSS
  ) {
    return (
      <UnstyledCreateUserContents
        params={params}
        disabled={disabled}
        isError={isError}
        statusText={statusText}
        email={email}
        password={password}
        personalName={personalName}
        onCreate={onCreate}
        setEmail={setEmail}
        setPassword={setPassword}
        setName={setName}
      />
    );
  }

  return (
    <StyledCreateUserContents
      params={params}
      disabled={disabled}
      isError={isError}
      statusText={statusText}
      email={email}
      password={password}
      personalName={personalName}
      onCreate={onCreate}
      setEmail={setEmail}
      setPassword={setPassword}
      setName={setName}
    />
  );
};

export default CreateUserContents;
