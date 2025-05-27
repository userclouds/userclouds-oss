import param from './PageParameterNames';

const mfaSubmitPageRequest = {
  pageType: 'plex_mfa_submit_page',
  parameterNames: [
    param.ActionButtonBorderColor,
    param.ActionButtonFillColor,
    param.ActionButtonTextColor,
    param.ActionSubmitButtonText,
    param.HeadingText,
    param.LogoImageFile,
    param.LoginFailStatusText,
    param.LoginStartStatusText,
    param.LoginSuccessStatusText,
    param.MFACodeLabel,
    param.PageBackgroundColor,
    param.PageTextColor,
  ],
};

export default mfaSubmitPageRequest;
