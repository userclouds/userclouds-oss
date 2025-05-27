import { Story } from '@storybook/react';

import { Accordion, AccordionItem } from './index';

const storybookData = {
  title: 'Components/Accordion',
  component: Accordion,
};

const ItemTemplate: Story<any> = (args) => <AccordionItem {...args} />;

export const Item = ItemTemplate.bind({});
Item.args = {
  title: 'Accordion Title',
  isOpen: true,
  children: <div>This is the content of the accordion item.</div>,
};

export default storybookData;

function FullTemplate() {
  return (
    <Accordion>
      <AccordionItem title="One">
        <div>Content of One</div>
      </AccordionItem>
      <AccordionItem title="Two">
        <div>Content of Two</div>
      </AccordionItem>
      <AccordionItem title="Three">
        <div>Content of Three</div>
      </AccordionItem>
    </Accordion>
  );
}

export const Full = FullTemplate.bind({});
