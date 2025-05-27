import CodeEditor from './index';

const storybookData = {
  title: 'Components/CodeEditor',
  component: CodeEditor,
};

export default storybookData;

function Template(args: object) {
  return (
    <CodeEditor
      javascriptExt
      value={`function fooBar() {
  for (const i = 0; i < 43; i++) {
    console.log(i);
    alert('hi');
  }
}`}
      {...args}
    />
  );
}

export const Default = Template.bind({});
