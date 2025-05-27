import IconButton from './index';
import { IconDeleteBin } from '../../icons';

const storybookData = {
  title: 'Components/IconButton',
  component: IconButton,
  args: {
    disabled: false,
  },
};

export default storybookData;

function Template(args: object) {
  return <IconButton icon={<IconDeleteBin />} title="Delete" {...args} />;
}

export const Medium = Template.bind({});
