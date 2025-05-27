@columns
Feature: columns page

  @a11y
  Scenario: Edit columns list accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema"
    ##the first one gets display columns. the second one gets all columns - needed for creating accessors and mutators.
    And a mocked "GET" request for "user store schema"
    And a mocked "GET" request for "accessors"
    And a mocked "GET" request for "mutators"
    And a mocked "GET" request for "dataTypes"
    And the following mocked requests:
      | Method | Path                                                                  | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes* | 200    | empty_paginated_response.json |
    When I navigate to the page with path "/columns?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    Then the page should have no accessibility violations

  Scenario: Edit columns list
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema"
    ##the first one gets display columns. the second one gets all columns - needed for creating accessors and mutators.
    And a mocked "GET" request for "user store schema"
    And a mocked "GET" request for "accessors"
    And a mocked "GET" request for "mutators"
    And a mocked "GET" request for "dataTypes"
    And the following mocked requests:
      | Method | Path                                                                  | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes* | 200    | empty_paginated_response.json |
    When I navigate to the page with path "/columns?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a "span" with the text "83cc42…"
    And I should see a table with ID "columnsTable" and the following data
      |  | Name           | Table | Column Type | ID           | Array |
      |  | email_verified | users | boolean     | Copy 032cae… | Off   |
      |  | email          | users | string      | Copy 2c7a7c… | Off   |
      |  | picture        | users | string      | Copy 4d4d07… | Off   |
      |  | name           | users | string      | Copy 62fcf8… | Off   |
      |  | nickname       | users | string      | Copy 83cc42… | Off   |
    When I toggle the checkbox in column 1 of row 5 of the table with ID "columnsTable"
    Then I should see a table with ID "columnsTable" and the following data
      |  | Name           | Table | Column Type | ID           | Array |
      |  | email_verified | users | boolean     | Copy 032cae… | Off   |
      |  | email          | users | string      | Copy 2c7a7c… | Off   |
      |  | picture        | users | string      | Copy 4d4d07… | Off   |
      |  | name           | users | string      | Copy 62fcf8… | Off   |
      |  | nickname       | users | string      | Copy 83cc42… | Off   |
    And I should not see an element matching selector "#userstoreColumns table > tbody > tr:first-child[class*='queuedfordelete']"
    And I should not see an element matching selector "#userstoreColumns table > tbody > tr:nth-child(2)[class*='queuedfordelete']"
    And I should not see an element matching selector "#userstoreColumns table > tbody > tr:nth-child(3)[class*='queuedfordelete']"
    And I should not see an element matching selector "#userstoreColumns table > tbody > tr:nth-child(4)[class*='queuedfordelete']"
    And I should see an element matching selector "#userstoreColumns table > tbody > tr:nth-child(5)[class*='queuedfordelete']"
    Given the following mocked requests:
      | Method | Path                                                                                                     | Status | Body                       |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns*                                     | 200    | userstoreschema_edit2.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns*                                     | 200    | userstoreschema_edit2.json |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns                                      | 200    | {}                         |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns/032cae17-df3a-4e87-82a0-c706ed0679ee | 200    | {}                         |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns/62fcf8b4-48d0-46d9-9f5b-d0813a478a2b | 200    | {}                         |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns/83cc42b0-da8c-4a61-9db1-da70f21bab60 | 200    | {}                         |
    When I click the button with ID "deleteColumnsButton"
    Then I should see the following text within the dialog titled "Delete Columns"
      | Selector | Text                                                                   |
      | div      | Are you sure you want to delete 1 column? This action is irreversible. |
    When I click the button with ID "cancelDeleteButton"
    Then I should not see an element matching selector "#cancelDeleteButton"
    When I click the button with ID "deleteColumnsButton"
    Then I should see the following text within the dialog titled "Delete Columns"
      | Selector | Text                                                                   |
      | div      | Are you sure you want to delete 1 column? This action is irreversible. |
    When I click the button with ID "confirmDeleteButton"
    Then I should see the following text on the page
      | TagName | TextContent         |
      | p       | Successfully saved. |
    And I should see a "span" with the text "62fcf8…"
    Then I should see a table with ID "columnsTable" and the following data
      |  | Name           | Table | Column Type | ID           | Array |
      |  | email_verified | users | boolean     | Copy 032cae… | Off   |
      |  | email          | users | string      | Copy 2c7a7c… | Off   |
      |  | picture        | users | string      | Copy 4d4d07… | Off   |
      |  | name           | users | string      | Copy 62fcf8… | Off   |
    When I click the "delete" button in row 1 of the table with ID "columnsTable"
    Then I should see the following text within the dialog titled "Delete Column"
      | Selector | Text                                                                      |
      | div      | Are you sure you want to delete this column? This action is irreversible. |
    When I click the button with ID "confirmDeleteButton"
    Then I should see the following text on the page
      | TagName            | TextContent         |
      | .success-message p | Successfully saved. |
