@policies
@transformers
@transformer_details
Feature: transformer details page

  @a11y
  Scenario: basic info transformer accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "dataTypes"
    And the following mocked requests:
      | Method | Path                                                                                                                     | Status | Body                                                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions/d2685634-ac22-4ce8-bdae-64892f17f4e9              | 200    | {"create":true,"read":true,"update":true,"delete":true} |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/d2685634-ac22-4ce8-bdae-64892f17f4e9?version=0 | 200    | single_transformer_get.json                             |
    When I navigate to the page with path "/transformers/d2685634-ac22-4ce8-bdae-64892f17f4e9/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: basic info transformer
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "dataTypes"
    And the following mocked requests:
      | Method | Path                                                                                                                     | Status | Body                                                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions/d2685634-ac22-4ce8-bdae-64892f17f4e9              | 200    | {"create":true,"read":true,"update":true,"delete":true} |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/d2685634-ac22-4ce8-bdae-64892f17f4e9?version=0 | 200    | single_transformer_get.json                             |
    When I navigate to the page with path "/transformers/d2685634-ac22-4ce8-bdae-64892f17f4e9/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching transformer..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a "h1" with the text "Transformer Details: Foo"
    And I should see the following text on the page
      | TagName         | TextContent                                 |
      | label           | Name                                        |
      | label > div > p | Foo                                         |
      | label           | Description                                 |
      | label > div > p | Do Kung Fu                                  |
      | label           | Transform Type                              |
      | label > div > p | Transform                                   |
      | label           | Input Data Type                             |
      | label > div > p | phonenumber                                 |
      | label           | Output Data Type                            |
      | label > div > p | boolean                                     |
      | label           | Reuse existing token?                       |
      | svg>title       | Off                                         |
      | label           | Function                                    |
      | label           | Parameters                                  |
      | div             | Parameters are identical on every execution |
    And I should see a code editor with the ID "transformer_function" and the value "function transform(data, params) {  return true;}"
    And I should see a code editor with the ID "transformer_params" and the value "{}"
    And I should not see a button labeled "Run Test"
    And I should not see a button labeled "Save Transformer"
    And I should not see a button labeled "Cancel"
    And I should see a button labeled "Edit transformer"
    And the button labeled "Edit transformer" should be enabled

  Scenario: only read permissons transformer
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "dataTypes"
    And the following mocked requests:
      | Method | Path                                                                                                                     | Status | Body                                                       |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions/d2685634-ac22-4ce8-bdae-64892f17f4e9              | 200    | {"create":false,"read":true,"update":false,"delete":false} |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/d2685634-ac22-4ce8-bdae-64892f17f4e9?version=0 | 200    | single_transformer_get.json                                |
    When I navigate to the page with path "/transformers/d2685634-ac22-4ce8-bdae-64892f17f4e9/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching transformer..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a "h1" with the text "Transformer Details: Foo"
    And I should see the following text on the page
      | TagName         | TextContent                                 |
      | label           | Name                                        |
      | label > div > p | Foo                                         |
      | label           | Description                                 |
      | label > div > p | Do Kung Fu                                  |
      | label           | Transform Type                              |
      | label > div > p | Transform                                   |
      | label           | Input Data Type                             |
      | label > div > p | phonenumber                                 |
      | label           | Output Data Type                            |
      | label > div > p | boolean                                     |
      | label           | Reuse existing token?                       |
      | svg>title       | Off                                         |
      | label           | Function                                    |
      | label           | Parameters                                  |
      | div             | Parameters are identical on every execution |
    And I should see a code editor with the ID "transformer_function" and the value "function transform(data, params) {  return true;}"
    And I should see a code editor with the ID "transformer_params" and the value "{}"
    And I should not see a button labeled "Run Test"
    And I should not see a button labeled "Save Transformer"
    And I should not see a button labeled "Cancel"
    And I should not see a button labeled "Edit transformer"

  Scenario: no delete permissions transformer
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "dataTypes"
    And the following mocked requests:
      | Method | Path                                                                                                                     | Status | Body                                                     |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions/d2685634-ac22-4ce8-bdae-64892f17f4e9              | 200    | {"create":true,"read":true,"update":true,"delete":false} |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/d2685634-ac22-4ce8-bdae-64892f17f4e9?version=0 | 200    | single_transformer_get.json                              |
    When I navigate to the page with path "/transformers/d2685634-ac22-4ce8-bdae-64892f17f4e9/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching transformer..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a "h1" with the text "Transformer Details: Foo"
    And I should see the following text on the page
      | TagName         | TextContent                                 |
      | label           | Name                                        |
      | label > div > p | Foo                                         |
      | label           | Description                                 |
      | label > div > p | Do Kung Fu                                  |
      | label           | Transform Type                              |
      | label > div > p | Transform                                   |
      | label           | Input Data Type                             |
      | label > div > p | phonenumber                                 |
      | label           | Output Data Type                            |
      | label > div > p | boolean                                     |
      | label           | Reuse existing token?                       |
      | svg>title       | Off                                         |
      | label           | Function                                    |
      | label           | Parameters                                  |
      | div             | Parameters are identical on every execution |
    And I should see a code editor with the ID "transformer_function" and the value "function transform(data, params) {  return true;}"
    And I should see a code editor with the ID "transformer_params" and the value "{}"
    And I should not see a button labeled "Run Test"
    And I should not see a button labeled "Save Transformer"
    And I should not see a button labeled "Cancel"
    And I should see a button labeled "Edit transformer"
    And the button labeled "Edit transformer" should be enabled

  @edit_transformer
  @a11y
  Scenario: edit transformer save error accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "dataTypes"
    And the following mocked requests:
      | Method | Path                                                                                                                     | Status | Body                                                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions/d2685634-ac22-4ce8-bdae-64892f17f4e9              | 200    | {"create":true,"read":true,"update":true,"delete":true} |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/d2685634-ac22-4ce8-bdae-64892f17f4e9?version=0 | 200    | single_transformer_get.json                             |
    When I navigate to the page with path "/transformers/d2685634-ac22-4ce8-bdae-64892f17f4e9/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching transformer..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a "h1" with the text "Transformer Details: Foo"
    And I should see a button labeled "Edit transformer"
    And the button labeled "Edit transformer" should be enabled
    When I click the button labeled "Edit transformer"
    Then the page should have no accessibility violations
    And I should see the following form elements
      | TagName  | Type | Name        | Value                 |
      | input    | text | name        | Foo                   |
      | input    | text | description | Do Kung Fu            |
      | textarea |      | test_data   | secret data goes here |
      | input    | text | test_result |                       |
    And I should not see a checkbox labeled "Reuse existing token?"
    And I should see a dropdown matching selector "[name='input_data_type']" with the following options
      | Text                | Value               | Selected |
      | address_eu          | address_eu          |          |
      | birthdate           | birthdate           |          |
      | birthdate_composite | birthdate_composite |          |
      | boolean             | boolean             |          |
      | canonical_address   | canonical_address   |          |
      | date                | date                |          |
      | e164_phonenumber    | e164_phonenumber    |          |
      | email               | email               |          |
      | integer             | integer             |          |
      | phonenumber         | phonenumber         | true     |
      | ssn                 | ssn                 |          |
      | string              | string              |          |
      | timestamp           | timestamp           |          |
      | uuid                | uuid                |          |
    And I should see a dropdown matching selector "[name='output_data_type']" with the following options
      | Text                | Value               | Selected |
      | address_eu          | address_eu          |          |
      | birthdate           | birthdate           |          |
      | birthdate_composite | birthdate_composite |          |
      | boolean             | boolean             | true     |
      | canonical_address   | canonical_address   |          |
      | date                | date                |          |
      | e164_phonenumber    | e164_phonenumber    |          |
      | email               | email               |          |
      | integer             | integer             |          |
      | phonenumber         | phonenumber         |          |
      | ssn                 | ssn                 |          |
      | string              | string              |          |
      | timestamp           | timestamp           |          |
      | uuid                | uuid                |          |
    And I should see a dropdown matching selector "[name='transform_type']" with the following options
      | Text                  | Value               | Selected |
      | Passthrough           | passthrough         |          |
      | Transform             | transform           | true     |
      | Tokenize by value     | tokenizebyvalue     |          |
      | Tokenize by reference | tokenizebyreference |          |
    And I should see a code editor with the ID "transformer_function" and the value "function transform(data, params) {  return true;}"
    And I should see a code editor with the ID "transformer_params" and the value "{}"
    And I should see a button labeled "Save Transformer"
    And the button labeled "Save Transformer" should be disabled
    And I should see a button labeled "Cancel"
    And the button labeled "Cancel" should be enabled
    When I replace the text in the "name" field with "ObfuscateEmailNaive"
    And I replace the text in the "description" field with "Takes foo@bar.com and returns foo[at]bar[dot]com"
    And I select the option labeled "Tokenize by value" in the dropdown matching selector "[name='transform_type']"
    And I select the option labeled "email" in the dropdown matching selector "[name='input_data_type']"
    And I select the option labeled "string" in the dropdown matching selector "[name='output_data_type']"
    And I replace the text in the code editor with the ID "transformer_function" with the value "function transform(data, params) { const [name, host] = data.split('@'); return `${name}[at]${host.replaceAll('.', '[dot]')}`;}"
    Then I should see the following form elements
      | TagName  | Type | Name        | Value                                            |
      | input    | text | name        | ObfuscateEmailNaive                              |
      | input    | text | description | Takes foo@bar.com and returns foo[at]bar[dot]com |
      | textarea |      | test_data   | secret data goes here                            |
      | input    | text | test_result |                                                  |
    And I should see a checkbox labeled "Reuse existing token?"
    And the checkbox labeled "Reuse existing token?" should be unchecked
    And I should see a code editor with the ID "transformer_function" and the value "function transform(data, params) { const [name, host] = data.split('@'); return `${name}[at]${host.replaceAll('.', '[dot]')}`;}"
    And I should see a code editor with the ID "transformer_params" and the value "{}"
    And the button labeled "Save Transformer" should be enabled
    And the button labeled "Cancel" should be enabled
    When I replace the text in the "name" field with ""
    Then I should see the following form elements
      | TagName  | Type | Name        | Value                                            |
      | input    | text | name        |                                                  |
      | input    | text | description | Takes foo@bar.com and returns foo[at]bar[dot]com |
      | textarea |      | test_data   | secret data goes here                            |
      | input    | text | test_result |                                                  |
    When I click the button labeled "Save Transformer"
    Then the input with the name "name" should be invalid
    When I replace the text in the "name" field with "(&$%"
    Then I should see the following form elements
      | TagName  | Type | Name        | Value                                            |
      | input    | text | name        | (&$%                                             |
      | input    | text | description | Takes foo@bar.com and returns foo[at]bar[dot]com |
      | textarea |      | test_data   | secret data goes here                            |
      | input    | text | test_result |                                                  |
    And the input with the name "name" should be invalid
    When I replace the text in the "name" field with "FooBar"
    Then the input with the name "name" should be valid
    Given the following mocked requests:
      | Method | Path                                                                                                           | Status | Body                                                                           |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/d2685634-ac22-4ce8-bdae-64892f17f4e9 | 400    | {"error":"not great, bob","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
    When I click the button labeled "Save Transformer"
    Then I should see the following text on the page
      | TagName | TextContent                                |
      | p       | Error updating transformer: not great, bob |
    And I should be on the page with the path "/transformers/d2685634-ac22-4ce8-bdae-64892f17f4e9/0"

  @edit_transformer
  Scenario: edit transformer save success
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "dataTypes"
    And the following mocked requests:
      | Method | Path                                                                                                                     | Status | Body                                                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions/d2685634-ac22-4ce8-bdae-64892f17f4e9              | 200    | {"create":true,"read":true,"update":true,"delete":true} |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/d2685634-ac22-4ce8-bdae-64892f17f4e9?version=0 | 200    | single_transformer_get.json                             |
    When I navigate to the page with path "/transformers/d2685634-ac22-4ce8-bdae-64892f17f4e9/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching transformer..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a "h1" with the text "Transformer Details: Foo"
    And I should see a button labeled "Edit transformer"
    And the button labeled "Edit transformer" should be enabled
    When I click the button labeled "Edit transformer"
    Then I should see the following form elements
      | TagName  | Type | Name        | Value                 |
      | input    | text | name        | Foo                   |
      | input    | text | description | Do Kung Fu            |
      | textarea |      | test_data   | secret data goes here |
      | input    | text | test_result |                       |
    And I should not see a checkbox labeled "Reuse existing token?"
    And I should see a dropdown matching selector "[name='input_data_type']" with the following options
      | Text                | Value               | Selected |
      | address_eu          | address_eu          |          |
      | birthdate           | birthdate           |          |
      | birthdate_composite | birthdate_composite |          |
      | boolean             | boolean             |          |
      | canonical_address   | canonical_address   |          |
      | date                | date                |          |
      | e164_phonenumber    | e164_phonenumber    |          |
      | email               | email               |          |
      | integer             | integer             |          |
      | phonenumber         | phonenumber         | true     |
      | ssn                 | ssn                 |          |
      | string              | string              |          |
      | timestamp           | timestamp           |          |
      | uuid                | uuid                |          |
    And I should see a dropdown matching selector "[name='output_data_type']" with the following options
      | Text                | Value               | Selected |
      | address_eu          | address_eu          |          |
      | birthdate           | birthdate           |          |
      | birthdate_composite | birthdate_composite |          |
      | boolean             | boolean             | true     |
      | canonical_address   | canonical_address   |          |
      | date                | date                |          |
      | e164_phonenumber    | e164_phonenumber    |          |
      | email               | email               |          |
      | integer             | integer             |          |
      | phonenumber         | phonenumber         |          |
      | ssn                 | ssn                 |          |
      | string              | string              |          |
      | timestamp           | timestamp           |          |
      | uuid                | uuid                |          |
    And I should see a dropdown matching selector "[name='transform_type']" with the following options
      | Text                  | Value               | Selected |
      | Passthrough           | passthrough         |          |
      | Transform             | transform           | true     |
      | Tokenize by value     | tokenizebyvalue     |          |
      | Tokenize by reference | tokenizebyreference |          |
    And I should see a code editor with the ID "transformer_function" and the value "function transform(data, params) {  return true;}"
    And I should see a code editor with the ID "transformer_params" and the value "{}"
    And I should see a button labeled "Save Transformer"
    And the button labeled "Save Transformer" should be disabled
    And I should see a button labeled "Cancel"
    And the button labeled "Cancel" should be enabled
    When I replace the text in the "name" field with "ObfuscateEmailNaive"
    And I replace the text in the "description" field with "Takes foo@bar.com and returns foo[at]bar[dot]com"
    And I select the option labeled "Tokenize by value" in the dropdown matching selector "[name='transform_type']"
    And I select the option labeled "email" in the dropdown matching selector "[name='input_data_type']"
    And I select the option labeled "string" in the dropdown matching selector "[name='output_data_type']"
    And I replace the text in the code editor with the ID "transformer_function" with the value "function transform(data, params) { const [name, host] = data.split('@'); return `${name}[at]${host.replaceAll('.', '[dot]')}`;}"
    Then I should see the following form elements
      | TagName  | Type | Name        | Value                                            |
      | input    | text | name        | ObfuscateEmailNaive                              |
      | input    | text | description | Takes foo@bar.com and returns foo[at]bar[dot]com |
      | textarea |      | test_data   | secret data goes here                            |
      | input    | text | test_result |                                                  |
    And I should see a checkbox labeled "Reuse existing token?"
    And the checkbox labeled "Reuse existing token?" should be unchecked
    When I toggle the checkbox labeled "Reuse existing token?"
    Then the checkbox labeled "Reuse existing token?" should be checked
    When I toggle the checkbox labeled "Reuse existing token?"
    Then the checkbox labeled "Reuse existing token?" should be unchecked
    And I should see a code editor with the ID "transformer_function" and the value "function transform(data, params) { const [name, host] = data.split('@'); return `${name}[at]${host.replaceAll('.', '[dot]')}`;}"
    And I should see a code editor with the ID "transformer_params" and the value "{}"
    And the button labeled "Save Transformer" should be enabled
    And the button labeled "Cancel" should be enabled
    Given a mocked "GET" request for "tenants"
    And the following mocked requests:
      | Method | Path                                                                                                                     | Status | Body                                                    |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/d2685634-ac22-4ce8-bdae-64892f17f4e9           | 200    | updated_transformer_get.json                            |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions/d2685634-ac22-4ce8-bdae-64892f17f4e9              | 200    | {"create":true,"read":true,"update":true,"delete":true} |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/d2685634-ac22-4ce8-bdae-64892f17f4e9?version=1 | 200    | updated_transformer_get.json                            |
    When I click the button labeled "Save Transformer"
    Then I should see a toast notification with the text "Successfully updated transformerClose"
    And I should see a "h1" with the text "Transformer Details: ObfuscateEmailNaive"
    And I should be on the page with the path "/transformers/d2685634-ac22-4ce8-bdae-64892f17f4e9/1"
    And I should see the following text on the page
      | TagName                                     | TextContent                                      |
      | label                                       | Name                                             |
      | label > div > p                             | ObfuscateEmailNaive                              |
      | label                                       | Description                                      |
      | label > div > p                             | Takes foo@bar.com and returns foo[at]bar[dot]com |
      | label                                       | Transform Type                                   |
      | label > div > p                             | Tokenize by value                                |
      | label                                       | Input Data Type                                  |
      | label:has-text('Input Data Type') > div > p | email                                            |
      | label                                       | Output Data Type                                 |
      | label > div > p                             | string                                           |
      | label                                       | Reuse existing token?                            |
      | label > div > div                           | Off                                              |

  @test_transformer
  Scenario: test transformer
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "dataTypes"
    And the following mocked requests:
      | Method | Path                                                                                                                     | Status | Body                                                     |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions/d2685634-ac22-4ce8-bdae-64892f17f4e9              | 200    | {"create":true,"read":true,"update":true,"delete":false} |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/d2685634-ac22-4ce8-bdae-64892f17f4e9?version=0 | 200    | single_transformer_get.json                              |
    When I navigate to the page with path "/transformers/d2685634-ac22-4ce8-bdae-64892f17f4e9/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching transformer..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a "h1" with the text "Transformer Details: Foo"
    And I should see a button labeled "Edit transformer"
    And the button labeled "Edit transformer" should be enabled
    And I should not see a button labeled "Run Test"
    When I click the button labeled "Edit transformer"
    Then I should see the following form elements
      | TagName  | Type | Name        | Value                 |
      | input    | text | name        | Foo                   |
      | input    | text | description | Do Kung Fu            |
      | textarea |      | test_data   | secret data goes here |
      | input    | text | test_result |                       |
    And I should see a code editor with the ID "transformer_function" and the value "function transform(data, params) { return true;}"
    And I should see a code editor with the ID "transformer_params" and the value "{}"
    And the button labeled "Save Transformer" should be disabled
    And the button labeled "Cancel" should be enabled
    And I should see a button labeled "Run Test"
    And the button labeled "Run Test" should be enabled
    Given the following mocked requests:
      | Method | Path                                                                                   | Status | Body                                                                                                                                                          |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/actions/test | 500    | {"error":{"http_status_code":500,"error":{"error":"javascript error: TypeError: Cannot read properties of undefined","request_id":"123"}},"request_id":"456"} |
    When I click the button labeled "Run Test"
    Then I should see the following text on the page
      | TagName          | TextContent                                                      |
      | .alert-message p | javascript error: TypeError: Cannot read properties of undefined |
    Given the following mocked requests:
      | Method | Path                                                                                   | Status | Body                                              |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/actions/test | 200    | {"value":"foo[at]bar[dot]baz[dot]com","debug":{}} |
    When I replace the text in the "test_data" field with "foo@bar.baz.com"
    And I click the button labeled "Run Test"
    Then I should see the following form elements
      | TagName  | Type | Name        | Value                      |
      | input    | text | name        | Foo                        |
      | input    | text | description | Do Kung Fu                 |
      | textarea |      | test_data   | foo@bar.baz.com            |
      | input    | text | test_result | foo[at]bar[dot]baz[dot]com |
    And I should not see an element matching selector ".alert-message p"

  Scenario: cancel transformer editing
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "dataTypes"
    And the following mocked requests:
      | Method | Path                                                                                                                     | Status | Body                                                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions/d2685634-ac22-4ce8-bdae-64892f17f4e9              | 200    | {"create":true,"read":true,"update":true,"delete":true} |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation/d2685634-ac22-4ce8-bdae-64892f17f4e9?version=0 | 200    | single_transformer_get.json                             |
    When I navigate to the page with path "/transformers/d2685634-ac22-4ce8-bdae-64892f17f4e9/0?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a "h1" with the text "Transformer Details: Foo"
    Then I should see a button labeled "Edit transformer"
    And the button labeled "Edit transformer" should be enabled
    When I click the button labeled "Edit transformer"
    Then I should see the following form elements
      | TagName  | Type | Name        | Value                 |
      | input    | text | name        | Foo                   |
      | input    | text | description | Do Kung Fu            |
      | textarea |      | test_data   | secret data goes here |
      | input    | text | test_result |                       |
    And I should see a button labeled "Run Test"
    And I should see the following text on the page
      | Selector               | TextContent      |
      | button:not([disabled]) | Run Test         |
      | button[disabled]       | Save Transformer |
    When I replace the text in the "name" field with "Our_Transformer"
    Then the button labeled "Save Transformer" should be enabled
    And the button labeled "Cancel" should be enabled
    Given I intend to dismiss the confirm dialog
    When I click the button labeled "Cancel"
    Then I should see the following text on the page
      | TagName | TextContent         |
      | h1      | Transformer Details |
    And I should be on the page with the path "/transformers/d2685634-ac22-4ce8-bdae-64892f17f4e9/0"
    Then I should see a "h1" with the text "Transformer Details: Foo"
    And the button labeled "Save Transformer" should be enabled
    And the button labeled "Cancel" should be enabled
    When I replace the text in the "name" field with "Foo"
    Then the button labeled "Save Transformer" should be disabled
    And the button labeled "Cancel" should be enabled
    # no confirm dialog
    When I click the button labeled "Cancel"
    Then I should see a button labeled "Edit transformer"
    And the button labeled "Edit transformer" should be enabled
    And I should not see a button labeled "Save Transformer"
    And I should not see a button labeled "Cancel"
    And I should not see a button labeled "Run Test"
    And I should be on the page with the path "/transformers/d2685634-ac22-4ce8-bdae-64892f17f4e9/0"
