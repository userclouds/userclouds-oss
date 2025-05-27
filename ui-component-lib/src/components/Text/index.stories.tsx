import Text from './index';

const storybookData = {
  title: 'Components/Text',
  component: Text,
  argTypes: {
    size: {
      control: 'select',
    },
  },
};

export default storybookData;

function Template(args: object) {
  return <Text {...args} />;
}

export const Sizes = Template.bind({});
Sizes.args = {
  size: '1',
  children: 'I’m a little bit of text',
  monospace: false,
};

export const Monospace = Template.bind({});
Monospace.args = {
  size: '3',
  children: 'I’m a little bit of text',
  monospace: true,
};
