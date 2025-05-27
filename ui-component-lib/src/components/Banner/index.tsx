import React from 'react';
import clsx from 'clsx';

import { IconInformation, IconAlert, IconClose } from '../../icons';
import IconButton from '../IconButton';

import styles from './index.module.scss';

export interface BannerProps {
  /** Theme of notification */
  theme?: 'info' | 'alert';
  /** Content of notification */
  children: React.ReactNode;
  /** If dismissable */
  isDismissable?: boolean;
  /** Function to execute on dismiss */
  onDismissClick?: React.MouseEventHandler<HTMLButtonElement>;
}

const BANNER_ICONS = {
  // Icon names map to the icon name
  info: <IconInformation className={styles.icon} />,
  alert: <IconAlert className={styles.icon} />,
};

/**
 * Notification to be used inside cards.
 */

const Banner: React.FC<BannerProps> = ({
  theme = 'info',
  children,
  isDismissable = true,
  onDismissClick,
}) => {
  const classes = clsx({
    [styles.root]: true,
    [styles[theme]]: theme,
  });

  return (
    <div className={classes}>
      {BANNER_ICONS[theme]}
      {children}
      {isDismissable && (
        <IconButton
          icon={<IconClose />}
          onClick={onDismissClick}
          size="tiny"
          className={styles.iconClose}
          theme="clear"
          title="dismiss"
          aria-label="dismiss"
        />
      )}
    </div>
  );
};

export default Banner;
