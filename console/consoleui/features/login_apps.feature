@authn
@login_apps
Feature: Authentication Page

  @a11y
  Scenario: view login apps list accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I wait for the network to be idle
    When I navigate to the page with path "/loginapps?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: view login apps list
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I wait for the network to be idle
    When I navigate to the page with path "/loginapps?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a link to "/loginapps/90ffb499-2549-470e-99cd-77f7008e2735?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "loginApps" and the following data
      | Name                     | Grant types                                           |
      | UserClouds Console (dev) | authorization_code, refresh_token, client_credentials |
      | Login for New Company    | refresh_token                                         |
      | Login for Hank Co        | authorization_code, refresh_token, client_credentials |
      | Login for Palmetto       |                                                       |
      | Login for AllBirds       | authorization_code, refresh_token                     |
      | Login for Contoso        | authorization_code, client_credentials                |
      | Login for Settings Check | client_credentials                                    |
      | Employee App             |                                                       |
    And I should see a button labeled "Create App"

  Scenario: add login app from login apps list
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    And an additional login app
    When I wait for the network to be idle
    When I navigate to the page with path "/loginapps?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a link to "/loginapps/90ffb499-2549-470e-99cd-77f7008e2735?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "loginApps" and the following data
      | Name                     | Grant types                                           |
      | UserClouds Console (dev) | authorization_code, refresh_token, client_credentials |
      | Login for New Company    | refresh_token                                         |
      | Login for Hank Co        | authorization_code, refresh_token, client_credentials |
      | Login for Palmetto       |                                                       |
      | Login for AllBirds       | authorization_code, refresh_token                     |
      | Login for Contoso        | authorization_code, client_credentials                |
      | Login for Settings Check | client_credentials                                    |
      | Employee App             |                                                       |
    And I should see a button labeled "Create App"
    When I click the button labeled "Create App"
    Then I should see a card with the title "General settings"
