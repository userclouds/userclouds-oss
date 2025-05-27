@tokenizer
@policy_templates
@access_policy_templates
@policy_template_details
Feature: create policy template page

  @a11y
  Scenario: policy template details accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                                | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8?version=0 | 200    | single_policy_template.json |
    When I navigate to the page with path "/policytemplates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: error editing policy template
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                                | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8?version=0 | 200    | single_policy_template.json |
    When I navigate to the page with path "/policytemplates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent      |
      | h1      | Template Details |
      | button  | Edit template    |
    And I should see the following text on the page
      | TagName            | TextContent                      |
      | label              | ID                               |
      | label > div > span | 1e7422…                          |
      | label              | Version                          |
      | label > div > p    | 0                                |
      | label              | Name                             |
      | label > div > p    | AllowAll                         |
      | label              | Description                      |
      | label + p          | This template allows all access. |
      | button             | Edit template                    |
    When I click the button labeled "Edit template"
    Then I should see the following form elements
      | TagName  | Type | Name                        | Value                            |
      | input    | text | policy_template_name        | AllowAll                         |
      | textarea |      | policy_template_description | This template allows all access. |
    # todo function
    And the button labeled "Save template" should be disabled
    And the button labeled "Cancel" should be enabled
    When I replace the text in the "policy_template_name" field with "Foo"
    And I replace the text in the "policy_template_description" field with "Bar"
    Then the button labeled "Save template" should be enabled
    Then I should see the following form elements
      | TagName  | Type | Name                        | Value |
      | input    | text | policy_template_name        | Foo   |
      | textarea |      | policy_template_description | Bar   |
    Given the following mocked requests:
      | Method | Path                                                                                                      | Status | Body                                                                  |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8 | 404    | {"error":"uh-oh","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
    When I click the button labeled "Save template"
    Then I should see the following text on the page
      | TagName | TextContent      |
      | h1      | Template Details |
      | button  | Save template    |
      | button  | Cancel           |
      | p       | uh-oh            |
    And the button labeled "Save template" should be enabled
    And I should see the following form elements
      | TagName  | Type | Name                        | Value |
      | input    | text | policy_template_name        | Foo   |
      | textarea |      | policy_template_description | Bar   |

  Scenario: successfully edit template
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                                | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8?version=0 | 200    | single_policy_template.json |
    When I navigate to the page with path "/policytemplates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent      |
      | h1      | Template Details |
      | button  | Edit template    |
    And I should see the following text on the page
      | TagName            | TextContent                      |
      | label              | ID                               |
      | label > div > span | 1e7422…                          |
      | label              | Version                          |
      | label > div > p    | 0                                |
      | label              | Name                             |
      | label > div > p    | AllowAll                         |
      | label              | Description                      |
      | label + p          | This template allows all access. |
      | button             | Edit template                    |
    When I click the button labeled "Edit template"
    Then I should see the following form elements
      | TagName  | Type | Name                        | Value                            |
      | input    | text | policy_template_name        | AllowAll                         |
      | textarea |      | policy_template_description | This template allows all access. |
    And the button labeled "Save template" should be disabled
    And the button labeled "Cancel" should be enabled
    When I replace the text in the "policy_template_name" field with "Allow All"
    Then the input with the name "policy_template_name" should be invalid
    When I replace the text in the "policy_template_name" field with "AllowAll2"
    And I replace the text in the "policy_template_description" field with "This template allows all access."
    Then the input with the name "policy_template_name" should be valid
    And the button labeled "Save template" should be enabled
    And I should see the following form elements
      | TagName  | Type | Name                        | Value                            |
      | input    | text | policy_template_name        | AllowAll2                        |
      | textarea |      | policy_template_description | This template allows all access. |
    # user is redirected to details page
    Given the following mocked requests:
      | Method | Path                                                                                                                | Status | Body                          |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8           | 200    | modified_policy_template.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8?version=1 | 200    | modified_policy_template.json |
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I click the button labeled "Save template"
    Then I should see the following text on the page
      | TagName            | TextContent                      |
      | label              | ID                               |
      | label > div > span | 1e7422…                          |
      | label              | Version                          |
      | label > div > p    | 1                                |
      | label              | Name                             |
      | label > div > p    | AllowAll2                        |
      | label              | Description                      |
      | label + p          | This template allows all access. |
      | button             | Edit template                    |
    And I should be on the page with the path "/policytemplates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8/1"

  Scenario: cancel editing policy template
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                                | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8?version=0 | 200    | single_policy_template.json |
    When I navigate to the page with path "/policytemplates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent      |
      | h1      | Template Details |
      | button  | Edit template    |
    And I should see the following text on the page
      | TagName            | TextContent                      |
      | label              | ID                               |
      | label > div > span | 1e7422…                          |
      | label              | Version                          |
      | label > div > p    | 0                                |
      | label              | Name                             |
      | label > div > p    | AllowAll                         |
      | label              | Description                      |
      | label + p          | This template allows all access. |
      | button             | Edit template                    |
    When I click the button labeled "Edit template"
    Then I should see the following form elements
      | TagName  | Type | Name                        | Value                            |
      | input    | text | policy_template_name        | AllowAll                         |
      | textarea |      | policy_template_description | This template allows all access. |
    # todo function
    And the button labeled "Save template" should be disabled
    And the button labeled "Cancel" should be enabled
    When I replace the text in the "policy_template_name" field with "Foo"
    And I replace the text in the "policy_template_description" field with "Bar"
    Then the button labeled "Save template" should be enabled
    Then I should see the following form elements
      | TagName  | Type | Name                        | Value |
      | input    | text | policy_template_name        | Foo   |
      | textarea |      | policy_template_description | Bar   |
    Given I intend to accept the confirm dialog
    When I click the button labeled "Cancel"
    And I should see the following text on the page
      | TagName            | TextContent                      |
      | label              | ID                               |
      | label > div > span | 1e7422…                          |
      | label              | Version                          |
      | label > div > p    | 0                                |
      | label              | Name                             |
      | label > div > p    | AllowAll                         |
      | label              | Description                      |
      | label + p          | This template allows all access. |
      | button             | Edit template                    |
