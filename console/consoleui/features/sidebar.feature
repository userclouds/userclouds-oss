@sidebar
Feature: sidebar links

  Scenario: click the links in the sidebar
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    When I navigate to the page with path "/"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a card with the title "Tenant Details"
    Given a mocked "GET" request for "tenants"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    Given the following mocked requests:
      | Method | Path                                                                       | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/datamapping/datasources* | 200    | empty_paginated_response.json |
    When I wait for the network to be idle
    And I click "User Data Mapping" header in the sidebar
    And I click "Data Sources" in the sidebar
    Then I should see the following text on the page
      | TagName | TextContent     |
      | button  | Add data source |
    Given the following mocked requests:
      | Method | Path                                                                       | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/datamapping/elements*    | 200    | empty_paginated_response.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/datamapping/datasources* | 200    | empty_paginated_response.json |
    When I wait for the network to be idle
    And I click "Data Source Schemas" in the sidebar
    Then I should see the following text on the page
      | TagName | TextContent |
      | h2      | No schema   |
    When I wait for the network to be idle
    And I click "User Data Storage" header in the sidebar
    And I click "Columns" in the sidebar
    And I should see the following text on the page
      | TagName | TextContent   |
      | button  | Create Column |
    Given a mocked "GET" request for "tenants"
    And the following mocked requests:
      | Method | Path                                                                                                                 | Status | Body                         |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns?version=3&limit=1500                             | 200    | userstoreschema_default.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations?limit=1500&version=3                                 | 200    | end_users_page_orgs.json     |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/users?company_id=*&tenant_id=*&limit=50&organization_id=&version=3 | 200    | users_for_org_page_1.json    |
    When I wait for the network to be idle
    And I click "Users" in the sidebar
    Then I should see 3 links to "/users/03f84ae4-0d1a-4c87-b2ec-069606e38bc6?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given the following mocked requests:
      | Method | Path                                                                   | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/datatypes* | 200    | empty_paginated_response.json |
    When I wait for the network to be idle
    And I click "Data Types" in the sidebar
    Then I should see the following text on the page
      | TagName | TextContent      |
      | button  | Create Data Type |
    Given the following mocked requests:
      | Method | Path                                                                      | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/objectstores* | 200    | empty_paginated_response.json |
    When I wait for the network to be idle
    And I click "Object Stores" in the sidebar
    Then I should see the following text on the page
      | TagName | TextContent         |
      | button  | Create Object Store |
    Given a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    ##the first one gets display columns. the second one gets all columns - needed for creating accessors and mutators.
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "accessors"
    And a mocked "GET" request for "mutators"
    And the following mocked requests:
      | Method | Path                                                                  | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/purposes* | 200    | empty_paginated_response.json |
    When I wait for the network to be idle
    And I click "Access Methods" header in the sidebar
    And I click "Accessors" in the sidebar
    Then I should see a "span" with the text "3aa449…"
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
    When I wait for the network to be idle
    And I click "Mutators" in the sidebar
    And I should see a "span" with the text "1aa449…"
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
    When I wait for the network to be idle
    And I click "Access Rules" header in the sidebar
    And I click "Purposes" in the sidebar
    And I should see a link to "/purposes/7def2e9f-0c6a-489a-8549-4a673ad001e9?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": true, "read": true, "update": true, "delete": true } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates*  | 200    | policy_templates.json                                            |
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "transformers"
    When I wait for the network to be idle
    And I click "Access Policies" in the sidebar
    And I should see a link to "/accesspolicies/0c0b7371-5175-405b-a17c-fec5969914b8/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": true, "read": true, "update": true, "delete": true } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates*  | 200    | policy_templates.json                                            |
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "transformers"
    When I wait for the network to be idle
    And I click "Policy Templates" in the sidebar
    And I should see a link to "/policytemplates/c9cfe092-dbc2-4aca-a68d-85eb85126526/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given the following mocked requests:
      | Method | Path                                                                | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/secrets* | 200    | empty_paginated_response.json |
    When I wait for the network to be idle
    And I click "Secrets" in the sidebar
    Then I should see the following text on the page
      | TagName | TextContent   |
      | button  | Create Secret |
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                             | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations* | 200    | organizations_page_1.json |
    When I wait for the network to be idle
    And I click "Access Permissions" header in the sidebar
    And I click "Organizations" in the sidebar
    And I should see a link to "/organizations/7def2e9f-0c6a-489a-8549-4a673ad001e9?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
    When I wait for the network to be idle
    And I click "Object Types" in the sidebar
    And I should see a link to "/objecttypes/1bf2b775-e521-41d3-8b7e-78e89427e6fe?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
    When I wait for the network to be idle
    And I click "Edge Types" in the sidebar
    And I should see a link to "/edgetypes/1569fa26-2458-4041-8cf3-5532f0d0670a?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects_page_1.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | []                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | []                        |
    When I wait for the network to be idle
    And I click "Objects" in the sidebar
    And I should see a link to "/objects/140e8dbb-085a-4cf5-be5c-9889c6a2c6cd?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edge*        | 200    | authz_edges.json        |
    When I wait for the network to be idle
    And I click "Edges" in the sidebar
    And I should see a link to "/edges/cc51df08-18a2-4a08-b479-d676b16e8584?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": true, "read": true, "update": true, "delete": true } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates*  | 200    | { "data": [], "has_next": "false", "has_prev": false }           |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access*     | 200    | { "data": [], "has_next": "false", "has_prev": false }           |
    And a mocked "GET" request for "transformers"
    When I wait for the network to be idle
    And I click "User Data Masking" header in the sidebar
    And I click "Transformers" in the sidebar
    And I should see a link to "/transformers/e3743f5b-521e-4305-b232-ee82549e1477/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I wait for the network to be idle
    And I click "User Authentication" header in the sidebar
    And I click "Login Apps" in the sidebar
    And I should see a button labeled "Create App"
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I wait for the network to be idle
    And I click "OAuth Connections" in the sidebar
    And I should see a button labeled "Create Provider"
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I wait for the network to be idle
    And I click "Identity Providers" in the sidebar
    And I should see a button labeled "Save providers"
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    And a mocked "GET" request for "keys"
    When I wait for the network to be idle
    And I click "Comms Channels" in the sidebar
    And I should see a card with the title "Telephony Provider Settings"
    Given a mocked "GET" request for "tenants"
    And the following mocked requests:
      | Method | Path                                                                | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/auditlog/entries* | 200    | empty_paginated_response.json |
    When I wait for the network to be idle
    And I click "Monitoring" header in the sidebar
    And I click "Audit Log" in the sidebar
    Then I should see the following text on the page
      | TagName | TextContent                  |
      | h2      | No entries in the audit log. |
    Given the following mocked requests:
      | Method | Path                                                                     | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/dataaccesslog/entries* | 200    | empty_paginated_response.json |
    When I wait for the network to be idle
    And I click "Data Access Log" in the sidebar
    Then I should see the following text on the page
      | TagName | TextContent                        |
      | h2      | No entries in the data access log. |
    Given a mocked "GET" request for "tenants"
    And the following mocked requests:
      | Method | Path                                                    | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/runs* | 200    | empty_paginated_response.json |
    When I wait for the network to be idle
    And I click "System Log" in the sidebar
    Then I should see the following text on the page
      | TagName | TextContent                   |
      | h2      | No entries in the system log. |
    Given a mocked "GET" request for "tenants"
    And a mocked "GET" request for "tenants_urls"
    When I wait for the network to be idle
    And I click "Status" in the sidebar
    And I should see a card with the title "Infrastructure Resources"
    Given a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "database"
    And a mocked "GET" request for "plex_config"
    When I wait for the network to be idle
    And I click "Tenant Settings" in the sidebar
    Then I should see a cardrow with the title "Database Connections"
