import '@fontsource/inter/variable.css';
import '@fontsource/dm-mono/400.css';
import '../styles/globals-sb.css';

export const parameters = {
  options: {
    storySort: {
      order: ['Introduction'],
    },
  },
  actions: { argTypesRegex: '^on[A-Z].*' },
  controls: {
    matchers: {
      color: /(background|color)$/i,
      date: /Date$/,
    },
  },
};
