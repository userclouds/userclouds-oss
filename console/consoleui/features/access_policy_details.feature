@policies
@access_policy
@access_policies
@access_policy_details
Feature: Access policy details page

  @a11y
  Scenario: access policy details page accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions_policy"
    And a mocked "GET" request for "companies"
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/* | 200    | access_policy_2.json |
    When I navigate to the page with path "/accesspolicies/1f44a215-2590-488f-a254-b6e8e4b67346/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: existing access policy basic
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions_policy"
    And a mocked "GET" request for "companies"
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/* | 200    | access_policy_2.json |
    When I navigate to the page with path "/accesspolicies/1f44a215-2590-488f-a254-b6e8e4b67346/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching policy..."
    And the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName              | TextContent      |
      | h2                   | Basic Details    |
      | main label           | Name             |
      | main label > div > p | complexPolicy    |
      | main label           | Version          |
      | main label > div > p | 0                |
      | h2                   | Compose Policy   |
      | h2                   | Metadata         |
      | main label           | Required context |
      | main label > div > p | {}               |
    And I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name    | Parameters |
      | Where | AllowAll       | N/A        |
      | AND   | AllowNone      | N/A        |
      | AND   | CheckAttribute | {}         |
      | AND   | checkIfEven    | {a:1}      |
    And I should see a button labeled "Edit access policy"
    And I should not see a button labeled "Add Policy"
    And I should not see a button labeled "Add Template"
    And I should not see a button labeled "Run Test"
    Given the following mocked requests:
      | Method | Path                                                                | Status | Body                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
    And a mocked "GET" request for "access_policy_templates"
    ##TODO - remove. get double dispatch fixed
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
    When I click the button labeled "Edit access policy"
    Then I should see the following inputs
      | Type | Name             | Value         | Disabled |
      | text | policy_name      | complexPolicy | false    |
      | text | required_context | {}            | false    |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Run Test"
    And I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name    | Parameters |
      | Where | AllowAll       |            |
      | AND   | AllowNone      |            |
      | AND   | CheckAttribute |            |
      | AND   | checkIfEven    |            |
    # TODO: we need a way to evaluate these table cells with text inputs in them
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Run Test"
    Then I should see the following text on the page
      | Selector               | TextContent        |
      | button:not([disabled]) | Run Test           |
      | button[disabled]       | Save access policy |

  Scenario: existing access policy good save
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions_policy"
    And a mocked "GET" request for "companies"
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/* | 200    | access_policy_2.json |
    When I navigate to the page with path "/accesspolicies/1f44a215-2590-488f-a254-b6e8e4b67346/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching policy..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName              | TextContent      |
      | h2                   | Basic Details    |
      | main label           | Name             |
      | main label > div > p | complexPolicy    |
      | main label           | Version          |
      | main label > div > p | 0                |
      | h2                   | Compose Policy   |
      | h2                   | Metadata         |
      | main label           | Required context |
      | main label > div > p | {}               |
    And I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name    | Parameters |
      | Where | AllowAll       | N/A        |
      | AND   | AllowNone      | N/A        |
      | AND   | CheckAttribute | {}         |
      | AND   | checkIfEven    | {a:1}      |
    And I should see a button labeled "Edit access policy"
    And I should not see a button labeled "Add Policy"
    And I should not see a button labeled "Add Template"
    And I should not see a button labeled "Run Test"
    Given the following mocked requests:
      | Method | Path                                                                | Status | Body                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
    And a mocked "GET" request for "access_policy_templates"
    ##TODO - remove. get double dispatch fixed
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
    When I click the button labeled "Edit access policy"
    Then I should see the following inputs
      | Type | Name             | Value         | Disabled |
      | text | policy_name      | complexPolicy | false    |
      | text | required_context | {}            | false    |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Run Test"
    When I click the button labeled "Add Policy"
    Then I should see a "dialog td" with the text "complexPolicy"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    When I click the radio input in row 1 of the table with ID "paginatedPolicyChooserPolicies"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name    | Parameters |
      | Where | AllowAll       |            |
      | AND   | AllowNone      |            |
      | AND   | CheckAttribute |            |
      | AND   | checkIfEven    |            |
      | AND   | complexPolicy  |            |
    When I click the delete button in row 5 of the table with ID "accessPolicyComponents"
    Then I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name    | Parameters |
      | Where | AllowAll       |            |
      | AND   | AllowNone      |            |
      | AND   | CheckAttribute |            |
      | AND   | checkIfEven    |            |
    When I click the delete button in row 4 of the table with ID "accessPolicyComponents"
    Then I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name    | Parameters |
      | Where | AllowAll       |            |
      | AND   | AllowNone      |            |
      | AND   | CheckAttribute |            |
    When I replace the text in the "policy_name" field with "aDifferentPolicy"
    Given a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "permissions_policy"
    And the following mocked requests:
      | Method | Path                                                                                                             | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access*                                               | 200    | access_policies_2.json  |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/1f44a215-2590-488f-a254-b6e8e4b67346           | 200    | access_policy_2_v1.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/1f44a215-2590-488f-a254-b6e8e4b67346?version=1 | 200    | access_policy_2_v1.json |
    When I click the button labeled "Save access policy"
    Then I should be navigated to the page with the path "/accesspolicies/1f44a215-2590-488f-a254-b6e8e4b67346/1"

  Scenario: existing access policy edit required context
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions_policy"
    And a mocked "GET" request for "companies"
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/* | 200    | access_policy_2.json |
    When I navigate to the page with path "/accesspolicies/1f44a215-2590-488f-a254-b6e8e4b67346/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching policy..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName              | TextContent      |
      | h2                   | Basic Details    |
      | main label           | Name             |
      | main label > div > p | complexPolicy    |
      | main label           | Version          |
      | main label > div > p | 0                |
      | h2                   | Compose Policy   |
      | h2                   | Metadata         |
      | main label           | Required context |
      | main label > div > p | {}               |
    And I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name    | Parameters |
      | Where | AllowAll       | N/A        |
      | AND   | AllowNone      | N/A        |
      | AND   | CheckAttribute | {}         |
      | AND   | checkIfEven    | {a:1}      |
    And I should see a button labeled "Edit access policy"
    And I should not see a button labeled "Add Policy"
    And I should not see a button labeled "Add Template"
    And I should not see a button labeled "Run Test"
    Given the following mocked requests:
      | Method | Path                                                                | Status | Body                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
    And a mocked "GET" request for "access_policy_templates"
    ##TODO - remove. get double dispatch fixed
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
    When I click the button labeled "Edit access policy"
    Then I should see the following inputs
      | Type | Name             | Value         | Disabled |
      | text | policy_name      | complexPolicy | false    |
      | text | required_context | {}            | false    |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Run Test"
    # test invalid JSON
    When I replace the text in the "required_context" field with "{ 'foo': 'bar' }"
    And I click the button labeled "Save access policy"
    Then I should see the following inputs
      | Type | Name             | Value            | Disabled |
      | text | policy_name      | complexPolicy    | false    |
      | text | required_context | { 'foo': 'bar' } | false    |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Run Test"
    And I should see the following text on the page
      | TagName          | TextContent                         |
      | .alert-message p | Required context must be valid JSON |
    And I should be on the page with the path "/accesspolicies/1f44a215-2590-488f-a254-b6e8e4b67346/0"
    # test good JSON
    When I replace the text in the "required_context" field with "{ \"foo\": \"bar\" }"
    Given a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "permissions_policy"
    And the following mocked requests:
      | Method | Path                                                                                                             | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access*                                               | 200    | access_policies_2.json  |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/1f44a215-2590-488f-a254-b6e8e4b67346           | 200    | access_policy_2_v1.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/1f44a215-2590-488f-a254-b6e8e4b67346?version=1 | 200    | access_policy_2_v1.json |
    When I click the button labeled "Save access policy"
    Then I should be navigated to the page with the path "/accesspolicies/1f44a215-2590-488f-a254-b6e8e4b67346/1"

  Scenario: existing access policy conduct test
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions_policy"
    And a mocked "GET" request for "companies"
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/* | 200    | access_policy_2.json |
    When I navigate to the page with path "/accesspolicies/1f44a215-2590-488f-a254-b6e8e4b67346/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching policy..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName              | TextContent      |
      | h2                   | Basic Details    |
      | main label           | Name             |
      | main label > div > p | complexPolicy    |
      | main label           | Version          |
      | main label > div > p | 0                |
      | h2                   | Compose Policy   |
      | h2                   | Metadata         |
      | main label           | Required context |
      | main label > div > p | {}               |
    And I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name    | Parameters |
      | Where | AllowAll       | N/A        |
      | AND   | AllowNone      | N/A        |
      | AND   | CheckAttribute | {}         |
      | AND   | checkIfEven    | {a:1}      |
    And I should see a button labeled "Edit access policy"
    And I should not see a button labeled "Add Policy"
    And I should not see a button labeled "Add Template"
    And I should not see a button labeled "Run Test"
    Given the following mocked requests:
      | Method | Path                                                                | Status | Body                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
    And a mocked "GET" request for "access_policy_templates"
    ##TODO - remove. get double dispatch fixed
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
    When I click the button labeled "Edit access policy"
    Then I should see the following inputs
      | Type | Name             | Value         | Disabled |
      | text | policy_name      | complexPolicy | false    |
      | text | required_context | {}            | false    |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Run Test"
    Given the following mocked requests:
      | Method | Path                                                                           | Status | Body                           |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/actions/test | 200    | {"allowed":false,"console":""} |
    When I click the button labeled "Run Test"
    Then I should see the following text on the page
      | TagName          | TextContent   |
      | .alert-message p | Access denied |
    When I click the button labeled "Add Policy"
    Then I should see a "dialog td" with the text "complexPolicy"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    When I click the radio input in row 1 of the table with ID "paginatedPolicyChooserPolicies"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name    | Parameters |
      | Where | AllowAll       |            |
      | AND   | AllowNone      |            |
      | AND   | CheckAttribute |            |
      | AND   | checkIfEven    |            |
      | AND   | complexPolicy  |            |
    Given the following mocked requests:
      | Method | Path                                                                           | Status | Body                          |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/actions/test | 200    | {"allowed":true,"console":""} |
    When I click the button labeled "Run Test"
    Then I should see the following text on the page
      | TagName            | TextContent    |
      | .success-message p | Access allowed |

  Scenario: existing access policy rate limiting
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions_policy"
    And a mocked "GET" request for "companies"
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/* | 200    | access_policy_2.json |
    When I navigate to the page with path "/accesspolicies/1f44a215-2590-488f-a254-b6e8e4b67346/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching policy..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName              | TextContent             |
      | h2                   | Basic Details           |
      | main label           | Name                    |
      | main label > div > p | complexPolicy           |
      | main label           | Version                 |
      | main label > div > p | 0                       |
      | h2                   | Compose Policy          |
      | h2                   | Execution Rate Limiting |
      | h2                   | Result Rate Limiting    |
      | h2                   | Metadata                |
      | main label           | Required context        |
      | main label > div > p | {}                      |
    When I click the button labeled "Edit access policy"
    Then I should see a checkbox labeled "Enforce Execution Rate Limiting"
    And I should see a checkbox labeled "Enforce Result Rate Limiting"
    And I should not see a checkbox labeled "Announce Max Execution Failure"
    And I should not see a checkbox labeled "Announce Max Result Failure"
    When I toggle the checkbox labeled "Enforce Execution Rate Limiting"
    Then the checkbox labeled "Enforce Execution Rate Limiting" should be checked
    And I should see a checkbox labeled "Announce Max Execution Failure"
    And I should see the following inputs
      | Type   | Name                 | Value | Disabled |
      | number | max_executions       | 1     | false    |
      | number | max_execution_window | 0     | false    |
    When I toggle the checkbox labeled "Enforce Result Rate Limiting"
    Then the checkbox labeled "Enforce Result Rate Limiting" should be checked
    And I should see a checkbox labeled "Announce Max Result Failure"
    And I should see the following inputs
      | Type   | Name        | Value | Disabled |
      | number | max_results | 1     | false    |
    When I toggle the checkbox labeled "Enforce Execution Rate Limiting"
    Then the checkbox labeled "Enforce Execution Rate Limiting" should be unchecked
    And I should not see a checkbox labeled "Announce Max Execution Failure"
    When I toggle the checkbox labeled "Enforce Result Rate Limiting"
    Then the checkbox labeled "Enforce Result Rate Limiting" should be unchecked
    And I should not see a checkbox labeled "Announce Max Result Failure"
    When I toggle the checkbox labeled "Enforce Execution Rate Limiting"
    Then the checkbox labeled "Enforce Execution Rate Limiting" should be checked
    And I should see a checkbox labeled "Announce Max Execution Failure"
    And I should see the following inputs
      | Type   | Name                 | Value | Disabled |
      | number | max_executions       | 1     | false    |
      | number | max_execution_window | 0     | false    |
    When I replace the text in the "max_executions" field with "10"
    And I replace the text in the "max_execution_window" field with "5"
    Then I should see the following inputs
      | Type   | Name                 | Value | Disabled |
      | number | max_executions       | 10    | false    |
      | number | max_execution_window | 5     | false    |
    Given a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "permissions_policy"
    And the following mocked requests:
      | Method | Path                                                                                                             | Status | Body                                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access*                                               | 200    | access_policies_2.json               |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/1f44a215-2590-488f-a254-b6e8e4b67346           | 200    | access_policy_2_thresholds_edit.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/1f44a215-2590-488f-a254-b6e8e4b67346?version=1 | 200    | access_policy_2_thresholds_edit.json |
    When I click the button labeled "Save access policy"
    Then I should be navigated to the page with the path "/accesspolicies/1f44a215-2590-488f-a254-b6e8e4b67346/1"
    And I should see a toast notification with the text "Successfully updated access policyClose"

  Scenario: cancel access policy editing
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions_policy"
    And a mocked "GET" request for "companies"
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/* | 200    | access_policy_2.json |
    When I navigate to the page with path "/accesspolicies/1f44a215-2590-488f-a254-b6e8e4b67346/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching policy..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName              | TextContent      |
      | h2                   | Basic Details    |
      | main label           | Name             |
      | main label > div > p | complexPolicy    |
      | main label           | Version          |
      | main label > div > p | 0                |
      | h2                   | Compose Policy   |
      | h2                   | Metadata         |
      | main label           | Required context |
      | main label > div > p | {}               |
    And I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name    | Parameters |
      | Where | AllowAll       | N/A        |
      | AND   | AllowNone      | N/A        |
      | AND   | CheckAttribute | {}         |
      | AND   | checkIfEven    | {a:1}      |
    And I should see a button labeled "Edit access policy"
    And I should not see a button labeled "Add Policy"
    And I should not see a button labeled "Add Template"
    And I should not see a button labeled "Run Test"
    Given the following mocked requests:
      | Method | Path                                                                | Status | Body                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
    And a mocked "GET" request for "access_policy_templates"
    ##TODO - remove. get double dispatch fixed
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
    When I click the button labeled "Edit access policy"
    Then I should see the following inputs
      | Type | Name             | Value         | Disabled |
      | text | policy_name      | complexPolicy | false    |
      | text | required_context | {}            | false    |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Run Test"
    And I should see a button labeled "Save access policy"
    And the button labeled "Save access policy" should be disabled
    When I replace the text in the "policy_name" field with "Our_Access_Policy"
    Then the button labeled "Save access policy" should be enabled
    And the button labeled "Cancel" should be enabled
    Given I intend to dismiss the confirm dialog
    When I click the button labeled "Cancel"
    Then I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic Details |
    And I should be on the page with the path "/accesspolicies/1f44a215-2590-488f-a254-b6e8e4b67346/0"
    And the button labeled "Save access policy" should be enabled
    And the button labeled "Cancel" should be enabled
    When I replace the text in the "policy_name" field with "complexPolicy"
    Then the button labeled "Save access policy" should be disabled
    And the button labeled "Cancel" should be enabled
    # no confirm dialog
    When I click the button labeled "Cancel"
    Then I should see a button labeled "Edit access policy"
    And the button labeled "Edit access policy" should be enabled
    And I should not see a button labeled "Save access policy"
    And I should not see a button labeled "Cancel"
    And I should not see a button labeled "Run Test"
    And I should be on the page with the path "/accesspolicies/1f44a215-2590-488f-a254-b6e8e4b67346/0"
