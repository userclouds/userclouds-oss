import ToolTip from './index';

const storybookData = {
  title: 'Components/ToolTip',
  component: ToolTip,
};

export default storybookData;

function Template(args: object) {
  return <ToolTip {...args} />;
}

export const Default = Template.bind({});
Default.args = {
  arrow: 'bottom',
};
