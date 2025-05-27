@end_users
@organizations
@users
@end_users_page
Feature: end users page

  @a11y
  Scenario: end users list and pagination accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                                 | Status | Body                         |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns?version=3&limit=1500                             | 200    | userstoreschema_default.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations?limit=1500&version=3                                 | 200    | end_users_page_orgs.json     |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/users?company_id=*&tenant_id=*&limit=50&organization_id=&version=3 | 200    | users_for_org_page_1.json    |
    When I navigate to the page with path "/users?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: end users list and pagination
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                                 | Status | Body                         |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns?version=3&limit=1500                             | 200    | userstoreschema_default.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations?limit=1500&version=3                                 | 200    | end_users_page_orgs.json     |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/users?company_id=*&tenant_id=*&limit=50&organization_id=&version=3 | 200    | users_for_org_page_1.json    |
    When I navigate to the page with path "/users?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And the page title should be "[dev] UserClouds Console"
    Then I should see a table with ID "usersTable" and the following data
      |  |  | Email                                | ID      |
      |  |  | bender@userclouds.com (not verified) | 03f84a… |
      |  |  | leela@userclouds.com (verified)      | 042878… |
      |  |  | zoidberg@firstorg.com (not verified) | 0dfd39… |
      |  |  | amy@secondorg.com (verified)         | 130400… |
      |  |  | farnsworth@firstorg.com (verified)   | 1700fc… |
      |  |  | zapp@userclouds.com (verified)       | 220462… |
      |  |  | kif@userclouds.com (not verified)    | 285567… |
      |  |  | hermes@userclouds.com (verified)     | 2b5653… |
      |  |  | labarbara@userclouds.com (verified)  | 32f175… |
      |  |  | fry@userclouds.com (not verified)    | 39daeb… |
    And I should see a dropdown matching selector "[name='organization_id']" with the following options
      | Text              | Value                                | Selected |
      | All organizations |                                      | true     |
      | UserClouds Dev    | 1ee4497e-c326-4068-94ed-3dcdaaaa53bc |          |
      | Kyle's First Org  | 10ccc4fd-0d6b-4ded-889b-41eb119c650a |          |
      | Kyle's Second Org | 1e01c626-4f69-44ae-822d-a76741e54bab |          |
    And I should see 1 buttons with ID "prev" and they should be disabled
    And I should see 1 buttons with ID "next" and they should be enabled
    # go to page 2
    Given a mocked request for users across all orgs (page 2)
    When I click the 1st button with ID "next"
    Then I should see a table with ID "usersTable" and the following data
      |  |  | Email                                | ID      |
      |  |  | fry@userclouds.com (not verified)    | 39daeb… |
      |  |  | labarbara@userclouds.com (verified)  | 32f175… |
      |  |  | hermes@userclouds.com (verified)     | 2b5653… |
      |  |  | kif@userclouds.com (not verified)    | 285567… |
      |  |  | zapp@userclouds.com (verified)       | 220462… |
      |  |  | farnsworth@firstorg.com (verified)   | 1700fc… |
      |  |  | amy@secondorg.com (verified)         | 130400… |
      |  |  | zoidberg@firstorg.com (not verified) | 0dfd39… |
      |  |  | leela@userclouds.com (verified)      | 042878… |
      |  |  | bender@userclouds.com (not verified) | 03f84a… |
    And I should see 1 buttons with ID "prev" and they should be enabled
    And I should see 1 buttons with ID "next" and they should be enabled
    # go to page 3
    Given a mocked request for users across all orgs (page 3)
    When I click the 1st button with ID "next"
    Then I should see a table with ID "usersTable" and the following data
      |  |  | Email                                | ID      |
      |  |  | zapp@userclouds.com (verified)       | 220462… |
      |  |  | kif@userclouds.com (not verified)    | 285567… |
      |  |  | hermes@userclouds.com (verified)     | 2b5653… |
      |  |  | labarbara@userclouds.com (verified)  | 32f175… |
      |  |  | fry@userclouds.com (not verified)    | 39daeb… |
      |  |  | bender@userclouds.com (not verified) | 03f84a… |
      |  |  | leela@userclouds.com (verified)      | 042878… |
      |  |  | zoidberg@firstorg.com (not verified) | 0dfd39… |
      |  |  | amy@secondorg.com (verified)         | 130400… |
      |  |  | farnsworth@firstorg.com (verified)   | 1700fc… |
    And I should see 1 buttons with ID "prev" and they should be enabled
    And I should see 1 buttons with ID "next" and they should be disabled
    # go back to page 2
    Given a mocked request for users across all orgs (page 2)
    When I click the 1st button with ID "prev"
    Then I should see a table with ID "usersTable" and the following data
      |  |  | Email                                | ID      |
      |  |  | fry@userclouds.com (not verified)    | 39daeb… |
      |  |  | labarbara@userclouds.com (verified)  | 32f175… |
      |  |  | hermes@userclouds.com (verified)     | 2b5653… |
      |  |  | kif@userclouds.com (not verified)    | 285567… |
      |  |  | zapp@userclouds.com (verified)       | 220462… |
      |  |  | farnsworth@firstorg.com (verified)   | 1700fc… |
      |  |  | amy@secondorg.com (verified)         | 130400… |
      |  |  | zoidberg@firstorg.com (not verified) | 0dfd39… |
      |  |  | leela@userclouds.com (verified)      | 042878… |
      |  |  | bender@userclouds.com (not verified) | 03f84a… |
    And I should see 1 buttons with ID "prev" and they should be enabled
    And I should see 1 buttons with ID "next" and they should be enabled
    # switch orgs
    Given a mocked request for users for the org with ID "10ccc4fd-0d6b-4ded-889b-41eb119c650a"
    When I select the option labeled "Kyle's First Org" in the dropdown matching selector "[name='organization_id']"
    Then I should see a table with ID "usersTable" and the following data
      |  |  | Email                                | ID      |
      |  |  | zoidberg@firstorg.com (not verified) | 0dfd39… |
      |  |  | farnsworth@firstorg.com (verified)   | 1700fc… |
    And the button with title "View previous page" should be disabled
    And the button with title "View next page" should be disabled

  Scenario: end users list for tenant with orgs disabled
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "selectedTenant2"
    And a mocked "GET" request for "tenants_urls"
    When I navigate to the page with path "/"
    Then the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent                |
      | h1      | Tenant Home: Console - Dev |
      | a[href] | Manage Team                |
    # this check is a way of making sure we've responded to the nav menu feature flag
    And I should see a custom dropdown matching selector "#pageHeader #tenantSelectDropdown" with the following options
      | Text           | Value                                | Selected |
      | Console - Dev  | 41ab79a8-0dff-418e-9d42-e1694469120a | true     |
      | Food Tenant    | 9835e88f-bd66-4cd2-b1c4-dfc61de9a2e1 |          |
      | Change Company |                                      |          |
    Given a mocked "GET" request for "tenants_urls"
    When I select the option labeled "Food Tenant" in the custom dropdown matching selector "#pageHeader #tenantSelectDropdown"
    Then I should see the following text on the page
      | TagName | TextContent              |
      | h1      | Tenant Home: Food Tenant |
    And I should be on the page with the relative URL "/?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=9835e88f-bd66-4cd2-b1c4-dfc61de9a2e1"
    Given a mocked request for users for a tenant without orgs
    And the following mocked requests:
      | Method | Path                                                                                                                 | Status | Body                         |
      | GET    | /api/tenants/9835e88f-bd66-4cd2-b1c4-dfc61de9a2e1/userstore/columns?version=3&limit=1500                             | 200    | userstoreschema_default.json |
      | GET    | /api/tenants/9835e88f-bd66-4cd2-b1c4-dfc61de9a2e1/users?company_id=*&tenant_id=*&limit=50&organization_id=&version=3 | 200    | users_for_org_page_1.json    |
    When I click "User Data Storage" header in the sidebar
    And I click "Users" in the sidebar
    And the page title should be "[dev] UserClouds Console"
    Then I should see a table with ID "usersTable" and the following data
      |  |  | Email                                | ID      |
      |  |  | bender@userclouds.com (not verified) | 03f84a… |
      |  |  | leela@userclouds.com (verified)      | 042878… |
      |  |  | zoidberg@firstorg.com (not verified) | 0dfd39… |
      |  |  | amy@secondorg.com (verified)         | 130400… |
      |  |  | farnsworth@firstorg.com (verified)   | 1700fc… |
      |  |  | zapp@userclouds.com (verified)       | 220462… |
      |  |  | kif@userclouds.com (not verified)    | 285567… |
      |  |  | hermes@userclouds.com (verified)     | 2b5653… |
      |  |  | labarbara@userclouds.com (verified)  | 32f175… |
      |  |  | fry@userclouds.com (not verified)    | 39daeb… |
    And I should not see a dropdown matching selector "[name='organization_id']"

  Scenario: end users list no users
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                                 | Status | Body                                                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns?version=3&limit=1500                             | 200    | userstoreschema_default.json                           |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations?limit=1500&version=3                                 | 200    | end_users_page_orgs.json                               |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/users?company_id=*&tenant_id=*&limit=50&organization_id=&version=3 | 200    | { "data": [], "has_next": "false", "has_prev": false } |
    When I navigate to the page with path "/users?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent                                    |
      | h2      | Nothing to display                             |
      | p       | This organization does not have any users yet. |
    And I should see a dropdown matching selector "[name='organization_id']" with the following options
      | Text              | Value                                | Selected |
      | All organizations |                                      | true     |
      | UserClouds Dev    | 1ee4497e-c326-4068-94ed-3dcdaaaa53bc |          |
      | Kyle's First Org  | 10ccc4fd-0d6b-4ded-889b-41eb119c650a |          |
      | Kyle's Second Org | 1e01c626-4f69-44ae-822d-a76741e54bab |          |

  # NOTE: this scenario officially can't happen. Tenants with use_organizations enabled
  # are provisioned an org that cannot be deleted.
  Scenario: end users list no orgs
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                                 | Status | Body                                                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns?version=3&limit=1500                             | 200    | userstoreschema_default.json                           |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations?limit=1500&version=3                                 | 200    | { "data": [], "has_next": "false", "has_prev": false } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/users?company_id=*&tenant_id=*&limit=50&organization_id=&version=3 | 200    | users_for_org_page_1.json                              |
    When I navigate to the page with path "/users?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    Then I should see a table with ID "usersTable" and the following data
      |  |  | Email                                | ID      |
      |  |  | bender@userclouds.com (not verified) | 03f84a… |
      |  |  | leela@userclouds.com (verified)      | 042878… |
      |  |  | zoidberg@firstorg.com (not verified) | 0dfd39… |
      |  |  | amy@secondorg.com (verified)         | 130400… |
      |  |  | farnsworth@firstorg.com (verified)   | 1700fc… |
      |  |  | zapp@userclouds.com (verified)       | 220462… |
      |  |  | kif@userclouds.com (not verified)    | 285567… |
      |  |  | hermes@userclouds.com (verified)     | 2b5653… |
      |  |  | labarbara@userclouds.com (verified)  | 32f175… |
      |  |  | fry@userclouds.com (not verified)    | 39daeb… |
    And I should not see a dropdown matching selector "[name='organization_id']"

  Scenario: end users list no users
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                                                                 | Status | Body                                                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns?version=3&limit=1500                             | 200    | userstoreschema_default.json                           |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations?limit=1500&version=3                                 | 200    | end_users_page_orgs.json                               |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/users?company_id=*&tenant_id=*&limit=50&organization_id=&version=3 | 200    | { "data": [], "has_next": "false", "has_prev": false } |
    When I navigate to the page with path "/users?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent                                    |
      | h2      | Nothing to display                             |
      | p       | This organization does not have any users yet. |
    And I should see a dropdown matching selector "[name='organization_id']" with the following options
      | Text              | Value                                | Selected |
      | All organizations |                                      | true     |
      | UserClouds Dev    | 1ee4497e-c326-4068-94ed-3dcdaaaa53bc |          |
      | Kyle's First Org  | 10ccc4fd-0d6b-4ded-889b-41eb119c650a |          |
      | Kyle's Second Org | 1e01c626-4f69-44ae-822d-a76741e54bab |          |
