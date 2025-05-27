@access_policies
@access_policies_list
Feature: access policies

  @a11y
  Scenario: access policies page accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": true, "read": true, "update": true, "delete": true } |
    And a mocked "GET" request for "access_policies"
    When I navigate to the page with path "/accesspolicies?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a&access_policies_limit=2"
    Then the page should have no accessibility violations

  @admin
  Scenario: access policies page
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": true, "read": true, "update": true, "delete": true } |
    And a mocked "GET" request for "access_policies"
    When I navigate to the page with path "/accesspolicies?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a&access_policies_limit=2"
    Then I should see a "p" with the text "Loading policies..."
    Then I should see a table with ID "accessPolicies" and the following data
      |  | Name           | Version | ID           |
      |  | Allow_all      | 0       | Copy 0c0b73… |
      |  | Dont_Allow_Any | 1       | Copy 000000… |
    And the page title should be "[dev] UserClouds Console"
    And I should see a link to "/accesspolicies/0c0b7371-5175-405b-a17c-fec5969914b8/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"

  @global_policies
  Scenario: access policies page with global policies
    Given I am a logged-in user
    And the following feature flags
      | Name                   | Value |
      | global-access-policies | true  |
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions                                 | 200    | { "create": true, "read": true, "update": true, "delete": true } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/a78f1f88-3684-4e59-a01d-c121e259ec96 | 200    | global_accessor_policy.json                                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/804e84f1-7fa4-4bb4-b785-4c89e1ceaba0 | 200    | global_mutator_policy.json                                       |
    And a mocked "GET" request for "access_policies"
    When I navigate to the page with path "/accesspolicies?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a&access_policies_limit=2"
    Then I should see a "p" with the text "Loading policies..."
    Then I should see a table with ID "accessPolicies" and the following data
      |  | Name                             | Version | ID           |
      |  | GlobalBaselinePolicyForAccessors | 0       | Copy a78f1f… |
      |  | GlobalBaselinePolicyForMutators  | 0       | Copy 804e84… |
      |  | Allow_all                        | 0       | Copy 0c0b73… |
      |  | Dont_Allow_Any                   | 1       | Copy 000000… |
    And I should see a link to "/accesspolicies/0c0b7371-5175-405b-a17c-fec5969914b8/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And the page title should be "[dev] UserClouds Console"

  @admin
  @delete_access_policies
  Scenario: access policies page delete
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": true, "read": true, "update": true, "delete": true } |
    And a mocked "GET" request for "access_policies"
    When I navigate to the page with path "/accesspolicies?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a&access_policies_limit=2"
    Then I should see a "p" with the text "Loading policies..."
    And the page title should be "[dev] UserClouds Console"
    And I should see a link to "/accesspolicies/0c0b7371-5175-405b-a17c-fec5969914b8/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a table with ID "accessPolicies" and the following data
      |  | Name           | Version | ID           |
      |  | Allow_all      | 0       | Copy 0c0b73… |
      |  | Dont_Allow_Any | 1       | Copy 000000… |
    # queue an item
    When I toggle the checkbox in column 1 of row 1 of the table with ID "accessPolicies"
    Then row 1 of the table with ID "accessPolicies" should be marked for delete
    And row 2 of the table with ID "accessPolicies" should not be marked for delete
    # unqueue an item
    When I toggle the checkbox in column 1 of row 1 of the table with ID "accessPolicies"
    Then row 1 of the table with ID "accessPolicies" should not be marked for delete
    And row 2 of the table with ID "accessPolicies" should not be marked for delete
    # queue a different item
    When I toggle the checkbox in column 1 of row 2 of the table with ID "accessPolicies"
    Then row 1 of the table with ID "accessPolicies" should not be marked for delete
    And row 2 of the table with ID "accessPolicies" should be marked for delete
    Given the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                                                                                                           |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/00000000-5175-405b-a17c-fec5969914b8 | 409    | {"error":{"http_status_code":409,"error":{"error":"foo"}},"request_id":"2d209fe1-3e67-46ae-9d42-e1694469120a"} |
    # failed request
    When I click the button with ID "deleteAccessPoliciesButton"
    Then I should see the following text within the dialog titled "Delete Access Policies"
      | Selector | Text                                                                          |
      | div      | Are you sure you want to delete 1 access policy? This action is irreversible. |
    When I click the button with ID "confirmDeleteButton"
    Then I should see a table with ID "accessPolicies" and the following data
      |  | Name           | Version | ID           |
      |  | Allow_all      | 0       | Copy 0c0b73… |
      |  | Dont_Allow_Any | 1       | Copy 000000… |
    Then row 1 of the table with ID "accessPolicies" should not be marked for delete
    And row 2 of the table with ID "accessPolicies" should be marked for delete
    And I should see the following text on the page
      | TagName                | TextContent                                                            |
      | .alert-message ul > li | Error deleting access policy 00000000-5175-405b-a17c-fec5969914b8: foo |
    Given the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access?*                                    | 200    | access_policies.json |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/00000000-5175-405b-a17c-fec5969914b8 | 200    | {}                   |
    # this shows the original two access policies, but that's fine
    # all requests succeed
    When I click the button with ID "deleteAccessPoliciesButton"
    Then I should see the following text within the dialog titled "Delete Access Policies"
      | Selector | Text                                                                          |
      | div      | Are you sure you want to delete 1 access policy? This action is irreversible. |
    When I click the button with ID "confirmDeleteButton"
    And I should see the following text on the page
      | TagName            | TextContent                           |
      | .success-message p | Successfully deleted access policies. |
    # again, this makes it look like we didn't delete anything, but that's not what we're testing
    And I should see a table with ID "accessPolicies" and the following data
      |  | Name           | Version | ID           |
      |  | Allow_all      | 0       | Copy 0c0b73… |
      |  | Dont_Allow_Any | 1       | Copy 000000… |
