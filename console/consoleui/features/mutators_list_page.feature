Feature: mutators page

  @a11y
  Scenario: Mutators list page accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "accessors"
    And a mocked "GET" request for "mutators"
    And the following mocked requests:
      | Method | Path                                                                  | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes* | 200    | empty_paginated_response.json |
    When I navigate to the page with path "/mutators?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  @mutators
  Scenario: Delete mutators
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
    When I navigate to the page with path "/mutators?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching tenant mutators..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a "span" with the text "1aa449…"
    And I should see a table with ID "mutators" and the following data
      |  | Name             | Table | Columns                                               | ID           | Version |
      |  | My_Mutator       | users | 1 Columns(baz_column)baz_column                       | Copy 2ee449… | 2       |
      |  | My other mutator | users | 2 Columns(bar_column, foo_column)bar_columnfoo_column | Copy 1aa449… | 1       |
    When I toggle the checkbox in column 1 of row 1 of the table with ID "mutators"
    Then I should see a table with ID "mutators" and the following data
      |  | Name             | Table | Columns                                               | ID           | Version |
      |  | My_Mutator       | users | 1 Columns(baz_column)baz_column                       | Copy 2ee449… | 2       |
      |  | My other mutator | users | 2 Columns(bar_column, foo_column)bar_columnfoo_column | Copy 1aa449… | 1       |
    And I should see an element matching selector "#userstoreMutators table > tbody > tr:first-child[class*='queuedfordelete']"
    And I should not see an element matching selector "#userstoreMutators table > tbody > tr:nth-child(2)[class*='queuedfordelete']"
    Given the following mocked requests:
      | Method | Path                                                                                                      | Status | Body |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/mutators/2ee4497e-c326-4068-94ed-3dcdaaaa53bc | 200    | null |
    When I click the button with ID "deleteMutatorsButton"
    Then I should see the following text within the dialog titled "Delete Mutators"
      | Selector | Text                                                                    |
      | div      | Are you sure you want to delete 1 mutator? This action is irreversible. |
    Then I should see a table with ID "mutators" and the following data
      |  | Name             | Table | Columns                                               | ID           | Version |
      |  | My_Mutator       | users | 1 Columns(baz_column)baz_column                       | Copy 2ee449… | 2       |
      |  | My other mutator | users | 2 Columns(bar_column, foo_column)bar_columnfoo_column | Copy 1aa449… | 1       |

  @mutators
  Scenario: No mutators
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    ##the first one gets display columns. the second one gets all columns - needed for creating accessors and mutators.
    And a mocked "GET" request for "user store schema edit"
    And the following mocked requests:
      | Method | Path                                                                  | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes* | 200    | empty_paginated_response.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/mutators* | 200    | empty_paginated_response.json |
    And a mocked "GET" request for "accessors"
    When I navigate to the page with path "/mutators?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should not see an element matching selector "#userstoreMutators table"
    And I should see the following text on the page
      | TagName                    | TextContent |
      | #userstoreMutators h2      | No mutators |
      | #userstoreMutators a[href] | Add Mutator |
