import { useEffect } from 'react';

const beforeUnloadHandler = (e: BeforeUnloadEvent) => {
  e.preventDefault();
  const ok = window.confirm('Leave page? Changes will not be saved.');
  if (ok) {
    window.removeEventListener('beforeunload', beforeUnloadHandler, {
      capture: true,
    });
  }
};
// NavigationBlocker allows a component to block navigation if `prompt` is true,
// and allow navigation if `prompt` is false. Typically you set `prompt` to be
// the value of a dirty flag.
// This component blocks nav if you go back/forward in the browser, click on any
// link or element that would cause navigation, enter a new URL in the nav bar, etc.
const NavigationBlocker = ({ showPrompt }: { showPrompt: boolean }) => {
  useEffect(() => {
    if (showPrompt) {
      window.addEventListener('beforeunload', beforeUnloadHandler, {
        capture: true,
      });
    }
  }, [showPrompt]);

  return <></>;
};

export default NavigationBlocker;
