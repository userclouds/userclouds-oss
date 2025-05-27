import { Story } from '@storybook/react';

import PopOverBox from './index';

const storybookData = {
  title: 'Components/PopOverBox',
  component: PopOverBox,
};

export default storybookData;

const Template: Story<any> = (args) => <PopOverBox {...args} />;

export const Default = Template.bind({});
Default.args = {
  children: (
    <div style={{ padding: 16 }}>
      This is just a box with a shadow and no default padding
    </div>
  ),
};
