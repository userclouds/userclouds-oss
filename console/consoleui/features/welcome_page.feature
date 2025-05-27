@welcome
Feature: Welcome page

  Scenario: Creating a company and its first tenant
    Given a mocked "GET" request for "UC admin service info"
    And a mocked "GET" request for "userinfo"
    And the following mocked requests:
      | Method | Path           | Status | Body |
      | GET    | /api/companies | 200    | []   |
    When I navigate to the page with path "/"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a card with the title "Welcome to UserClouds, Kyle Jacobson!" and the description "Let's get you set up with a new company."
    Given a mocked "POST" request for "companies"
    # Playwright suggests mocking in reverse order, with a fallback
    # POST happens first, then GET
    And a mocked "GET" request for "a list containing one tenant"
    And a mocked "GET" request for "selected new tenant"
    And a mocked "POST" request for "tenants"
    And a mocked "GET" request for "serviceinfo"
    And a mocked "GET" request for "userinfo"
    And a mocked "GET" request for "companies"
    And a mocked "GET" request for "tenants_urls"
    # when company_id or tenant_id changes in the querystring, we re-fetch flags
    And a mocked response for no enabled feature flags
    When I type "UserClouds Dev" into the "company_name" field
    And I type "Foo" into the "tenant_name" field
    And I click the button labeled "Let's go!"
    Then I should see the following text on the page
      | TagName      | TextContent                           |
      | h1           | Tenant Home: Foo                      |
      | section > h1 | Welcome to UserClouds, Kyle Jacobson! |
    And I should be on the page with the path "/"
