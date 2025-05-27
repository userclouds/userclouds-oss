@policies
@access_policy
@access_policies
@create_access_policy
Feature: create access policy page

  @a11y
  Scenario: new access policy basic accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "companies"
    And a mocked "GET" request for "access_policies"
    When I navigate to the page with path "/accesspolicies/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: new access policy basic info
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "companies"
    And a mocked "GET" request for "access_policies"
    When I navigate to the page with path "/accesspolicies/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic Details |
    And I should see the following inputs
      | Type | Name             | Value | Disabled |
      | text | policy_name      |       | false    |
      | text | required_context | {}    | false    |
    And I should see a table with ID "accessPolicyComponents" and the following data
      |                           |
      | No policy components yet. |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Run Test"
    And I should see a button labeled "Save access policy"

  Scenario: new access policy enabled save
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "companies"
    And a mocked "GET" request for "access_policies"
    When I navigate to the page with path "/accesspolicies/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic Details |
    And I should see the following inputs
      | Type | Name             | Value | Disabled |
      | text | policy_name      |       | false    |
      | text | required_context | {}    | false    |
    And I should see a table with ID "accessPolicyComponents" and the following data
      |                           |
      | No policy components yet. |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Run Test"
    Then I should see the following text on the page
      | Selector               | TextContent        |
      | button:not([disabled]) | Run Test           |
      | button[disabled]       | Save access policy |
    When I replace the text in the "policy_name" field with "Our_Access_Policy"
    Then I should see the following text on the page
      | Selector               | TextContent        |
      | button:not([disabled]) | Run Test           |
      | button:not([disabled]) | Save access policy |

  Scenario: new access policy component
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "companies"
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
    When I navigate to the page with path "/accesspolicies/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic Details |
    And I should see the following inputs
      | Type | Name             | Value | Disabled |
      | text | policy_name      |       | false    |
      | text | required_context | {}    | false    |
    And I should see a table with ID "accessPolicyComponents" and the following data
      |                           |
      | No policy components yet. |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Run Test"
    And I should see a button labeled "Save access policy"
    When I click the button labeled "Add Policy"
    Then I should see a "dialog td" with the text "complexPolicy"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    When I click the radio input in row 1 of the table with ID "paginatedPolicyChooserPolicies"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name   | Parameters |
      | Where | complexPolicy |            |
    When I click the button labeled "Add Policy"
    Then I should see a "dialog td" with the text "complexPolicy"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    When I click the radio input in row 1 of the table with ID "paginatedPolicyChooserPolicies"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name   | Parameters | Delete     |
      | Where | complexPolicy |            | Delete Bin |
      | AND   | complexPolicy |            | Delete Bin |
    When I click the button labeled "Add Template"
    Then I should see a "dialog td" with the text "hellopolicy4"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    When I click the radio input in row 1 of the table with ID "paginatedPolicyChooserTemplates"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name   | Parameters | Delete     |
      | Where | complexPolicy |            | Delete Bin |
      | AND   | complexPolicy |            | Delete Bin |
      | AND   | hellopolicy4  |            | Delete Bin |
    When I click the delete button in row 1 of the table with ID "accessPolicyComponents"
    Then I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name   | Parameters | Delete     |
      | Where | complexPolicy |            | Delete Bin |
      | AND   | hellopolicy4  |            | Delete Bin |

  Scenario: new access policy bad save
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "companies"
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
    When I navigate to the page with path "/accesspolicies/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic Details |
    And I should see the following inputs
      | Type | Name             | Value |
      | text | policy_name      |       |
      | text | required_context | {}    |
    And I should see a table with ID "accessPolicyComponents" and the following data
      |                           |
      | No policy components yet. |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Run Test"
    Then I should see the following text on the page
      | Selector               | TextContent        |
      | button:not([disabled]) | Run Test           |
      | button[disabled]       | Save access policy |
    When I click the button labeled "Add Policy"
    Then I should see a "dialog td" with the text "complexPolicy"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    When I click the radio input in row 1 of the table with ID "paginatedPolicyChooserPolicies"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name   | Parameters |
      | Where | complexPolicy |            |
    When I click the button labeled "Add Policy"
    Then I should see a "dialog td" with the text "complexPolicy"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    When I click the radio input in row 1 of the table with ID "paginatedPolicyChooserPolicies"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name   | Parameters | Delete     |
      | Where | complexPolicy |            | Delete Bin |
      | AND   | complexPolicy |            | Delete Bin |
    When I click the button labeled "Add Template"
    Then I should see a "dialog td" with the text "hellopolicy4"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    When I click the radio input in row 1 of the table with ID "paginatedPolicyChooserTemplates"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name   | Parameters | Delete     |
      | Where | complexPolicy |            | Delete Bin |
      | AND   | complexPolicy |            | Delete Bin |
      | AND   | hellopolicy4  |            | Delete Bin |
    When I click the button labeled "Save access policy"
    Then the input with the name "policy_name" should be invalid
    When I replace the text in the "policy_name" field with "(&$%"
    Then the input with the name "policy_name" should be invalid
    When I replace the text in the "policy_name" field with "Valid Name"
    Then the input with the name "policy_name" should be invalid
    When I replace the text in the "policy_name" field with "ValidName"
    Then the input with the name "policy_name" should be valid
    Given the following mocked requests:
      | Method | Path                                                              | Status | Body                                                                           |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access | 400    | {"error":"not great, bob","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
    When I click the button labeled "Save access policy"
    Then I should see the following text on the page
      | TagName | TextContent                           |
      | p       | error creating policy: not great, bob |
    And I should be on the page with the path "/accesspolicies/create"

  Scenario: new access policy good save
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "companies"
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
    When I navigate to the page with path "/accesspolicies/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic Details |
    And I should see the following inputs
      | Type | Name             | Value | Disabled |
      | text | policy_name      |       | false    |
      | text | required_context | {}    | false    |
    And I should see a table with ID "accessPolicyComponents" and the following data
      |                           |
      | No policy components yet. |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Run Test"
    Then I should see the following text on the page
      | Selector               | TextContent        |
      | button:not([disabled]) | Run Test           |
      | button[disabled]       | Save access policy |
    When I click the button labeled "Add Policy"
    Then I should see a "dialog td" with the text "complexPolicy"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    When I click the radio input in row 1 of the table with ID "paginatedPolicyChooserPolicies"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name   | Parameters |
      | Where | complexPolicy |            |
    When I click the button labeled "Add Policy"
    Then I should see a "dialog td" with the text "complexPolicy"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    When I click the radio input in row 1 of the table with ID "paginatedPolicyChooserPolicies"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name   | Parameters | Delete     |
      | Where | complexPolicy |            | Delete Bin |
      | AND   | complexPolicy |            | Delete Bin |
    When I replace the text in the "policy_name" field with "Our_Access_Policy"
    # test invalid JSON
    And I replace the text in the "required_context" field with "{ 'foo': 'bar' }"
    And I click the button labeled "Save access policy"
    Then I should see the following inputs
      | Type | Name             | Value             | Disabled |
      | text | policy_name      | Our_Access_Policy | false    |
      | text | required_context | { 'foo': 'bar' }  | false    |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Run Test"
    And I should see the following text on the page
      | TagName          | TextContent                         |
      | .alert-message p | Required context must be valid JSON |
    And I should be on the page with the path "/accesspolicies/create"
    When I replace the text in the "required_context" field with "{ \"foo\": \"bar\" }"
    Then I should see the following text on the page
      | Selector               | TextContent        |
      | button:not([disabled]) | Run Test           |
      | button:not([disabled]) | Save access policy |
    Given the following mocked requests:
      | Method | Path                                                              | Status | Body                                                                                                                                                                                                                                                                                                                                                                   |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access | 200    | {"id":"3303dc3d-0aed-47d2-b718-28c7df1dd252","name":"our_access_policy","description":"","policy_type":"composite_and","tag_ids":[],"version":0,"required_context":"{\"foo\":\"bar\"}","components":[{"policy":{"id":"1f44a215-2590-488f-a254-b6e8e4b67346","name":"complexPolicy"}},{"policy":{"id":"1f44a215-2590-488f-a254-b6e8e4b67346","name":"complexPolicy"}}]} |
    When I click the button labeled "Save access policy"
    Then I should be navigated to the page with the path "/accesspolicies/3303dc3d-0aed-47d2-b718-28c7df1dd252/0"

  Scenario: new access policy conduct test
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "companies"
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?* | 200    | access_policies_2.json |
    When I navigate to the page with path "/accesspolicies/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent    |
      | h2      | Basic Details  |
      | h2      | Compose Policy |
      | h2      | Test Policy    |
    And I should see the following inputs
      | Type | Name             | Value | Disabled |
      | text | policy_name      |       | false    |
      | text | required_context | {}    | false    |
    And I should see a table with ID "accessPolicyComponents" and the following data
      |                           |
      | No policy components yet. |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a code editor with the ID "context" and the value "// Context changes on a per-resolution basis{  "server": {    "ip_address": "127.0.0.1",    "claims": {      "sub": "bob"    },    "action": "resolve"  },  "client": {    "purpose": "marketing"  },  "user": {    "name": "Jane Doe",    "email": "jane@doe.com",    "phone": "+15703211564"  }}"
    And I should see a button labeled "Run Test"
    And I should see a button labeled "Save access policy"
    When I click the button labeled "Add Policy"
    Then I should see a "dialog td" with the text "complexPolicy"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    When I click the radio input in row 1 of the table with ID "paginatedPolicyChooserPolicies"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessPolicyComponents" and the following data
      |       | Policy name   | Parameters |
      | Where | complexPolicy |            |
    Given the following mocked requests:
      | Method | Path                                                                           | Status | Body                           |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/actions/test | 200    | {"allowed":false,"console":""} |
    When I click the button labeled "Run Test"
    Then I should see the following text on the page
      | TagName          | TextContent   |
      | .alert-message p | Access denied |
    Given the following mocked requests:
      | Method | Path                                                                           | Status | Body                          |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/actions/test | 200    | {"allowed":true,"console":""} |
    When I click the button labeled "Run Test"
    Then I should see the following text on the page
      | TagName            | TextContent    |
      | .success-message p | Access allowed |

  Scenario: cancel access policy creation
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "companies"
    And a mocked "GET" request for "access_policies"
    When I navigate to the page with path "/accesspolicies/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic Details |
    And I should see the following inputs
      | Type | Name        | Value | Disabled |
      | text | policy_name |       | false    |
    And I should see a table with ID "accessPolicyComponents" and the following data
      |                           |
      | No policy components yet. |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Run Test"
    Then I should see the following text on the page
      | Selector               | TextContent        |
      | button:not([disabled]) | Run Test           |
      | button[disabled]       | Save access policy |
    When I replace the text in the "policy_name" field with "Our_Access_Policy"
    Then the button labeled "Save access policy" should be enabled
    And the button labeled "Cancel" should be enabled
    Given I intend to dismiss the confirm dialog
    When I click the button labeled "Cancel"
    Then I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic Details |
    And I should be on the page with the path "/accesspolicies/create"
    And the button labeled "Save access policy" should be enabled
    And the button labeled "Cancel" should be enabled
    When I replace the text in the "policy_name" field with ""
    Then the button labeled "Save access policy" should be disabled
    And the button labeled "Cancel" should be enabled
    Given a mocked "GET" request for "tenants"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": true, "read": true, "update": true, "delete": true } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates*  | 200    | policy_templates.json                                            |
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "transformers"
    # no confirm dialog
    When I click the button labeled "Cancel"
    Then I should be on the page with the path "/accesspolicies"
