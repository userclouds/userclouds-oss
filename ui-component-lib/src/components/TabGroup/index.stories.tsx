import { Story } from '@storybook/react';

import TabGroup from './index';
import { IconCheck, IconUser3, IconSettingsGear } from '../../icons';

const storybookData = {
  title: 'Components/TabGroup',
  component: TabGroup,
};

export default storybookData;

const Template: Story<any> = (args) => <TabGroup {...args} />;

export const Basic = Template.bind({});
Basic.args = {
  items: [
    {
      id: '1',
      children: 'Tab 1',
    },
    {
      id: '2',
      children: 'Tab 2',
    },
  ],
  tabContent: {
    '1': <div>Content for Tab 1</div>,
    '2': <div>Content for Tab 2</div>,
  },
};

export const WithIcons = Template.bind({});
WithIcons.args = {
  items: [
    {
      id: '1',
      children: 'Tab 1',
      iconLeft: <IconCheck size="medium" />,
    },
    {
      id: '2',
      children: 'Tab 2',
      iconLeft: <IconUser3 size="medium" />,
    },
    {
      id: '3',
      children: 'Tab 3',
      iconLeft: <IconSettingsGear size="medium" />,
    },
  ],
  tabContent: {
    '1': <div>Content for Tab 1 with check icon</div>,
    '2': <div>Content for Tab 2 with user icon</div>,
    '3': <div>Content for Tab 3 with settings icon</div>,
  },
  defaultActiveTab: '2',
};

export const WithDisabledTab = Template.bind({});
WithDisabledTab.args = {
  items: [
    {
      id: '1',
      children: 'Tab 1',
    },
    {
      id: '2',
      children: 'Tab 2',
      disabled: true,
    },
    {
      id: '3',
      children: 'Tab 3',
    },
  ],
  tabContent: {
    '1': <div>Content for Tab 1</div>,
    '2': (
      <div>This content won't be accessible because the tab is disabled</div>
    ),
    '3': <div>Content for Tab 3</div>,
  },
};

export const FullWidth = Template.bind({});
FullWidth.args = {
  items: [
    {
      id: '1',
      children: 'Tab 1',
    },
    {
      id: '2',
      children: 'Tab 2',
    },
    {
      id: '3',
      children: 'Tab 3',
    },
  ],
  tabContent: {
    '1': <div>Content for Tab 1</div>,
    '2': <div>Content for Tab 2</div>,
    '3': <div>Content for Tab 3</div>,
  },
  fullWidth: true,
};
