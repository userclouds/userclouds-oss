import React, { useState, useRef, useEffect } from 'react';
import clsx from 'clsx';

import styles from './index.module.scss';

export interface TabItemProps {
  /** ClassName for overrides if needed, best to avoid */
  className?: string;
  /** Text of the button */
  children: React.ReactNode;
  /** Disabled state */
  disabled?: boolean;
  /** React icon to right of the text */
  iconRight?: React.ReactNode;
  /** React icon to left of the text */
  iconLeft?: React.ReactNode;
  /** Event when clicking button */
  onClick?: React.MouseEventHandler<HTMLButtonElement>;
  /** ID for button */
  id: string;
  /** Whether this tab is active */
  isActive?: boolean;
}

export interface TabGroupProps {
  /** ClassName for overrides if needed, best to avoid */
  className?: string;
  /** Array of tab items */
  items: TabItemProps[];
  /** Default active tab ID */
  defaultActiveTab?: string;
  /** Content to display for each tab */
  tabContent?: Record<string, React.ReactNode>;
  /** Callback when tab changes */
  onTabChange?: (tabId: string) => void;
  /** Spread the button text/icon to far left and right */
  fullWidth?: boolean;
}

const TabItem: React.FC<TabItemProps> = ({
  className,
  children,
  disabled,
  iconRight,
  iconLeft,
  onClick,
  id,
  isActive,
}) => {
  const classes = clsx({
    [styles.item]: true,
    [styles.active]: isActive,
    [styles.disabled]: disabled,
    [className]: className,
  });

  return (
    <button
      className={classes}
      onClick={onClick}
      disabled={disabled}
      type="button"
      id={id}
    >
      {iconLeft && <span className={styles.iconLeft}>{iconLeft}</span>}
      <span className={styles.label}>{children}</span>
      {iconRight && <span className={styles.iconRight}>{iconRight}</span>}
    </button>
  );
};

const TabGroup: React.FC<TabGroupProps> = ({
  className,
  items,
  defaultActiveTab,
  tabContent,
  fullWidth,
  onTabChange,
}) => {
  const [activeTab, setActiveTab] = useState<string>(
    defaultActiveTab || (items.length > 0 ? items[0].id : '')
  );
  const tabsRef = useRef<HTMLDivElement>(null);
  const indicatorRef = useRef<HTMLDivElement>(null);

  const handleTabClick = (tabId: string) => {
    setActiveTab(tabId);
    if (onTabChange) {
      onTabChange(tabId);
    }
  };

  // Update the indicator position when the active tab changes
  useEffect(() => {
    if (tabsRef.current && indicatorRef.current) {
      const tabsContainer = tabsRef.current;
      const activeTabElement = tabsContainer.querySelector(
        `button[id="${activeTab}"]`
      ) as HTMLElement;

      if (activeTabElement) {
        const { left: tabsLeft } = tabsContainer.getBoundingClientRect();
        const { left: activeLeft, width: activeWidth } =
          activeTabElement.getBoundingClientRect();

        // Calculate the position relative to the tabs container
        const indicatorLeft = activeLeft - tabsLeft;

        // Update the indicator position
        const indicator = indicatorRef.current;
        indicator.style.left = `${indicatorLeft}px`;
        indicator.style.width = `${activeWidth}px`;
        indicator.style.opacity = '1';
      }
    }
  }, [activeTab]);

  const classes = clsx({
    [styles.root]: true,
    [className]: className,
    [styles.fullWidth]: fullWidth,
  });

  return (
    <div className={styles.container}>
      <div className={classes} ref={tabsRef}>
        {items.map((item) => (
          <TabItem
            key={item.id}
            {...item}
            isActive={activeTab === item.id}
            onClick={(e) => {
              if (item.onClick) {
                item.onClick(e);
              }
              handleTabClick(item.id);
            }}
          />
        ))}
        <div className={styles.indicator} ref={indicatorRef} />
      </div>
      {tabContent && (
        <div className={styles.tabContent}>{tabContent[activeTab]}</div>
      )}
    </div>
  );
};

export default TabGroup;
