import { connect } from 'react-redux';
import { useState, useEffect } from 'react';

import {
  Heading,
  IconBookOpen,
  IconFileCode,
  IconNews,
  LoaderDots,
} from '@userclouds/ui-component-lib';

import { RootState } from '../store';
import { MyProfile } from '../models/UserProfile';

import Styles from './OnboardingPage.module.css';
import PageCommon from './PageCommon.module.css';

const OnboardingPage = ({
  myProfile,
}: {
  myProfile: MyProfile | undefined;
}) => {
  const [loadingMessage, setLoadingMessage] = useState(
    'Please wait while we provision your tenant'
  );
  const [messageOpacity, setMessageOpacity] = useState(1);
  const [elapsedTime, setElapsedTime] = useState(0);

  // Change message with simple fade effect
  const changeMessageWithFade = (newMessage: string) => {
    setMessageOpacity(0);

    setTimeout(() => {
      setLoadingMessage(newMessage);
      setMessageOpacity(1);
    }, 500);
  };

  useEffect(() => {
    const timer = setInterval(() => {
      setElapsedTime((prevTime) => prevTime + 1);
    }, 1000);

    return () => clearInterval(timer);
  }, []);

  useEffect(() => {
    if (elapsedTime === 10) {
      changeMessageWithFade('This can take up to a minute');
    } else if (elapsedTime === 20) {
      changeMessageWithFade("We're setting up your environment");
    } else if (elapsedTime === 30) {
      changeMessageWithFade('Almost there, finalizing your tenant setup');
    }
  }, [elapsedTime]);

  return (
    <div className={Styles.onboardingPageContent}>
      <section className={Styles.section}>
        <header>
          <h1>
            Welcome to <em>UserClouds{myProfile ? ', ' : ''}</em>
            {myProfile ? myProfile.userProfile.name() : ''}!
          </h1>
        </header>
        <main>
          <LoaderDots size="medium" assistiveText="Loading…" />
          <Heading
            size="3"
            headingLevel="2"
            className={Styles.loadingText}
            style={{ opacity: messageOpacity }}
          >
            {loadingMessage}
          </Heading>
        </main>
        <footer>
          <h3>Resources</h3>
          <ul className={Styles.resourceList}>
            <li>
              <IconBookOpen />
              <a
                href="https://docs.userclouds.com/docs/"
                title="Open documentation site in a new tab"
                target="_blank"
                rel="noreferrer"
                className={PageCommon.link}
              >
                Documentation
              </a>
            </li>
            <li>
              <IconFileCode />
              <a
                href="https://docs.userclouds.com/reference"
                title="Open API reference in a new tab"
                target="_blank"
                rel="noreferrer"
                className={PageCommon.link}
              >
                API reference
              </a>
            </li>
            <li>
              <IconNews />
              <a
                href="https://www.userclouds.com/blog"
                title="Visit the UserClouds blog (opens in a new tab)"
                target="_blank"
                rel="noreferrer"
                className={PageCommon.link}
              >
                Blog
              </a>
            </li>
          </ul>
        </footer>
      </section>
    </div>
  );
};

export default connect((state: RootState) => {
  return {
    myProfile: state.myProfile,
    companyID: state.selectedCompanyID,
    selectedTenant: state.selectedTenant,
    fetchingTenants: state.fetchingTenants,
    fetchingSelectedTenant: state.fetchingSelectedTenant,
  };
})(OnboardingPage);
