import { Story } from '@storybook/react';
import DateTimePicker from './index';

const storybookData = {
  title: 'Components/DateTimePicker',
  component: DateTimePicker,
};

export default storybookData;

const Template: Story<any> = (args) => <DateTimePicker {...args} />;

export const Default = Template.bind({});
Default.args = {
  value: new Date(),
};
