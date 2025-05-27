@comms_channels
Feature: comms channel page

  @a11y
  Scenario: edit telephony details on comms channel page accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I navigate to the page with path "/commschannels?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: edit telephony details on comms channel page
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I navigate to the page with path "/commschannels?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading telephony…"
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following inputs
      | Type | Name                                  | Value | Disabled |
      | text | telephony_provider_twilio_account_sid |       | false    |
      | text | telephony_provider_twilio_api_key_sid |       | false    |
      | text | telephony_provider_twilio_api_secret  |       | false    |
    When I type "my sid" into the input with ID "telephony_provider_twilio_account_sid"
    And I type "my api sid" into the input with ID "telephony_provider_twilio_api_key_sid"
    And I type "my api secret" into the input with ID "telephony_provider_twilio_api_secret"
    Then I should see the following inputs
      | Type | Name                                  | Value         | Disabled |
      | text | telephony_provider_twilio_account_sid | my sid        | false    |
      | text | telephony_provider_twilio_api_key_sid | my api sid    | false    |
      | text | telephony_provider_twilio_api_secret  | my api secret | false    |
    Given a mocked "POST" request for "plex_config"
    When I click the button labeled "Save"
    Then I should see a "p" with the text "Successfully updated telephony provider"

  Scenario: edit telephony details error on comms channel page
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I navigate to the page with path "/commschannels?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading telephony…"
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following inputs
      | Type | Name                                  | Value | Disabled |
      | text | telephony_provider_twilio_account_sid |       | false    |
      | text | telephony_provider_twilio_api_key_sid |       | false    |
      | text | telephony_provider_twilio_api_secret  |       | false    |
    When I type "0=0; DROP TABLE Users;" into the input with ID "telephony_provider_twilio_account_sid"
    And I type "my api sid" into the input with ID "telephony_provider_twilio_api_key_sid"
    And I type "my api secret" into the input with ID "telephony_provider_twilio_api_secret"
    Then I should see the following inputs
      | Type | Name                                  | Value                  | Disabled |
      | text | telephony_provider_twilio_account_sid | 0=0; DROP TABLE Users; | false    |
      | text | telephony_provider_twilio_api_key_sid | my api sid             | false    |
      | text | telephony_provider_twilio_api_secret  | my api secret          | false    |
    Given the following mocked requests:
      | Method | Path                                                         | Status | Body                       |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/plexconfig | 400    | {"error":"bad table name"} |
    When I click the button labeled "Save"
    Then I should see 2 elements matching selector "p:has-text('bad table name')"

  Scenario: edit email settings details on comms channel page
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I navigate to the page with path "/commschannels?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading email settings..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following inputs
      | Type     | Name           | Value | Disabled |
      | text     | email_host     |       | false    |
      | number   | email_port     |       | false    |
      | text     | email_username |       | false    |
      | password | email_password |       | false    |
    When I type "my host" into the input with ID "email_host"
    And I type "123" into the input with ID "email_port"
    And I type "my_un" into the input with ID "email_username"
    And I type "my_pass" into the input with ID "email_password"
    Then I should see the following inputs
      | Type     | Name           | Value   | Disabled |
      | text     | email_host     | my host | false    |
      | number   | email_port     | 123     | false    |
      | text     | email_username | my_un   | false    |
      | password | email_password | my_pass | false    |
    Given a mocked "POST" request for "plex_config"
    When I click the button labeled "Save email settings"
    Then I should see a "p" with the text "Successfully updated email settings"

  Scenario: edit email settings details error on comms channel page
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I navigate to the page with path "/commschannels?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading email settings..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following inputs
      | Type     | Name           | Value | Disabled |
      | text     | email_host     |       | false    |
      | number   | email_port     |       | false    |
      | text     | email_username |       | false    |
      | password | email_password |       | false    |
    When I type "0=0; DROP TABLE Users;" into the input with ID "email_host"
    And I type "123" into the input with ID "email_port"
    And I type "my_un" into the input with ID "email_username"
    And I type "my_pass" into the input with ID "email_password"
    Then I should see the following inputs
      | Type     | Name           | Value                  | Disabled |
      | text     | email_host     | 0=0; DROP TABLE Users; | false    |
      | number   | email_port     | 123                    | false    |
      | text     | email_username | my_un                  | false    |
      | password | email_password | my_pass                | false    |
    Given the following mocked requests:
      | Method | Path                                                         | Status | Body                           |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/plexconfig | 400    | {"error":"bad email settings"} |
    When I click the button labeled "Save email settings"
    Then I should see 2 elements matching selector "p:has-text('bad email settings')"

  Scenario: view JWT keys on comms channel page
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I navigate to the page with path "/commschannels?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent                |
      | p       | -----BEGIN PUBLIC KEY----- |
    And I should see the following text on the page
      | TagName | TextContent                                                      |
      | p       | MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA61hdA8fRvjRynndI/oXc |
    And I should see a button labeled "Rotate Keys"
    And I should see a button labeled "Download Private Key"

  Scenario: rotate JWT keys on comms channel page
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I navigate to the page with path "/commschannels?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent                                                      |
      | p       | -----BEGIN PUBLIC KEY-----                                       |
      | p       | MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA61hdA8fRvjRynndI/oXc |
    Given the following mocked requests:
      | Method | Path                                                                  | Status | Body               |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/keys/actions/rotate | 204    | {}                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/keys                | 200    | authn_new_key.json |
    When I click the button labeled "Rotate Keys"
    Then I should see the following text on the page
      | TagName | TextContent                                                     |
      | p       | -----BEGIN PUBLIC KEY-----                                      |
      | p       | MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA7rywZBh7I+txo1Bk7OF |
    And I should see a button labeled "Rotate Keys"
    And I should see a button labeled "Download Private Key"
