@tokenizer
@policy_templates
@access_policy_templates
@create_policy_template
Feature: create policy template page

  @a11y
  Scenario: creating policy template accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I navigate to the page with path "/policytemplates/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: error creating policy template
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I navigate to the page with path "/policytemplates/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see the following text on the page
      | TagName          | TextContent                   |
      | h1               | Create Template: New Template |
      | button[disabled] | Create template               |
      | button           | Cancel                        |
    And I should see the following form elements
      | TagName  | Type | Name                        | Value |
      | input    | text | policy_template_name        |       |
      | textarea |      | policy_template_description |       |
    And the page title should be "[dev] UserClouds Console"
    When I replace the text in the "policy_template_name" field with "Foo Bar"
    Then the input with the name "policy_template_name" should be invalid
    # todo function
    When I replace the text in the "policy_template_name" field with "Foo"
    And I replace the text in the "policy_template_description" field with "Bar"
    Then the button labeled "Create template" should be enabled
    And the input with the name "policy_template_name" should be valid
    And I should see the following form elements
      | TagName  | Type | Name                        | Value |
      | input    | text | policy_template_name        | Foo   |
      | textarea |      | policy_template_description | Bar   |
    Given the following mocked requests:
      | Method | Path                                                                 | Status | Body                                                                  |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates | 404    | {"error":"uh-oh","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
    When I click the button labeled "Create template"
    Then I should see the following text on the page
      | TagName | TextContent                   |
      | h1      | Create Template: New Template |
      | button  | Create template               |
      | button  | Cancel                        |
      | p       | uh-oh                         |
    And the button labeled "Create template" should be enabled
    And I should see the following form elements
      | TagName  | Type | Name                        | Value |
      | input    | text | policy_template_name        | Foo   |
      | textarea |      | policy_template_description | Bar   |

  Scenario: successfully create policy template
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I navigate to the page with path "/policytemplates/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName          | TextContent                   |
      | h1               | Create Template: New Template |
      | button[disabled] | Create template               |
      | button           | Cancel                        |
    And I should see the following form elements
      | TagName  | Type | Name                        | Value |
      | input    | text | policy_template_name        |       |
      | textarea |      | policy_template_description |       |
    When I replace the text in the "policy_template_name" field with "AllowAll"
    And I replace the text in the "policy_template_description" field with "This template allows all access."
    Then the button labeled "Create template" should be enabled
    Then I should see the following form elements
      | TagName  | Type | Name                        | Value                            |
      | input    | text | policy_template_name        | AllowAll                         |
      | textarea |      | policy_template_description | This template allows all access. |
    # user is redirected to details page
    Given the following mocked requests:
      | Method | Path                                                                                                                | Status | Body                        |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates                                                | 200    | single_policy_template.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8?version=0 | 200    | single_policy_template.json |
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I click the button labeled "Create template"
    Then I should see the following text on the page
      | TagName            | TextContent                      |
      | label              | ID                               |
      | label > div > span | 1e7422…                          |
      | label              | Name                             |
      | label > div > p    | AllowAll                         |
      | label              | Description                      |
      | label + p          | This template allows all access. |
      | button             | Edit template                    |
    And I should be on the page with the path "/policytemplates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8/0"
    And I should see a toast notification with the text "Successfully created policy template 'AllowAll'Close"
