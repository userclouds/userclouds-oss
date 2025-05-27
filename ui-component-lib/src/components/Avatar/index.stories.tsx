import { Story } from '@storybook/react';

import { UserAvatar, EntityAvatar } from './index';

const storybookData = {
  title: 'Components/Avatar',
  component: UserAvatar,
};

export default storybookData;

const UserTemplate: Story<any> = (args) => <UserAvatar {...args} />;

export const UserDefault = UserTemplate.bind({});
UserDefault.args = {
  src: './avatar.jpg',
  fullName: 'Jeff Jones',
};

export const UserEmpty = UserTemplate.bind({});
UserEmpty.args = {};

const InitialTemplate: Story<any> = (args) => <EntityAvatar {...args} />;

export const InitialDefault = InitialTemplate.bind({});
InitialDefault.args = {
  initial: 'W',
};
