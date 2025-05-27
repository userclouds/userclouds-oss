import React, { useState } from 'react';

import Label from './index';
import TextInput from '../TextInput';

const storybookData = {
  title: 'Components/Label',
  component: Label,
};

export default storybookData;

function Isolated(args: object) {
  return <Label {...args} htmlFor="label-example-id" />;
}

export const Default = Isolated.bind({});
Default.args = {
  htmlFor: 'label-example-id',
  children: 'I’m the text of the label',
  hasError: false,
  disabled: false,
  id: 'label-example-id',
};

function Template(args: object) {
  const [value, setValue] = useState('');

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setValue(event.target.value);
  };

  return (
    <>
      <Label {...args} htmlFor="label-example-id" />
      <TextInput
        placeholder="Placeholder text"
        id="label-example"
        value={value}
        onChange={handleChange}
      />
    </>
  );
}

export const WithInput = Template.bind({});
WithInput.args = {
  htmlFor: 'label-example',
  children: 'I’m the text of the label',
  hasError: false,
  disabled: false,
  id: 'label-example-id',
};
