import React, { useState, useRef, Fragment } from 'react';
import clsx from 'clsx';

import styles from './index.module.scss';

enum SelectActions {
  Close = 0,
  CloseSelect = 1,
  First = 2,
  Last = 3,
  Next = 4,
  Open = 5,
  PageDown = 6,
  PageUp = 7,
  Previous = 8,
  Select = 9,
  Type = 10,
}

const getActionFromKey = (e: React.KeyboardEvent, menuOpen) => {
  const { key, altKey } = e;

  // all keys that will do the default open action
  const openKeys = ['ArrowDown', 'ArrowUp', 'Enter', ' '];

  // handle opening when closed
  if (!menuOpen && openKeys.includes(key)) {
    return SelectActions.Open;
  }

  // home and end move the selected option when open or closed
  if (key === 'Home') {
    return SelectActions.First;
  }
  if (key === 'End') {
    return SelectActions.Last;
  }

  // handle keys when open
  if (menuOpen) {
    if (key === 'ArrowUp' && altKey) {
      return SelectActions.CloseSelect;
    }
    if (key === 'ArrowDown' && !altKey) {
      return SelectActions.Next;
    }
    if (key === 'ArrowUp') {
      return SelectActions.Previous;
    }
    if (key === 'PageUp') {
      return SelectActions.PageUp;
    }
    if (key === 'PageDown') {
      return SelectActions.PageDown;
    }
    if (key === 'Escape') {
      return SelectActions.Close;
    }
    if (key === 'Enter' || key === ' ') {
      return SelectActions.CloseSelect;
    }
  }
  return undefined;
};

const handleKeyPress =
  (
    isOpen: boolean,
    focusedIndex: number,
    options: { label: string; value: string }[],
    actions: React.ReactElement[] | undefined,
    setFocusedIndex: (i: number) => void,
    setIsOpen: (isOpen: boolean) => void,
    changeHandler: (val: string) => void
  ) =>
  (e: React.KeyboardEvent) => {
    let max: number = options.length - 1;
    if (actions?.length) {
      max += actions.length;
    }
    const action: number | undefined = getActionFromKey(e, isOpen);

    switch (action) {
      case SelectActions.Last:
        setFocusedIndex(max);
        break;
      case SelectActions.First:
        setFocusedIndex(0);
        break;
      case SelectActions.Next:
        e.preventDefault();
        setFocusedIndex(Math.min(max, focusedIndex + 1));
        break;
      case SelectActions.Previous:
        e.preventDefault();
        setFocusedIndex(Math.max(0, focusedIndex - 1));
        break;
      case SelectActions.PageUp:
        e.preventDefault();
        setFocusedIndex(Math.max(0, focusedIndex - options.length));
        break;
      case SelectActions.PageDown:
        e.preventDefault();
        setFocusedIndex(Math.min(max, focusedIndex + options.length));
        break;
      case SelectActions.CloseSelect: {
        e.preventDefault();
        if (focusedIndex > -1) {
          // if this is an option, not an action
          // if it's an action, let's just let the browser
          // handle
          if (focusedIndex <= options.length - 1) {
            const newVal = options[focusedIndex].value;
            changeHandler(newVal);
          } else {
            const focusedAction = e.currentTarget.querySelector(
              '[class*="has-focus"]'
            );
            if (focusedAction?.firstChild instanceof HTMLElement) {
              focusedAction.firstChild.click();
            }
          }
          setFocusedIndex(-1);
        }
        setIsOpen(false);
        break;
      }
      case SelectActions.Close:
        e.preventDefault();
        setFocusedIndex(0);
        setIsOpen(false);
        break;
      case SelectActions.Open:
        e.preventDefault();
        setIsOpen(true);
        break;
      default:
        break;
    }
  };

export interface PseudoSelectProps extends React.ComponentProps<'ol'> {
  options: { label: string; value: string }[];
  actions: React.ReactElement[] | undefined;
  labeledBy: string; // we expect label to be created outside this component
  disabled?: boolean;
  value?: string;
  changeHandler: (val: string) => void;
}

export const PseudoSelect = ({
  options,
  actions,
  id,
  labeledBy,
  disabled = false,
  value = '',
  changeHandler,
  className = '',
  ...args
}: PseudoSelectProps) => {
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [focusedIndex, setFocusedIndex] = useState<number>(-1);
  const selfRef = useRef(null);
  const labelEl = document.getElementById(labeledBy);
  if (labelEl) {
    labelEl.addEventListener('click', (e: Event) => {
      e.preventDefault();

      selfRef.current.focus();
    });
  }

  const classes = clsx(
    {
      [styles.pseudoSelect]: true,
    },
    className
  );

  const selectedEl = options.find(({ value: val }) => val === value);
  let focusedEl;
  let focusedElID = null;
  if (focusedIndex && focusedIndex > options.length - 1) {
    focusedElID = `${id}-action-${focusedIndex - options.length}`;
  } else if (focusedIndex && focusedIndex > -1) {
    focusedEl = options[focusedIndex];
    focusedElID = `${id}-${focusedEl.value}`;
  }

  /* eslint-disable jsx-a11y/click-events-have-key-events */
  return (
    <ol
      id={id}
      role="listbox"
      tabIndex={0}
      aria-labelledby={labeledBy}
      aria-expanded={isOpen}
      aria-disabled={disabled}
      aria-activedescendant={focusedElID}
      onClick={() => {
        if (!disabled && !isOpen) {
          setIsOpen(true);
        }
      }}
      onKeyDown={handleKeyPress(
        isOpen,
        focusedIndex,
        options,
        actions,
        setFocusedIndex,
        setIsOpen,
        changeHandler
      )}
      onBlur={(e) => {
        // don't let blur prevent click handlers inside actions from triggering
        if (isOpen && !e.currentTarget.contains(e.relatedTarget)) {
          setIsOpen(false);
        }
      }}
      ref={selfRef}
      className={classes}
      {...args}
    >
      <li className={styles.currentSelection}>{selectedEl?.label}</li>
      <ol aria-hidden={!isOpen}>
        {options.map(({ label, value: val }, i: number) => (
          <li
            key={val}
            id={`${id}-${val}`}
            role="option"
            aria-selected={value === val}
            data-value={val}
            className={focusedIndex === i ? styles['has-focus'] : ''}
            onClick={(e: React.MouseEvent<HTMLLIElement>) => {
              e.preventDefault();

              // we need this check so that clicking to open the dropdown
              // doesn't select the first item
              if (isOpen) {
                changeHandler(val);
                setIsOpen(false);
              }
            }}
          >
            {label}
          </li>
        ))}
        {actions?.length > 0 &&
          actions.map((action: React.ReactElement, i: number) => (
            <Fragment key={action.props.title || `action_${i}`}>
              <hr />
              <li
                role="option"
                aria-selected={false /* these aren't selectable */}
                id={`${id}-action-${i}`}
                className={clsx(
                  styles['listbox-action'],
                  focusedIndex - options.length === i ? styles['has-focus'] : ''
                )}
              >
                {action}
              </li>
            </Fragment>
          ))}
      </ol>
    </ol>
  );
  /* eslint-enable jsx-a11y/click-events-have-key-events */
};

export interface SelectProps extends React.ComponentProps<'select'> {
  disabled?: boolean;
  /** If has error */
  hasError?: boolean;
  /** Full width input */
  full?: boolean;
  /** Size theme */
  themeSize?: 'medium' | 'small';
  /** Char width (determines width of the select based on character count) */
  charWidth?: number;
}

/** Dropdown menu for mutually exclusive options. */

const Select: React.FC<SelectProps> = ({
  children,
  hasError = false,
  disabled = false,
  full = false,
  themeSize = 'medium',
  charWidth,
  className,
  ...args
}) => {
  // Calculate width based on the number of characters (using 'em' for dynamic font scaling)
  const widthStyle = charWidth ? { width: `${charWidth}ch` } : {};

  return (
    <div
      className={clsx(
        {
          [styles.root]: true,
          [styles.full]: full,
          [styles.hasError]: hasError,
          [styles.disabled]: disabled,
          [styles[themeSize]]: themeSize,
        },
        className
      )}
    >
      <select
        className={styles.select}
        disabled={disabled}
        style={widthStyle} // Dynamically apply the width based on charWidth
        {...args}
      >
        {children}
      </select>

      {!args.multiple && (
        <svg
          viewBox="0 0 24 24"
          width="16"
          height="16"
          className={styles.caret}
          stroke="currentColor"
          strokeWidth="2"
          fill="none"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <polyline points="6 9 12 15 18 9" />
        </svg>
      )}
    </div>
  );
};

export default Select;
