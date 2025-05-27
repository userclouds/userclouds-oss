@organizations
@org_details
Feature: organization details page

  @a11y
  Scenario: basic info organization accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema"
    And the following mocked requests:
      | Method | Path                                                              | Status | Body                              |
      | GET    | /api/tenants/*/organizations/72157ddd-60db-4fcf-a124-2b14098bc091 | 200    | single_organization.json          |
      | GET    | /api/tenants/*/users?*                                            | 200    | { "data": [], "has_next": false } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/loginapps?*     | 200    | []                                |
    When I navigate to the page with path "/organizations/72157ddd-60db-4fcf-a124-2b14098bc091?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: basic info organization
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema"
    And the following mocked requests:
      | Method | Path                                                              | Status | Body                              |
      | GET    | /api/tenants/*/organizations/72157ddd-60db-4fcf-a124-2b14098bc091 | 200    | single_organization.json          |
      | GET    | /api/tenants/*/users?*                                            | 200    | { "data": [], "has_next": false } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/loginapps?*     | 200    | []                                |
    When I navigate to the page with path "/organizations/72157ddd-60db-4fcf-a124-2b14098bc091?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading ..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent       |
      | h2      | Basic Details     |
      | button  | Edit Organization |
    And I should see a cardrow with the title "Basic Details"
    And I should see a cardrow with the title "Login Apps"
    And I should see a cardrow with the title "Users"
    And I should see the following text on the page
      | TagName            | TextContent   |
      | label              | ID            |
      | label > div > span | 72157d…       |
      | label              | Name          |
      | label > div > p    | First Org     |
      | label              | Region        |
      | label > div > p    | aws-us-east-1 |

  Scenario: edit organization name
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema"
    And the following mocked requests:
      | Method | Path                                                              | Status | Body                              |
      | GET    | /api/tenants/*/organizations/72157ddd-60db-4fcf-a124-2b14098bc091 | 200    | single_organization.json          |
      | GET    | /api/tenants/*/users?*                                            | 200    | { "data": [], "has_next": false } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/loginapps?*     | 200    | []                                |
    When I navigate to the page with path "/organizations/72157ddd-60db-4fcf-a124-2b14098bc091?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading ..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent       |
      | h2      | Basic Details     |
      | button  | Edit Organization |
      | p       | First Org         |
    And I should see a cardrow with the title "Basic Details"
    And I should see a cardrow with the title "Login Apps"
    And I should see a cardrow with the title "Users"
    When I click the button labeled "Edit Organization"
    Then I should not see a button labeled "Edit Organization"
    And the button labeled "Save Organization" should be disabled
    And the button labeled "Cancel" should be enabled
    And I should see the following text on the page
      | TagName            | TextContent   |
      | label              | ID            |
      | label > div > span | 72157d…       |
      | label              | Region        |
      | label > div > p    | aws-us-east-1 |
    And I should see the following form elements
      | TagName | Type | Name              | Value     |
      | input   | text | organization_name | First Org |
    # enter an empty name
    When I replace the text in the "organization_name" field with ""
    Then the button labeled "Save Organization" should be disabled
    When I replace the text in the "organization_name" field with "Second Org"
    Then the button labeled "Save Organization" should be enabled
    # error for no particular reason
    # the asterisk here is just to save space
    Given the following mocked requests:
      | Method | Path                                                              | Status | Body                                                                  |
      | PUT    | /api/tenants/*/organizations/72157ddd-60db-4fcf-a124-2b14098bc091 | 500    | {"error":"uh-oh","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
    When I click the button labeled "Save Organization"
    Then I should see the following text on the page
      | TagName | TextContent |
      | p       | uh-oh       |
    # try again
    Given the following mocked requests:
      | Method | Path                                                              | Status | Body                     |
      | PUT    | /api/tenants/*/organizations/72157ddd-60db-4fcf-a124-2b14098bc091 | 200    | single_organization.json |
    When I click the button labeled "Save Organization"
    And I should see the following text on the page
      | TagName            | TextContent       |
      | label              | ID                |
      | label > div > span | 72157d…           |
      | label              | Name              |
      | label > div > p    | First Org         |
      | label              | Region            |
      | label > div > p    | aws-us-east-1     |
      | button             | Edit Organization |
    Then I should not see a button labeled "Save Organization"
    Then I should not see a button labeled "Cancel"

  Scenario: view login apps organization
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema"
    And the following mocked requests:
      | Method | Path                                                              | Status | Body                              |
      | GET    | /api/tenants/*/organizations/72157ddd-60db-4fcf-a124-2b14098bc091 | 200    | single_organization.json          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/users?*         | 200    | { "data": [], "has_next": false } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/loginapps?*     | 200    | loginapps.json                    |
    When I navigate to the page with path "/organizations/72157ddd-60db-4fcf-a124-2b14098bc091?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading ..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a cardrow with the title "Basic Details"
    And I should see a cardrow with the title "Login Apps"
    And I should see a cardrow with the title "Users"
    Then I should see a "td" with the text "Login for The Gates Foundation"
    And I should see a table with ID "loginApps" and the following data
      | Name                           |
      | UserClouds Console (dev)       |
      | Login for The Gates Foundation |
