@purposes
@create_purpose
Feature: create purpose page

  @a11y
  Scenario: creating purpose accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I navigate to the page with path "/purposes/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: error creating purpose
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I navigate to the page with path "/purposes/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName          | TextContent    |
      | h2               | Basic Details  |
      | button[disabled] | Create Purpose |
      | button           | Cancel         |
    And I should see a cardrow with the title "Basic Details"
    And I should see the following form elements
      | TagName  | Type | Name                | Value |
      | input    | text | purpose_name        |       |
      | textarea |      | purpose_description |       |
    When I replace the text in the "purpose_name" field with "Operational"
    Then the button labeled "Create Purpose" should be disabled
    And I replace the text in the "purpose_description" field with "For operations, duh"
    Then the button labeled "Create Purpose" should be enabled
    Then I should see the following form elements
      | TagName  | Type | Name                | Value               |
      | input    | text | purpose_name        | Operational         |
      | textarea |      | purpose_description | For operations, duh |
    Given the following mocked requests:
      | Method | Path                                                                 | Status | Body                                                                  |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes | 404    | {"error":"uh-oh","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
    When I click the button labeled "Create Purpose"
    Then I should see the following text on the page
      | TagName | TextContent    |
      | h2      | Basic Details  |
      | button  | Create Purpose |
      | button  | Cancel         |
      | p       | uh-oh          |
    And the button labeled "Create Purpose" should be enabled
    And I should see the following form elements
      | TagName  | Type | Name                | Value               |
      | input    | text | purpose_name        | Operational         |
      | textarea |      | purpose_description | For operations, duh |

  Scenario: successfully create purpose
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I navigate to the page with path "/purposes/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName          | TextContent    |
      | h2               | Basic Details  |
      | button[disabled] | Create Purpose |
      | button           | Cancel         |
    And I should see a cardrow with the title "Basic Details"
    And I should see the following form elements
      | TagName  | Type | Name                | Value |
      | input    | text | purpose_name        |       |
      | textarea |      | purpose_description |       |
    When I replace the text in the "purpose_name" field with "Operational"
    Then the button labeled "Create Purpose" should be disabled
    And I replace the text in the "purpose_description" field with "For operations, duh"
    Then the button labeled "Create Purpose" should be enabled
    Then I should see the following form elements
      | TagName  | Type | Name                | Value               |
      | input    | text | purpose_name        | Operational         |
      | textarea |      | purpose_description | For operations, duh |
    # user is redirected to details page
    Given the following mocked requests:
      | Method | Path                                                                                                      | Status | Body                |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes                                      | 200    | single_purpose.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes/9a0f0b22-dabf-40b2-8f82-4a2caab9e605 | 200    | single_purpose.json |
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I click the button labeled "Create Purpose"
    Then I should see the following text on the page
      | TagName            | TextContent         |
      | label              | ID                  |
      | label > div > span | 9a0f0b…             |
      | label              | Name                |
      | label > div > p    | Operational         |
      | label              | Description         |
      | label + p          | For operations, duh |
      | button             | Edit Purpose        |
    And I should be on the page with the path "/purposes/9a0f0b22-dabf-40b2-8f82-4a2caab9e605"
    And I should see a toast notification with the text "Successfully created purposeClose"
