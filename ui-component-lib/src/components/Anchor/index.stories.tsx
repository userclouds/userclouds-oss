import { Story } from '@storybook/react';

import Anchor from './index';

const storybookData = {
  title: 'Components/Anchor',
  component: Anchor,
};

export default storybookData;

const Template: Story<any> = (args) => (
  <div>
    This is a <Anchor {...args} /> and more text.
  </div>
);

export const Default = Template.bind({});
Default.args = {
  href: 'https://userclouds.com',
  children: 'link',
};

export const Button = Template.bind({});
Button.args = {
  children: 'link',
  onClick: () => alert('I’ve been clicked'),
};
