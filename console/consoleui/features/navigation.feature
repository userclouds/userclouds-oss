@navigation
Feature: navigation

  Scenario: clear ephemeral state on navigation
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                      | Status | Body                |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes/9a0f0b22-dabf-40b2-8f82-4a2caab9e605 | 200    | single_purpose.json |
    When I navigate to the page with path "/purposes/9a0f0b22-dabf-40b2-8f82-4a2caab9e605?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic Details |
      | button  | Edit Purpose  |
      | p       | Operational   |
    When I click the button labeled "Edit Purpose"
    Then I should see the following text on the page
      | TagName          | TextContent   |
      | h2               | Basic Details |
      | button[disabled] | Save Purpose  |
      | button           | Cancel        |
      | p                | Operational   |
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                      | Status | Body                |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes/9a0f0b22-dabf-40b2-8f82-4a2caab9e605 | 200    | single_purpose.json |
    When I navigate to the page with path "/purposes/9a0f0b22-dabf-40b2-8f82-4a2caab9e605?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic Details |
      | button  | Edit Purpose  |
      | p       | Operational   |
