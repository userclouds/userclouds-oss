@authz
@object_type
Feature: object type page

  @a11y
  Scenario: object type list accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json            |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json       |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json       |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edges*       | 200    | empty_paginated_response.json |
    When I navigate to the page with path "/objecttypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: create object type
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json            |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json       |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json       |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edges*       | 200    | empty_paginated_response.json |
    When I navigate to the page with path "/objecttypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a card with the title "Object types"
    And the button labeled "Create Object Type" should be enabled
    Given the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
    When I click the button labeled "Create Object Type"
    Then I should see a card with the title "Basic Details"
    And the button labeled "Create Object Type" should be disabled
    When I type "new object type" into the input with ID "name"
    Then the button labeled "Create Object Type" should be enabled
    Given the following mocked requests:
      | Method | Path                                                                | Status | Body                             |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes | 200    | authz_created_object_type_1.json |
    When I click the button labeled "Create Object Type"
    Then I should see a card with the title "Object types"
    And I should see a toast notification with the text "Successfully created object typeClose"
    And I should be on the page with the path "/objecttypes/2e0631ca-d43c-4a66-9579-7eabfeaf3740/"

  Scenario: create object type error
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
    When I navigate to the page with path "/objecttypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a card with the title "Object types"
    Given the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
    When I click the button labeled "Create Object Type"
    Then I should see a card with the title "Basic Details"
    And the button labeled "Create Object Type" should be disabled
    When I type "new object type" into the input with ID "name"
    Then the button labeled "Create Object Type" should be enabled
    Given the following mocked requests:
      | Method | Path                                                                | Status | Body                                                                                              |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes | 409    | {"error":"This object type already exists", "request_id": "114a04ac-f9b2-4341-b8ed-1c48f049ded4"} |
    When I click the button labeled "Create Object Type"
    Then I should see a card with the title "Basic Details"
    And the button labeled "Create Object Type" should be enabled
    And I should see a "p" with the text "error creating object type: This object type already exists"

  Scenario: object type detail displays basic details
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                            |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json              |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types_created.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json         |
    When I navigate to the page with path "/objecttypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a card with the title "Object types"
    And I should see a link to "/objecttypes/2e0631ca-d43c-4a66-9579-7eabfeaf3740?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given the following mocked requests:
      | Method | Path                                                                                                                            | Status | Body                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes/2e0631ca-d43c-4a66-9579-7eabfeaf3740                        | 200    | authz_created_object_type_1.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects?limit=50&version=3&type_id=2e0631ca-d43c-4a66-9579-7eabfeaf3740 | 200    | {}                               |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes*                                                            | 200    | authz_object_types_created.json  |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*                                                              | 200    | authz_edge_types_1.json          |
    When I click the link with the href "/objecttypes/2e0631ca-d43c-4a66-9579-7eabfeaf3740?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching Object Type..."
    And I should see a card with the title "Basic Details"
    And I should see the following text on the page
      | TagName | TextContent     |
      | label   | Name            |
      | p       | new object type |
      | label   | ID              |
      | span    | 2e0631…         |

  Scenario: object type detail displays with edge types and objects of object type
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
    When I navigate to the page with path "/objecttypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a card with the title "Object types"
    And I should see a link to "/objecttypes/f5bce640-f866-4464-af1a-9e7474c4a90c?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given the following mocked requests:
      | Method | Path                                                                                                                            | Status | Body                                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes/f5bce640-f866-4464-af1a-9e7474c4a90c                        | 200    | authz_group_object_type.json         |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects?limit=50&version=3&type_id=f5bce640-f866-4464-af1a-9e7474c4a90c | 200    | authz_objects_for_object_type_1.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes*                                                            | 200    | authz_object_types.json              |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*                                                              | 200    | authz_edge_types_1.json              |
    When I click the link with the href "/objecttypes/f5bce640-f866-4464-af1a-9e7474c4a90c?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching Object Type..."
    And I should see a card with the title "Basic Details"
    And I should see the following text on the page
      | TagName | TextContent |
      | label   | Name        |
      | p       | _group      |
      | label   | ID          |
      | span    | f5bce6…     |
    And I should see a card with the title "Edge Types with this Source"
    And I should see a table within the "Edge Types with this Source" card row with the following data
      | Type Name          | Source Object Type | Target Object Type | ID           |
      | _group_policy_full | _group             | _policies          | Copy 1569fa… |
      | _group_policy_read | _group             | _policies          | Copy 1eb8cb… |
      | _member_deprecated | _group             | _user              | Copy 1eec16… |
      | _admin_deprecated  | _group             | _user              | Copy 60b696… |
    And I should see a card with the title "Edge Types with this Target"
    And I should see a table with ID "edgeTypesWithThisTarget" and the following data
      | Type Name               | Source Object Type | Target Object Type | ID           |
      | _admin                  | _user              | _group             | Copy 237aba… |
      | _user_group_policy_full | _user              | _group             | Copy 517348… |
      | _member                 | _user              | _group             | Copy e5eb50… |
      | _user_group_policy_read | _user              | _group             | Copy f11c57… |
    And I should see a card with the title "Objects"
    And I should see a table with ID "objectsTable" and the following data
      |  | ID      | Alias                                                 | Type Name |
      |  | 1ee449… | UserClouds Dev (1ee4497e-c326-4068-94ed-3dcdaaaa53bc) | _group    |
      |  | a40097… | company1 (a400970c-5a1d-46df-8050-31566d7da586)       | _group    |

  Scenario: object type detail displays empty state for edge types
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
    When I navigate to the page with path "/objecttypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a card with the title "Object types"
    And I should see a link to "/objecttypes/f5bce640-f866-4464-af1a-9e7474c4a90c?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given the following mocked requests:
      | Method | Path                                                                                                                            | Status | Body                                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes/f5bce640-f866-4464-af1a-9e7474c4a90c                        | 200    | authz_group_object_type.json         |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects?limit=50&version=3&type_id=f5bce640-f866-4464-af1a-9e7474c4a90c | 200    | authz_objects_for_object_type_1.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes*                                                            | 200    | authz_object_types.json              |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*                                                              | 200    | []                                   |
    When I click the link with the href "/objecttypes/f5bce640-f866-4464-af1a-9e7474c4a90c?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching Object Type..."
    And I should see a card with the title "Basic Details"
    And I should see the following text on the page
      | TagName | TextContent |
      | label   | Name        |
      | p       | _group      |
      | label   | ID          |
      | span    | f5bce6…     |
    And I should see a card with the title "Edge Types with this Source"
    And I should see a "h2" with the text "No edges with this source"
    And I should see a card with the title "Edge Types with this Target"
    And I should see a "h2" with the text "No edges with this target"
    And I should see a card with the title "Objects"
    And I should see a table with ID "objectsTable" and the following data
      |  | ID      | Alias                                                 | Type Name |
      |  | 1ee449… | UserClouds Dev (1ee4497e-c326-4068-94ed-3dcdaaaa53bc) | _group    |
      |  | a40097… | company1 (a400970c-5a1d-46df-8050-31566d7da586)       | _group    |
