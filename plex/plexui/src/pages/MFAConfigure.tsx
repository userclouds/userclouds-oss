import React, { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { APIError, Ellipsis, LoginStyles } from '@userclouds/sharedui';
import {
  IconAuthenticatorApp,
  IconButton,
  IconDeleteBin,
  IconEmail,
  IconRecoveryCode,
  IconRotate,
  IconSms,
  IconStarLine,
  IconStarSolid,
  IconToggleOn,
  IconToggleOff,
  InlineNotification,
  Label,
  Text,
} from '@userclouds/ui-component-lib';
import API, { makePlexURL } from '../API';
import requestParams from '../models/MFAChannelPageRequest';
import { mungePageParameters } from '../models/PageParametersResponse';
import { MFAChannelsResponse } from '../models/MFAChannelsResponse';
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

const MFAConfigure: React.FC = () => {
  const [params, setParams] = useState<Record<string, string>>();
  const [mfaChannelsResponse, setMFAChannelsResponse] =
    useState<MFAChannelsResponse>();
  const [error, setError] = useState<string>();
  const [mfaEnabled, setMFAEnabled] = useState<boolean>();

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
        if (mfaChannels) {
          setMFAChannelsResponse(mfaChannels);
          setMFAEnabled(hasPrimary(mfaChannels));
        }
      });
    }
  }, [sessionID]);

  const sortChannels = (mfaChannels: MFAChannelsResponse) => {
    mfaChannels.mfa_channels.sort((a, b) => {
      if (a.primary) {
        return -1;
      }
      if (b.primary) {
        return 1;
      }
      if (a.mfa_channel_type !== b.mfa_channel_type) {
        return a.mfa_channel_type.localeCompare(b.mfa_channel_type);
      }
      return a.mfa_channel_description.localeCompare(b.mfa_channel_description);
    });
  };

  const hasPrimary = (mfaChannels: MFAChannelsResponse) => {
    return (
      mfaChannels.mfa_channels.findIndex((channel) => channel.primary) >= 0
    );
  };

  const enableMFA = (sessionId: string, mfaChannels: MFAChannelsResponse) => {
    mfaChannels.mfa_channels.forEach((channel) => {
      if (channel.can_make_primary) {
        onMakePrimary(sessionId, channel.mfa_channel_id);
      }
    });
    setError('No channels configured. Configure a channel to enable MFA.');
  };

  const onMakePrimary = async (sID: string, channelID: string) => {
    setError('');
    const maybeError = await API.mfaMakePrimaryChannel(sID, channelID);

    if (maybeError instanceof APIError) {
      setError(maybeError.message);
    } else {
      setError('');
    }
  };

  const onRemovePrimary = async (sID: string) => {
    setError('');
    const maybeError = await API.mfaClearPrimaryChannel(sID);
    if (maybeError instanceof APIError) {
      setError(maybeError.message);
    } else {
      setError('');
    }
  };

  const onRotate = async (sID: string, channelID: string) => {
    setError('');
    const maybeError = await API.mfaReissueRecoveryCode(sID, channelID);

    if (maybeError instanceof APIError) {
      setError(maybeError.message);
    } else {
      setError('');
    }
  };

  const onDelete = async (sID: string, channelID: string) => {
    setError('');
    const maybeError = await API.mfaDeleteChannel(sID, channelID);

    if (maybeError instanceof APIError) {
      setError(maybeError.message);
    } else {
      setError('');
    }
  };

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
  if (mfaChannelsResponse.mfa_channels) {
    sortChannels(mfaChannelsResponse);
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
          {mfaChannelsResponse.description}
        </h1>

        {!mfaChannelsResponse.can_dismiss && ( // initial set up
          <table className={Styles.table}>
            <tbody>
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
                          className={Styles.tableRow}
                        >
                          <td className={Styles.tableCellIcon}>
                            {getIcon(channel.mfa_channel_type)}
                          </td>
                          <td className={Styles.tableCell}>
                            <a
                              href={makePlexURL(
                                '/plexui/mfachannel/configurechannel',
                                {
                                  session_id: sessionID,
                                  channelType: channel.mfa_channel_type,
                                }
                              )}
                              onClick={(e: React.MouseEvent) => {
                                e.preventDefault();
                                API.navigateToConfigureMFAChannel(
                                  sessionID,
                                  channel.mfa_channel_type
                                );
                              }}
                            >
                              {channel.mfa_channel_type
                                .charAt(0)
                                .toUpperCase() +
                                channel.mfa_channel_type.substring(1)}
                            </a>
                          </td>
                        </tr>
                      )
                  )}
            </tbody>
          </table>
        )}
        {mfaChannelsResponse.can_dismiss && ( // configuring additional channels
          <div className={Styles.fitContainer}>
            {error && (
              <InlineNotification theme="alert">{error}</InlineNotification>
            )}
            <Text>Modify Existing Channels</Text>
            {(mfaChannelsResponse.can_disable || !mfaEnabled) &&
              (mfaEnabled ? (
                <>
                  <IconButton
                    icon={<IconToggleOn />}
                    className={Styles.on}
                    onClick={() => {
                      onRemovePrimary(sessionID);
                    }}
                    title="Turn MFA Off"
                    aria-label="Turn MFA Off"
                  />
                  MFA Enabled
                </>
              ) : (
                <>
                  <IconButton
                    icon={<IconToggleOff />}
                    className={Styles.off}
                    onClick={() => {
                      enableMFA(sessionID, mfaChannelsResponse);
                    }}
                    title="Turn MFA On"
                    aria-label="Turn MFA On"
                  />
                  MFA Disabled
                </>
              ))}
            <table className={Styles.configureTable}>
              <tbody>
                {mfaChannelsResponse &&
                  mfaChannelsResponse.mfa_channels &&
                  mfaChannelsResponse.mfa_channels.map(
                    (channel) =>
                      channel && (
                        <tr
                          key={channel.mfa_channel_id}
                          className={Styles.configureTableRow}
                        >
                          <td>{getIcon(channel.mfa_channel_type)}</td>
                          <td>
                            <p className={Styles.emailText}>
                              {channel.mfa_channel_description}
                            </p>
                          </td>
                          <td>
                            {channel.primary && (
                              <IconButton
                                icon={<IconStarSolid />}
                                className={Styles.primary}
                                onClick={() => {}}
                                title="Primary MFA Method"
                                aria-label="Primary MFA Method"
                              />
                            )}
                            {channel.can_make_primary && !channel.primary && (
                              <IconButton
                                icon={<IconStarLine />}
                                className={Styles.makePrimary}
                                title="Make Primary"
                                aria-label="Make Primary"
                                onClick={() =>
                                  onMakePrimary(
                                    sessionID,
                                    channel.mfa_channel_id
                                  )
                                }
                              />
                            )}
                          </td>
                          <td>
                            {channel.mfa_channel_type === 'recovery_code' && (
                              <IconButton
                                icon={<IconRotate />}
                                title="Rotate Recovery Codes"
                                aria-label="Rotate Recovery Codes"
                                onClick={() =>
                                  onRotate(sessionID, channel.mfa_channel_id)
                                }
                              />
                            )}
                            {channel.can_delete && !channel.primary && (
                              <IconButton
                                icon={<IconDeleteBin />}
                                title="Delete"
                                aria-label="Delete"
                                onClick={() =>
                                  onDelete(sessionID, channel.mfa_channel_id)
                                }
                              />
                            )}
                          </td>
                        </tr>
                      )
                  )}
              </tbody>
            </table>
            <br />
            {mfaChannelsResponse &&
            (!mfaChannelsResponse.mfa_channels ||
              mfaChannelsResponse.max_mfa_channels >
                mfaChannelsResponse.mfa_channels.length) ? (
              <>
                <Text>Add Additional Channels</Text>

                <table className={Styles.table}>
                  <tbody>
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
                                    channel.mfa_channel_type
                                      .charAt(0)
                                      .toUpperCase() +
                                    channel.mfa_channel_type.substring(1)}
                                </td>
                              </tr>
                            )
                        )}
                  </tbody>
                </table>
              </>
            ) : (
              <Label htmlFor="add">
                Add Additional Channels
                <Text id="add">
                  You have configured the max number of channels. Delete one to
                  create another.
                </Text>
              </Label>
            )}
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
          </div>
        )}
      </div>
    </main>
  );
};

export default MFAConfigure;
