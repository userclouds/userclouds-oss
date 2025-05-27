import param from './PageParameterNames';

const createUserPageRequest = {
  pageType: 'plex_create_user_page',
  parameterNames: [
    param.ActionButtonBorderColor,
    param.ActionButtonFillColor,
    param.ActionButtonTextColor,
    param.ActionSubmitButtonText,
    param.AllowCreation,
    param.CreateUserDisabledText,
    param.CreateUserFailStatusText,
    param.CreateUserStartStatusText,
    param.CustomCreateUserPagePostFieldsetHTMLCSS,
    param.CustomCreateUserPagePreFieldsetHTMLCSS,
    param.CustomCreateUserPagePostFormHTMLCSS,
    param.CustomCreateUserPagePreFormHTMLCSS,
    param.CustomCreateUserPagePreMainHTMLCSS,
    param.EmailLabel,
    param.HeadingText,
    param.LoginFailStatusText,
    param.LoginSuccessStatusText,
    param.LogoImageFile,
    param.PasswordLabel,
    param.PageBackgroundColor,
    param.PageTextColor,
    param.RequireName,
    param.SubheadingText,
  ],
};

export default createUserPageRequest;
