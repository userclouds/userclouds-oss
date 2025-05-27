import { Story } from '@storybook/react';
import ToastSolid from './index';

const storybookData = {
  title: 'Components/ToastSolid',
  component: ToastSolid,
};

export default storybookData;

const Template: Story<any> = (args) => <ToastSolid {...args} />;

export const Info = Template.bind({});
Info.args = {
  isDismissable: false,
  children: (
    <>
      Get ready. <a href="/foo">Add connection.</a>
    </>
  ),
};

export const Alert = Template.bind({});
Alert.args = {
  theme: 'alert',
  children: (
    <>
      Your cloud app “The Machine” is disconnected.{' '}
      <a href="/bar">Reestablish connection.</a>
    </>
  ),
};

export const Success = Template.bind({});
Success.args = {
  theme: 'success',
  children: 'Invite Sent',
};
