import TextShortener from './index';

const storybookData = {
  title: 'Components/TextShortener',
  component: TextShortener,
  argTypes: {
    length: 8,
    isCopyable: true,
    firstChars: true,
  },
};

export default storybookData;

function Template(args: object) {
  return <TextShortener {...args} />;
}

export const Default = Template.bind({});
Default.args = {
  uuid: '00000000-0000-0000-0000-000000000000',
};
