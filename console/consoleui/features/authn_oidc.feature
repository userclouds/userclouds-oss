@oidc
Feature: oidc

  @a11y
  Scenario: create oidc accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "serviceinfo"
    And a mocked "GET" request for "userinfo"
    And a mocked "GET" request for "plex_config"
    When I navigate to the page with path "/oauthconnections/oidc_provider/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: create oidc
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "serviceinfo"
    And a mocked "GET" request for "userinfo"
    And a mocked "GET" request for "plex_config"
    When I navigate to the page with path "/oauthconnections/oidc_provider/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following form elements
      | TagName | Type     | Name                    | Value   | Disabled |
      | input   | text     | provider_type           | custom  | true     |
      | input   | text     | provider_name           |         | false    |
      | input   | text     | provider_url            |         | false    |
      | input   | text     | provider_description    |         | false    |
      | input   | text     | client_id               |         | false    |
      | input   | password | client_secret           |         | false    |
      | input   | text     | defaultScopeNameopenid  | openid  | true     |
      | input   | text     | defaultScopeNameprofile | profile | true     |
      | input   | text     | defaultScopeNameemail   | email   | true     |
    And I should see a button labeled "Add Scope"
    And I should see a button labeled "Create Connection"
    When I click the button labeled "Add Scope"
    Then I should see the following form elements
      | TagName | Type     | Name                    | Value     | Disabled |
      | input   | text     | provider_type           | custom    | true     |
      | input   | text     | provider_name           |           | false    |
      | input   | text     | provider_url            |           | false    |
      | input   | text     | provider_description    |           | false    |
      | input   | text     | client_id               |           | false    |
      | input   | password | client_secret           |           | false    |
      | input   | text     | defaultScopeNameopenid  | openid    | true     |
      | input   | text     | defaultScopeNameprofile | profile   | true     |
      | input   | text     | defaultScopeNameemail   | email     | true     |
      | input   | text     | additionalScopeName0    | new_scope | false    |

  Scenario: create oidc bad save
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "serviceinfo"
    And a mocked "GET" request for "userinfo"
    And a mocked "GET" request for "plex_config"
    When I navigate to the page with path "/oauthconnections/oidc_provider/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                                  |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/oidcproviders/create | 500    | {"error":"uh-oh","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
    When I click the button labeled "Create Connection"
    Then I should see the following text on the page
      | TagName | TextContent |
      | p       | uh-oh       |

  ##TODO Good save test when querying for individual OIDC providers becomes possible (non monolithic plex_config)
  Scenario: edit oidc basic functionality
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "serviceinfo"
    And a mocked "GET" request for "userinfo"
    And a mocked "GET" request for "plex_config_custom_oidc"
    When I navigate to the page with path "/oauthconnections/oidc_provider/greatUniqueName?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following form elements
      | TagName | Type     | Name                    | Value                 | Disabled |
      | input   | text     | provider_type           | custom                | true     |
      | input   | text     | provider_name           | greatUniqueName       | true     |
      | input   | text     | provider_url            | myurl.com             | true     |
      | input   | text     | provider_description    | wonderful description | true     |
      | input   | text     | client_id               | aclientid             | true     |
      | input   | password | client_secret           | secret                | true     |
      | input   | text     | defaultScopeNameopenid  | openid                | true     |
      | input   | text     | defaultScopeNameprofile | profile               | true     |
      | input   | text     | defaultScopeNameemail   | email                 | true     |
    And I should see a button labeled "Add Scope"
    And the button labeled "Add Scope" should be disabled
    And I should see a button labeled "Edit Connection"
    And the button labeled "Edit Connection" should be enabled
    And I should not see a button labeled "Delete"
    When I click the button labeled "Edit Connection"
    Then I should see the following form elements
      | TagName | Type     | Name                    | Value                 | Disabled |
      | input   | text     | provider_type           | custom                | true     |
      | input   | text     | provider_name           | greatUniqueName       | true     |
      | input   | text     | provider_url            | myurl.com             | false    |
      | input   | text     | provider_description    | wonderful description | false    |
      | input   | text     | client_id               | aclientid             | false    |
      | input   | password | client_secret           | secret                | false    |
      | input   | text     | defaultScopeNameopenid  | openid                | true     |
      | input   | text     | defaultScopeNameprofile | profile               | true     |
      | input   | text     | defaultScopeNameemail   | email                 | true     |
    And I should see a button labeled "Add Scope"
    And the button labeled "Add Scope" should be enabled
    And I should see a button labeled "Save Connection"
    And the button labeled "Save Connection" should be enabled
    And I should see a button labeled "Cancel"
    And the button labeled "Cancel" should be enabled
    And I should see a button labeled "Delete"
    And the button labeled "Delete" should be enabled

  Scenario: edit oidc add scope and cancel
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "serviceinfo"
    And a mocked "GET" request for "userinfo"
    And a mocked "GET" request for "plex_config_custom_oidc"
    When I navigate to the page with path "/oauthconnections/oidc_provider/greatUniqueName?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following form elements
      | TagName | Type     | Name                    | Value                 | Disabled |
      | input   | text     | provider_type           | custom                | true     |
      | input   | text     | provider_name           | greatUniqueName       | true     |
      | input   | text     | provider_url            | myurl.com             | true     |
      | input   | text     | provider_description    | wonderful description | true     |
      | input   | text     | client_id               | aclientid             | true     |
      | input   | password | client_secret           | secret                | true     |
      | input   | text     | defaultScopeNameopenid  | openid                | true     |
      | input   | text     | defaultScopeNameprofile | profile               | true     |
      | input   | text     | defaultScopeNameemail   | email                 | true     |
    When I click the button labeled "Edit Connection"
    And I click the button labeled "Add Scope"
    Then I should see the following form elements
      | TagName | Type     | Name                    | Value                 | Disabled |
      | input   | text     | provider_type           | custom                | true     |
      | input   | text     | provider_name           | greatUniqueName       | true     |
      | input   | text     | provider_url            | myurl.com             | false    |
      | input   | text     | provider_description    | wonderful description | false    |
      | input   | text     | client_id               | aclientid             | false    |
      | input   | password | client_secret           | secret                | false    |
      | input   | text     | defaultScopeNameopenid  | openid                | true     |
      | input   | text     | defaultScopeNameprofile | profile               | true     |
      | input   | text     | defaultScopeNameemail   | email                 | true     |
    And I should see an element matching selector "input#additionalScopeName0"
    When I type "Foo" into the input with ID "provider_description"
    And I type "Foo" into the input with ID "provider_url"
    And I type "Foo" into the input with ID "client_id"
    And I type "Foo" into the input with ID "client_secret"
    Then I should see the following form elements
      | TagName | Type     | Name                    | Value                    | Disabled |
      | input   | text     | provider_type           | custom                   | true     |
      | input   | text     | provider_name           | greatUniqueName          | true     |
      | input   | text     | provider_url            | Foomyurl.com             | false    |
      | input   | text     | provider_description    | Foowonderful description | false    |
      | input   | text     | client_id               | Fooaclientid             | false    |
      | input   | password | client_secret           | Foosecret                | false    |
      | input   | text     | defaultScopeNameopenid  | openid                   | true     |
      | input   | text     | defaultScopeNameprofile | profile                  | true     |
      | input   | text     | defaultScopeNameemail   | email                    | true     |
    Given a mocked "GET" request for "plex_config_custom_oidc"
    When I click the button labeled "Cancel"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following form elements
      | TagName | Type     | Name                    | Value                 | Disabled |
      | input   | text     | provider_type           | custom                | true     |
      | input   | text     | provider_name           | greatUniqueName       | true     |
      | input   | text     | provider_url            | myurl.com             | true     |
      | input   | text     | provider_description    | wonderful description | true     |
      | input   | text     | client_id               | aclientid             | true     |
      | input   | password | client_secret           | secret                | true     |
      | input   | text     | defaultScopeNameopenid  | openid                | true     |
      | input   | text     | defaultScopeNameprofile | profile               | true     |
      | input   | text     | defaultScopeNameemail   | email                 | true     |
    And I should see a button labeled "Add Scope"
    And the button labeled "Add Scope" should be disabled
    And I should see a button labeled "Edit Connection"
    And the button labeled "Edit Connection" should be enabled
    And I should not see a button labeled "Delete"
    Then I should not see an element matching selector "input#additionalScopeName0"
