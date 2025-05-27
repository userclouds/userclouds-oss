import clsx from 'clsx';
import React from 'react';

import styles from './index.module.scss';

export interface IconButtonProps {
  /** ClassName for overrides if needed, best to avoid */
  className?: string;
  /** If button should be an anchor tag */
  href?: string;
  /** If Href should not follow the link */
  hrefOnClick?: React.MouseEventHandler<HTMLAnchorElement>;
  /** React icon */
  icon: React.ReactNode;
  /** Theme theme */
  theme?: 'ghost' | 'clear';
  /** Size theme */
  size?: 'medium' | 'small' | 'tiny';
  /** If is disabled */
  disabled?: boolean;
  /** onClick action */
  onClick?: React.MouseEventHandler<HTMLButtonElement>;
  /** Alternate Text */
  title: string;
  /** ID for button */
  id?: string;
  /** Aria label for button */
  ariaLabel?: string;
}

/**
 * Button used exclusively with single icons. This behaves similarly to
 * the `<Button>` component in that if the `href` prop is present the underlying
 * HTML will be an `<a>` tag, otherwise `<button>`.
 */

const IconButton: React.FC<IconButtonProps> = ({
  className,
  href,
  hrefOnClick,
  icon,
  size = 'medium',
  disabled = false,
  theme = 'ghost',
  onClick,
  title,
  id,
  ariaLabel,
}) => {
  const classes = clsx({
    [styles.root]: true,
    [styles[size]]: size,
    [styles[theme]]: theme,
    [className]: className,
  });

  // If "href" is present use `<a>`, otherwise use `<button>`
  const htmlOutput = (): React.ReactElement => {
    if (href) {
      if (hrefOnClick) {
        return (
          <a className={classes} href={href} onClick={hrefOnClick}>
            {icon}
          </a>
        );
      }
      return (
        <a className={classes} href={href}>
          {icon}
        </a>
      );
    }
    return (
      <button
        className={classes}
        disabled={disabled}
        onClick={onClick}
        type="button"
        title={title}
        aria-label={ariaLabel}
        id={id}
      >
        {icon}
      </button>
    );
  };

  return htmlOutput();
};

export default IconButton;
