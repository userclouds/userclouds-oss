import { Card, CardRow, CardColumns, CardColumn, CardFooter } from './index';
import Label from '../../components/Label';
import Button from '../../components/Button';
import Checkbox from '../../components/Checkbox';
import InputReadOnly from '../../components/InputReadOnly';
import Text from '../../components/Text';
import TextInput from '../../components/TextInput';
import ButtonGroup from '../ButtonGroup';
import InputGroup from '../InputGroup';

const storybookData = {
  title: 'Layouts/Card',
  component: Card,
};

export default storybookData;

function Template(args: object) {
  return (
    <Card {...args}>
      <CardRow>
        <Label htmlFor="tenant-url">Tenant URL</Label>
        <InputReadOnly id="tenant-url">
          https://testingtenant.tenant.userclouds.com
        </InputReadOnly>
      </CardRow>
      <CardRow>
        <Label htmlFor="group-title">Group Title</Label>
        <InputGroup>
          <InputReadOnly type="checkbox" isChecked id="group-title">
            This is a read-only checkbox
          </InputReadOnly>
          <InputReadOnly type="checkbox" id="group-title">
            This is a read-only checkbox
          </InputReadOnly>
          <InputReadOnly type="checkbox" id="group-title">
            Read-only checkbox
          </InputReadOnly>
        </InputGroup>
      </CardRow>
      <CardFooter>
        <Button theme="secondary">Edit Settings</Button>
      </CardFooter>
    </Card>
  );
}

export const Default = Template.bind({});
Default.args = {
  hasError: false,
  isDirty: false,
  title: 'Applications',
  description:
    'Configure Authentication for your applications. Each Plex Application represents an OAuth2/OIDC client.',
};

function DirtyTemplate(args: object) {
  return (
    <Card {...args}>
      <CardRow>
        <Label htmlFor="tenant-url">Tenant URL</Label>
        <TextInput
          value="https://testingtenant.tenant.userclouds.com"
          id="tenant-url"
        />
      </CardRow>
      <CardRow>
        <Label htmlFor="group-title">Group Title</Label>
        <InputGroup>
          <Checkbox id="group-title">This is a checkbox</Checkbox>
          <Checkbox id="group-title">This is a checkbox</Checkbox>
          <Checkbox id="group-title">This is a checkbox</Checkbox>
        </InputGroup>
      </CardRow>
      <CardFooter>
        <ButtonGroup>
          <Button>Save</Button>
          <Button theme="ghost">Cancel</Button>
        </ButtonGroup>
      </CardFooter>
    </Card>
  );
}

export const Dirty = DirtyTemplate.bind({});
Dirty.args = {
  isDirty: true,
  title: 'Applications',
  description:
    'Configure Authentication for your applications. Each Plex Application represents an OAuth2/OIDC client.',
};

function ErrorTemplate(args: object) {
  return (
    <Card {...args}>
      <CardRow>
        <Label htmlFor="tenant-url">Tenant URL</Label>
        <TextInput
          value="https://testingtenant.tenant.userclouds.com"
          id="tenant-url"
        />
      </CardRow>
      <CardRow>
        <Label htmlFor="group-title">Group Title</Label>
        <InputGroup>
          <Checkbox id="group-title">This is a checkbox</Checkbox>
          <Checkbox id="group-title">This is a checkbox</Checkbox>
          <Checkbox id="group-title">This is a checkbox</Checkbox>
        </InputGroup>
      </CardRow>
      <CardFooter>
        <ButtonGroup>
          <Button>Save</Button>
          <Button theme="ghost">Cancel</Button>
        </ButtonGroup>
      </CardFooter>
    </Card>
  );
}

export const Error = ErrorTemplate.bind({});
Error.args = {
  hasError: true,
  title: 'Applications',
  description:
    'Configure Authentication for your applications. Each Plex Application represents an OAuth2/OIDC client.',
};

function LockedTemplate(args: object) {
  return (
    <Card {...args}>
      <CardRow>
        <Label htmlFor="tenant-url">Tenant URL</Label>
        <InputReadOnly id="tenant-url">
          https://testingtenant.tenant.userclouds.com
        </InputReadOnly>
      </CardRow>
      <CardRow>
        <Label htmlFor="group-title">Group Title</Label>
        <InputGroup>
          <InputReadOnly type="checkbox" isChecked id="group-title">
            This is a read-only checkbox
          </InputReadOnly>
          <InputReadOnly type="checkbox" id="group-title">
            This is a read-only checkbox
          </InputReadOnly>
          <InputReadOnly type="checkbox" id="group-title">
            Read-only checkbox
          </InputReadOnly>
        </InputGroup>
      </CardRow>
    </Card>
  );
}

export const Locked = LockedTemplate.bind({});
Locked.args = {
  lockedMessage: 'Edits require tenant admin privileges',
  title: 'Applications',
  description:
    'Configure Authentication for your applications. Each Plex Application represents an OAuth2/OIDC client.',
};

function ColumnsTemplate(args: object) {
  return (
    <Card {...args}>
      <CardColumns>
        <CardColumn>
          <CardRow>
            <Label htmlFor="options-here">Options here</Label>
            <InputGroup>
              <Checkbox id="options-here">This is a checkbox</Checkbox>
              <Checkbox id="options-here">This is a checkbox</Checkbox>
              <Checkbox id="options-here">This is a checkbox</Checkbox>
            </InputGroup>
          </CardRow>
        </CardColumn>
        <CardColumn>
          <CardRow>
            <Label htmlFor="options-over-here">Options over here</Label>
            <InputGroup>
              <Checkbox id="options-over-here">This is a checkbox</Checkbox>
              <Checkbox id="options-over-here">This is a checkbox</Checkbox>
              <Checkbox id="options-over-here">This is a checkbox</Checkbox>
            </InputGroup>
          </CardRow>
        </CardColumn>
      </CardColumns>
      <CardFooter>
        <Button theme="secondary">Edit Settings</Button>
      </CardFooter>
    </Card>
  );
}

export const Columns = ColumnsTemplate.bind({});
Columns.args = {
  title: 'Applications',
  description:
    'Configure Authentication for your applications. Each Plex Application represents an OAuth2/OIDC client.',
};

const TextOnlyTemplate = () => (
  <Card title="Applications">
    <CardRow>
      <Text>
        Ownership and control of sensitive data with a private-by-design AuthN &
        AuthZ platform. Centralize your sensitive data & your enforcement of
        access policies to minimize your risk of fines and data breaches, with
        minimal effort.
      </Text>
      <Text>
        Reclaim ownership and control of sensitive data with a private-by-design
        AuthN & AuthZ platform. Centralize your sensitive data & your
        enforcement of access policies to minimize your risk of fines and data
        breaches, with minimal effort.
      </Text>
    </CardRow>
  </Card>
);

export const TextOnly = TextOnlyTemplate.bind({});
TextOnly.args = {};
