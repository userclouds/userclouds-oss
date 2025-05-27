import InputReadOnly from './index';
import Label from '../Label';
import InputGroup from '../../layouts/InputGroup';

const storybookData = {
  title: 'Components/InputReadOnly',
  component: InputReadOnly,
};

export default storybookData;

function Template(args: object) {
  return (
    <InputReadOnly {...args}>
      This is the read-only text input used for select, radio, and text input
      values.
    </InputReadOnly>
  );
}

export const Default = Template.bind({});
Default.args = {
  isChecked: true,
};

function LabelTemplate() {
  return (
    <>
      <Label htmlFor="readOnlyInput">This is the label</Label>
      <InputReadOnly id="readOnlyInput">
        Read-only input used for select, radio, and text input values.
      </InputReadOnly>
    </>
  );
}

export const TextWithLabel = LabelTemplate.bind({});
TextWithLabel.args = {};

function CheckBoxTemplate(args: object) {
  return (
    <InputReadOnly {...args}>This is the read-only checkbox</InputReadOnly>
  );
}

export const Checkbox = CheckBoxTemplate.bind({});
Checkbox.args = {
  isChecked: true,
  type: 'checkbox',
};

function GroupTemplate() {
  return (
    <div>
      <Label htmlFor="readOnlyInput">Label Text</Label>
      <InputGroup>
        <InputReadOnly isChecked type="checkbox" id="readOnlyInput">
          This is a checkbox
        </InputReadOnly>
        <InputReadOnly type="checkbox">This is a checkbox</InputReadOnly>
        <InputReadOnly type="checkbox">This is a checkbox</InputReadOnly>
      </InputGroup>
    </div>
  );
}

export const CheckboxGroup = GroupTemplate.bind({});
CheckboxGroup.args = {};
