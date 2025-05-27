import { Story } from '@storybook/react';
import Banner from './index';

const storybookData = {
  title: 'Components/Banner',
  component: Banner,
};

export default storybookData;

const Template: Story<any> = (args) => <Banner {...args} />;

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
