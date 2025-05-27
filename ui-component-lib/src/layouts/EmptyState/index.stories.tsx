import EmptyState from './index';
import { Card, CardRow } from '../Card';
import Button from '../../components/Button';
import { IconDashboard3 } from '../../icons';

const storybookData = {
  title: 'Layouts/EmptyState',
  component: EmptyState,
};

export default storybookData;

function Template(args: object) {
  return (
    <Card>
      <CardRow>
        <EmptyState {...args}>
          <Button theme="outline">Do something</Button>
        </EmptyState>
      </CardRow>
    </Card>
  );
}

export const Default = Template.bind({});
Default.args = {
  title: 'This is an empty state',
  subTitle: 'This is a subtitle',
  image: <IconDashboard3 size="large" />,
};
