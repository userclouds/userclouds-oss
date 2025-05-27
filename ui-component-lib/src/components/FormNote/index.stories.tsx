import { Story } from '@storybook/react';

import FormNote from './index';
import Label from '../Label';
import TextInput from '../TextInput';

const storybookData = {
  title: 'Components/FormNote',
  component: FormNote,
};

export default storybookData;

const Template: Story<any> = (args) => <FormNote {...args} />;

export const Default = Template.bind({});
Default.args = {
  children: 'Here is a form note',
};

export const Full = Template.bind({});
Full.args = {
  children: (
    <>
      <Label htmlFor="input-label">Input label</Label>
      <TextInput placeholder="This is an input" id="input-label" />
      <FormNote>Here is a form note</FormNote>
    </>
  ),
};
