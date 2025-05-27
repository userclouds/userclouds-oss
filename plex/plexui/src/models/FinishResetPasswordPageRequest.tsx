import param from './PageParameterNames';

const finishResetPasswordPageRequest = {
  pageType: 'plex_finish_reset_password_page',
  parameterNames: [
    param.ActionButtonBorderColor,
    param.ActionButtonFillColor,
    param.ActionButtonTextColor,
    param.ActionSubmitButtonText,
    param.HeadingText,
    param.LogoImageFile,
    param.MissingOTPCodeText,
    param.NewPasswordLabel,
    param.PageBackgroundColor,
    param.PageTextColor,
    param.ResetPasswordFailStatusText,
    param.ResetPasswordStartStatusText,
    param.ResetPasswordSuccessStatusText,
  ],
};

export default finishResetPasswordPageRequest;
