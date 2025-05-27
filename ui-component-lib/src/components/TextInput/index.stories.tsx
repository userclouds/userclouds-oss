import React, { useState, useRef } from 'react';

import TextInput from './index';
import IconButton from '../IconButton';
import { IconClose, IconSearch } from '../../icons';

const storybookData = {
  title: 'Components/TextInput',
  component: TextInput,
};

export default storybookData;

function DefaultTemplate(args: object) {
  const [value, setValue] = useState('');

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setValue(event.target.value);
  };

  return <TextInput {...args} value={value} onChange={handleChange} />;
}

export const Default = DefaultTemplate.bind({});
Default.args = {
  value: '',
  disabled: false,
  hasError: false,
  placeholder: 'Placeholder text',
};

function IconTemplate(args: object) {
  const [value, setValue] = useState('');

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setValue(event.target.value);
  };

  const inputEl = useRef(null);
  const handleIconClick = () => {
    setValue('');
    inputEl.current.focus();
  };

  return (
    <TextInput
      {...args}
      value={value}
      onChange={handleChange}
      ref={inputEl}
      innerLeft={<IconSearch />}
      innerRight={
        <IconButton
          theme="clear"
          icon={<IconClose />}
          onClick={handleIconClick}
          title="Close"
          aria-label="Close"
        />
      }
    />
  );
}

export const Icons = IconTemplate.bind({});
Icons.args = {
  value: '',
  disabled: false,
  hasError: false,
  placeholder: 'Placeholder text',
};
