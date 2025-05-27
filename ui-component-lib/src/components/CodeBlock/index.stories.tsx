import { Story } from '@storybook/react';

import CodeBlock from './index';
import Text from '../Text';

const storybookData = {
  title: 'Components/CodeBlock',
  component: CodeBlock,
};

export default storybookData;

const DefaultTemplate: Story<any> = (args) => <CodeBlock {...args} />;

export const Default = DefaultTemplate.bind({});
Default.args = {
  children: (
    <Text monospace>
      Our souls are like those orphans whose unwedded mothers die in bearing
      them: the secret of our paternity lies in their grave, and we must there
      to learn it.
    </Text>
  ),
};
