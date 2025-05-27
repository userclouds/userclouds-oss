import React, { useState, useRef, useEffect } from 'react';
import clsx from 'clsx';

import { Dropdown } from '../../layouts/Dropdown';
import { IconInformation } from '../../icons';
import IconButton from '../IconButton';
import Styles from './index.module.scss';

export interface ToolTipProps {
  children?: JSX.Element | JSX.Element[];
  direction?: 'left' | 'right';
  className?: string;
}

const ToolTip: React.FC<ToolTipProps> = ({
  children,
  direction = 'left',
  className,
}) => {
  const [dropDownActive, setDropDownActive] = useState(false);
  const toolTipRef = useRef<HTMLDivElement>(null);
  const toggleDropDown = () => setDropDownActive(!dropDownActive);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        toolTipRef.current &&
        !toolTipRef.current.contains(event.target as Node)
      ) {
        setDropDownActive(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, []);

  const classes = clsx({
    [Styles.tooltip]: true,
    [className]: className,
  });

  return (
    <div ref={toolTipRef} className={classes}>
      <IconButton
        onClick={(e) => {
          e.stopPropagation();
          toggleDropDown();
        }}
        icon={<IconInformation size="small" />}
        title="information"
        size="small"
        aria-label="information"
      />
      {dropDownActive && (
        <Dropdown direction={direction} onClick={(e) => e.stopPropagation()}>
          {children}
        </Dropdown>
      )}
    </div>
  );
};

export default ToolTip;
