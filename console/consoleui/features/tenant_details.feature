@tenant_details
@tenants
Feature: Tenant Details Page

  Scenario: Basic info tenant
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "database"
    And a mocked "GET" request for "plex_config"
    When I navigate to the page with path "/tenants/41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent                   |
      | h1      | Tenant Details: Console - Dev |
    And I should see the following text on the page
      | TagName            | TextContent                                          |
      | label              | Name                                                 |
      | label > div > p    | Console - Dev                                        |
      | label              | URL (SDK or HTTP connections)                        |
      | label > div > span | https://console-dev.tenant.dev.userclouds.tools:3333 |
    And I should not see an element matching selector "#pageContent button:has-text('Delete tenant')"
    And I should see a "h2" with the text "Database Connections"
    And I should see a "h2" with the text "Custom Domain"
    And I should see a "h2" with the text "Trusted Issuer"
    And I should see a table with ID "databases" and the following data
      | Database Name | Proxy Host Address | Proxy Port |
      | database      | hostname           | 123        |
    And I should see a table with ID "tenant_url_table" and the following data
      | Tenant URL                 | Verified                 |
      | https://customurl.com      | not verified Information |
      | https://othercustomurl.com | verified                 |
    And I should see a table with ID "trusted_issuers" and the following data
      | URL                    |
      | https://amazon.com     |
      | https://www.google.com |
      | http://facebook.com    |

  Scenario: Non-admin tenant details
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenantNonAdmin"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "database"
    And a mocked "GET" request for "plex_config"
    When I navigate to the page with path "/tenants/41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should not see an element matching selector "#pageContent button:has-text('Delete tenant')"
    And I should not see a "h2" with the text "Database Connections"
    And I should not see a "h2" with the text "Custom Domain"
    And I should not see a "h2" with the text "Trusted JWT Issuer"

  @edit_tenant
  Scenario: edit tenant URL
    Given I am a logged-in user
    And the following mocked requests:
      | Method | Path                                                                  | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/database* | 200    | empty_paginated_response.json |
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "plex_config"
    When I navigate to the page with path "/tenants/41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent                   |
      | h1      | Tenant Details: Console - Dev |
    And I should see a "h2" with the text "Database Connections"
    And I should see a "h2" with the text "Custom Domain"
    And I should see a "h2" with the text "Trusted Issuer"
    And I should not see a button labeled "Remove tenant URL"
    And I should not see a button labeled "Add URL"
    When I click the button labeled "Edit Settings"
    When I click the button labeled "Add Custom URL"
    Then I should see a dialog with the title "Tenant URL"
    Then the input with the name "tenant_url" should have the value ""
    And the input with the name "tenant_url" should be invalid
    # field is empty
    And the input with the name "tenant_url" should be invalid
    When I type "foo" into the "tenant_url" field
    # field has a non-URL value
    Then the input with the name "tenant_url" should have the value "foo"
    And the input with the name "tenant_url" should be invalid
    And the button labeled "Save" should be enabled
    # cancel editing
    When I click the button with ID "cancelURL"
    Then I should see a button labeled "Add Custom URL"
    # re-enter edit mode
    When I click the button labeled "Add Custom URL"
    Then the input with the name "tenant_url" should have the value ""
    When I replace the text in the "tenant_url" field with "http://foo.domain.com"
    # URL is not HTTPS
    Then the input with the name "tenant_url" should have the value "http://foo.domain.com"
    And the input with the name "tenant_url" should be invalid
    And the button labeled "Save" should be enabled
    When I replace the text in the "tenant_url" field with "https://foo.domain.com"
    # Valid HTTPS url
    Then the input with the name "tenant_url" should have the value "https://foo.domain.com"
    And the input with the name "tenant_url" should be valid
    And the button labeled "Save" should be enabled
    # click save
    Given I am a logged-in user
    And a mocked "GET" request for "plex_config"
    Given the following mocked requests:
      | Method | Path                                                                  | Status | Body                                                                                                                                                                                                                                                                                                                                                                                                                                 |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/urls                | 200    | [{"id":"286c7032-7555-40e3-abfd-413d431d75f0","created":"2024-03-29T18:28:16.653894Z","updated":"2024-03-29T18:28:22.776335Z","deleted":"0001-01-01T00:00:00Z","tenant_id":"41ab79a8-0dff-418e-9d42-e1694469120a","tenant_url":"https://foo.domain.com","validated":false,"system":false,"active":false,"dns_verifier":"Prr7TsfFBR6n4aVGDL4VUqZfZ3K0So2w6fAnH157-CE","certificate_valid_until":"0001-01-01T00:00:00Z"}]              |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/urls                | 201    | {"tenant_url":{"id":"286c7032-7555-40e3-abfd-413d431d75f0","created":"2024-03-29T18:28:16.653894Z","updated":"2024-03-29T18:28:16.653894Z","deleted":"0001-01-01T00:00:00Z","tenant_id":"41ab79a8-0dff-418e-9d42-e1694469120a","tenant_url":"https://foo.domain.com","validated":false,"system":false,"active":false,"dns_verifier":"Prr7TsfFBR6n4aVGDL4VUqZfZ3K0So2w6fAnH157-CE","certificate_valid_until":"0001-01-01T00:00:00Z"}} |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/database* | 200    | empty_paginated_response.json                                                                                                                                                                                                                                                                                                                                                                                                        |
    When I click the button with ID "saveURL"
    When I click the button with ID "savePage"
    Then I should see a "td" with the text "not verified Information"
    When I click the button labeled "Edit Settings"
    When I click the "edit" button in row 1 of the table with ID "tenant_url_table"
    Then I should see a button labeled "Refresh"
    And the input with the name "tenant_url" should have the value "https://foo.domain.com"
    And I should see the following text on the page
      | TagName | TextContent                  |
      | div     | DNS ownership not validated. |
    # Get the new validation status back from the server
    Then I should see the following text on the page
      | TagName | TextContent                                                                                                                                          |
      | div     | DNS ownership not validated. Please create a DNS TXT record at _acme-challenge.foo.domain.com with value Prr7TsfFBR6n4aVGDL4VUqZfZ3K0So2w6fAnH157-CE |
    # Refresh, get valid
    Given the following mocked requests:
      | Method | Path                                                   | Status | Body                                                                                                                                                                                                                                                                                                                                                                                                                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/urls | 200    | [{"id":"286c7032-7555-40e3-abfd-413d431d75f0","created":"2024-03-29T18:28:16.653894Z","updated":"2024-03-29T18:28:22.776335Z","deleted":"0001-01-01T00:00:00Z","tenant_id":"41ab79a8-0dff-418e-9d42-e1694469120a","tenant_url":"https://foo.domain.com","validated":true,"system":false,"active":false,"dns_verifier":"Prr7TsfFBR6n4aVGDL4VUqZfZ3K0So2w6fAnH157-CE","certificate_valid_until":"0001-01-01T00:00:00Z"}] |
    When I click the button labeled "Refresh"
    Then I should not see a dialog with the title "Tenant URL"
    And I should see a "td" with the text "verified"
    When I click the "edit" button in row 1 of the table with ID "tenant_url_table"
    Then I should see the following text on the page
      | TagName | TextContent                                            |
      | div     | DNS ownership verified. Certificate is not yet issued. |
    And I should not see a button labeled "Refresh"
    # Edit URL
    And the input with the name "tenant_url" should have the value "https://foo.domain.com"
    And the input with the name "tenant_url" should be valid
    And the button with ID "saveURL" should be enabled
    When I replace the text in the "tenant_url" field with "https://bar.domain.com"
    Then the input with the name "tenant_url" should have the value "https://bar.domain.com"
    And the input with the name "tenant_url" should be valid
    And the button with ID "saveURL" should be enabled
    # click save
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "plex_config"
    And the following mocked requests:
      | Method | Path                                                                                        | Status | Body                                                                                                                                                                                                                                                                                                                                                                                                                                 |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/urls/286c7032-7555-40e3-abfd-413d431d75f0 | 200    | {"tenant_url":{"id":"286c7032-7555-40e3-abfd-413d431d75f0","created":"2024-03-29T18:28:16.653894Z","updated":"2024-03-29T18:28:16.653894Z","deleted":"0001-01-01T00:00:00Z","tenant_id":"41ab79a8-0dff-418e-9d42-e1694469120a","tenant_url":"https://bar.domain.com","validated":false,"system":false,"active":false,"dns_verifier":"Prr7TsfFBR6n4aVGDL4VUqZfZ3K0So2w6fAnH157-CE","certificate_valid_until":"0001-01-01T00:00:00Z"}} |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/database*                       | 200    | empty_paginated_response.json                                                                                                                                                                                                                                                                                                                                                                                                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/urls                                      | 200    | tenants_urls_updated.json                                                                                                                                                                                                                                                                                                                                                                                                            |
    When I click the button with ID "saveURL"
    When I click the button with ID "savePage"
    Then I should see a button labeled "Edit Settings"
    Given I am a logged-in user
    And the following mocked requests:
      | Method | Path                                                   | Status | Body                                                                                                                                                                                                                                                                                                                                                                                                                   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/urls | 200    | [{"id":"286c7032-7555-40e3-abfd-413d431d75f0","created":"2024-03-29T18:28:16.653894Z","updated":"2024-03-29T18:28:22.776335Z","deleted":"0001-01-01T00:00:00Z","tenant_id":"41ab79a8-0dff-418e-9d42-e1694469120a","tenant_url":"https://bar.domain.com","validated":true,"system":false,"active":false,"dns_verifier":"Prr7TsfFBR6n4aVGDL4VUqZfZ3K0So2w6fAnH157-CE","certificate_valid_until":"0001-01-01T00:00:00Z"}] |
    When I click the button labeled "Edit Settings"
    And I should see a "td" with the text "https://bar.domain.com"
    When I click the "edit" button in row 1 of the table with ID "tenant_url_table"
    Then I should not see a button labeled "Refresh"
    And the input with the name "tenant_url" should have the value "https://bar.domain.com"
    And I should see the following text on the page
      | TagName | TextContent                                                                                                                            |
      | div     | DNS ownership verified. Certificate is not yet issued. Domain is not currently CNAMEd to console-dev.tenant.dev.userclouds.tools:3333. |
    When I click the button with ID "cancelURL"
    # delete URL
    Given I intend to accept the confirm dialog
    And the following mocked requests:
      | Method | Path                                                                                        | Status | Body |
      | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/urls/286c7032-7555-40e3-abfd-413d431d75f0 | 204    | null |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/urls                                      | 200    | []   |
    When I click the "delete" button in row 1 of the table with ID "tenant_url_table"
    Then I should see a table with ID "tenant_url_table" and the following data
      | Tenant URL                 | Verified |
      | https://othercustomurl.com | verified |
    And I should see a button labeled "Add Custom URL"
