@transformers
@transformers_list
Feature: transformers

  @a11y
  Scenario: transformers list accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": true, "read": true, "update": true, "delete": true } |
    And a mocked "GET" request for "transformers"
    When I navigate to the page with path "/transformers?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  @admin
  Scenario: transformers list
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": true, "read": true, "update": true, "delete": true } |
    And a mocked "GET" request for "transformers"
    When I navigate to the page with path "/transformers?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading transformers..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a link to "/transformers/e3743f5b-521e-4305-b232-ee82549e1477/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "transformers" and the following data
      |  | Name                     | Transform Type    | ID           |
      |  | Passthrough              | Passthrough       | Copy 405d7c… |
      |  | Always_foo               | Passthrough       | Copy 000000… |
      |  | EmailToID                | Tokenize by value | Copy 0cedf7… |
      |  | SSNToID                  | Transform         | Copy 3f65ee… |
      |  | CreditCardToID           | Transform         | Copy 618a4a… |
      |  | FullNameToID             | Transform         | Copy b9bf35… |
      |  | PassthroughUnchangedData | Passthrough       | Copy c0b5b2… |
      |  | UUID                     | Tokenize by value | Copy e3743f… |
      |  | UUIDShouldntShowUp       | Tokenize by value | Copy 000000… |
      |  | UUIDShouldShowUpMutator  | Tokenize by value | Copy 000000… |
      |  | UUIDShouldShowUpAccessor | Tokenize by value | Copy 000000… |

  Scenario: transformers list without read permissions
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": false, "read": false, "update": false, "delete": false } |
    When I navigate to the page with path "/transformers?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading transformers..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a "p" with the text "You do not have permission to view any transformers. Please contact your administrator to request access."
    And I should see a card with the title "Request access" and the description "You do not have permission to view any transformers. Please contact your administrator to request access."

  @admin
  @delete_transformers
  Scenario: delete transformers list partial success
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": true, "read": true, "update": true, "delete": true } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates*  | 200    | { "data": [], "has_next": "false", "has_prev": false }           |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access*     | 200    | { "data": [], "has_next": "false", "has_prev": false }           |
    And a mocked "GET" request for "transformers"
    When I navigate to the page with path "/transformers?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading transformers..."
    And the page title should be "[dev] UserClouds Console"
    And I should see a link to "/transformers/e3743f5b-521e-4305-b232-ee82549e1477/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "transformers" and the following data
      |  | Name                     | Transform Type    | ID           |
      |  | Passthrough              | Passthrough       | Copy 405d7c… |
      |  | Always_foo               | Passthrough       | Copy 000000… |
      |  | EmailToID                | Tokenize by value | Copy 0cedf7… |
      |  | SSNToID                  | Transform         | Copy 3f65ee… |
      |  | CreditCardToID           | Transform         | Copy 618a4a… |
      |  | FullNameToID             | Transform         | Copy b9bf35… |
      |  | PassthroughUnchangedData | Passthrough       | Copy c0b5b2… |
      |  | UUID                     | Tokenize by value | Copy e3743f… |
      |  | UUIDShouldntShowUp       | Tokenize by value | Copy 000000… |
      |  | UUIDShouldShowUpMutator  | Tokenize by value | Copy 000000… |
      |  | UUIDShouldShowUpAccessor | Tokenize by value | Copy 000000… |
    # queue a transformer for delete
    When I toggle the checkbox in column 1 of row 7 of the table with ID "transformers"
    Then the button with ID "deleteTransformersButton" should be enabled
    And row 1 of the table with ID "transformers" should not be marked for delete
    And row 2 of the table with ID "transformers" should not be marked for delete
    And row 3 of the table with ID "transformers" should not be marked for delete
    And row 4 of the table with ID "transformers" should not be marked for delete
    And row 5 of the table with ID "transformers" should not be marked for delete
    And row 6 of the table with ID "transformers" should not be marked for delete
    And row 7 of the table with ID "transformers" should be marked for delete
    And row 8 of the table with ID "transformers" should not be marked for delete
    # queue multiple for delete
    When I toggle the checkbox in column 1 of row 4 of the table with ID "transformers"
    And I toggle the checkbox in column 1 of row 3 of the table with ID "transformers"
    And I toggle the checkbox in column 1 of row 8 of the table with ID "transformers"
    Then row 1 of the table with ID "transformers" should not be marked for delete
    And row 2 of the table with ID "transformers" should not be marked for delete
    And row 3 of the table with ID "transformers" should be marked for delete
    And row 4 of the table with ID "transformers" should be marked for delete
    And row 5 of the table with ID "transformers" should not be marked for delete
    And row 6 of the table with ID "transformers" should not be marked for delete
    And row 7 of the table with ID "transformers" should be marked for delete
    And row 8 of the table with ID "transformers" should be marked for delete
    And the button with ID "deleteTransformersButton" should be enabled
    # unqueue an item
    When I toggle the checkbox in column 1 of row 8 of the table with ID "transformers"
    Then row 1 of the table with ID "transformers" should not be marked for delete
    And row 2 of the table with ID "transformers" should not be marked for delete
    And row 3 of the table with ID "transformers" should be marked for delete
    And row 4 of the table with ID "transformers" should be marked for delete
    And row 5 of the table with ID "transformers" should not be marked for delete
    And row 6 of the table with ID "transformers" should not be marked for delete
    And row 7 of the table with ID "transformers" should be marked for delete
    And row 8 of the table with ID "transformers" should not be marked for delete
    And the button with ID "deleteTransformersButton" should be enabled
    Given the following mocked requests:
      | Method | Path                                                                                                           | Status | Body                                                                                                           |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/3f65ee22-2241-4694-bbe3-72cefbe59ff2 | 200    | {}                                                                                                             |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/0cedf7a4-86ab-450a-9426-478ad0a60faa | 409    | {"error":{"http_status_code":409,"error":{"error":"foo"}},"request_id":"2d209fe1-3e67-46ae-8aff-2930c705046d"} |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a | 409    | {"error":{"http_status_code":409,"error":{"error":"bar"}},"request_id":"2d209fe1-3e67-4068-94ed-3dcdaaaa53bc"} |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation*                                     | 200    | delete_transformers_partial_success.json                                                                       |
    When I click the button with ID "deleteTransformersButton"
    Then I should see the following text within the dialog titled "Delete Transformers"
      | Selector | TextContent                                                                  |
      | div      | Are you sure you want to delete 3 transformers? This action is irreversible. |
    When I click the button with ID "confirmDeleteButton"
    Then the button with ID "deleteTransformersButton" should be enabled
    Then I should see a table with ID "transformers" and the following data
      |  | Name                     | Transform Type    | ID           |
      |  | Passthrough              | Passthrough       | Copy 405d7c… |
      |  | Always_foo               | Passthrough       | Copy 000000… |
      |  | EmailToID                | Tokenize by value | Copy 0cedf7… |
      |  | CreditCardToID           | Transform         | Copy 618a4a… |
      |  | FullNameToID             | Transform         | Copy b9bf35… |
      |  | PassthroughUnchangedData | Passthrough       | Copy c0b5b2… |
      |  | UUID                     | Tokenize by value | Copy e3743f… |
    Then row 1 of the table with ID "transformers" should not be marked for delete
    And row 2 of the table with ID "transformers" should not be marked for delete
    And row 3 of the table with ID "transformers" should be marked for delete
    And row 4 of the table with ID "transformers" should not be marked for delete
    And row 5 of the table with ID "transformers" should not be marked for delete
    And row 6 of the table with ID "transformers" should be marked for delete
    And row 7 of the table with ID "transformers" should not be marked for delete
    And I should see the following text on the page
      | TagName                | TextContent                                                          |
      | .alert-message ul > li | Error deleting transformer 0cedf7a4-86ab-450a-9426-478ad0a60faa: foo |
      | .alert-message ul > li | Error deleting transformer c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a: bar |
