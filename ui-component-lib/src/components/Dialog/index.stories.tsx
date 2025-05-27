import { Story } from '@storybook/react';
import Dialog, { DialogBody, DialogFooter } from './index';
import Heading from '../Heading';
import Button from '../Button';
import ButtonGroup from '../../layouts/ButtonGroup';
import Text from '../Text';

const storybookData = {
  title: 'Components/Dialog',
  component: Dialog,
};

export default storybookData;

const Template: Story<any> = (args) => (
  <>
    <Dialog id="myDialog" {...args}>
      <Heading headingLevel={2} size={2}>
        Cool dialog
      </Heading>
      <DialogBody>
        <Text>
          This is a dialog and it has some other components inside it.
        </Text>
      </DialogBody>
      <DialogFooter>
        <Button
          theme="primary"
          onClick={() => {
            const myDialog = document.getElementById(
              'myDialog'
            ) as HTMLDialogElement;
            myDialog.close();
          }}
        >
          Close dialog
        </Button>
      </DialogFooter>
    </Dialog>
    <ButtonGroup>
      <Button
        theme="outline"
        onClick={() => {
          const myDialog = document.getElementById(
            'myDialog'
          ) as HTMLDialogElement;
          myDialog.showModal();
        }}
      >
        Show modal dialog
      </Button>
      <Button
        theme="outline"
        onClick={() => {
          const myDialog = document.getElementById(
            'myDialog'
          ) as HTMLDialogElement;
          myDialog.show();
        }}
      >
        Show non-modal dialog
      </Button>
    </ButtonGroup>
  </>
);

export const Basic = Template.bind({});
