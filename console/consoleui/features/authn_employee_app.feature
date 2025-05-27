@authn
@plex
@employee_app
Feature: employee app page

  @a11y
  Scenario: edit application settings employee app
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    When I navigate to the page with path "/loginapps/plex_employee_app?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  @general_settings
  Scenario: edit application settings employee app
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    When I navigate to the page with path "/loginapps/plex_employee_app?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a "h1" with the text "General settings"
    And I should see a card with the title "General settings"
    And the button labeled "Save" within the "General settings" card should be disabled
    When I click to expand the "Application Settings" accordion
    Then I should see the following text on the page
      | TagName | TextContent       |
      | span    | 6ed148…           |
      | p       | Employee Plex App |
    And I should see the following inputs
      | Type | Name          | Value                                |
      | text | client_id     | 6fa61fbf-f572-4178-aefd-111ba560da6c |
      | text | client_secret | 755921f2-e8c0-41ce-8a12-068bb9ed863b |
    When I type "id" into the input with ID "client_id"
    And I type "secret" into the input with ID "client_secret"
    Then the button labeled "Save" within the "General settings" card should be enabled
    And I should see the following inputs
      | Type | Name          | Value                                      |
      | text | client_id     | id6fa61fbf-f572-4178-aefd-111ba560da6c     |
      | text | client_secret | secret755921f2-e8c0-41ce-8a12-068bb9ed863b |
    When I click to expand the "Allowed Redirect URLs" accordion
    Then the button labeled "Add Redirect URL" should be enabled
    When I click to expand the "Allowed Logout URLs" accordion
    Then the button labeled "Add Logout URL" should be enabled
    Given the following mocked requests:
      | Method | Path                                                         | Status | Body            |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/plexconfig | 200    | plexconfig.json |
    When I click the button labeled "Save"
    Then I should see the following text on the page
      | TagName | TextContent                             |
      | p       | Successfully updated employee login app |
    And the button labeled "Save" within the "General settings" card should be disabled

  Scenario: edit allowed redirect urls employee app
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    When I navigate to the page with path "/loginapps/plex_employee_app?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a "h1" with the text "General settings"
    And I should see a card with the title "General settings"
    And the button labeled "Save" within the "General settings" card should be disabled
    When I click to expand the "Allowed Redirect URLs" accordion
    Then the button labeled "Add Redirect URL" should be enabled
    When I click the button labeled "Add Redirect URL"
    And I click the button labeled "Add Redirect URL"
    And I change the text in row 1 of the editable list with ID "allowedRedirectURIs" to "https://foo1.com"
    And I change the text in row 2 of the editable list with ID "allowedRedirectURIs" to "https://foo2.com"
    Then the values in the editable list with ID "allowedRedirectURIs" should be
      | Value            |
      | https://foo1.com |
      | https://foo2.com |
    When I click the delete icon next to row 2 of the editable list with ID "allowedRedirectURIs"
    Then the values in the editable list with ID "allowedRedirectURIs" should be
      | Value            |
      | https://foo1.com |
    Given the following mocked requests:
      | Method | Path                                                         | Status | Body            |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/plexconfig | 200    | plexconfig.json |
    When I click the button labeled "Save"
    Then I should see the following text on the page
      | TagName | TextContent                             |
      | p       | Successfully updated employee login app |
    And the button labeled "Save" within the "General settings" card should be disabled

  Scenario: edit allowed logout urls employee app
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    When I navigate to the page with path "/loginapps/plex_employee_app?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a "h1" with the text "General settings"
    And I should see a card with the title "General settings"
    And the button labeled "Save" within the "General settings" card should be disabled
    When I click to expand the "Allowed Logout URLs" accordion
    Then the button labeled "Add Logout URL" should be enabled
    When I click the button labeled "Add Logout URL"
    And I click the button labeled "Add Logout URL"
    And I change the text in row 1 of the editable list with ID "allowedLogoutURIs" to "https://foo1.com"
    And I change the text in row 2 of the editable list with ID "allowedLogoutURIs" to "https://foo2.com"
    Then the values in the editable list with ID "allowedLogoutURIs" should be
      | Value            |
      | https://foo1.com |
      | https://foo2.com |
    When I click the delete icon next to row 2 of the editable list with ID "allowedLogoutURIs"
    Then the values in the editable list with ID "allowedLogoutURIs" should be
      | Value            |
      | https://foo1.com |
    Given the following mocked requests:
      | Method | Path                                                         | Status | Body            |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/plexconfig | 200    | plexconfig.json |
    When I click the button labeled "Save"
    Then I should see the following text on the page
      | TagName | TextContent                             |
      | p       | Successfully updated employee login app |
    And the button labeled "Save" within the "General settings" card should be disabled

  Scenario: save error employee app
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    When I navigate to the page with path "/loginapps/plex_employee_app?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a "h1" with the text "General settings"
    And I should see a card with the title "General settings"
    And the button labeled "Save" within the "General settings" card should be disabled
    When I click to expand the "Allowed Logout URLs" accordion
    Then the button labeled "Add Logout URL" should be enabled
    When I click the button labeled "Add Logout URL"
    And I click the button labeled "Add Logout URL"
    And I change the text in row 1 of the editable list with ID "allowedLogoutURIs" to "--drop table users;"
    And I change the text in row 2 of the editable list with ID "allowedLogoutURIs" to "https://foo2.com"
    Then the values in the editable list with ID "allowedLogoutURIs" should be
      | Value               |
      | --drop table users; |
      | https://foo2.com    |
    When I click the delete icon next to row 2 of the editable list with ID "allowedLogoutURIs"
    Then the values in the editable list with ID "allowedLogoutURIs" should be
      | Value               |
      | --drop table users; |
    Given the following mocked requests:
      | Method | Path                                                         | Status | Body                                    |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/plexconfig | 409    | {"error":"stop sql injecting on tests"} |
    When I click the button labeled "Save"
    Then I should see the following text on the page
      | TagName | TextContent                 |
      | p       | stop sql injecting on tests |
    And the button labeled "Save" within the "General settings" card should be enabled
