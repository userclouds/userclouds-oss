Feature: identity providers list

  @a11y
  Scenario: view and add identity providers list accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I navigate to the page with path "/identityproviders?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: view and add identity providers list
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I navigate to the page with path "/identityproviders?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a table with ID "identityProviders" and the following data
      | Identity Provider    | Type       |
      | UC IDP Dev (Console) | UserClouds |
      | New auth0 provider   | Auth0      |
    And I should see a dropdown matching selector "[name='active_provider']" with the following options
      | Text                 | Value                                | Selected |
      | UC IDP Dev (Console) | a83f8eed-0b5e-4f3f-bcff-ad695d502849 | true     |
      | New auth0 provider   | 0cf72991-f1bd-4146-a8e5-5a73791595b2 |          |
    And I should see a button labeled "Create Provider"
    And I should see a button labeled "Save providers"
    And the button labeled "Save providers" should be disabled
    When I click the button labeled "Create Provider"
    And I should see a table with ID "identityProviders" and the following data
      | Identity Provider    | Type       |
      | UC IDP Dev (Console) | UserClouds |
      | New auth0 provider   | Auth0      |
      | New Plex Provider    | UserClouds |
    And I should see a dropdown matching selector "[name='active_provider']" with the following options
      | Text                 | Value                                | Selected |
      | UC IDP Dev (Console) | a83f8eed-0b5e-4f3f-bcff-ad695d502849 | true     |
      | New auth0 provider   | 0cf72991-f1bd-4146-a8e5-5a73791595b2 |          |
      | New Plex Provider    | *                                    |          |
