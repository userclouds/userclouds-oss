import React from 'react';
import clsx from 'clsx';

import { IconClose } from '../../icons';
import IconButton from '../IconButton';

import styles from './index.module.scss';

export interface ToastSolidProps {
  /** Theme of notification */
  theme?: 'info' | 'alert' | 'success';
  /** Content of notification */
  children: React.ReactNode;
  /** If dismissable */
  isDismissable?: boolean;
  /** Function to execute on dismiss */
  onDismissClick?: React.MouseEventHandler<HTMLButtonElement>;
}

/**
 * Notifications that appear at center bottom.
 */

const ToastSolid: React.FC<
  React.HTMLProps<HTMLDivElement> & ToastSolidProps
> = ({
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

export default ToastSolid;
