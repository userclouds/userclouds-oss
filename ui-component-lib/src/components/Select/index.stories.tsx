import React, { useState } from 'react';

import Select from './index';

const storybookData = {
  title: 'Components/Select',
  component: Select,
  args: {
    disabled: false,
    hasError: false,
    full: false,
    id: 'id-of-input',
  },
};

export default storybookData;

function Template({ ...args }) {
  const [value, setValue] = useState('ny');

  const handleChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    setValue(event.target.value);
  };

  return (
    <Select value={value} onChange={handleChange} {...args}>
      <option value="ny">New York</option>
      <option value="ca">California</option>
      <option value="tn">Tennessee</option>
      <option value="fl">Florida</option>
    </Select>
  );
}

export const Default = Template.bind({});
Default.args = {
  id: 'states-list',
};
