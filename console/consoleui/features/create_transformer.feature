@userstore
@transformers
@create_transformer
@new_transformer
Feature: create transformer page

  @a11y
  Scenario: create transformer accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "dataTypes"
    When I navigate to the page with path "/transformers/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: create transformer with basic info
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "dataTypes"
    When I navigate to the page with path "/transformers/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent                                 |
      | h1      | Create Transformer                          |
      | label   | Name                                        |
      | label   | Description                                 |
      | label   | Transform Type                              |
      | label   | Input Data Type                             |
      | label   | Output Data Type                            |
      | label   | Reuse existing token?                       |
      | label   | Function                                    |
      | label   | Parameters (JSON dictionary or array)       |
      | div     | Parameters are identical on every execution |
      | label   | Test data                                   |
      | label   | Test result                                 |
    And I should see a "h1" with the text "Create Transformer: New Transformer"
    And I should see the following form elements
      | TagName  | Type | Name        | Value                 |
      | input    | text | name        |                       |
      | input    | text | description |                       |
      | textarea |      | test_data   | secret data goes here |
      | input    | text | test_result |                       |
    And I should see a checkbox labeled "Reuse existing token?"
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
      | phonenumber         | phonenumber         |          |
      | ssn                 | ssn                 |          |
      | string              | string              | true     |
      | timestamp           | timestamp           |          |
      | uuid                | uuid                |          |
    And I should see a dropdown matching selector "[name='output_data_type']" with the following options
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
      | phonenumber         | phonenumber         |          |
      | ssn                 | ssn                 |          |
      | string              | string              | true     |
      | timestamp           | timestamp           |          |
      | uuid                | uuid                |          |
    And I should see a dropdown matching selector "[name='transform_type']" with the following options
      | Text                  | Value               | Selected |
      | Passthrough           | passthrough         |          |
      | Transform             | transform           |          |
      | Tokenize by value     | tokenizebyvalue     | true     |
      | Tokenize by reference | tokenizebyreference |          |
    And I should see a code editor with the ID "transformer_function" and the value "function transform(data, params) {  return data;}"
    And I should see a code editor with the ID "transformer_params" and the value "{}"
    And I should see a button labeled "Run Test"
    And I should see a button labeled "Save Transformer"
    And the button labeled "Save Transformer" should be disabled
    And I should see a button labeled "Cancel"
    And the button labeled "Cancel" should be enabled

  Scenario: create transformer save error
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "dataTypes"
    When I navigate to the page with path "/transformers/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a "h1" with the text "Create Transformer: New Transformer"
    And I should see the following form elements
      | TagName  | Type | Name        | Value                 |
      | input    | text | name        |                       |
      | input    | text | description |                       |
      | textarea |      | test_data   | secret data goes here |
      | input    | text | test_result |                       |
    And I should see a checkbox labeled "Reuse existing token?"
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
      | phonenumber         | phonenumber         |          |
      | ssn                 | ssn                 |          |
      | string              | string              | true     |
      | timestamp           | timestamp           |          |
      | uuid                | uuid                |          |
    And I should see a dropdown matching selector "[name='output_data_type']" with the following options
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
      | phonenumber         | phonenumber         |          |
      | ssn                 | ssn                 |          |
      | string              | string              | true     |
      | timestamp           | timestamp           |          |
      | uuid                | uuid                |          |
    And I should see a dropdown matching selector "[name='transform_type']" with the following options
      | Text                  | Value               | Selected |
      | Passthrough           | passthrough         |          |
      | Transform             | transform           |          |
      | Tokenize by value     | tokenizebyvalue     | true     |
      | Tokenize by reference | tokenizebyreference |          |
    And I should see a code editor with the ID "transformer_function" and the value "function transform(data, params) {  return data;}"
    And I should see a code editor with the ID "transformer_params" and the value "{}"
    And I should see a button labeled "Save Transformer"
    And the button labeled "Save Transformer" should be disabled
    And I should see a button labeled "Cancel"
    And the button labeled "Cancel" should be enabled
    When I type "Foo" into the "name" field
    And I type "Do Kung Fu" into the "description" field
    And I select the option labeled "phonenumber" in the dropdown matching selector "[name='input_data_type']"
    And I select the option labeled "boolean" in the dropdown matching selector "[name='output_data_type']"
    And I select the option labeled "Transform" in the dropdown matching selector "[name='transform_type']"
    Then I should see the following form elements
      | TagName  | Type | Name        | Value                 |
      | input    | text | name        | Foo                   |
      | input    | text | description | Do Kung Fu            |
      | textarea |      | test_data   | secret data goes here |
      | input    | text | test_result |                       |
    And I should not see a checkbox labeled "Reuse existing token?"
    # no function body if transform type is Passthrough
    When I select the option labeled "Passthrough" in the dropdown matching selector "[name='transform_type']"
    Then I should see the following form elements
      | TagName  | Type | Name        | Value                 |
      | input    | text | name        | Foo                   |
      | input    | text | description | Do Kung Fu            |
      | textarea |      | test_data   | secret data goes here |
      | input    | text | test_result |                       |
    And the input with the name "input_data_type" should be disabled
    And the input with the name "output_data_type" should be disabled
    And I should see the following text on the page
      | TagName | TextContent                                                                             |
      | div     | For "passthrough" transformers, input and output data types are automatically inferred. |
    And I should not see a checkbox labeled "Reuse existing token?"
    And I should not see a code editor with the ID "transformer_function"
    And I should not see a code editor with the ID "transformer_params"
    # go back to type = transform
    When I select the option labeled "Transform" in the dropdown matching selector "[name='transform_type']"
    And I replace the text in the code editor with the ID "transformer_function" with the value "function transform(data, params) {  return true;}"
    Then I should see a code editor with the ID "transformer_function" and the value "function transform(data, params) {  return true;}"
    And I should see a button labeled "Save Transformer"
    And the button labeled "Save Transformer" should be enabled
    And I should see a button labeled "Cancel"
    And the button labeled "Cancel" should be enabled
    When I replace the text in the "name" field with ""
    Then I should see the following form elements
      | TagName  | Type | Name        | Value                 |
      | input    | text | name        |                       |
      | input    | text | description | Do Kung Fu            |
      | textarea |      | test_data   | secret data goes here |
      | input    | text | test_result |                       |
    When I click the button labeled "Save Transformer"
    Then the input with the name "name" should be invalid
    When I replace the text in the "name" field with "(&$%"
    Then I should see the following form elements
      | TagName  | Type | Name        | Value                 |
      | input    | text | name        | (&$%                  |
      | input    | text | description | Do Kung Fu            |
      | textarea |      | test_data   | secret data goes here |
      | input    | text | test_result |                       |
    And the input with the name "name" should be invalid
    When I replace the text in the "name" field with "FooBar"
    Then the input with the name "name" should be valid
    Given the following mocked requests:
      | Method | Path                                                                      | Status | Body                                                                           |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation | 400    | {"error":"not great, bob","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
    When I click the button labeled "Save Transformer"
    Then I should see the following text on the page
      | TagName | TextContent                                |
      | p       | Error creating transformer: not great, bob |
    And I should be on the page with the path "/transformers/create"

  Scenario: create transformer save success
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "dataTypes"
    When I navigate to the page with path "/transformers/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a "h1" with the text "Create Transformer: New Transformer"
    And I should see the following form elements
      | TagName  | Type | Name        | Value                 |
      | input    | text | name        |                       |
      | input    | text | description |                       |
      | textarea |      | test_data   | secret data goes here |
      | input    | text | test_result |                       |
    And I should see a checkbox labeled "Reuse existing token?"
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
      | phonenumber         | phonenumber         |          |
      | ssn                 | ssn                 |          |
      | string              | string              | true     |
      | timestamp           | timestamp           |          |
      | uuid                | uuid                |          |
    And I should see a dropdown matching selector "[name='output_data_type']" with the following options
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
      | phonenumber         | phonenumber         |          |
      | ssn                 | ssn                 |          |
      | string              | string              | true     |
      | timestamp           | timestamp           |          |
      | uuid                | uuid                |          |
    And I should see a dropdown matching selector "[name='transform_type']" with the following options
      | Text                  | Value               | Selected |
      | Passthrough           | passthrough         |          |
      | Transform             | transform           |          |
      | Tokenize by value     | tokenizebyvalue     | true     |
      | Tokenize by reference | tokenizebyreference |          |
    And I should see a code editor with the ID "transformer_function" and the value "function transform(data, params) {  return data;}"
    And I should see a code editor with the ID "transformer_params" and the value "{}"
    And I should see a button labeled "Save Transformer"
    And the button labeled "Save Transformer" should be disabled
    And I should see a button labeled "Cancel"
    And the button labeled "Cancel" should be enabled
    When I type "Foo" into the "name" field
    And I type "Do Kung Fu" into the "description" field
    And I select the option labeled "phonenumber" in the dropdown matching selector "[name='input_data_type']"
    And I select the option labeled "boolean" in the dropdown matching selector "[name='output_data_type']"
    And I select the option labeled "Transform" in the dropdown matching selector "[name='transform_type']"
    Then I should see the following form elements
      | TagName  | Type | Name        | Value                 |
      | input    | text | name        | Foo                   |
      | input    | text | description | Do Kung Fu            |
      | textarea |      | test_data   | secret data goes here |
      | input    | text | test_result |                       |
    And I should not see a checkbox labeled "Reuse existing token?"
    When I replace the text in the code editor with the ID "transformer_function" with the value "function transform(data, params) {  return true;}"
    Then I should see a code editor with the ID "transformer_function" and the value "function transform(data, params) {  return true;}"
    And I should see a button labeled "Save Transformer"
    And the button labeled "Save Transformer" should be enabled
    And I should see a button labeled "Cancel"
    And the button labeled "Cancel" should be enabled
    Given the following mocked requests:
      | Method | Path                                                                      | Status | Body                                                             |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/transformation | 200    | single_transformer_get.json                                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions    | 200    | { "create": true, "read": true, "update": true, "delete": true } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates*     | 200    | { "data": [], "has_next": "false", "has_prev": false }           |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access*        | 200    | { "data": [], "has_next": "false", "has_prev": false }           |
    And a mocked "GET" request for "transformers"
    When I click the button labeled "Save Transformer"
    Then I should see the following text on the page
      | TagName | TextContent             |
      | p       | Fetching transformer... |
    And I should be on the page with the path "/transformers/d2685634-ac22-4ce8-bdae-64892f17f4e9/0"
    And I should see a toast notification with the text "Successfully created transformerClose"

  @test_transformer
  Scenario: test transformer
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "dataTypes"
    When I navigate to the page with path "/transformers/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a "h1" with the text "Create Transformer: New Transformer"
    And I should see the following form elements
      | TagName  | Type | Name        | Value                 |
      | input    | text | name        |                       |
      | input    | text | description |                       |
      | textarea |      | test_data   | secret data goes here |
      | input    | text | test_result |                       |
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
    And I should see a code editor with the ID "transformer_function" and the value "function transform(data, params) { const [name, host] = data.split('@'); return `${name}[at]${host.replaceAll('.', '[dot]')}`;}"
    And I should see a code editor with the ID "transformer_params" and the value "{}"
    And the button labeled "Save Transformer" should be enabled
    And the button labeled "Cancel" should be enabled
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
      | TagName  | Type | Name        | Value                                            |
      | input    | text | name        | ObfuscateEmailNaive                              |
      | input    | text | description | Takes foo@bar.com and returns foo[at]bar[dot]com |
      | textarea |      | test_data   | foo@bar.baz.com                                  |
      | input    | text | test_result | foo[at]bar[dot]baz[dot]com                       |
    And I should not see an element matching selector ".alert-message p"

  Scenario: cancel transformer creation
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "tenants_urls"
    And a mocked "GET" request for "permissions"
    And a mocked "GET" request for "dataTypes"
    And a mocked "GET" request for "companies"
    And a mocked "GET" request for "access_policies"
    When I navigate to the page with path "/transformers/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent        |
      | h1      | Create Transformer |
    And I should see a button labeled "Run Test"
    And I should see a button labeled "Save Transformer"
    And I should see a button labeled "Cancel"
    And the button labeled "Run Test" should be enabled
    And the button labeled "Save Transformer" should be disabled
    And the button labeled "Cancel" should be enabled
    When I replace the text in the "name" field with "Our_Transformer"
    Then the button labeled "Save Transformer" should be enabled
    And the button labeled "Cancel" should be enabled
    Given I intend to dismiss the confirm dialog
    When I click the button labeled "Cancel"
    Then I should see the following text on the page
      | TagName | TextContent        |
      | h1      | Create Transformer |
    And I should be on the page with the path "/transformers/create"
    Then I should see a "h1" with the text "Create Transformer: New Transformer"
    And the button labeled "Save Transformer" should be enabled
    And the button labeled "Cancel" should be enabled
    When I replace the text in the "name" field with ""
    Then the button labeled "Save Transformer" should be disabled
    And the button labeled "Cancel" should be enabled
    Given a mocked "GET" request for "tenants"
    And the following mocked requests:
      | Method | Path                                                                   | Status | Body                                                             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/permissions | 200    | { "create": true, "read": true, "update": true, "delete": true } |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/templates*  | 200    | policy_templates.json                                            |
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "transformers"
    # no confirm dialog
    When I click the button labeled "Cancel"
    Then I should see the following text on the page
      | TagName | TextContent |
      | td      | Passthrough |
    And I should be on the page with the path "/transformers"
