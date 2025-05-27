@authn
@plex
@login_app
Feature: login app page

  @a11y
  Scenario: general settings login app accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And the following mocked requests:
      | Method | Path                                                                                                     | Status | Body                |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/emailelements                                          | 200    | email_elements.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/smselements                                            | 200    | sms_elements.json   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/apppageparameters/90ffb499-2549-470e-99cd-77f7008e2735 | 200    | pageparameters.json |
    When I navigate to the page with path "/loginapps/90ffb499-2549-470e-99cd-77f7008e2735?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  @general_settings
  Scenario: edit general settings login app
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And the following mocked requests:
      | Method | Path                                                                                                     | Status | Body                |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/emailelements                                          | 200    | email_elements.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/smselements                                            | 200    | sms_elements.json   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/apppageparameters/90ffb499-2549-470e-99cd-77f7008e2735 | 200    | pageparameters.json |
    When I navigate to the page with path "/loginapps/90ffb499-2549-470e-99cd-77f7008e2735?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a card with the title "General settings"
    And I should see a card with the title "Login settings"
    And I should see a card with the title "Email settings"
    And I should see a card with the title "SMS settings"
    And I should see a card with the title "Advanced"
    When I click to expand the "Application Settings" accordion
    And I replace the text in the "name" field with "Foo"
    Then the button labeled "Save" within the "General settings" card should be enabled
    When I replace the text in the "description" field with "Foo"
    And I replace the text in the "client_id" field with "ed79c6c4d8ee35cb50f3e5ed6d788509"
    And I replace the text in the "client_secret" field with "4DzhbWxUu4X/igtUZ0pvt/vC+lQM9nOaXOmBdyn4pxksNFRxYuD/yFLF6NMoBfTf"
    When I click to expand the "Allowed Redirect URLs" accordion
    Then the values in the editable list with ID "allowedRedirectURIs" should be
      | Value                                                         |
      | https://console.dev.userclouds.tools:3333/auth/callback       |
      | https://console.dev.userclouds.tools:3010/auth/callback       |
      | https://console.dev.userclouds.tools:3333/auth/invitecallback |
      | https://console.dev.userclouds.tools:3010/auth/invitecallback |
    And I change the text in row 1 of the editable list with ID "allowedRedirectURIs" to "https://foo.com"
    And I click the delete icon next to row 3 of the editable list with ID "allowedRedirectURIs"
    Then the values in the editable list with ID "allowedRedirectURIs" should be
      | Value                                                         |
      | https://foo.com                                               |
      | https://console.dev.userclouds.tools:3010/auth/callback       |
      | https://console.dev.userclouds.tools:3010/auth/invitecallback |
    When I click to expand the "Allowed Logout URLs" accordion
    Then the values in the editable list with ID "allowedLogoutURLs" should be
      | Value                                      |
      | https://console.dev.userclouds.tools:3333/ |
      | https://console.dev.userclouds.tools:3010/ |
    And I click the delete icon next to row 1 of the editable list with ID "allowedLogoutURLs"
    And I click the delete icon next to row 1 of the editable list with ID "allowedLogoutURLs"
    Then the values in the editable list with ID "allowedLogoutURLs" should be
      | Value |
    When I click the button labeled "Add Logout URL"
    And I change the text in row 1 of the editable list with ID "allowedLogoutURLs" to "https://foo.com"
    Then the values in the editable list with ID "allowedLogoutURLs" should be
      | Value           |
      | https://foo.com |
    When I click to expand the "OAuth Grant Types" accordion
    And I toggle the checkbox labeled "Client Credentials"
    Then the checkbox labeled "Client Credentials" should be unchecked
    Given the following mocked requests:
      | Method | Path                                                                                                                    | Status | Body            |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/loginapps/actions/samlidp?app_id=90ffb499-2549-470e-99cd-77f7008e2735 | 200    | enable_sso.json |
    When I click to expand the "SAML IDP" accordion
    And I click the button labeled "Enable"
    Then I should see the following text on the page
      | TagName | TextContent               |
      | label   | Entity ID                 |
      | label   | SSO URL                   |
      | label   | Certificate               |
      | label   | Trusted Service Providers |
    And I should see a button labeled "Add Trusted Service Provider"
    Given a mocked "POST" request for "plex_config"
    When I click the button labeled "Save"
    Then I should see the following text on the page
      | TagName | TextContent                    |
      | p       | Successfully updated login app |
    And the button labeled "Save" within the "General settings" card should be disabled

  @login_settings
  Scenario: edit login settings
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And the following mocked requests:
      | Method | Path                                                                                                     | Status | Body                |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/emailelements                                          | 200    | email_elements.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/smselements                                            | 200    | sms_elements.json   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/apppageparameters/90ffb499-2549-470e-99cd-77f7008e2735 | 200    | pageparameters.json |
    When I navigate to the page with path "/loginapps/90ffb499-2549-470e-99cd-77f7008e2735?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a card with the title "General settings"
    And I should see a card with the title "Login settings" and the description "Customize this application’s login methods, security settings and user interface. To set up new connections, go back to the Authentication page."
    And I should see a card with the title "Email settings"
    And I should see a card with the title "SMS settings"
    And I should see a card with the title "Advanced"
    And I should see the following text on the page
      | TagName                             | TextContent                  |
      | details > summary > div             | Authentication methods       |
      | details > summary > div             | Authentication methods       |
      | details > summary > div             | Logo & colors                |
      | details > summary > div             | Login page settings          |
      | details > summary > div             | Create account page settings |
      | #loginSettingsForm button[disabled] | Save login settings          |
      | #loginSettingsForm button[disabled] | Cancel                       |
    And I should see the following text in the login preview pane
      | TagName                              | TextContent                   |
      | button                               | See create user page          |
      | h1                                   | Sign in to UserClouds Console |
      | label                                | Username                      |
      | label                                | Password                      |
      | a[href="/plexui/startresetpassword"] | Forgot username or password?  |
      | a[href="/plexui/passwordlesslogin"]  | Hate Passwords?...            |
      | button[type="submit"]                | Sign in                       |
      | button                               | Sign in with Facebook         |
      | button                               | Sign in with Google           |
      | button                               | Sign in with LinkedIn         |
    And I should see the following form elements within the form with ID "loginForm"
      | TagName | Type     | Name          | Value |
      | input   | text     | form_username |       |
      | input   | password | form_password |       |
    When I click to expand the "Authentication methods" accordion
    Then I should see the following text on the page
      | TagName | TextContent           |
      | label   | Username and password |
      | label   | Email - passwordless  |
      | label   | Facebook              |
      | label   | Google                |
      | label   | LinkedIn              |
    And the checkbox labeled "Username and password" should be checked
    And the checkbox labeled "Email - passwordless" should be checked
    And the checkbox labeled "Facebook" should be checked
    And the checkbox labeled "Google" should be checked
    And the checkbox labeled "LinkedIn" should be checked
    When I toggle the checkbox labeled "Google"
    Then I should not see a button labeled "Sign in with Google"
    And the checkbox labeled "Google" should be unchecked
    And the button labeled "Save login settings" should be enabled
    And the button labeled "Cancel" within the "Login settings" card should be enabled
    When I toggle the checkbox labeled "Username and password"
    Then I should not see an element matching selector "#loginForm label:has-text('Username')"
    And I should not see an element matching selector "#loginForm input[name='form_username']"
    And I should not see an element matching selector "#loginForm label:has-text('Password')"
    And I should not see an element matching selector "#loginForm input[name='form_password']"
    And I should not see an element matching selector "a[href='/plexui/startresetpassword']"
    And the checkbox labeled "Username and password" should be unchecked
    When I click to expand the "Authentication settings" accordion
    Then the checkbox labeled "Require MFA for all non-social logins" should be unchecked
    And the checkbox labeled "Email" should be unchecked
    And the checkbox labeled "SMS" should be unchecked
    # And the checkbox labeled "SMS" should be disabled
    And the checkbox labeled "Authenticator app" should be unchecked
    And the checkbox labeled "Recovery code" should be unchecked
    # this won't occasion a change in the login preview
    When I toggle the checkbox labeled "Authenticator app"
    Then the checkbox labeled "Authenticator app" should be checked
    When I toggle the checkbox labeled "Username and password"
    Then the checkbox labeled "Username and password" should be checked
    And social login should follow other methods in the login form
    When I toggle the radio labeled "Social first"
    Then the radio labeled "Social first" should be checked
    And social login should precede other methods in the login form
    When I click to expand the "Logo & colors" accordion
    Then I should see the following form elements
      | TagName | Type | Name               | Value   |
      | input   | text | actionButtonFill   | #1090FF |
      | input   | text | actionButtonText   | #FFFFFF |
      | input   | text | actionButtonBorder | #1090FF |
      | input   | text | pageBackground     | #F8F8F8 |
      | input   | text | pageText           | #505060 |
    When I replace the text in the "actionButtonFill" field with "#00FF00"
    And I replace the text in the "actionButtonBorder" field with "#FF0000"
    Then the submit button in the login preview pane should have a "background-color" of "#00FF00"
    And the submit button in the login preview pane should have a "border-color" of "#FF0000"
    When I click to expand the "Login page settings" accordion
    Then the input with the name "loginPageHeadingCopy" should have the value "Sign in to UserClouds Console"
    And the input with the name "loginPageSubheadingCopy" should have the value ""
    And the input with the name "loginPageFooterContents" should have the value ""
    When I replace the text in the "loginPageHeadingCopy" field with "Abracadabra"
    And I replace the text in the "loginPageSubheadingCopy" field with "...is the password"
    And I replace the text in the "loginPageFooterContents" field with "(Taylor's Version)"
    Then the input with the name "loginPageHeadingCopy" should have the value "Abracadabra"
    And the input with the name "loginPageSubheadingCopy" should have the value "...is the password"
    And the input with the name "loginPageFooterContents" should have the value "(Taylor's Version)"
    And I should see the following text in the login preview pane
      | TagName                              | TextContent                  |
      | button                               | See create user page         |
      | h1                                   | Abracadabra                  |
      | h2                                   | ...is the password           |
      | label                                | Username                     |
      | label                                | Password                     |
      | a[href="/plexui/startresetpassword"] | Forgot username or password? |
      | a[href="/plexui/passwordlesslogin"]  | Hate Passwords?...           |
      | button[type="submit"]                | Sign in                      |
      | button                               | Sign in with Facebook        |
      | button                               | Sign in with LinkedIn        |
      | p                                    | (Taylor's Version)           |
    When I click to expand the "Create account page settings" accordion
    Then I should see the following form elements
      | TagName  | Type | Name                            | Value       |
      | input    | text | createAccountPageHeadingCopy    | Create User |
      | input    | text | createAccountPageSubheadingCopy |             |
      | textarea |      | createAccountPageFooterContents |             |
    And the checkbox labeled "Name" should be checked
    When I click the button labeled "See create user page"
    Then I should see the following text in the login preview pane
      | TagName               | TextContent    |
      | button                | See login page |
      | h1                    | Create User    |
      | label                 | Full name      |
      | label                 | Email          |
      | label                 | Password       |
      | button[type="submit"] | Create         |
    And I should see the following form elements within the form with ID "createUserForm"
      | TagName | Type     | Name          | Value |
      | input   | text     | form_name     |       |
      | input   | email    | form_email    |       |
      | input   | password | form_password |       |
    And the submit button in the login preview pane should have a "background-color" of "#00FF00"
    And the submit button in the login preview pane should have a "border-color" of "#FF0000"
    When I toggle the checkbox labeled "Name"
    And the checkbox labeled "Name" should be unchecked
    And I replace the text in the "createAccountPageHeadingCopy" field with "Sign up for Insta"
    And I replace the text in the "createAccountPageFooterContents" field with "Some restrictions apply"
    And I should see the following text in the login preview pane
      | TagName               | TextContent             |
      | button                | See login page          |
      | h1                    | Sign up for Insta       |
      | label                 | Email                   |
      | label                 | Password                |
      | p                     | Some restrictions apply |
      | button[type="submit"] | Create                  |
    And I should see the following form elements within the form with ID "createUserForm"
      | TagName | Type     | Name          | Value |
      | input   | email    | form_email    |       |
      | input   | password | form_password |       |
    Given modified login settings for a login app
    When I click the button labeled "Save login settings"
    Then I should see the following text on the page
      | TagName            | TextContent                       |
      | .success-message p | Successfully saved login settings |
    And the button labeled "Save login settings" should be disabled

  @email_settings
  Scenario: edit email settings
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And the following mocked requests:
      | Method | Path                                                                                                     | Status | Body                |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/emailelements                                          | 200    | email_elements.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/smselements                                            | 200    | sms_elements.json   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/apppageparameters/90ffb499-2549-470e-99cd-77f7008e2735 | 200    | pageparameters.json |
    When I navigate to the page with path "/loginapps/90ffb499-2549-470e-99cd-77f7008e2735?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a card with the title "General settings"
    And I should see a card with the title "Login settings"
    And I should see a card with the title "Email settings" and the description "Customize your end-user-facing emails for this application."
    And I should see a card with the title "SMS settings"
    And I should see a card with the title "Advanced"
    And I should see the following text on the page
      | TagName | TextContent           |
      | label   | Choose email template |
      | label   | Sender email          |
      | label   | Sender name           |
      | label   | Subject               |
      | label   | HTML body             |
      | label   | Text body             |
    And I should see the following form elements
      | TagName  | Type  | Name               | Value                                                                                         |
      | select   |       | email_message_type | invite_new                                                                                    |
      | input    | email | sender             | info@userclouds.com                                                                           |
      | input    | text  | sender_name        | UserClouds                                                                                    |
      | input    | text  | subject_template   | Invitation to {{.AppName}}                                                                    |
      | textarea |       | html_template      | <h1>{{.InviterName}} has invited you to {{.AppName}}.</h1>                                    |
      | textarea |       | text_template      | {{.InviterName}} has invited you to {{.AppName}}. {{.InviteText}} Click {{.Link}} to sign up. |
    And I should see a dropdown matching selector "[name='email_message_type']" with the following options
      | Text                   | Value               | Selected |
      | Invite existing user   | invite_existing     |          |
      | Invite new user        | invite_new          | true     |
      | MFA email challenge    | mfa_email_challenge |          |
      | MFA email verification | mfa_email_verify    |          |
      | Passwordless login     | passwordless_login  |          |
      | Reset password         | reset_password      |          |
      | Verify email           | verify_email        |          |
    And the button labeled "Save Email Configuration" should be disabled
    # modify variables in templates
    When I type " from " into the "subject_template" field
    And I click the "InviterName" button associated with the "subject_template" input in the email settings form
    Then the input with the name "subject_template" should have the value "Invitation to {{.AppName}} from {{.InviterName}}"
    When I replace the text in the "text_template" field with "Insta: new invite!"
    And I select "Insta" in the "text_template" field
    And I click the "AppName" button associated with the "text_template" input in the email settings form
    Then the input with the name "text_template" should have the value "{{.AppName}}: new invite!"
    And the button labeled "Save Email Configuration" should be enabled
    And the button labeled "Cancel" within the "Email settings" card should be enabled
    # cancel editing. form should reset
    When I click the button labeled "Cancel" in the "Email settings" card
    Then I should see the following form elements
      | TagName  | Type  | Name               | Value                                                                                         |
      | select   |       | email_message_type | invite_new                                                                                    |
      | input    | email | sender             | info@userclouds.com                                                                           |
      | input    | text  | sender_name        | UserClouds                                                                                    |
      | input    | text  | subject_template   | Invitation to {{.AppName}}                                                                    |
      | textarea |       | html_template      | <h1>{{.InviterName}} has invited you to {{.AppName}}.</h1>                                    |
      | textarea |       | text_template      | {{.InviterName}} has invited you to {{.AppName}}. {{.InviteText}} Click {{.Link}} to sign up. |
    And the button labeled "Save Email Configuration" should be disabled
    And the button labeled "Cancel" within the "Email settings" card should be disabled
    # change template
    When I select the option labeled "Passwordless login" in the dropdown matching selector "[name='email_message_type']"
    Then I should see the following form elements
      | TagName  | Type  | Name               | Value                                                                                                                                     |
      | select   |       | email_message_type | passwordless_login                                                                                                                        |
      | input    | email | sender             | info@userclouds.com                                                                                                                       |
      | input    | text  | sender_name        | UserClouds                                                                                                                                |
      | input    | text  | subject_template   | Login Request                                                                                                                             |
      | textarea |       | html_template      | <h1>Login Request to {{.AppName}}</h1><p>Use code {{.Code}} to complete your sign in or click <a href="{{.Link}}">here</a> to sign in</p> |
      | textarea |       | text_template      | Confirm your email address to sign in to {{.AppName}}. Use the code {{.Code}} or click {{.Link}} to sign in                               |
    And the button labeled "Save Email Configuration" should be disabled
    And the button labeled "Cancel" within the "Email settings" card should be disabled
    # modify variables
    When I type " from " into the "subject_template" field
    And I click the "AppName" button associated with the "subject_template" input in the email settings form
    Then the input with the name "subject_template" should have the value "Login Request from {{.AppName}}"
    When I replace the text in the "text_template" field with "Insta: confirm your email address to sign in."
    And I select "Insta" in the "text_template" field
    And I click the "AppName" button associated with the "text_template" input in the email settings form
    Then the input with the name "text_template" should have the value "{{.AppName}}: confirm your email address to sign in."
    And the button labeled "Save Email Configuration" should be enabled
    And the button labeled "Cancel" within the "Email settings" card should be enabled
    # save and get error
    Given the following mocked requests:
      | Method | Path                                                            | Status | Body                                                                  |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/emailelements | 400    | {"error":"uh-oh","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
    When I click the button labeled "Save Email Configuration"
    Then I should see the following text on the page
      | TagName                         | TextContent                               |
      | #emailSettings .alert-message p | Error saving tenant email elements: uh-oh |
    And I should see the following form elements
      | TagName  | Type  | Name               | Value                                                                                                                                     |
      | select   |       | email_message_type | passwordless_login                                                                                                                        |
      | input    | email | sender             | info@userclouds.com                                                                                                                       |
      | input    | text  | sender_name        | UserClouds                                                                                                                                |
      | input    | text  | subject_template   | Login Request from {{.AppName}}                                                                                                           |
      | textarea |       | html_template      | <h1>Login Request to {{.AppName}}</h1><p>Use code {{.Code}} to complete your sign in or click <a href="{{.Link}}">here</a> to sign in</p> |
      | textarea |       | text_template      | {{.AppName}}: confirm your email address to sign in.                                                                                      |
    And the button labeled "Save Email Configuration" should be enabled
    And the button labeled "Cancel" within the "Email settings" card should be enabled
    # successful save
    Given modified email settings for a login app
    When I click the button labeled "Save Email Configuration"
    Then I should see the following text on the page
      | TagName                           | TextContent                 |
      | #emailSettings .success-message p | Changes successfully saved! |
    And I should not see an element matching selector "#emailSettings .alert-message"
    And I should see the following form elements
      | TagName  | Type  | Name               | Value                                                                                                                                     |
      | select   |       | email_message_type | passwordless_login                                                                                                                        |
      | input    | email | sender             | info@userclouds.com                                                                                                                       |
      | input    | text  | sender_name        | UserClouds                                                                                                                                |
      | input    | text  | subject_template   | Login Request from {{.AppName}}                                                                                                           |
      | textarea |       | html_template      | <h1>Login Request to {{.AppName}}</h1><p>Use code {{.Code}} to complete your sign in or click <a href="{{.Link}}">here</a> to sign in</p> |
      | textarea |       | text_template      | {{.AppName}}: confirm your email address to sign in.                                                                                      |
    And the button labeled "Save Email Configuration" should be disabled
    And the button labeled "Cancel" within the "Email settings" card should be disabled

  @sms_settings
  Scenario: edit SMS settings
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And the following mocked requests:
      | Method | Path                                                                                                     | Status | Body                |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/emailelements                                          | 200    | email_elements.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/smselements                                            | 200    | sms_elements.json   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/apppageparameters/90ffb499-2549-470e-99cd-77f7008e2735 | 200    | pageparameters.json |
    When I navigate to the page with path "/loginapps/90ffb499-2549-470e-99cd-77f7008e2735?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a card with the title "General settings"
    And I should see a card with the title "Login settings"
    And I should see a card with the title "Email settings" and the description "Customize your end-user-facing emails for this application."
    And I should see a card with the title "SMS settings" and the description "Customize your end-user-facing SMS messages for this application."
    And I should see a card with the title "Advanced"
    And I should see the following text on the page
      | TagName             | TextContent            |
      | label               | Choose SMS template    |
      | label               | Sender phone number    |
      | label               | Message body           |
      | button              | Save SMS Configuration |
      | #smsSettings button | Cancel                 |
    And I should see the following form elements
      | TagName  | Type | Name              | Value                                                                              |
      | select   |      | SMS_message_type  | sms_mfa_challenge                                                                  |
      | input    | text | sms_sender        | +1111111111                                                                        |
      | textarea |      | sms_body_template | Login Request to {{.AppName}}. Use code {{.Code}} to complete the sign in process. |
    And I should see a dropdown matching selector "[name='SMS_message_type']" with the following options
      | Text                 | Value             | Selected |
      | MFA SMS challenge    | sms_mfa_challenge | true     |
      | MFA SMS verification | sms_mfa_verify    |          |
    And the button labeled "Save SMS Configuration" should be disabled
    # modify variables in templates
    When I replace the text in the "sms_sender" field with "+15555552343"
    And I replace the text in the "sms_body_template" field with "Log in to Insta with code CODE"
    And I select "Insta" in the "sms_body_template" field
    And I click the "AppName" button associated with the "sms_body_template" input in the SMS settings form
    And I select "CODE" in the "sms_body_template" field
    And I click the "Code" button associated with the "sms_body_template" input in the SMS settings form
    Then the input with the name "sms_body_template" should have the value "Log in to {{.AppName}} with code {{.Code}}"
    And the input with the name "sms_sender" should have the value "+15555552343"
    And the button labeled "Save SMS Configuration" should be enabled
    And the button labeled "Cancel" within the "SMS settings" card should be enabled
    # cancel editing. form should reset
    When I click the button labeled "Cancel" in the "SMS settings" card
    Then I should see the following form elements
      | TagName  | Type | Name              | Value                                                                              |
      | select   |      | SMS_message_type  | sms_mfa_challenge                                                                  |
      | input    | text | sms_sender        | +1111111111                                                                        |
      | textarea |      | sms_body_template | Login Request to {{.AppName}}. Use code {{.Code}} to complete the sign in process. |
    And the button labeled "Save SMS Configuration" should be disabled
    And the button labeled "Cancel" within the "SMS settings" card should be disabled
    # change template
    When I select the option labeled "MFA SMS verification" in the dropdown matching selector "[name='SMS_message_type']"
    Then I should see the following form elements
      | TagName  | Type | Name              | Value                                                                                                        |
      | select   |      | SMS_message_type  | sms_mfa_verify                                                                                               |
      | input    | text | sms_sender        | +1111111111                                                                                                  |
      | textarea |      | sms_body_template | MFA Verification Request for {{.AppName}}. Use code {{.Code}} to confirm your phone number for use with MFA. |
    And the button labeled "Save SMS Configuration" should be disabled
    And the button labeled "Cancel" within the "Email settings" card should be disabled
    # modify variables in templates
    When I replace the text in the "sms_sender" field with "+15555552343"
    And I replace the text in the "sms_body_template" field with "Verify your phone number with Insta. Use code CODE"
    And I select "Insta" in the "sms_body_template" field
    And I click the "AppName" button associated with the "sms_body_template" input in the SMS settings form
    And I select "CODE" in the "sms_body_template" field
    And I click the "Code" button associated with the "sms_body_template" input in the SMS settings form
    Then the input with the name "sms_body_template" should have the value "Verify your phone number with {{.AppName}}. Use code {{.Code}}"
    And the input with the name "sms_sender" should have the value "+15555552343"
    And the button labeled "Save SMS Configuration" should be enabled
    And the button labeled "Cancel" within the "SMS settings" card should be enabled
    # save and get error
    Given the following mocked requests:
      | Method | Path                                                          | Status | Body                                                                  |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/smselements | 400    | {"error":"uh-oh","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
    When I click the button labeled "Save SMS Configuration"
    Then I should see the following text on the page
      | TagName                       | TextContent |
      | #smsSettings .alert-message p | uh-oh       |
    And I should see the following form elements
      | TagName  | Type | Name              | Value                                                          |
      | select   |      | SMS_message_type  | sms_mfa_verify                                                 |
      | input    | text | sms_sender        | +15555552343                                                   |
      | textarea |      | sms_body_template | Verify your phone number with {{.AppName}}. Use code {{.Code}} |
    And the button labeled "Save SMS Configuration" should be enabled
    And the button labeled "Cancel" within the "SMS settings" card should be enabled
    # successful save
    Given modified SMS settings for a login app
    When I click the button labeled "Save SMS Configuration"
    Then I should see the following text on the page
      | TagName                         | TextContent                 |
      | #smsSettings .success-message p | Changes successfully saved! |
    And I should not see an element matching selector "#smsSettings .alert-message"
    And I should see the following form elements
      | TagName  | Type | Name              | Value                                                          |
      | select   |      | SMS_message_type  | sms_mfa_verify                                                 |
      | input    | text | sms_sender        | +15555552343                                                   |
      | textarea |      | sms_body_template | Verify your phone number with {{.AppName}}. Use code {{.Code}} |
    And the button labeled "Save SMS Configuration" should be disabled
    And the button labeled "Cancel" within the "SMS settings" card should be disabled
