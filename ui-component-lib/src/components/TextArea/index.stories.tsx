import TextArea from './index';

const storybookData = {
  title: 'Components/TextArea',
  component: TextArea,
};

export default storybookData;

function TemplateDefault(args: object) {
  return <TextArea {...args} />;
}

export const Default = TemplateDefault.bind({});
Default.args = {
  readOnly: false,
  monospace: false,
  hasError: false,
  disabled: false,
  id: 'textarea-id',
  placeholder: 'Hi, this is placeholder if needed.',
};

function TemplateMonospace(args: object) {
  return <TextArea {...args} />;
}

export const MonospaceReadOnly = TemplateMonospace.bind({});
MonospaceReadOnly.args = {
  readOnly: true,
  monospace: true,
  hasError: false,
  disabled: false,
  id: 'textarea-id',
  value:
    'Whenever I find myself growing grim about the mouth; whenever it is a damp, drizzly November in my soul; whenever I find myself involuntarily pausing before coffin warehouses, and bringing up the rear of every funeral I meet; and especially whenever my hypos get such an upper hand of me, that it requires a strong moral principle to prevent me from deliberately stepping into the street, and methodically knocking hats off - then, I account it high time to get to sea as soon as I can.',
};
