import React from 'react';
import clsx from 'clsx';

import styles from './index.module.scss';

import { IconArrowRight } from '../../icons';

export interface BreadcrumbsProps {
  /** Text and URL of each crumb */
  links?: JSX.Element[];
  /** Overriding classes */
  className?: string;
}

/** Takes in array of names and urls and outputs line of links. */

const Breadcrumbs: React.FC<BreadcrumbsProps> = ({ links, className }) => {
  const classes = clsx({
    [styles.root]: true,
    [className]: className,
  });

  return (
    <div className={classes}>
      {links &&
        links.map((link, index) => (
          <div className={styles.crumb} key={`breadcrumb-${link.key}`}>
            <>
              {index !== 0 && (
                <IconArrowRight size="small" className={styles.icon} />
              )}
              {link}
            </>
          </div>
        ))}
    </div>
  );
};

export default Breadcrumbs;
