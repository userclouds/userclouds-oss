import React from 'react';
import clsx from 'clsx';

import { IconClose } from '../../icons';
import IconButton from '../IconButton';

import styles from './index.module.scss';

export interface AppliedFilterProps {
  /** Text of AppliedFilter source */
  source?: string;
  /** Text of AppliedFilter */
  text: string;
  /** If dismissable */
  isDismissable?: boolean;
  /** Function to execute on dismiss */
  onDismissClick?: React.MouseEventHandler<HTMLButtonElement>;
}

/**
 * AppliedFilters with the option to dimiss.
 */

const AppliedFilter: React.FC<
  React.HTMLProps<HTMLDivElement> & AppliedFilterProps
> = ({
  source,
  text,
  isDismissable = false,
  onDismissClick,
  ...htmlAttributes
}) => {
  const classes = clsx({
    [styles.root]: true,
    [styles.isDismissable]: isDismissable,
  });

  return (
    <div className={classes} {...htmlAttributes}>
      {source && <span className={styles.source}>{source}</span>}
      <span className={styles.text}>{text}</span>
      {isDismissable && (
        <IconButton
          icon={<IconClose />}
          onClick={onDismissClick}
          size="tiny"
          className={styles.iconClose}
          title="Remove this AppliedFilter"
          theme="clear"
          aria-label="Remove this AppliedFilter"
        />
      )}
    </div>
  );
};

export default AppliedFilter;
