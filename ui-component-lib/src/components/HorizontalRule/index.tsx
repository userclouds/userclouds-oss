import React from 'react';
import clsx from 'clsx';

import styles from './index.module.scss';

type HorizontalRuleProps = React.ComponentProps<'hr'>;

/** Divides sections within Cards. Used to help visual separation. */

const HorizontalRule: React.FC<HorizontalRuleProps> = ({
  className,
  ...args
}) => {
  const classes = clsx({
    [styles.root]: true,
    [className]: className,
  });

  return <hr className={classes} {...args} />;
};

export default HorizontalRule;
