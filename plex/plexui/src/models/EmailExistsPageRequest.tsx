import param from './PageParameterNames';

const emailExistsPageRequest = {
  pageType: 'plex_email_exists_page',
  parameterNames: [
    param.ActionButtonBorderColor,
    param.ActionButtonFillColor,
    param.ActionButtonTextColor,
    param.ActionSubmitButtonText,
    param.EmailExistsCreateAccountText,
    param.EmailExistsLoginText,
    param.FooterHTML,
    param.HeadingText,
    param.LoginFailStatusText,
    param.LoginStartStatusText,
    param.LoginSuccessStatusText,
    param.LogoImageFile,
    param.OIDCAuthenticationSettings,
    param.PageBackgroundColor,
    param.PageTextColor,
    param.PasswordLabel,
    param.SocialRedirectStatusText,
    param.SubheadingText,
    param.UserNameLabel,
  ],
};

export default emailExistsPageRequest;
