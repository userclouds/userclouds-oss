import HiddenTextInput from './index';

const storybookData = {
  title: 'Components/HiddenTextInput',
  component: HiddenTextInput,
};

export default storybookData;

function DefaultTemplate(args: object) {
  return <HiddenTextInput {...args} />;
}

export const Default = DefaultTemplate.bind({});
Default.args = {
  value: 'mypassword',
  disabled: false,
  hasError: false,
  placeholder: 'Placeholder text',
};
