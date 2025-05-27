import React from 'react';
import clsx from 'clsx';

import {
  IconInformation,
  IconAlert,
  IconCheckCircle,
  IconClose,
} from '../../icons';
import IconButton from '../IconButton';

import styles from './index.module.scss';

export interface ToastProps {
  /** Theme of notification */
  theme?: 'info' | 'alert' | 'success';
  /** Content of notification */
  children: React.ReactNode;
  /** If dismissable */
  isDismissable?: boolean;
  /** Function to execute on dismiss */
  onDismissClick?: React.MouseEventHandler<HTMLButtonElement>;
}

const TOAST_ICONS = {
  // Icon names map to the icon name
  info: <IconInformation className={styles.icon} />,
  alert: <IconAlert className={styles.icon} />,
  success: <IconCheckCircle className={styles.icon} />,
};

/**
 * Notifications that appear in lower right corner.
 */

const Toast: React.FC<React.HTMLProps<HTMLDivElement> & ToastProps> = ({
  theme = 'info',
  children,
  isDismissable = true,
  onDismissClick,
  ...htmlAttributes
}) => {
  const classes = clsx({
    [styles.root]: true,
    [styles[theme]]: theme,
  });

  return (
    <div className={classes} {...htmlAttributes}>
      {TOAST_ICONS[theme]}
      {children}
      {isDismissable && (
        <IconButton
          icon={<IconClose />}
          onClick={onDismissClick}
          size="tiny"
          className={styles.iconClose}
          title="dismiss this notification"
          aria-label="dismiss this notification"
        />
      )}
    </div>
  );
};

export default Toast;
