Feature: oauth connections list

  @a11y
  Scenario: oauth connections list accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I navigate to the page with path "/oauthconnections?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: view oauth connections list
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I navigate to the page with path "/oauthconnections?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a table with ID "oauthConnections" and the following data
      | Type                             | Provider Name | Provider URL                | Configured |
      | Facebook black and white version | facebook      | https://www.facebook.com    | Yes        |
      | Google black and white version   | google        | https://accounts.google.com | Yes        |
      | LinkedIn black and white version | linkedin      | https://www.linkedin.com    | Yes        |
    And I should see a button labeled "Create Provider"
    And I should see a link to "/oauthconnections/oidc_provider/facebook?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/oauthconnections/oidc_provider/google?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/oauthconnections/oidc_provider/linkedin?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"

  Scenario: add oauth connections on oauth connections list
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I navigate to the page with path "/oauthconnections?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a card with the title "OAuth Connections"
    And I should see a table with ID "oauthConnections" and the following data
      | Type                             | Provider Name | Provider URL                | Configured |
      | Facebook black and white version | facebook      | https://www.facebook.com    | Yes        |
      | Google black and white version   | google        | https://accounts.google.com | Yes        |
      | LinkedIn black and white version | linkedin      | https://www.linkedin.com    | Yes        |
    And I should see a link to "/oauthconnections/oidc_provider/facebook?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/oauthconnections/oidc_provider/google?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/oauthconnections/oidc_provider/linkedin?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a button labeled "Create Provider"
    Given a mocked "GET" request for "plex_config"
    When I click the button labeled "Create Provider"
    #handle navigation. the rest of the tests are in authn_oidc.feature
    Then I should see a card with the title "OIDC Connection"
