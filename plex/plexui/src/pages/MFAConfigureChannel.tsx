import React, { useState, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { APIError, Ellipsis, LoginStyles } from '@userclouds/sharedui';
import {
  InlineNotification,
  Label,
  Select,
} from '@userclouds/ui-component-lib';
import API from '../API';
import requestParams from '../models/MFAChannelPageRequest';
import { mungePageParameters } from '../models/PageParametersResponse';
import Styles from './MFA.module.css';
import { MFAChannelsResponse } from '../models/MFAChannelsResponse';

const getSecurityTextPrompt = (channelType: string) => {
  if (channelType === 'email') {
    return 'Enter an email we can send a verification code to.';
  }
  if (channelType === 'authenticator') {
    return 'Select your Authenticator App (Duo, Google, etc.).';
  }
  if (channelType === 'sms') {
    return 'Enter a phone number we can text a verification code to.';
  }
  return '';
};

const MFAConfigureChannel: React.FC = () => {
  const [params, setParams] = useState<Record<string, string>>();
  const [channelTypeID, setChannelTypeID] = useState<string | null>('');
  const [email, setEmail] = useState<string>('');
  const [mfaChannelsResponse, setMFAChannelsResponse] =
    useState<MFAChannelsResponse>();

  const [error, setError] = useState<string>('');

  const location = useLocation();
  const queryParams = new URLSearchParams(location.search);
  const channelType = queryParams.get('channelType');

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
        (e) => {
          setError(e.error);
        }
      );
      API.fetchMFAChannels(sessionID).then((mfaChannels) => {
        if (mfaChannels) {
          setMFAChannelsResponse(mfaChannels);
        }
      });
    }
  }, [sessionID, channelType, channelTypeID]);

  useEffect(() => {
    if (sessionID && mfaChannelsResponse?.mfa_authenticator_types?.length) {
      if (channelType === 'authenticator' && !channelTypeID) {
        setChannelTypeID(
          mfaChannelsResponse.mfa_authenticator_types[0].authenticator_type
        );
      }
    }
  }, [sessionID, mfaChannelsResponse, channelType, channelTypeID]);

  const onSubmit = async () => {
    if (sessionID) {
      setError('');
      if (channelType && channelTypeID) {
        const maybeError = await API.mfaCreateChannel(
          sessionID,
          channelType,
          channelTypeID
        );
        if (maybeError instanceof APIError) {
          const message = maybeError.message
            ? maybeError.message
            : 'Unable to create channel.';
          setError(message);
        } else {
          setError('');
        }
      }
    }
  };

  if (!sessionID) {
    return (
      <InlineNotification theme="alert">
        {missingSessionIDMsg}
      </InlineNotification>
    );
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
          Secure your Account
        </h1>
        {channelType && (
          <p className={Styles.text}>{getSecurityTextPrompt(channelType)}</p>
        )}
        {error !== '' && (
          <InlineNotification theme="alert">{error}</InlineNotification>
        )}

        <>
          <form
            className={Styles.form}
            onSubmit={(e) => {
              e.preventDefault();
              onSubmit();
            }}
          >
            {channelType === 'email' && (
              <>
                <Label className={Styles.formElement} htmlFor="form_mfa">
                  Email
                  <input
                    className={Styles.textInput}
                    name="form_mfa"
                    value={email}
                    required
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      setEmail(e.target.value.trim());
                      setChannelTypeID(e.target.value.trim());
                    }}
                  />
                </Label>
              </>
            )}
            {channelType === 'authenticator' && (
              <>
                <Label className={Styles.formElement} htmlFor="form_mfa">
                  Authenticator
                </Label>

                <Select
                  name="form_mfa"
                  value={channelTypeID || ''}
                  required
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    e.preventDefault();
                    setChannelTypeID(e.target.value);
                  }}
                >
                  {mfaChannelsResponse.mfa_authenticator_types &&
                    mfaChannelsResponse.mfa_authenticator_types.map(
                      (channel) => (
                        <option
                          value={channel.authenticator_type}
                          key={channel.authenticator_type}
                        >
                          {channel.description}
                        </option>
                      )
                    )}
                </Select>
              </>
            )}
            <button
              type="submit"
              className={Styles.formElement + ' ' + Styles.submitButton}
              style={{
                color: `${params.actionButtonTextColor}`,
                borderColor: `${params.actionButtonBorderColor}`,
                backgroundColor: `${params.actionButtonFillColor}`,
              }}
            >
              Continue
            </button>
          </form>
        </>
      </div>
    </main>
  );
};

export default MFAConfigureChannel;
