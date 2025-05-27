import React, { useState, useEffect } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import {
  Ellipsis,
  CreateUserContents,
  LoginStyles,
} from '@userclouds/sharedui';
import API from '../API';
import requestParams from '../models/CreateUserPageRequest';
import { mungePageParameters } from '../models/PageParametersResponse';

const CreateUser: React.FC = () => {
  const [params, setParams] = useState<Record<string, string>>();
  const [disabled, setDisabled] = useState<boolean>(false);
  const [statusText, setStatusText] = useState<string>('');
  const [isError, setIsError] = useState<boolean>(false);
  const [personalName, setName] = useState<string>('');
  const [email, setEmail] = useState<string>('');
  const [password, setPassword] = useState<string>('');

  const location = useLocation();
  const navigate = useNavigate();
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
        (error) => {
          setIsError(true);
          setStatusText(error.message);
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

  if (!params.allowCreation) {
    return <div>params.createUserDisabledText</div>;
  }

  const onCreate = async () => {
    setStatusText(params.createUserStartStatusText);
    setIsError(false);
    setDisabled(true);
    API.createUser(
      sessionID,
      params.clientID,
      email,
      email,
      password,
      personalName
    ).then(
      async (result) => {
        if (result === true) {
          const maybeError = await API.usernamePasswordLogin(
            sessionID,
            email,
            password
          );
          if (maybeError) {
            setStatusText(
              `${params.loginFailStatusText}: ${maybeError.message}`
            );
            setIsError(true);
            setDisabled(false);
          } else {
            // Technically this already should have started redirecting, but it can be async.
            setStatusText(params.loginSuccessStatusText);
            setIsError(false);
          }
        } else {
          // this code path means that the email address is already in use
          // redirect to userwithemailexists page, passing along necessary variables in state
          const existingUserLoginProviders = result.join(',');
          navigate({
            pathname: '/userwithemailexists',
            search: `?session_id=${sessionID}&email=${email}&authns=${existingUserLoginProviders}`,
          });
        }
      },
      (error) => {
        setStatusText(`${params.createUserFailStatusText}: ${error.message}`);
        setIsError(true);
        setDisabled(false);
      }
    );
  };

  return (
    <CreateUserContents
      params={params}
      disabled={disabled}
      isError={isError}
      statusText={statusText}
      email={email}
      password={password}
      personalName={personalName}
      onCreate={onCreate}
      setName={setName}
      setEmail={setEmail}
      setPassword={setPassword}
    />
  );
};

export default CreateUser;
