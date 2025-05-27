import { useArgs } from '@storybook/client-api';

import Checkbox from './index';

const storybookData = {
  title: 'Components/Checkbox',
  component: Checkbox,
  args: {
    checked: false,
    disabled: false,
    id: 'id-of-input',
    children:
      'This is the text of the checkbox and if it runs long it will wrap onto next line.',
  },
};

export default storybookData;

export function Default({ ...args }) {
  const [{ checked }, updateArgs] = useArgs();

  return (
    <Checkbox {...args} onChange={() => updateArgs({ checked: !checked })} />
  );
}
