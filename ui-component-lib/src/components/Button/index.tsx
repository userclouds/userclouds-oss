import React from 'react';
import clsx from 'clsx';

import LoaderDots from '../LoaderDots';

import styles from './index.module.scss';

export interface ButtonProps {
  /** ClassName for overrides if needed, best to avoid */
  className?: string;
  /** Text of the button */
  children: React.ReactNode;
  /** Theme theme */
  theme?: 'primary' | 'dangerous' | 'secondary' | 'outline' | 'ghost';
  /** Size theme */
  size?: 'medium' | 'small' | 'pagination';
  /** 100% width of parent container */
  full?: boolean;
  /** If button should be an anchor tag */
  href?: string;
  /** Disabled state */
  disabled?: boolean;
  /** Loading state */
  isLoading?: boolean;
  /** React icon to right of the text */
  iconRight?: React.ReactNode;
  /** React icon to left of the text */
  iconLeft?: React.ReactNode;
  /** Spread the button text/icon to far left and right */
  spaceBetween?: boolean;
  /** Event when clicking button */
  onClick?: React.MouseEventHandler<HTMLButtonElement>;
  /** Type for button */
  type?: 'button' | 'submit' | 'reset';
  /** ID for button */
  id?: string;

  value?: string;
  title?: string;
}
/**
 * Button component that behave as a `button` or an `a` element when needed but will visually look the same to the user.
 * Allows for icon on right and left of text as well as a loading state. Use small icons with small buttons, medium icons
 * with medium buttons.
 */

const Button: React.FC<ButtonProps> = ({
  className,
  children,
  theme = 'primary',
  size = 'medium',
  full = false,
  href,
  iconRight,
  iconLeft,
  disabled = false,
  isLoading = false,
  spaceBetween = false,
  onClick,
  type = 'button',
  id,
  value,
  title,
}) => {
  const classes = clsx({
    [styles.root]: true,
    [className]: className,
    [styles.full]: full,
    [styles.isLoading]: isLoading,
    [styles.spaceBetween]: spaceBetween,
    [styles[theme]]: theme,
    [styles[size]]: size,
  });

  // If no icon is present just pass the "children" value, otherwise pass in icon HTML and wrapper
  const buttonContent = () => {
    // De
    const loaderTheme =
      theme === 'primary' || theme === 'dangerous' ? 'inverse' : 'brand';

    return (
      <>
        <span className={styles.wrap}>
          {iconLeft && <span className={styles.iconLeft}>{iconLeft}</span>}
          {children && <span className={styles.label}>{children}</span>}
          {iconRight && <span className={styles.iconRight}>{iconRight}</span>}
        </span>
        {isLoading && (
          <span className={styles.loaderDots}>
            <LoaderDots size={size} theme={loaderTheme} />
          </span>
        )}
      </>
    );
  };

  // If "href" is present use `<a>`, otherwise use `<button>`
  const htmlOutput = () => {
    if (href) {
      return (
        <a href={href} className={classes}>
          {buttonContent()}
        </a>
      );
    }
    return (
      <button
        disabled={disabled || isLoading}
        className={classes}
        onClick={onClick}
        type={type || 'button'}
        value={value}
        id={id}
        title={title}
      >
        {buttonContent()}
      </button>
    );
  };

  return htmlOutput();
};

export default Button;
