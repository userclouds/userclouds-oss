import param from './PageParameterNames';

const startPasswordResetPageRequest = {
  pageType: 'plex_start_reset_password_page',
  parameterNames: [
    param.ActionButtonBorderColor,
    param.ActionButtonFillColor,
    param.ActionButtonTextColor,
    param.ActionSubmitButtonText,
    param.EmailLabel,
    param.HeadingText,
    param.LogoImageFile,
    param.PageBackgroundColor,
    param.PageTextColor,
    param.ResetPasswordFailStatusText,
    param.ResetPasswordStartStatusText,
    param.ResetPasswordSuccessStatusText,
  ],
};

export default startPasswordResetPageRequest;
