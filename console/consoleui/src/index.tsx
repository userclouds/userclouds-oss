import { createRoot } from 'react-dom/client';
import { Provider } from 'react-redux';
import { init as initSentry } from '@sentry/react';
import { BrowserTracing } from '@sentry/browser';

import '@userclouds/ui-component-lib';

import { getEnvData } from './models/EnvData';
import App from './App';
import store from './store';
import { startListening } from './routing';
import reportWebVitals from './reportWebVitals';
import './index.css';

const container = document.getElementById('root');
if (container) {
  const root = createRoot(container);
  root.render(
    <Provider store={store}>
      <App />
    </Provider>
  );
  startListening(store.dispatch);
  // If you want to start measuring performance in your app, pass a function
  // to log results (for example: reportWebVitals(console.log))
  // or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
  reportWebVitals();
} else {
  throw new Error('Error mounting React App');
}
const { Universe, SentryDsn } = getEnvData();
if (SentryDsn !== '') {
  initSentry({
    dsn: SentryDsn,
    integrations: [new BrowserTracing()],
    debug: false,
    environment: Universe,
    maxBreadcrumbs: 50,
  });
}
