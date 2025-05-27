import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
} from './index';

const storybookData = {
  title: 'Components/Table',
  component: Table,
  args: {
    spacing: 'default',
    hasLines: true,
  },
};

export default storybookData;

function Template(args: object) {
  return (
    <Table {...args}>
      <TableHead>
        <TableRow>
          <TableRowHead>Name</TableRowHead>
          <TableRowHead align="right">Text</TableRowHead>
        </TableRow>
      </TableHead>
      <TableBody>
        <TableRow>
          <TableCell data-title="Name">Example Tenant Default App 1</TableCell>
          <TableCell data-title="Text" align="right">
            Text aligned right
          </TableCell>
        </TableRow>
        <TableRow>
          <TableCell data-title="Name">Example Tenant Default App 2</TableCell>
          <TableCell data-title="Text" align="right">
            Text aligned right
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  );
}

export const Default = Template.bind({});
