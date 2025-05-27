import ButtonGroup from './index';
import Button from '../../components/Button';

const storybookData = {
  title: 'Layouts/ButtonGroup',
  component: ButtonGroup,
};

export default storybookData;

function Template(args: object) {
  return (
    <ButtonGroup {...args}>
      <Button>Save</Button>
      <Button theme="ghost">Cancel</Button>
    </ButtonGroup>
  );
}

export const Default = Template.bind({});
Default.args = {};
