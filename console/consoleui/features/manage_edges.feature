@authz
Feature: manage edges page

  Scenario: manage edges empty state
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edge*        | 200    | {}                      |

    When I navigate to the page with path "/edges?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a card with the title "Edge types"
    And I should see a button labeled "Create Edge"
    And I should see a "h2" with the text "No Edges"

  Scenario: manage edges full state
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edge*        | 200    | authz_edges.json        |

    When I navigate to the page with path "/edges?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And the button with title "View previous page" should be disabled
    And the button with title "View next page" should be disabled
    And I should see a link to "/edges/cc51df08-18a2-4a08-b479-d676b16e8584?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "edges" and the following data
      |  | Edge ID | Edge Type | Source Object | Target Object |
      |  | 149fe5… | 1569fa…   | 1ee449…       | 61553e…       |
      |  | 188f0d… | ca3d97…   | 61553e…       | 3f65ee…       |
      |  | 35d08a… | ca3d97…   | 61553e…       | b9bf35…       |
      |  | 60b54e… | ca3d97…   | 61553e…       | c0b5b2…       |
      |  | 62e1a6… | ca3d97…   | 61553e…       | 0cedf7…       |
      |  | 9625c2… | ca3d97…   | 61553e…       | e3743f…       |
      |  | b8ca2b… | 237aba…   | de6874…       | 1ee449…       |
      |  | c3bf38… | ca3d97…   | 61553e…       | 618a4a…       |
      |  | cb1716… | 237aba…   | de6874…       | a40097…       |
      |  | cc51df… | 2dbccb…   | 61553e…       | 3f380e…       |
    And I should see a button labeled "Create Edge"
