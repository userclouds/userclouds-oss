@header
Feature: header

  Scenario: change selected company and tenant
    Given I am a logged-in UC admin
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
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
      | Create Tenant  |                                      |          |
      | Change Company |                                      |          |
    When I select the option labeled "Change Company" in the custom dropdown matching selector "#pageHeader #tenantSelectDropdown"
    Then I should see a dialog with the title "Switch Company"
    And I should see the following text within the dialog titled "Switch Company"
      | Selector         | Text                         |
      | ol > li > button | UserClouds Dev               |
      | ol > li > button | Hobart and William and Smith |
      | ol > li > button | Test Test Test               |
      | a[href]          | Create Company               |
    Given the following mocked requests:
      | Method | Path                                                        | Status | Body |
      | GET    | /api/companies/5ded72de-c606-48ce-9675-df88557d56fe/tenants | 200    | []   |
    When I click the button labeled "Test Test Test"
    Then I should see the following text on the page
      | TagName | TextContent             |
      | h1      | Tenant Home: No tenants |
    And I should be on the page with the relative URL "/?company_id=5ded72de-c606-48ce-9675-df88557d56fe"
    And I should not see a dialog with the title "Switch Company"
    When I select the option labeled "Change Company" in the custom dropdown matching selector "#pageHeader #tenantSelectDropdown"
    Then I should see a dialog with the title "Switch Company"
    And I should see the following text within the dialog titled "Switch Company"
      | Selector         | Text                         |
      | ol > li > button | UserClouds Dev               |
      | ol > li > button | Hobart and William and Smith |
      | ol > li > button | Test Test Test               |
      | a[href]          | Create Company               |
    Given a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    When I click the button labeled "UserClouds dev"
    Then I should see the following text on the page
      | TagName | TextContent                |
      | h1      | Tenant Home: Console - Dev |
      | a[href] | Manage Team                |
    # this check is a way of making sure we've responded to the nav menu feature flag
    And I should be on the page with the relative URL "/?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a custom dropdown matching selector "#pageHeader #tenantSelectDropdown" with the following options
      | Text           | Value                                | Selected |
      | Console - Dev  | 41ab79a8-0dff-418e-9d42-e1694469120a | true     |
      | Food Tenant    | 9835e88f-bd66-4cd2-b1c4-dfc61de9a2e1 |          |
      | Create Tenant  |                                      |          |
      | Change Company |                                      |          |
    Given a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "selectedTenant2"
    When I select the option labeled "Food Tenant" in the custom dropdown matching selector "#pageHeader #tenantSelectDropdown"
    Then I should see the following text on the page
      | TagName | TextContent              |
      | h1      | Tenant Home: Food Tenant |
      | a[href] | Manage Team              |
    # this check is a way of making sure we've responded to the nav menu feature flag
    And I should be on the page with the relative URL "/?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=9835e88f-bd66-4cd2-b1c4-dfc61de9a2e1"
    And I should see a custom dropdown matching selector "#pageHeader #tenantSelectDropdown" with the following options
      | Text           | Value                                | Selected |
      | Console - Dev  | 41ab79a8-0dff-418e-9d42-e1694469120a |          |
      | Food Tenant    | 9835e88f-bd66-4cd2-b1c4-dfc61de9a2e1 | true     |
      | Create Tenant  |                                      |          |
      | Change Company |                                      |          |

  Scenario: create company
    Given I am a logged-in UC admin
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
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
      | Create Tenant  |                                      |          |
      | Change Company |                                      |          |
    When I select the option labeled "Change Company" in the custom dropdown matching selector "#pageHeader #tenantSelectDropdown"
    Then I should see a dialog with the title "Switch Company"
    And I should see the following text within the dialog titled "Switch Company"
      | Selector         | Text                         |
      | ol > li > button | UserClouds Dev               |
      | ol > li > button | Hobart and William and Smith |
      | ol > li > button | Test Test Test               |
      | a[href]          | Create Company               |
    When I click the link with the text "Create Company"
    Then I should see a dialog with the title "Create Company"
    And I should not see a dialog with the title "Switch Company"
    Given the following mocked requests:
      | Method | Path                                                        | Status | Body                              |
      | POST   | /api/companies                                              | 200    | create_company_from_header.json   |
      | GET    | /api/serviceinfo                                            | 200    | service_info_for_new_company.json |
      | GET    | /api/companies/1e01c626-4f69-44ae-822d-a76741e54bab/tenants | 200    | []                                |
    When I type "my new company" into the "company_name" field
    And I click the button labeled "Save"
    Then I should see the following text on the page
      | TagName | TextContent                  |
      | h1      | Add a new tenant: New tenant |
    And I should be on the page with the relative URL "/tenants/create?company_id=1e01c626-4f69-44ae-822d-a76741e54bab"

  Scenario: create company failure
    Given I am a logged-in UC admin
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
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
      | Create Tenant  |                                      |          |
      | Change Company |                                      |          |
    When I select the option labeled "Change Company" in the custom dropdown matching selector "#pageHeader #tenantSelectDropdown"
    Then I should see a dialog with the title "Switch Company"
    And I should see the following text within the dialog titled "Switch Company"
      | Selector         | Text                         |
      | ol > li > button | UserClouds Dev               |
      | ol > li > button | Hobart and William and Smith |
      | ol > li > button | Test Test Test               |
      | a[href]          | Create Company               |
    When I click the link with the text "Create Company"
    Then I should see a dialog with the title "Create Company"
    And I should not see a dialog with the title "Switch Company"
    Given the following mocked requests:
      | Method | Path           | Status | Body                                                                  |
      | POST   | /api/companies | 400    | {"error":"uh-oh","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
    When I type "my new company" into the "company_name" field
    And I click the button labeled "Save"
    Then I should see the following text within the dialog titled "Create Company"
      | Selector       | Text                                               |
      | .alert-message | Error creating company named my new company: uh-oh |
    And I should see the following text on the page
      | TagName | TextContent                |
      | h1      | Tenant Home: Console - Dev |

  Scenario: choose "Create new" from the tenants dropdown
    Given I am a logged-in UC admin
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
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
      | Create Tenant  |                                      |          |
      | Change Company |                                      |          |
    Given a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    When I select the option labeled "Create Tenant" in the custom dropdown matching selector "#pageHeader #tenantSelectDropdown"
    Then I should be on the page with the path "/tenants/create"
    # Tenant is ready
    Given a mocked "GET" request for "a single tenant"
    # Tenant is being provisioned
    And a mocked "GET" request for "a single tenant being created"
    # Tenant is created but authz edges haven't been written
    And a mocked "GET" request for "a single tenant being created" that returns a "403"
    # Tenant hasn't yet even been saved
    # Playwright suggests mocking in reverse order, with a fallback
    # POST happens first, then GET
    And a mocked "GET" request for "a single tenant being created" that returns a "404"
    And a mocked "POST" request for "tenants"
    And a mocked "GET" request for "a single tenant"
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "tenants_urls"
    When I type "Foo" into the "name" field
    And I click the button labeled "Create Tenant"
    Then I should see the following text on the page
      | TagName      | TextContent                           |
      | h1           | Tenant Home: Foo                      |
      | section > h1 | Welcome to UserClouds, Kyle Jacobson! |
    And I should be on the page with the path "/"

  Scenario: non-admin user
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
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

  Scenario: click logo to return home
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema"
    ##the first one gets display columns. the second one gets all columns - needed for creating accessors and mutators.
    And a mocked "GET" request for "user store schema"
    And a mocked "GET" request for "accessors"
    And a mocked "GET" request for "mutators"
    And a mocked "GET" request for "purposes"
    When I navigate to the page with path "/purposes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see 1 buttons labeled "Create Purpose" and they should be enabled
    Given a mocked "GET" request for "tenants"
    And the following mocked requests:
      | Method | Path                                                                  | Status | Body                       |
      | GET    | /api/tenants/*/userstore/columns/032cae17-df3a-4e87-82a0-c706ed0679ee | 200    | email_verified_column.json |
      | GET    | /api/tenants/*/userstore/purposes?limit=50&version=3                  | 200    | purposes.json              |
    And a mocked request for column retention durations with only two supported units
    When I click the link with the href "/purposes/0b112683-aa23-4269-b098-ae6fdc1a9d8d?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic Details |
    Given a mocked "GET" request for "tenants"
    And a mocked "GET" request for "tenants_urls"
    When I click the element matching selector "header a[href='/'][title='Return to home']"
    Then I should see the following text on the page
      | TagName | TextContent                |
      | h1      | Tenant Home: Console - Dev |
    And I should be on the page with the path "/"
