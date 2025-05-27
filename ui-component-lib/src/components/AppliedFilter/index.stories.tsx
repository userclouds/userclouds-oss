import { Story } from '@storybook/react';
import AppliedFilter from './index';

const storybookData = {
  title: 'Components/AppliedFilter',
  component: AppliedFilter,
};

export default storybookData;

const Template: Story<any> = (args) => <AppliedFilter {...args} />;

export const Default = Template.bind({});
Default.args = {
  source: 'All columns',
  text: 'Policy',
};

export const Dismissable = Template.bind({});
Dismissable.args = {
  source: 'All columns',
  text: 'Policy',
  isDismissable: true,
};
