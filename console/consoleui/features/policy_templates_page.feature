Feature: policy templates page

  @admin
  @policy_templates
  Scenario: basic info
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": true, "read": true, "update": true, "delete": true } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates*  | 200    | policy_templates.json                                            |
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "transformers"

    When I navigate to the page with path "/policytemplates?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a&templates_limit=3"
    And the page title should be "[dev] UserClouds Console"
    And I should see a link to "/policytemplates/c9cfe092-dbc2-4aca-a68d-85eb85126526/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "policyTemplates" and the following data
      |  | Name           | Description                                                                | Version | ID           |
      |  | AllowAll       | This template allows all access.                                           | 0       | Copy 1e7422… |
      |  | CheckAttribute | This template returns the value of checkAttribute on the given parameters. | 0       | Copy aad2bf… |
      |  | foo            | bar                                                                        | 0       | Copy c9cfe0… |
    And I should see a button labeled "Create Template"

  @admin
  @policy_templates
  @delete_policy_templates
  Scenario: policy templates page delete templates
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": true, "read": true, "update": true, "delete": true } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates*  | 200    | policy_templates.json                                            |
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "transformers"

    When I navigate to the page with path "/policytemplates?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a&templates_limit=3"
    And the page title should be "[dev] UserClouds Console"
    And I should see a link to "/policytemplates/c9cfe092-dbc2-4aca-a68d-85eb85126526/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "policyTemplates" and the following data
      |  | Name           | Description                                                                | Version | ID           |
      |  | AllowAll       | This template allows all access.                                           | 0       | Copy 1e7422… |
      |  | CheckAttribute | This template returns the value of checkAttribute on the given parameters. | 0       | Copy aad2bf… |
      |  | foo            | bar                                                                        | 0       | Copy c9cfe0… |
    And I should see a button labeled "Create Template"

    When I toggle the checkbox in column 1 of row 1 of the table with ID "policyTemplates"
    Then row 1 of the table with ID "policyTemplates" should be marked for delete

    When I toggle the checkbox in column 1 of row 2 of the table with ID "policyTemplates"
    And I toggle the checkbox in column 1 of row 3 of the table with ID "policyTemplates"
    Then row 2 of the table with ID "policyTemplates" should be marked for delete
    And row 3 of the table with ID "policyTemplates" should be marked for delete

    # unqueue an item
    When I toggle the checkbox in column 1 of row 2 of the table with ID "policyTemplates"
    Then row 1 of the table with ID "policyTemplates" should be marked for delete
    And row 2 of the table with ID "policyTemplates" should not be marked for delete
    And row 3 of the table with ID "policyTemplates" should be marked for delete

    Given the following mocked requests:
      | Method | Path                                                                                                      | Status | Body                                                                                                           |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates/1e742248-fdde-4c88-9ea7-2c2106ec7aa8 | 409    | {"error":{"http_status_code":409,"error":{"error":"foo"}},"request_id":"2d209fe1-3e67-46ae-9d42-e1694469120a"} |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates/c9cfe092-dbc2-4aca-a68d-85eb85126526 | 409    | {"error":{"http_status_code":409,"error":{"error":"bar"}},"request_id":"2d209fe1-3e67-4068-94ed-3dcdaaaa53bc"} |
    And I intend to accept the confirm dialog

    # all requests fail
    When I click the button with ID "deletePolicyTemplatesButton"
    Then I should see the following text within the dialog titled "Delete Policy Templates"
      | Selector | Text                                                                             |
      | div      | Are you sure you want to delete 2 policy templates? This action is irreversible. |

    When I click the button with ID "confirmDeleteButton"
    Then I should see a table with ID "policyTemplates" and the following data
      |  | Name           | Description                                                                | Version | ID           |
      |  | AllowAll       | This template allows all access.                                           | 0       | Copy 1e7422… |
      |  | CheckAttribute | This template returns the value of checkAttribute on the given parameters. | 0       | Copy aad2bf… |
      |  | foo            | bar                                                                        | 0       | Copy c9cfe0… |
    Then row 1 of the table with ID "policyTemplates" should be marked for delete
    And row 2 of the table with ID "policyTemplates" should not be marked for delete
    And row 3 of the table with ID "policyTemplates" should be marked for delete
    And I should see the following text on the page
      | TagName                | TextContent                                                                     |
      | .alert-message ul > li | Error deleting access policy template 1e742248-fdde-4c88-9ea7-2c2106ec7aa8: foo |
      | .alert-message ul > li | Error deleting access policy template c9cfe092-dbc2-4aca-a68d-85eb85126526: bar |

    When I toggle the checkbox in column 1 of row 1 of the table with ID "policyTemplates"
    And I toggle the checkbox in column 1 of row 2 of the table with ID "policyTemplates"
    Then row 1 of the table with ID "policyTemplates" should not be marked for delete
    And row 2 of the table with ID "policyTemplates" should be marked for delete
    And row 3 of the table with ID "policyTemplates" should be marked for delete

    Given the following mocked requests:
      | Method | Path                                                                                                      | Status | Body                                    |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates/aad2bf25-311f-467e-9169-a6a89b6d34a6 | 200    | {}                                      |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates/c9cfe092-dbc2-4aca-a68d-85eb85126526 | 200    | {}                                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates*                                     | 200    | policy_templates_successful_delete.json |

    # all requests succeed
    When I click the button with ID "deletePolicyTemplatesButton"
    Then I should see the following text within the dialog titled "Delete Policy Templates"
      | Selector | Text                                                                             |
      | div      | Are you sure you want to delete 2 policy templates? This action is irreversible. |

    When I click the button with ID "confirmDeleteButton"
    Then I should see the following text on the page
      | TagName            | TextContent                                   |
      | .success-message p | Successfully deleted access policy templates. |
    And I should see a table with ID "policyTemplates" and the following data
      |  | Name     | Description                      | Version | ID           |
      |  | AllowAll | This template allows all access. | 0       | Copy 1e7422… |

  @admin
  @pagination
  @policy_templates
  @policy_templates_pagination
  Scenario: policy templates page pagination
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": true, "read": true, "update": true, "delete": true } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates*  | 200    | policy_templates.json                                            |
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "transformers"

    When I navigate to the page with path "/policytemplates?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a&templates_limit=3"
    And the page title should be "[dev] UserClouds Console"
    And I should see a link to "/policytemplates/c9cfe092-dbc2-4aca-a68d-85eb85126526/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "policyTemplates" and the following data
      |  | Name           | Description                                                                | Version | ID           |
      |  | AllowAll       | This template allows all access.                                           | 0       | Copy 1e7422… |
      |  | CheckAttribute | This template returns the value of checkAttribute on the given parameters. | 0       | Copy aad2bf… |
      |  | foo            | bar                                                                        | 0       | Copy c9cfe0… |
    And I should see a button labeled "Create Template"

    Given the following mocked requests:
      | Method | Path                                                                  | Status | Body                                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates* | 200    | policy_templates_successful_delete.json |
    # When the querystring changes we refetch all lists.
    # TODO: this is a bug, but we can leave it in for now since we'll be splitting these out into separate pages
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "transformers"

    When I click the button with ID "next"
    Then I should see a table with ID "policyTemplates" and the following data
      |  | Name     | Description                      | Version | ID           |
      |  | AllowAll | This template allows all access. | 0       | Copy 1e7422… |

    Given the following mocked requests:
      | Method | Path                                                                  | Status | Body                  |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates* | 200    | policy_templates.json |
    # When the querystring changes we refetch all lists.
    # TODO: this is a bug, but we can leave it in for now since we'll be splitting these out into separate pages
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "transformers"

    When I click the button with ID "prev"
    Then I should see a table with ID "policyTemplates" and the following data
      |  | Name           | Description                                                                | Version | ID           |
      |  | AllowAll       | This template allows all access.                                           | 0       | Copy 1e7422… |
      |  | CheckAttribute | This template returns the value of checkAttribute on the given parameters. | 0       | Copy aad2bf… |
      |  | foo            | bar                                                                        | 0       | Copy c9cfe0… |
