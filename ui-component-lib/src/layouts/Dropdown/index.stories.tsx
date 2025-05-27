import { Dropdown, DropdownSection, DropdownButton } from './index';

const storybookData = {
  title: 'Layouts/Dropdown',
  component: Dropdown,
};

export default storybookData;

const Template = (args: object) => (
  <Dropdown {...args}>
    <DropdownSection>
      <DropdownButton>One</DropdownButton>
      <DropdownButton>Two</DropdownButton>
    </DropdownSection>
    <DropdownSection>
      <DropdownButton>Three</DropdownButton>
      <DropdownButton>Four</DropdownButton>
    </DropdownSection>
  </Dropdown>
);

export const Default = Template.bind({});
Default.args = {};
