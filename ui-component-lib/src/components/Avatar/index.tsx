import React from 'react';
import clsx from 'clsx';

import styles from './index.module.scss';

import DefaultUserAvatar from './default_user-avatar.svg';

export interface UserAvatarProps {
  /** Source URL of the UserAvatar image */
  src?: string;
  /** Alt tag for the image */
  fullName?: string;
  /** Size of Avatar */
  size?: 'medium' | 'small';
}

/**
 * User images with fallback if no `src` is present.
 */

export const UserAvatar: React.FC<UserAvatarProps> = ({
  src,
  fullName,
  size = 'medium',
}) => {
  const imageSource: string = src || DefaultUserAvatar;

  const classes = clsx({
    [styles.picture]: true,
    [styles[size]]: size,
  });

  return (
    <picture className={classes}>
      <img
        className={styles.img}
        src={imageSource}
        alt={fullName && `Avatar for ${fullName}`}
      />
    </picture>
  );
};

interface EntityAvatarProps {
  /** Single initial */
  initial: string;
  /** Background color */
  bgColor?: string;
  /** Size of Avatar */
  size?: 'medium' | 'small';
}

/**
 * User images with fallback if no `src` is present. Currently only one size.
 */

export const EntityAvatar: React.FC<EntityAvatarProps> = ({
  initial,
  bgColor,
  size = 'medium',
}) => {
  const classes = clsx({
    [styles.initialWrap]: true,
    [styles[size]]: size,
  });

  return (
    <div className={classes} style={{ backgroundColor: bgColor }}>
      <span className={styles.initial}>{initial}</span>
    </div>
  );
};
