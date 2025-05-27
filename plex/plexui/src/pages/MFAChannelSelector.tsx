import React, { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Ellipsis, LoginStyles } from '@userclouds/sharedui';
import {
  IconEmail,
  IconRecoveryCode,
  IconSms,
  IconAuthenticatorApp,
  Text,
} from '@userclouds/ui-component-lib';
import API, { makePlexURL } from '../API';
import requestParams from '../models/MFAChannelPageRequest';
import { mungePageParameters } from '../models/PageParametersResponse';
import { MFAChannelsResponse } from '../models/MFAChannelsResponse';
import { MFAPurpose } from '../models/MFAPurpose';
import Styles from './MFA.module.css';

const getIcon = (channelType: string) => {
  if (channelType === 'email') {
    return <IconEmail />;
  }
  if (channelType === 'recovery_code') {
    return <IconRecoveryCode />;
  }
  if (channelType === 'sms') {
    return <IconSms />;
  }
  if (channelType === 'authenticator') {
    return <IconAuthenticatorApp />;
  }
};

const MFAChannelSelector: React.FC = () => {
  const [params, setParams] = useState<Record<string, string>>();
  const [mfaChannelsResponse, setMFAChannelsResponse] =
    useState<MFAChannelsResponse>();

  const [queryParams] = useSearchParams();

  const missingSessionIDMsg = `Missing required 'session_id' parameter`;
  const sessionID = queryParams.get('session_id');

  useEffect(() => {
    if (sessionID) {
      API.fetchPageParameters(
        sessionID,
        requestParams.pageType,
        requestParams.parameterNames
      ).then((pageParameters) => {
        setParams(mungePageParameters(pageParameters));
      });
      API.fetchMFAChannels(sessionID).then((mfaChannels) => {
        setMFAChannelsResponse(mfaChannels);
      });
    }
  }, [sessionID]);

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

      <div className={Styles.container}>
        <h1
          className={Styles.heading}
          style={{
            color: `${params.pageTextColor}`,
          }}
        >
          {params.headingText}
        </h1>
        {mfaChannelsResponse.description && (
          <Text>{mfaChannelsResponse.description}</Text>
        )}
        <div>
          {mfaChannelsResponse.mfa_purpose !== MFAPurpose.Configure && (
            <table className={Styles.table}>
              <tbody>
                {mfaChannelsResponse &&
                  mfaChannelsResponse.mfa_channels &&
                  mfaChannelsResponse.mfa_channels.map(
                    (channel) =>
                      channel && (
                        <tr>
                          <td className={Styles.tableCellIcon}>
                            {getIcon(channel.mfa_channel_type)}
                          </td>
                          <td className={Styles.tableCell}>
                            <a
                              title="mfaSubmit"
                              href={makePlexURL('/plexui/mfasubmit', {
                                session_id: sessionID,
                                channelType: channel.mfa_channel_type,
                              })}
                              onClick={(e: React.MouseEvent) => {
                                e.preventDefault();
                                API.mfaChallenge(
                                  sessionID,
                                  channel.mfa_channel_id
                                );
                              }}
                            >
                              {channel.mfa_channel_description}
                            </a>
                          </td>
                        </tr>
                      )
                  )}
              </tbody>
            </table>
          )}
          <table className={Styles.table}>
            <tbody className={Styles.clickableBody}>
              {mfaChannelsResponse &&
                mfaChannelsResponse.mfa_channel_types &&
                mfaChannelsResponse.mfa_channel_types
                  .sort((a, b) =>
                    a.mfa_channel_type.localeCompare(b.mfa_channel_type)
                  )
                  .map(
                    (channel) =>
                      channel &&
                      channel.can_create && (
                        <tr
                          key={channel.mfa_channel_type}
                          className={Styles.clickableRow}
                          onClick={(e: React.MouseEvent) => {
                            e.preventDefault();
                            API.navigateToConfigureMFAChannel(
                              sessionID,
                              channel.mfa_channel_type
                            );
                          }}
                        >
                          <td className={Styles.tableCellSingle}>
                            {'Add ' +
                              channel.mfa_channel_type.charAt(0).toUpperCase() +
                              channel.mfa_channel_type.substring(1)}
                          </td>
                        </tr>
                      )
                  )}
            </tbody>
          </table>
          {mfaChannelsResponse.mfa_purpose === MFAPurpose.Configure && (
            <button
              onClick={() => API.mfaEndConfiguration(sessionID)}
              className={Styles.submitButton}
              style={{
                color: `${params.actionButtonTextColor}`,
                borderColor: `${params.actionButtonBorderColor}`,
                backgroundColor: `${params.actionButtonFillColor}`,
              }}
            >
              Exit Configuration and Log In
            </button>
          )}
        </div>
      </div>
    </main>
  );
};

export default MFAChannelSelector;
