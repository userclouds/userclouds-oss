import React from 'react';
import clsx from 'clsx';

import { IconInformation, IconAlert, IconCheckCircle } from '../../icons';

import styles from './index.module.scss';

export interface InlineNotificationProps extends React.ComponentProps<'p'> {
  /** Theme of notification */
  theme?: 'info' | 'alert' | 'success';
  /** HTML tag to use to wrap children * */
  elementName?: keyof React.ReactHTML;
  /** Content of notification */
  children: React.ReactNode;
}

const NOTIFICATION_ICONS = {
  // Icon names map to the icon name
  info: <IconInformation className={styles.icon} />,
  alert: <IconAlert className={styles.icon} />,
  success: <IconCheckCircle className={styles.icon} />,
};

/**
 * Notification to be used inside cards.
 */

const InlineNotification: React.FC<InlineNotificationProps> = ({
  theme = 'info',
  elementName = 'p',
  children,
  ...otherProps
}) => {
  const classes = clsx({
    [styles.root]: true,
    [styles[theme]]: theme,
  });

  return (
    <div className={clsx(classes, otherProps.className, `${theme}-message`)}>
      {NOTIFICATION_ICONS[theme]}
      {React.createElement(elementName, {}, children)}
    </div>
  );
};

export default InlineNotification;
