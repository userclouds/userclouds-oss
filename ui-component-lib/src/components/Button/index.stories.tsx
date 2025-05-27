import { Story } from '@storybook/react';

import Button from './index';
import ButtonGroup from '../../layouts/ButtonGroup';
import { IconCheck } from '../../icons';

const storybookData = {
  title: 'Components/Button',
  component: Button,
};

export default storybookData;

const Template: Story<any> = (args) => <Button {...args} />;

export const PrimaryMedium = Template.bind({});
PrimaryMedium.args = {
  full: false,
  disabled: false,
  children: 'Save Changes',
  iconLeft: <IconCheck size="medium" />,
};

export const PrimarySmall = Template.bind({});
PrimarySmall.args = {
  full: false,
  size: 'small',
  disabled: false,
  children: 'Save Changes',
  iconLeft: <IconCheck size="small" />,
};

function GroupTemplate() {
  return (
    <ButtonGroup>
      <Button theme="primary">Edit Settings</Button>
      <Button theme="secondary">Edit Settings</Button>
      <Button theme="dangerous">Edit Settings</Button>
      <Button theme="outline">Edit Settings</Button>
      <Button theme="ghost">Edit Settings</Button>
    </ButtonGroup>
  );
}

export const Group = GroupTemplate.bind({});

function GroupTemplateLoading() {
  return (
    <ButtonGroup>
      <Button isLoading theme="primary">
        Edit Settings
      </Button>
      <Button isLoading theme="secondary">
        Edit Settings
      </Button>
      <Button isLoading theme="dangerous">
        Edit Settings
      </Button>
      <Button isLoading theme="outline">
        Edit Settings
      </Button>
      <Button isLoading theme="ghost">
        Edit Settings
      </Button>
    </ButtonGroup>
  );
}

export const GroupLoading = GroupTemplateLoading.bind({});
