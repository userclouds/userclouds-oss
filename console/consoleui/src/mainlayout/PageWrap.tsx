import React from 'react';
import { connect } from 'react-redux';

import {
  Heading,
  Text,
  ToastSolid,
  PageWrapStyles as styles,
  PageTitleStyles,
} from '@userclouds/ui-component-lib';

import Header from './Header';
import Footer from './Footer';
import SideBar from './SideBar';
import { RootState, AppDispatch } from '../store';
import Notification from '../models/Notification';
import { removePostedNotification } from '../actions/notifications';
import Styles from './PageWrap.module.css';
import { MyProfile } from '../models/UserProfile';

interface PageTitleProps {
  title: string;
  description?: string | JSX.Element;
  itemName?: string;
  id?: string;
}

const ConnectedPageTitle: React.FC<PageTitleProps> = ({
  title,
  description,
  itemName,
  id,
}) => {
  return (
    <div className={Styles.title} id={id}>
      <Heading size={2} headingLevel={1}>
        {title}
        {itemName ? (
          <>
            {': '}
            <b>{itemName}</b>
          </>
        ) : (
          ''
        )}
      </Heading>
      {description && (
        <Text className={PageTitleStyles.description}>{description}</Text>
      )}
    </div>
  );
};

export const PageTitle = connect((state: RootState) => ({
  featureFlags: state.featureFlags,
}))(ConnectedPageTitle);

interface PageWrapProps {
  children: React.ReactNode;
  notifications: Notification[];
  dispatch: AppDispatch;
  myProfile?: MyProfile;
}

export const PageWrap = ({
  children,
  notifications,
  dispatch,
  myProfile,
}: PageWrapProps) => {
  // Only show the sidebar when a user is logged in
  const isAuthenticated = !!myProfile;

  return (
    <div className={Styles.root}>
      <Header />

      <div className={styles.inner}>
        {isAuthenticated && <SideBar isOpen />}
        <main className={Styles.main} id="pageContent">
          {notifications.length > 0 && (
            <ol id="notificationCenter" className={styles.toastNotifications}>
              {notifications.map((notification: Notification) => (
                <li className={styles.animate} key={notification.message}>
                  <ToastSolid
                    theme={notification.type as string}
                    isDismissable
                    onDismissClick={() => {
                      dispatch(removePostedNotification(notification.id));
                    }}
                  >
                    {notification.message}
                  </ToastSolid>
                </li>
              ))}
            </ol>
          )}
          <div className={styles.content}>
            <div className={Styles.contentInner}>{children}</div>
            <div className={Styles.footer}>
              <Footer />
            </div>
          </div>
        </main>
      </div>
    </div>
  );
};

export default connect((state: RootState) => ({
  notifications: state.notifications,
  myProfile: state.myProfile,
}))(PageWrap);
