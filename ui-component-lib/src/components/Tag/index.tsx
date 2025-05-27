import React from 'react';
import clsx from 'clsx';

import { IconClose } from '../../icons';
import IconButton from '../IconButton';

import styles from './index.module.scss';

export interface TagProps {
  /** Content of tag */
  tag: string;
  /** If dismissable */
  isRemovable?: boolean;
  /** Function to execute on dismiss */
  onDismissClick?: React.MouseEventHandler<HTMLButtonElement>;
}

/**
 * Tags with the option to dismiss.
 */

const Tag: React.FC<React.HTMLProps<HTMLDivElement> & TagProps> = ({
  tag,
  isRemovable = false,
  onDismissClick,
  className,
  ...htmlAttributes
}) => {
  const classes = clsx(
    {
      [styles.root]: true,
      [styles.isDismissable]: isRemovable,
    },
    className
  );

  return (
    <div className={classes} {...htmlAttributes} key={tag}>
      {tag}
      {isRemovable && (
        <IconButton
          title="remove"
          icon={<IconClose />}
          onClick={onDismissClick}
          size="tiny"
          className={styles.iconClose}
          theme="clear"
          aria-label="remove"
        />
      )}
    </div>
  );
};

export default Tag;
