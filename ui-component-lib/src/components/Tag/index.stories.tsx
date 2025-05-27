import { Story } from '@storybook/react';
import Tag from './index';

const storybookData = {
  title: 'Components/Tag',
  component: Tag,
};

export default storybookData;

const Template: Story<any> = (args) => <Tag {...args} />;

export const Default = Template.bind({});
Default.args = {
  tag: 'one / two / three',
};

export const Dismissable = Template.bind({});
Dismissable.args = {
  tag: 'onelongword',
  isDismissable: true,
};
