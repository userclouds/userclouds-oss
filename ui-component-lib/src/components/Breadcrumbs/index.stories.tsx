import { Story } from '@storybook/react';

import Breadcrumbs from './index';

const storybookData = {
  title: 'Components/Breadcrumbs',
  component: Breadcrumbs,
};

const Link = ({
  children,
  ...props
}: {
  children: string;
  [key: string]: any;
}) => (
  <a {...props} style={{ color: '#555', textDecoration: 'none' }}>
    {children}
  </a>
);

export default storybookData;

const Template: Story<any> = (args) => <Breadcrumbs {...args} />;

export const Default = Template.bind({});
Default.args = {
  links: [
    <Link href="#home" key="home">
      Home
    </Link>,
    <Link href="#library" key="library">
      Library
    </Link>,
  ],
};
