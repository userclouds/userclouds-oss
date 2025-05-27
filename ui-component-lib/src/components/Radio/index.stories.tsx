import React from 'react';
import { useArgs } from '@storybook/client-api';

import Radio from './index';
import FormNote from '../FormNote';
import InputGroup from '../../layouts/InputGroup';

const storybookData = {
  title: 'Components/Radio',
  component: Radio,
  args: {
    checked: false,
    disabled: false,
    id: 'id-of-input',
    children:
      'This is the text of the radio and if it runs long it will wrap onto next line.',
  },
};

export default storybookData;

export function Default({ ...args }) {
  const [{ isChecked }, updateArgs] = useArgs();

  return (
    <Radio {...args} onChange={() => updateArgs({ isChecked: !isChecked })} />
  );
}

export function Group() {
  const [selectedId, setSelectedId] = React.useState('a');

  return (
    <InputGroup>
      <Radio
        id="a"
        checked={selectedId === 'a'}
        name="radio-example"
        onChange={() => {
          setSelectedId('a');
        }}
      >
        Option A
      </Radio>
      <Radio
        id="b"
        checked={selectedId === 'b'}
        name="radio-example"
        onChange={() => setSelectedId('b')}
      >
        <div>Option B that goes there</div>
        <FormNote>A FormNote about option C.</FormNote>
      </Radio>
      <Radio
        id="c"
        checked={selectedId === 'c'}
        disabled
        name="radio-example"
        onChange={() => setSelectedId('c')}
      >
        Option C
      </Radio>
    </InputGroup>
  );
}
