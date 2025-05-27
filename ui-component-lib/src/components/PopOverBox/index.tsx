import React from 'react';
import styles from './index.module.scss';

export interface PopOverBoxProps extends React.ComponentProps<'div'> {
  /** Content of box */
  children: React.ReactNode;
}

/** Simple wrapper for dropdown menus or other popups. */

const PopOverBox: React.FC<PopOverBoxProps> = ({ children }) => (
  <div className={styles.root}>{children}</div>
);

export default PopOverBox;
