import React, { useState, useMemo } from 'react';
import clsx from 'clsx';
import styles from './index.module.scss';

import IconButton from '../IconButton';
import { IconEye, IconEyeOff, IconLock2 } from '../../icons';

import Text from '../Text';

export interface InputReadOnlyProps {
  id?: string;
  /** Input components to render in a vertical space. */
  children?: React.ReactNode;
  /** If item is a checkox and is checked. */
  isChecked?: boolean;
  /** If text should be monospace. */
  monospace?: boolean;
  /** Show lock icon for items that cannot be edited. */
  isLocked?: boolean;
  /** Prevent wrapping of text, useful in tables so they don't collapse. */
  noWrap?: boolean;
  /** Style of read-only input to display, defaults to text. */
  type?: 'checkbox' | 'text';
  /** Title attribute for displaying a tooltip * */
  title?: string;
  /** Use sparingly * */
  className?: string;
}

/** Text for read-only states of selects, radio, checkbox, and text inputs. */

export default function InputReadOnly({
  id,
  monospace = false,
  type = 'text',
  children = null,
  isChecked = false,
  isLocked = false,
  noWrap = false,
  title = null,
  className = '',
}: InputReadOnlyProps): JSX.Element {
  return (
    <div className={clsx(styles.root, className)} title={title}>
      {type === 'checkbox' && (
        <div>
          {isChecked ? (
            <svg
              width="15"
              height="15"
              viewBox="0 0 15 11"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <title>On</title>
              <path
                d="M6.15099 8.19086L13.811 0.530029L14.9902 1.70836L6.15099 10.5475L0.847656 5.2442L2.02599 4.06586L6.15099 8.19086Z"
                fill="#199C16"
              />
            </svg>
          ) : (
            <svg
              width="15"
              height="15"
              viewBox="0 0 15 3"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <title>Off</title>
              <path
                d="M14.1669 0.666793V2.33321L0.833208 2.33321V0.666792L14.1669 0.666793Z"
                fill="#283354"
              />
            </svg>
          )}
        </div>
      )}
      {isLocked && <IconLock2 size="small" className={styles.icon} />}
      <Text size={1} monospace={monospace} noWrap={noWrap} id={id}>
        {children}
      </Text>
    </div>
  );
}

const fudgeStringLength = (len: number): number => {
  // generate a whole number from 3 to 5 inclusive
  const fudgeFactor: number = Math.floor(Math.random() * 3 + 3);
  // whether to add or subtract the fudge factor
  // always add if string is short (<= 8)
  const addToString: boolean = len <= 8 || Math.random() >= 0.5;

  return addToString ? len + fudgeFactor : len - fudgeFactor;
};
const maskString = (str: string): string => {
  const newLength = fudgeStringLength(str.length);
  // u2006 is a 1/6em space
  return new Array(newLength).fill('*').join('\u2006');
};

export interface InputReadOnlyHiddenProps {
  /** Text to render inside component */
  value: string;
  /** If text should be monospace. */
  monospace?: boolean;
  /** Show lock icon for items that cannot be edited. */
  isLocked?: boolean;
  /** Prevent wrapping of text, useful in tables so they don't collapse. */
  noWrap?: boolean;
  /** Determines whether visibility can be set to visible. */
  canBeShown?: boolean;
  /** Use sparingly * */
  className?: string;
}
export const InputReadOnlyHidden = ({
  value = '',
  monospace = false,
  isLocked = false,
  noWrap = false,
  canBeShown = true,
  className = '',
}: InputReadOnlyHiddenProps) => {
  const [isHidden, setIsHidden] = useState<boolean>(true);
  const masked = useMemo<string>(() => maskString(value), [value]);
  return (
    <div className={clsx(styles.root, className)}>
      {isLocked && <IconLock2 size="small" className={styles.icon} />}
      <Text size={1} monospace={monospace} noWrap={noWrap}>
        {isHidden ? masked : value}
      </Text>
      {canBeShown && (
        <IconButton
          theme="clear"
          icon={isHidden ? <IconEye /> : <IconEyeOff />}
          title="Toggle text visibility"
          onClick={() => setIsHidden(!isHidden)}
          aria-label="Toggle text visibility"
        />
      )}
    </div>
  );
};
