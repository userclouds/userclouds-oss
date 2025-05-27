@accessors
Feature: Accessors page

  @a11y
  Scenario: Accessors list page accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    ##the first one gets display columns. the second one gets all columns - needed for creating accessors and mutators.
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "accessors"
    And a mocked "GET" request for "mutators"
    And the following mocked requests:
      | Method | Path                                                                  | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes* | 200    | empty_paginated_response.json |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/counters/query*     | 200    | accessor_metrics.json         |
    When I navigate to the page with path "/accessors?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: Delete accessors
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    ##the first one gets display columns. the second one gets all columns - needed for creating accessors and mutators.
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "accessors"
    And a mocked "GET" request for "mutators"
    And the following mocked requests:
      | Method | Path                                                                  | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes* | 200    | empty_paginated_response.json |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/counters/query*     | 200    | accessor_metrics.json         |
    When I navigate to the page with path "/accessors?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching tenant accessors..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a "span" with the text "3aa449…"
    And I should see a table with ID "accessors" and the following data
      |  | Name              | Tables | Columns                                               | ID           | Executions (30 days) | Version |
      |  | My_Accessor       | users  | 1 Columns(baz_column)baz_column                       | Copy 2ee449… | 22                   | 2       |
      |  | My_Other_Accessor | users  | 2 Columns(bar_column, foo_column)bar_columnfoo_column | Copy 3aa449… | 33                   | 1       |
    When I toggle the checkbox in column 1 of row 1 of the table with ID "accessors"
    Then I should see a "span" with the text "3aa449…"
    And I should see a table with ID "accessors" and the following data
      |  | Name              | Tables | Columns                                               | ID           | Executions (30 days) | Version |
      |  | My_Accessor       | users  | 1 Columns(baz_column)baz_column                       | Copy 2ee449… | 22                   | 2       |
      |  | My_Other_Accessor | users  | 2 Columns(bar_column, foo_column)bar_columnfoo_column | Copy 3aa449… | 33                   | 1       |
    And I should see an element matching selector "#userstoreAccessors table > tbody > tr:first-child[class*='queuedfordelete']"
    And I should not see an element matching selector "#userstoreAccessors table > tbody > tr:nth-child(2)[class*='queuedfordelete']"
    Given the following mocked requests:
      | Method | Path                                                                                                       | Status | Body |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc | 200    | null |
    When I click the button with ID "deleteAccessorsButton"
    Then I should see the following text within the dialog titled "Delete Accessors"
      | Selector | Text                                                                     |
      | div      | Are you sure you want to delete 1 accessor? This action is irreversible. |
    When I click the button with ID "cancelDeleteButton"
    Then I should not see an element matching selector "#cancelDeleteButton"
    When I click the button with ID "deleteAccessorsButton"
    Then I should see the following text within the dialog titled "Delete Accessors"
      | Selector | Text                                                                     |
      | div      | Are you sure you want to delete 1 accessor? This action is irreversible. |
    When I click the button with ID "confirmDeleteButton"
    Then I should see the following text on the page
      | TagName            | TextContent                    |
      | .success-message p | Successfully deleted accessors |
    Then I should see a "span" with the text "3aa449…"
    And I should see a table with ID "accessors" and the following data
      |  | Name              | Tables | Columns                                               | ID           | Executions (30 days) | Version |
      |  | My_Other_Accessor | users  | 2 Columns(bar_column, foo_column)bar_columnfoo_column | Copy 3aa449… | 33                   | 1       |

  @accessors
  Scenario: No accessors
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    ##the first one gets display columns. the second one gets all columns - needed for creating accessors and mutators.
    And a mocked "GET" request for "user store schema edit"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes*  | 200    | empty_paginated_response.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/accessors* | 200    | empty_paginated_response.json |
    And a mocked "GET" request for "mutators"
    When I navigate to the page with path "/accessors?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should not see an element matching selector "#userstoreAccessors table"
    And I should see the following text on the page
      | TagName                     | TextContent  |
      | #userstoreAccessors h2      | No accessors |
      | #userstoreAccessors a[href] | Add Accessor |
