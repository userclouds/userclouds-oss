@organizations
@create_organization
Feature: create organizations page

  @a11y
  Scenario: creating org accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I navigate to the page with path "/organizations/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: error creating org
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I navigate to the page with path "/organizations/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName          | TextContent         |
      | button[disabled] | Create Organization |
    And I should see a cardrow with the title "Basic Details"
    Then I should see the following form elements
      | TagName | Type | Name              | Value |
      | input   | text | organization_name |       |
    And I should see a dropdown matching selector "[name='organization_region']" with the following options
      | Text          | Value         | Selected |
      | aws-us-west-2 | aws-us-west-2 |          |
      | aws-us-east-1 | aws-us-east-1 | true     |
    When I replace the text in the "organization_name" field with "First Org"
    Then I should see the following form elements
      | TagName | Type | Name              | Value     |
      | input   | text | organization_name | First Org |
    And the button labeled "Create Organization" should be enabled
    Given the following mocked requests:
      | Method | Path                                                             | Status | Body                                                                  |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations  | 500    | {"error":"uh-oh","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations* | 200    | organizations_page_1.json                                             |
    When I click the button labeled "Create Organization"
    Then I should see the following text on the page
      | TagName | TextContent |
      | p       | uh-oh       |
    And I should be on the page with the path "/organizations/create"

  Scenario: success creating org
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I navigate to the page with path "/organizations/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName          | TextContent         |
      | button[disabled] | Create Organization |
    And I should see a cardrow with the title "Basic Details"
    Then I should see the following form elements
      | TagName | Type | Name              | Value |
      | input   | text | organization_name |       |
    And I should see a dropdown matching selector "[name='organization_region']" with the following options
      | Text          | Value         | Selected |
      | aws-us-west-2 | aws-us-west-2 |          |
      | aws-us-east-1 | aws-us-east-1 | true     |
    Given the following mocked requests:
      | Method | Path                                                              | Status | Body                      |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations   | 200    | single_organization.json  |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations?* | 200    | organizations_page_1.json |
    When I replace the text in the "organization_name" field with "First Org"
    And I select the option labeled "aws-us-east-1" in the dropdown matching selector "[name='organization_region']"
    Then I should see the following form elements
      | TagName | Type | Name              | Value     |
      | input   | text | organization_name | First Org |
    And I should see a dropdown matching selector "[name='organization_region']" with the following options
      | Text          | Value         | Selected |
      | aws-us-west-2 | aws-us-west-2 |          |
      | aws-us-east-1 | aws-us-east-1 | true     |
    And the button labeled "Create Organization" should be enabled
    When I click the button labeled "Create Organization"
    Then I should see the following text on the page
      | TagName                  | TextContent                                   |
      | #notificationCenter > li | Organization "First Org" successfully created |
    And I should be on the page with the path "/organizations/72157ddd-60db-4fcf-a124-2b14098bc091"
