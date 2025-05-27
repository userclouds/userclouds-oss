@authz
Feature: edge types page

  @a11y
  Scenario: edge types page accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | {}   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | []   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | []   |
    When I navigate to the page with path "/edgetypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: edge types page empty states
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | {}   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | []   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | []   |
    When I navigate to the page with path "/edgetypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent      |
      | h2      | No edge types    |
      | button  | Create Edge Type |

  Scenario: edge types page full states
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
    When I navigate to the page with path "/edgetypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a table with ID "edgeTypes" and the following data
      |  | Type Name                    | Source Object Type | Target Object Type | ID           |
      |  | _group_policy_full           | _group             | _policies          | Copy 1569fa… |
      |  | _group_policy_read           | _group             | _policies          | Copy 1eb8cb… |
      |  | _member_deprecated           | _group             | _user              | Copy 1eec16… |
      |  | _admin                       | _user              | _group             | Copy 237aba… |
      |  | _policy_exist_access         | _policies          | _access_policy     | Copy 2dbccb… |
      |  | _user_group_policy_full      | _user              | _group             | Copy 517348… |
      |  | _user_policy_full            | _user              | _policies          | Copy 58840f… |
      |  | _admin_deprecated            | _group             | _user              | Copy 60b696… |
      |  | _user_policy_read            | _user              | _policies          | Copy bb2baf… |
      |  | _policy_exist_transformation | _policies          | _transformer       | Copy ca3d97… |
      |  | _member                      | _user              | _group             | Copy e5eb50… |
      |  | _can_login                   | _user              | _login_app         | Copy ea7239… |
      |  | _user_group_policy_read      | _user              | _group             | Copy f11c57… |
