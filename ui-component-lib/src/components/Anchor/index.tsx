import React from 'react';
import clsx from 'clsx';

import styles from './index.module.scss';

export interface AnchorProps {
  /** className for overrides if needed, best to avoid */
  className?: string;
  /** Content of the Anchor */
  children: string | React.ReactNode;
  /** URL of Anchor, if exists */
  href?: string;
  /** Hide default underline */
  noUnderline?: boolean;
  /** React icon to right of the text */
  iconRight?: React.ReactNode;
  /** React icon to left of the text */
  iconLeft?: React.ReactNode;
  /** onClick action, use only if no `href` is present */
  onClick?: React.MouseEventHandler<HTMLButtonElement>;
}

/**
 * Commonly used as an `a` tag but without an `href` prop it will render as `button`.
 */

const Anchor: React.FC<AnchorProps> = ({
  className,
  children,
  href,
  noUnderline,
  iconRight,
  iconLeft,
  onClick,
}) => {
  const commonClasses = clsx({
    [styles.root]: true,
    [styles.noUnderline]: noUnderline,
    [className]: className,
  });

  const anchorClasses = clsx(commonClasses, {
    [styles.anchor]: true,
  });

  const buttonClasses = clsx(commonClasses, {
    [styles.button]: true,
  });

  // If no icon is present just pass the "children" value, otherwise pass in icon HTML and wrapper
  const anchorContent = () => (
    <span className={styles.wrap}>
      {iconLeft && <span className={styles.iconLeft}>{iconLeft}</span>}
      {children && <span className={styles.label}>{children}</span>}
      {iconRight && <span className={styles.iconRight}>{iconRight}</span>}
    </span>
  );

  // If "href" is present use `<a>`, otherwise use `<button>`
  const htmlOutput = (): React.ReactElement => {
    if (href) {
      return (
        <a className={anchorClasses} href={href}>
          {anchorContent()}
        </a>
      );
    }
    return (
      <button className={buttonClasses} onClick={onClick} type="button">
        {anchorContent()}
      </button>
    );
  };

  return htmlOutput();
};

export default Anchor;
