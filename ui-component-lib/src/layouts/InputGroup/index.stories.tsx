import InputGroup from './index';
import Checkbox from '../../components/Checkbox';
import Label from '../../components/Label';
import InputReadOnly from '../../components/InputReadOnly';

const storybookData = {
  title: 'Layouts/InputGroup',
  component: InputGroup,
};

export default storybookData;

function DefaultTemplate() {
  return (
    <div>
      <Label htmlFor="default-label">This is the label</Label>
      <InputGroup>
        <Checkbox id="default-label">This is a checkbox</Checkbox>
        <Checkbox id="default-label">This is a checkbox</Checkbox>
        <Checkbox id="default-label">This is a checkbox</Checkbox>
      </InputGroup>
    </div>
  );
}

export const Default = DefaultTemplate.bind({});
Default.args = {};

function ReadOnlyTemplate() {
  return (
    <div>
      <Label htmlFor="read-only-label">This is the label</Label>
      <InputGroup>
        <InputReadOnly isChecked type="checkbox" id="read-only-label">
          This is a read-only checkbox
        </InputReadOnly>
        <InputReadOnly type="checkbox" id="read-only-label">
          This is a read-only checkbox
        </InputReadOnly>
        <InputReadOnly isChecked type="checkbox" id="read-only-label">
          This is a read-only checkbox
        </InputReadOnly>
      </InputGroup>
    </div>
  );
}

export const ReadOnly = ReadOnlyTemplate.bind({});
Default.args = {};
