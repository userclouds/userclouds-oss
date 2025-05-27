import { Story } from '@storybook/react';
import InlineNotification from './index';

const storybookData = {
  title: 'Components/InlineNotification',
  component: InlineNotification,
};

export default storybookData;

const Template: Story<any> = (args) => <InlineNotification {...args} />;

export const Info = Template.bind({});
Info.args = {
  children: (
    <>
      You may need to add a second connection.{' '}
      <a href="/foo">Add connection.</a>
    </>
  ),
};

export const Alert = Template.bind({});
Alert.args = {
  theme: 'alert',
  children: (
    <>
      Your cloud app “The Machine” has been disconnected.{' '}
      <a href="/bar">Reestablish connection.</a>
    </>
  ),
};

export const Success = Template.bind({});
Success.args = {
  theme: 'success',
  children: (
    <>
      You did a great thing. <a href="/baz">Celebrate.</a>
    </>
  ),
};
