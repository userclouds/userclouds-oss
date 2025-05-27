Feature: purposes

  @a11y
  Scenario: purposes list accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    ##the first one gets display columns. the second one gets all columns - needed for creating accessors and mutators.
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "accessors"
    And a mocked "GET" request for "mutators"
    And the following mocked requests:
      | Method | Path                                                                  | Status | Body                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes* | 200    | purposes_page_1.json |
    When I navigate to the page with path "/purposes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  @purposes
  Scenario: No purposes
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
    When I navigate to the page with path "/purposes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should not see an element matching selector "#userstorePurposes table"
    And I should see the following text on the page
      | TagName                    | TextContent                                          |
      | #userstorePurposes h2      | Nothing to display                                   |
      | #userstorePurposes p       | No purposes have been specified yet for this tenant. |
      | #userstorePurposes a[href] | Create Purpose                                       |

  @purposes
  Scenario: Error fetching purposes
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    ##the first one gets display columns. the second one gets all columns - needed for creating accessors and mutators.
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "accessors"
    And a mocked "GET" request for "mutators"
    And the following mocked requests:
      | Method | Path                                                                  | Status | Body                                                                  |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes* | 404    | {"error":"uh-oh","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
    When I navigate to the page with path "/purposes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should not see an element matching selector "#userstorePurposes table"
    And I should see the following text on the page
      | TagName              | TextContent |
      | #userstorePurposes p | uh-oh       |

  @purposes
  Scenario: View purposes
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    ##the first one gets display columns. the second one gets all columns - needed for creating accessors and mutators.
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "accessors"
    And a mocked "GET" request for "mutators"
    And the following mocked requests:
      | Method | Path                                                                  | Status | Body                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes* | 200    | purposes_page_1.json |
    When I navigate to the page with path "/purposes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching purposes…"
    And the page title should be "[dev] UserClouds Console"
    Then I should see a link to "/purposes/7def2e9f-0c6a-489a-8549-4a673ad001e9?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a "span" with the text "7def2e…"
    And I should see a table with ID "purposes" and the following data
      |  | Name              | Description                 | ID           |
      |  | Marketing         | barraging you with emails   | Copy 1ee449… |
      |  | Analytics         | profiling you               | Copy 508620… |
      |  | Order fulfillment | Sending you stuff           | Copy 5ded72… |
      |  | Age verification  | Keeping your children safe  | Copy 79b600… |
      |  | Personalization   | Anything we want it to mean | Copy 7def2e… |

  @purposes
  @delete_purposes
  Scenario: Delete purposes
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    ##the first one gets display columns. the second one gets all columns - needed for creating accessors and mutators.
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "accessors"
    And a mocked "GET" request for "mutators"
    And the following mocked requests:
      | Method | Path                                                                  | Status | Body                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes* | 200    | purposes_page_1.json |
    When I navigate to the page with path "/purposes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching purposes…"
    And the page title should be "[dev] UserClouds Console"
    Then I should see a link to "/purposes/7def2e9f-0c6a-489a-8549-4a673ad001e9?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a "span" with the text "7def2e…"
    And I should see a table with ID "purposes" and the following data
      |  | Name              | Description                 | ID           |
      |  | Marketing         | barraging you with emails   | Copy 1ee449… |
      |  | Analytics         | profiling you               | Copy 508620… |
      |  | Order fulfillment | Sending you stuff           | Copy 5ded72… |
      |  | Age verification  | Keeping your children safe  | Copy 79b600… |
      |  | Personalization   | Anything we want it to mean | Copy 7def2e… |
    When I toggle the checkbox in column 1 of row 4 of the table with ID "purposes"
    Then I should see a table with ID "purposes" and the following data
      |  | Name              | Description                 | ID           |
      |  | Marketing         | barraging you with emails   | Copy 1ee449… |
      |  | Analytics         | profiling you               | Copy 508620… |
      |  | Order fulfillment | Sending you stuff           | Copy 5ded72… |
      |  | Age verification  | Keeping your children safe  | Copy 79b600… |
      |  | Personalization   | Anything we want it to mean | Copy 7def2e… |
    And row 1 of the table with ID "purposes" should not be marked for delete
    And row 2 of the table with ID "purposes" should not be marked for delete
    And row 3 of the table with ID "purposes" should not be marked for delete
    And row 4 of the table with ID "purposes" should be marked for delete
    And row 5 of the table with ID "purposes" should not be marked for delete
    Given the following mocked requests:
      | Method | Path                                                                                                      | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes*                                     | 200    | purposes_page_1_edit.json |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes/79b600f1-87a8-4a77-a206-9111b52efa84 | 200    | {}                        |
    When I click the button with ID "deletePurposesButton"
    Then I should see the following text within the dialog titled "Delete Purposes"
      | Selector | TextContent                                                             |
      | div      | Are you sure you want to delete 1 purpose? This action is irreversible. |
    When I click the button with ID "confirmDeleteButton"
    Then I should see the following text on the page
      | TagName | TextContent         |
      | p       | Successfully saved. |
    And I should see a link to "/purposes/7def2e9f-0c6a-489a-8549-4a673ad001e1?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "purposes" and the following data
      |  | Name              | Description                 | ID           |
      |  | Marketing         | barraging you with emails   | Copy 1ee449… |
      |  | Analytics         | profiling you               | Copy 508620… |
      |  | Order fulfillment | Sending you stuff           | Copy 5ded72… |
      |  | Personalization   | Anything we want it to mean | Copy 7def2e… |

  @purposes
  @delete_purposes
  Scenario: delete multiple purposes list partial success
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    ##the first one gets display columns. the second one gets all columns - needed for creating accessors and mutators.
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "accessors"
    And a mocked "GET" request for "mutators"
    And the following mocked requests:
      | Method | Path                                                                  | Status | Body                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes* | 200    | purposes_page_1.json |
    When I navigate to the page with path "/purposes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching purposes…"
    And the page title should be "[dev] UserClouds Console"
    Then I should see a link to "/purposes/7def2e9f-0c6a-489a-8549-4a673ad001e9?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a "span" with the text "7def2e…"
    And I should see a table with ID "purposes" and the following data
      |  | Name              | Description                 | ID           |
      |  | Marketing         | barraging you with emails   | Copy 1ee449… |
      |  | Analytics         | profiling you               | Copy 508620… |
      |  | Order fulfillment | Sending you stuff           | Copy 5ded72… |
      |  | Age verification  | Keeping your children safe  | Copy 79b600… |
      |  | Personalization   | Anything we want it to mean | Copy 7def2e… |
    When I toggle the checkbox in column 1 of row 4 of the table with ID "purposes"
    And I toggle the checkbox in column 1 of row 3 of the table with ID "purposes"
    Then I should see a table with ID "purposes" and the following data
      |  | Name              | Description                 | ID           |
      |  | Marketing         | barraging you with emails   | Copy 1ee449… |
      |  | Analytics         | profiling you               | Copy 508620… |
      |  | Order fulfillment | Sending you stuff           | Copy 5ded72… |
      |  | Age verification  | Keeping your children safe  | Copy 79b600… |
      |  | Personalization   | Anything we want it to mean | Copy 7def2e… |
    And row 1 of the table with ID "purposes" should not be marked for delete
    And row 2 of the table with ID "purposes" should not be marked for delete
    And row 3 of the table with ID "purposes" should be marked for delete
    And row 4 of the table with ID "purposes" should be marked for delete
    And row 5 of the table with ID "purposes" should not be marked for delete
    Given the following mocked requests:
      | Method | Path                                                                                                      | Status | Body                                                                                                           |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes/79b600f1-87a8-4a77-a206-9111b52efa84 | 200    | {}                                                                                                             |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes/5ded72de-c606-48ce-9675-df88557d56fe | 409    | {"error":{"http_status_code":409,"error":{"error":"foo"}},"request_id":"2d209fe1-3e67-46ae-8aff-2930c705046d"} |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes*                                     | 200    | purposes_page_1_edit.json                                                                                      |
    When I click the button with ID "deletePurposesButton"
    Then I should see the following text within the dialog titled "Delete Purposes"
      | Selector | TextContent                                                              |
      | div      | Are you sure you want to delete 2 purposes? This action is irreversible. |
    When I click the button with ID "confirmDeleteButton"
    Then I should see a table with ID "purposes" and the following data
      |  | Name              | Description                 | ID           |
      |  | Marketing         | barraging you with emails   | Copy 1ee449… |
      |  | Analytics         | profiling you               | Copy 508620… |
      |  | Order fulfillment | Sending you stuff           | Copy 5ded72… |
      |  | Personalization   | Anything we want it to mean | Copy 7def2e… |
    And row 1 of the table with ID "purposes" should not be marked for delete
    And row 2 of the table with ID "purposes" should not be marked for delete
    And row 3 of the table with ID "purposes" should be marked for delete
    And row 4 of the table with ID "purposes" should not be marked for delete
    And I should see the following text on the page
      | TagName                | TextContent                                                      |
      | .alert-message ul > li | error deleting purpose 5ded72de-c606-48ce-9675-df88557d56fe: foo |
