@tenants
@create_tenant
Feature: Create tenant page

  Scenario: Creating a tenant for a company with no tenants
    Given I am a logged-in user
    And the following mocked requests:
      | Method | Path                                                        | Status | Body |
      | GET    | /api/companies/1ee4497e-c326-4068-94ed-3dcdaaaa53bc/tenants | 200    | []   |
    When I navigate to the page with path "/"
    Then I should see a "p" with the text "This company doesn't have any tenants yet. You can create one now."
    And I should see the following text on the page
      | TagName | TextContent             |
      | h1      | Tenant Home: No tenants |
    And I should see a card with the title "Tenant details"
    And the page title should be "[dev] UserClouds Console"
    Given the following mocked requests:
      | Method | Path                                                        | Status | Body |
      | GET    | /api/companies/1ee4497e-c326-4068-94ed-3dcdaaaa53bc/tenants | 200    | []   |
    # when company_id changes in the querystring, we re-fetch flags
    And a mocked response for no enabled feature flags
    When I navigate to the page with path "/tenants/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc"
    Then I should see the following text on the page
      | TagName | TextContent      |
      | h1      | Add a new tenant |
    # Playwright suggests mocking in reverse order, with a fallback
    # POST happens first, then GET
    #
    # Tenant is ready
    Given a mocked "GET" request for "a single tenant"
    # Tenant is being provisioned
    And a mocked "GET" request for "a single tenant being created"
    # Tenant is created but authz edges haven't been written
    And a mocked "GET" request for "a single tenant being created" that returns a "403"
    # Tenant hasn't yet even been saved
    And a mocked "GET" request for "a single tenant being created" that returns a "404"
    And a mocked "POST" request for "tenants"
    # when tenant_id changes in the querystring, we re-fetch flags
    And a mocked response for no enabled feature flags
    When I type "Foo" into the "name" field
    And I click the button labeled "Create"
    Then I should see the following text on the page
      | TagName      | TextContent                           |
      | h1           | Tenant Home: Foo                      |
      | section > h1 | Welcome to UserClouds, Kyle Jacobson! |
    And I should be on the page with the path "/"


# TODO: use dropdown