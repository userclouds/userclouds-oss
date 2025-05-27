@authz
@authz_objects
Feature: authz objects list

  @a11y
  Scenario: objects list accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects_page_1.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json   |
    When I navigate to the page with path "/objects?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: empty objects list
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | []                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
    When I navigate to the page with path "/objects?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a "h2" with the text "No objects"

  Scenario: objects list
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects_page_1.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json   |
    When I navigate to the page with path "/objects?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    And I should see a link to "/objects/140e8dbb-085a-4cf5-be5c-9889c6a2c6cd?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "objectsTable" and the following data
      |  | ID      | alias          | Type Name                            |
      |  | 07aab4… | Azure AVM - 17 | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 09742f… | GCP GCE - 16   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 0cedf7… | AWS EC2 - 13   | _transformer                         |
      |  | 11c239… | AWS EC2 - 12   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 140e8d… | GCP GCE - 14   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
    And the page title should be "[dev] UserClouds Console"
    Given the following mocked requests:
      | Method | Path                                                             | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects* | 200    | authz_objects_page_2.json |
    When I click the 1st button with ID "next"
    And I should see a table with ID "objectsTable" and the following data
      |  | ID      | alias                                           | Type Name                            |
      |  | 1fb536… | Azure AVM - 20                                  | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 25310e… | GCP GCE - 1                                     | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 27dc69… | AWS EC2 - 14                                    | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 2b92f8… | Azure AVM - 5                                   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 2c83cf… | company1 (2c83cf53-cfe2-46c6-88b1-8a2a5d45e3d0) | _group                               |
    Given the following mocked requests:
      | Method | Path                                                             | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects* | 200    | authz_objects_page_3.json |
    When I click the 1st button with ID "next"
    And I should see a table with ID "objectsTable" and the following data
      |  | ID      | alias          | Type Name                            |
      |  | 45b3cb… | Azure AVM - 11 | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 491ad0… | GCP GCE - 8    | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 4e5020… | GCP GCE - 4    | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 4e723f… | Azure AVM - 8  | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 4ee97a… | AWS EC2 - 1    | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
    Given the following mocked requests:
      | Method | Path                                                             | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects* | 200    | authz_objects_page_2.json |
    When I click the 1st button with ID "prev"
    And I should see a table with ID "objectsTable" and the following data
      |  | ID      | alias                                           | Type Name                            |
      |  | 1fb536… | Azure AVM - 20                                  | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 25310e… | GCP GCE - 1                                     | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 27dc69… | AWS EC2 - 14                                    | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 2b92f8… | Azure AVM - 5                                   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 2c83cf… | company1 (2c83cf53-cfe2-46c6-88b1-8a2a5d45e3d0) | _group                               |

  Scenario: objects list delete
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects_page_1.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json   |
    When I navigate to the page with path "/objects?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a link to "/objects/140e8dbb-085a-4cf5-be5c-9889c6a2c6cd?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "objectsTable" and the following data
      |  | ID      | alias          | Type Name                            |
      |  | 07aab4… | Azure AVM - 17 | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 09742f… | GCP GCE - 16   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 0cedf7… | AWS EC2 - 13   | _transformer                         |
      |  | 11c239… | AWS EC2 - 12   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 140e8d… | GCP GCE - 14   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
    And the page title should be "[dev] UserClouds Console"
    When I toggle the checkbox in column 1 of row 2 of the table with ID "objectsTable"
    Then I should see a table with ID "objectsTable" and the following data
      |  | ID      | alias          | Type Name                            |
      |  | 07aab4… | Azure AVM - 17 | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 09742f… | GCP GCE - 16   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 0cedf7… | AWS EC2 - 13   | _transformer                         |
      |  | 11c239… | AWS EC2 - 12   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 140e8d… | GCP GCE - 14   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
    And I should not see an element matching selector "table > tbody > tr:first-child[class*='queuedfordelete']"
    And I should see an element matching selector "table > tbody > tr:nth-child(2)[class*='queuedfordelete']"
    When I toggle the checkbox in column 1 of row 5 of the table with ID "objectsTable"
    Then I should see a table with ID "objectsTable" and the following data
      |  | ID      | alias          | Type Name                            |
      |  | 07aab4… | Azure AVM - 17 | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 09742f… | GCP GCE - 16   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 0cedf7… | AWS EC2 - 13   | _transformer                         |
      |  | 11c239… | AWS EC2 - 12   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 140e8d… | GCP GCE - 14   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
    And I should not see an element matching selector "table > tbody > tr:first-child[class*='queuedfordelete']"
    And I should see an element matching selector "table > tbody > tr:nth-child(5)[class*='queuedfordelete']"
    Given the following mocked requests:
      | Method | Path                                                                                                 | Status | Body |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects/115de207-a4e6-4c56-a62e-b1856c13a7e2 | 200    | null |
    When I click the button with ID "deleteObjectsButton"
    Then I should see the following text within the dialog titled "Delete Objects"
      | Selector | Text                                                                    |
      | div      | Are you sure you want to delete 2 objects? This action is irreversible. |
    When I click the button with ID "confirmDeleteButton"
    And I should not see an element matching selector "#objectsTable table > tbody > tr:first-child[class*='queuedfordelete']"

  Scenario: objects list filter
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects_page_1.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json   |
    When I navigate to the page with path "/objects?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a link to "/objects/140e8dbb-085a-4cf5-be5c-9889c6a2c6cd?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "objectsTable" and the following data
      |  | ID      | alias          | Type Name                            |
      |  | 07aab4… | Azure AVM - 17 | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 09742f… | GCP GCE - 16   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 0cedf7… | AWS EC2 - 13   | _transformer                         |
      |  | 11c239… | AWS EC2 - 12   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
      |  | 140e8d… | GCP GCE - 14   | 115de207-a4e6-4c56-a62e-b1856c13a7e2 |
    And the page title should be "[dev] UserClouds Console"
    When I replace the text in the "search_value_1" field with "140e8dbb-085a-4cf5-be5c-9889c6aaaaaa"
    When I click the button labeled "Add Filter"
    Then I should see an element matching selector "#searchBarFilters span:has-text('id')"
    And I should see an element matching selector "#searchBarFilters span:has-text('EQ 140e8dbb-085a-4cf5-be5c-9889c6aaaaaa')"
    When I click the button labeled "Clear Filters"
    Then I should not see an element matching selector "#searchBarFilters span:has-text('id')"
    And I should not see an element matching selector "#searchBarFilters span:has-text('EQ 140e8dbb-085a-4cf5-be5c-9889c6aaaaaa')"
    When I replace the text in the "search_value_1" field with "search text example"
    And I select the option labeled "Filter by Alias" in the dropdown with ID "objectscolumns"
    And I click the button labeled "Add Filter"
    Then I should see an element matching selector "#searchBarFilters span:has-text('alias')"
    And I should see an element matching selector "#searchBarFilters span:has-text('LK %search text example%')"
    When I select the option labeled "Filter by Created" in the dropdown with ID "objectscolumns"
    And I replace the text in the "search_value_1" field with "2022/12/05"
    And I replace the text in the "search_value_2" field with "2023/12/05"
    And I click the button labeled "Add Filter"
    Then I should see 2 elements matching selector "#searchBarFilters span:has-text('created')"
    And I should see an element matching selector "#searchBarFilters span:has-text('GE 12/5/2022')"
    And I should see an element matching selector "#searchBarFilters span:has-text('LE 12/5/2023')"
