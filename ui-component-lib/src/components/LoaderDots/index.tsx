import clsx from 'clsx';
import styles from './index.module.scss';

export interface LoaderDotsProps {
  /** Text that describes the current status and is only visible to screenreaders. */
  assistiveText?: string;
  /** Controls the size of the dots. */
  size?: 'pagination' | 'small' | 'medium';
  /** Changes the dot colors. */
  theme?: 'brand' | 'inverse' | 'muted';
}

export default function LoaderDots({
  assistiveText = 'Loading',
  size = 'medium',
  theme = 'brand',
}: LoaderDotsProps): JSX.Element {
  const rootClasses = clsx({
    [styles.root]: true,
    [styles.pagination]: size === 'pagination',
    [styles.small]: size === 'small',
    [styles.medium]: size === 'medium',
  });

  const dotClasses = clsx({
    [styles.dot]: true,
    [styles.dotThemeBrand]: theme === 'brand',
    [styles.dotThemeInverse]: theme === 'inverse',
    [styles.dotThemeMuted]: theme === 'muted',
  });

  return (
    <ul className={rootClasses} role="status">
      <li className={dotClasses} />
      <li className={dotClasses} />
      <li className={dotClasses} />
      <li className={styles.hiddenText}>{assistiveText}</li>
    </ul>
  );
}
