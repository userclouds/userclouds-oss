import { Story } from '@storybook/react';

import Heading from './index';

const storybookData = {
  title: 'Components/Heading',
  component: Heading,
  argTypes: {
    size: {
      control: 'select',
    },
  },
};

export default storybookData;

const Template: Story<any> = (args) => <Heading {...args} />;

export const Size1 = Template.bind({});
Size1.args = {
  size: '1',
  children: 'I’m a headline size 1',
};

export const Size2 = Template.bind({});
Size2.args = {
  size: '2',
  children: 'I’m a headline size 2',
};

export const Size3 = Template.bind({});
Size3.args = {
  size: '3',
  children: 'I’m a headline size 3',
};
