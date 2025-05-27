@purposes
@purpose_details
Feature: purpose details page

  @a11y
  Scenario: purpose details accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                      | Status | Body                |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes/9a0f0b22-dabf-40b2-8f82-4a2caab9e605 | 200    | single_purpose.json |
    When I navigate to the page with path "/purposes/9a0f0b22-dabf-40b2-8f82-4a2caab9e605?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  @edit_purpose
  Scenario: error editing purpose
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                      | Status | Body                |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes/9a0f0b22-dabf-40b2-8f82-4a2caab9e605 | 200    | single_purpose.json |
    When I navigate to the page with path "/purposes/9a0f0b22-dabf-40b2-8f82-4a2caab9e605?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a cardrow with the title "Basic Details"
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
    And I should not see an element matching selector " button:has-text('Edit Purpose')"
    And I should see the following form elements
      | TagName  | Type | Name                | Value               |
      | textarea |      | purpose_description | For operations, duh |
    When I replace the text in the "purpose_description" field with "Not operational"
    Then the button labeled "Save Purpose" should be enabled
    Then I should see the following form elements
      | TagName  | Type | Name                | Value           |
      | textarea |      | purpose_description | Not operational |
    Given the following mocked requests:
      | Method | Path                                                                                                      | Status | Body                                                                  |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes/9a0f0b22-dabf-40b2-8f82-4a2caab9e605 | 404    | {"error":"uh-oh","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
    When I click the button labeled "Save Purpose"
    Then I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic Details |
      | button  | Save Purpose  |
      | button  | Cancel        |
      | p       | uh-oh         |
      | p       | Operational   |
    And the button labeled "Save Purpose" should be enabled
    And I should see the following form elements
      | TagName  | Type | Name                | Value           |
      | textarea |      | purpose_description | Not operational |

  @edit_purpose
  Scenario: successfully create purpose
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                      | Status | Body                |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes/9a0f0b22-dabf-40b2-8f82-4a2caab9e605 | 200    | single_purpose.json |
    When I navigate to the page with path "/purposes/9a0f0b22-dabf-40b2-8f82-4a2caab9e605?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a cardrow with the title "Basic Details"
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
    And I should not see an element matching selector "button:has-text('Edit Purpose')"
    And I should see the following form elements
      | TagName  | Type | Name                | Value               |
      | textarea |      | purpose_description | For operations, duh |
    When I replace the text in the "purpose_description" field with "Not operational"
    Then the button labeled "Save Purpose" should be enabled
    Then I should see the following form elements
      | TagName  | Type | Name                | Value           |
      | textarea |      | purpose_description | Not operational |
    # user is redirected to details page
    Given the following mocked requests:
      | Method | Path                                                                                                      | Status | Body                  |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes/9a0f0b22-dabf-40b2-8f82-4a2caab9e605 | 200    | single_purpose_1.json |
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I click the button labeled "Save Purpose"
    Then I should see the following text on the page
      | TagName            | TextContent     |
      | label              | ID              |
      | label > div > span | 9a0f0b…         |
      | label              | Name            |
      | label > div > p    | Operational     |
      | label              | Description     |
      | label + p          | Not operational |
      | button             | Edit Purpose    |
