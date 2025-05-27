import LoaderDots from './index';

const storybookData = {
  title: 'Components/LoaderDots',
  component: LoaderDots,
};

export default storybookData;

function Template(args: object) {
  return <LoaderDots {...args} />;
}

export const Default = Template.bind({});
Default.args = {
  assistiveText: 'text',
  size: 'medium',
  theme: 'brand',
};

function InverseTemplate(args: object) {
  return (
    <div style={{ backgroundColor: 'black', padding: 16 }}>
      <LoaderDots {...args} />
    </div>
  );
}

export const Inverse = InverseTemplate.bind({});
Inverse.args = {
  assistiveText: 'text',
  size: 'medium',
  theme: 'inverse',
};
