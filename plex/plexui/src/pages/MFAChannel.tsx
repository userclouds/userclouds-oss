import React, { useState, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { Ellipsis, LoginStyles } from '@userclouds/sharedui';
import API from '../API';
import requestParams from '../models/MFAChannelPageRequest';
import { mungePageParameters } from '../models/PageParametersResponse';
import { MFAChannelsResponse } from '../models/MFAChannelsResponse';
import { MFAPurpose } from '../models/MFAPurpose';
import Styles from './MFA.module.css';

const MFAChannel: React.FC = () => {
  const [params, setParams] = useState<Record<string, string>>();
  const [descriptionText, setDescriptionText] = useState<string>('');
  const [isError, setIsError] = useState<boolean>(false);
  const [mfaChannelsResponse, setMFAChannelsResponse] =
    useState<MFAChannelsResponse>();
  const [purpose, setPurpose] = useState<string>('');

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
      API.fetchMFAChannels(sessionID).then(
        (mfaChannels) => {
          setMFAChannelsResponse(mfaChannels);
          setDescriptionText(mfaChannels.description);
          setPurpose(mfaChannels.mfa_purpose);
        },
        () => {
          setIsError(true);
        }
      );
      if (purpose === MFAPurpose.LoginSetup) {
        API.navigateToConfigureMFA(sessionID);
      }
      if (purpose === MFAPurpose.Configure) {
        API.navigateToConfigureMFA(sessionID);
      }
      if (purpose === MFAPurpose.Login) {
        API.navigateToChooseMFAChannel(sessionID);
      }
    }
  }, [sessionID, descriptionText, purpose]);

  if (!sessionID) {
    return <div>{missingSessionIDMsg}</div>;
  }
  if (!params || !mfaChannelsResponse) {
    return (
      <main className={LoginStyles.login} aria-busy>
        <form className={LoginStyles.loginForm}>
          <p>Loading ...</p>
          <Ellipsis id="loadingIndicator" />
        </form>
      </main>
    );
  }

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

      <h1
        className={Styles.heading}
        style={{
          color: `${params.pageTextColor}`,
        }}
      >
        {params.headingText}
      </h1>
      <div className={`${LoginStyles.statusText} ${statusTypeClass}`}>
        {descriptionText}
      </div>
    </main>
  );
};

export default MFAChannel;
