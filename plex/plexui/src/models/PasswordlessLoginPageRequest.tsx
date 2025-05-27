import param from './PageParameterNames';

const passwordlessLoginPageRequest = {
  pageType: 'plex_passwordless_login_page',
  parameterNames: [
    param.ActionButtonBorderColor,
    param.ActionButtonFillColor,
    param.ActionButtonTextColor,
    param.ActionSubmitButtonText,
    param.EmailLabel,
    param.HeadingText,
    param.LogoImageFile,
    param.LoginFailStatusText,
    param.LoginStartStatusText,
    param.LoginSuccessStatusText,
    param.OtpCodeLabel,
    param.PageBackgroundColor,
    param.PageTextColor,
    param.PasswordlessSendEmailStartStatusText,
    param.PasswordlessSendEmailSuccessStatusText,
  ],
};

export default passwordlessLoginPageRequest;
