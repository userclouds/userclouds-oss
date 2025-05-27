import { useState, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { Ellipsis, LoginStyles } from '@userclouds/sharedui';
import { Card, ErrorText, SuccessText } from '@userclouds/ui-component-lib';
import API from '../API';
import requestParams from '../models/CreateUserPageRequest';
import { mungePageParameters } from '../models/PageParametersResponse';

const OTPVerifyEmail = () => {
  const [params, setParams] = useState<Record<string, string>>();
  const [email, setEmail] = useState<string>('');
  const [OTP, setOTP] = useState<string>('');
  const [statusText, setStatusText] = useState<string>('');
  const [isError, setIsError] = useState<boolean>(false);

  const location = useLocation();
  const queryParams = new URLSearchParams(location.search);
  const missingPathParamsMessage = `Missing required parameters`;
  const sessionID = queryParams.get('session_id');
  const pathEmail = queryParams.get('email');
  const pathOTP = queryParams.get('otp_code');

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
      if (email && OTP) {
        API.verifyOTPEmail(sessionID, email, OTP).then(
          (response) => {
            setIsError(false);
            setStatusText(response);
          },
          (error) => {
            setIsError(true);
            setStatusText(error.message);
          }
        );
      }
    }
  }, [sessionID, email, OTP]);

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

  if (!sessionID || !pathEmail || !pathOTP) {
    return (
      <Card>
        <ErrorText>{missingPathParamsMessage}</ErrorText>
      </Card>
    );
  }

  if (!email) {
    setEmail(pathEmail);
  }
  if (!OTP) {
    setOTP(pathOTP);
  }
  return (
    <Card title="Email Verification">
      {isError ? (
        <ErrorText>{statusText}</ErrorText>
      ) : (
        <SuccessText>{statusText}</SuccessText>
      )}
    </Card>
  );
};

export default OTPVerifyEmail;
