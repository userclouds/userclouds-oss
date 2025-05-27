function isDevOrContainer(): boolean {
  return (
    window.location.hostname.endsWith('dev.userclouds.tools') ||
    window.location.hostname.endsWith('test.userclouds.tools')
  );
}

function GetAuthURL(redirectPath: string): string {
  let redirectURL = redirectPath;
  if (isDevOrContainer()) {
    // For local development include full origin because it is on a different port.
    redirectURL = `${window.location.origin}${redirectPath}`;
  }
  return `/auth/redirect?redirect_to=${encodeURIComponent(redirectURL)}`;
}

function GetLogoutURL(redirectPath: string): string {
  let redirectURL = redirectPath;
  if (isDevOrContainer()) {
    // For local development include full origin because it is on a different port.
    redirectURL = `${window.location.origin}${redirectPath}`;
  }
  return `/auth/logout?redirect_to=${encodeURIComponent(redirectURL)}`;
}

export { GetAuthURL, GetLogoutURL };
