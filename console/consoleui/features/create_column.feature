@userstore
@columns
@create_column
Feature: create column page

  @a11y
  Scenario: create column
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "dataTypes"
    And the following mocked requests:
      | Method | Path                               | Status | Body                          |
      | GET    | /api/tenants/*/userstore/purposes* | 200    | empty_paginated_response.json |
      | GET    | /api/tenants/*/policies*           | 200    | access_policies.json          |
    When I navigate to the page with path "/columns/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations
    Then the page title should be "[dev] UserClouds Console"
    And I should see a cardrow with the title "Basic Details"
    And I should see the following inputs within the "Basic Details" cardrow
      | Type     | Name                   | Value | Disabled |
      | text     | name                   |       | false    |
      | checkbox | is_array               | on    | false    |
      | checkbox | unique_values_for_user | on    | true     |
      | checkbox | unique_ids_for_user    | on    | false    |
      | checkbox | allow_partial_updates  | on    | true     |
      | checkbox | immutable              | on    | true     |
      | checkbox | unique                 | on    | false    |
      | checkbox | searched_index         | on    | false    |
    And I should see a dropdown matching selector "[name='field_type']" with the following options
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
      | string              | string              |          |
      | timestamp           | timestamp           |          |
      | uuid                | uuid                |          |
    When I replace the text in the "name" field with "email_verified"
    And I select the option labeled "boolean" in the dropdown matching selector "[name='field_type']"
    Then I should see the following inputs within the "Basic Details" cardrow
      | Type | Name | Value          | Disabled |
      | text | name | email_verified | false    |
    And I should see a cardrow with the title "Access Policy"
    Given a mocked "GET" request for "access_policies"
    And I should see a table with ID "column-access-policy" and the following data
      |                           |
      | No policy components yet. |
    When I click the button labeled "Add Policy"
    Then I should see a "p" with the text "Loading policies..."
    Then I should see a "dialog td" with the text "Allow_all"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    And I should see a table with ID "paginatedPolicyChooserPolicies" and the following data
      | Select | Name           | Type   | Version | ID                                   |
      |        | Allow_all      | Policy | 0       | 0c0b7371-5175-405b-a17c-fec5969914b8 |
      |        | Dont_Allow_Any | Policy | 1       | 00000000-5175-405b-a17c-fec5969914b8 |
    When I click the radio input in row 1 of the table with ID "paginatedPolicyChooserPolicies"
    And I click the button labeled "Save selection"
    Then I should see a table with ID "column-access-policy" and the following data
      |       | Policy name | Parameters |
      | Where | Allow_all   |            |
    Given a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "POST" request for "columns"
    When I click the button labeled "Save Column"
    Then I should see a "p" with the text "Loading ..."
