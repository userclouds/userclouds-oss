import {
  Anchor,
  Text,
  FooterStyles as styles,
} from '@userclouds/ui-component-lib';

const year = new Date().getFullYear();
const Footer = () => {
  return (
    <footer className={styles.root}>
      <div className={styles.links}>
        <Text elementName="div" size={2}>
          <Anchor href="https://www.userclouds.com/about" noUnderline>
            About
          </Anchor>
        </Text>
        <Text elementName="div" size={2}>
          <Anchor href="https://www.userclouds.com/contact" noUnderline>
            Contact Us
          </Anchor>
        </Text>
        <Text elementName="div" size={2}>
          <Anchor href="https://www.userclouds.com/careers" noUnderline>
            Careers
          </Anchor>
        </Text>
        <Text elementName="div" size={2}>
          <Anchor href="https://docs.userclouds.com" noUnderline>
            Docs
          </Anchor>
        </Text>
      </div>
      <Text size={2} className={styles.copyright} elementName="div">
        &copy;{year} UserClouds
      </Text>
    </footer>
  );
};

export default Footer;
