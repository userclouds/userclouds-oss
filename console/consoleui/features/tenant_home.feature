@home_page
@tenants
Feature: Tenant Home Page

  @a11y
  Scenario: Basic info tenant accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    When I navigate to the page with path "/"
    Then the page should have no accessibility violations

  Scenario: Basic info tenant
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    When I navigate to the page with path "/"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName      | TextContent                                                          |
      | h1           | Tenant Home: Console - Dev                                           |
      | section > h1 | Welcome to UserClouds, Kyle Jacobson!                                |
      | section > h2 | How are you keeping users' data safe?Use the side menu to get going. |
    And I should see a card with the title "Tenant Details"
    And I should see a "span" with the text "Edit Tenant"
    And I should see the following text within the "Tenant Details" card
      | TagName         | TextContent                                          |
      | label           | ID                                                   |
      | label + div > p | 41ab79a8-0dff-418e-9d42-e1694469120a                 |
      | label           | URL                                                  |
      | label + div > p | https://console-dev.tenant.dev.userclouds.tools:3333 |
    # we should test that clicking these buttons works, but browser permissions
    # make this hard or impossible
    And I should see an icon button with the title "Copy tenant ID to clipboard" within the "Tenant Details" card
    And I should see an icon button with the title "Copy tenant URL to clipboard" within the "Tenant Details" card
    And I should see a card with the title "Codegen SDKs"
    And I should see a button labeled "Download Go SDK" within the "Codegen SDKs" card
    And the button labeled "Download Go SDK" within the "Codegen SDKs" card should be enabled
    And I should see a button labeled "Download Python SDK" within the "Codegen SDKs" card
    And the button labeled "Download Python SDK" within the "Codegen SDKs" card should be enabled
    And I should see a card with the title "Resources"
    And I should see the following text within the "Resources" card
      | TagName     | TextContent   |
      | ul > li > a | Documentation |
      | ul > li > a | API reference |
      | ul > li > a | Blog          |

  Scenario: redirect to login on 401
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    When I navigate to the page with path "/"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName      | TextContent                           |
      | h1           | Tenant Home: Console - Dev            |
      | section > h1 | Welcome to UserClouds, Kyle Jacobson! |
    Given I am an unauthenticated user
    And a mocked request for the auth redirect page
    When I reload the page
    Then I should be navigated to the page with the path "/auth/redirect"
