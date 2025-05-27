@columns
@column_details
@userstore
Feature: column details page

  Background:
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "dataTypes"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "accessor details"
    And a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    And a mocked "GET" request for "access_policies"
    And the following mocked requests:
      | Method | Path                                                                                                                                    | Status | Body                          |
      | GET    | /api/tenants/*/userstore/columns/032cae17-df3a-4e87-82a0-c706ed0679ee                                                                   | 200    | email_verified_column.json    |
      | GET    | /api/tenants/*/userstore/purposes*                                                                                                      | 200    | empty_paginated_response.json |
      | POST   | /api/tenants/*/userstore/columns/retentiondurations/actions/get                                                                         | 200    | column_purpose_durations.json |
      | GET    | /api/tenants/*/userstore/accessors?limit=50&filter=%28%27column_ids%27%2CHAS%2C%27032cae17-df3a-4e87-82a0-c706ed0679ee%27%29&version=3* | 200    | empty_paginated_response.json |
    When I navigate to the page with path "/columns/032cae17-df3a-4e87-82a0-c706ed0679ee?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent                 |
      | h1      | View Column: email_verified |
    And I should see a button labeled "Edit Column"
    And I should see a cardrow with the title "Basic Details"
    And I should see a cardrow with the title "Default Transformer"
    And I should see a cardrow with the title "Access Policy"
    And I should see a cardrow with the title "Accessors That Read Column"

  @a11y
  Scenario: column details page accessibility
    Then the page should have no accessibility violations

  @array_column
  Scenario: view details for an array column
    And I should see the following text on the page
      | TagName                                        | TextContent            |
      | label                                          | Name                   |
      | label > div > p                                | email_verified         |
      | label                                          | ID                     |
      | label                                          | Column Type            |
      | label > div > p                                | boolean                |
      | label                                          | Table                  |
      | label                                          | Array                  |
      | label:has-text("Array?") > div                 | Off                    |
      | label                                          | Unique Values For User |
      | label:has-text("Unique Values For User") > div | Off                    |
      | label                                          | Unique IDs For User    |
      | label:has-text("Unique IDs For User") > div    | Off                    |
      | label                                          | Partial Updates        |
      | label:has-text("Partial Updates") > div        | Off                    |
      | label                                          | Unique within Tenant   |
      | label:has-text("Unique within Tenant") > div   | Off                    |
      | label                                          | Search Indexed         |
      | label:has-text("Search Indexed") > div         | Off                    |
    And I should see a table with ID "default-transformer-table" and the following data
      | Transformer              | Token Access Policy |
      | PassthroughUnchangedData | n/a                 |
    And I should see a table with ID "column-access-policy" and the following data
      |                           |
      | No policy components yet. |

  @edit_column_name
  Scenario: Edit Column name
    And I should see the following text on the page
      | TagName         | TextContent    |
      | label           | Name           |
      | label > div > p | email_verified |
    And I should see a table with ID "default-transformer-table" and the following data
      | Transformer              | Token Access Policy |
      | PassthroughUnchangedData | n/a                 |
    And I should see a table with ID "column-access-policy" and the following data
      |                           |
      | No policy components yet. |
    When I click the button labeled "Edit Column"
    Then I should see a button labeled "Cancel"
    And I should see a button labeled "Save changes"
    And I should not see a button labeled "Edit Column"
    And I should see the following inputs within the "Basic Details" cardrow
      | Type | Name | Value          | Disabled |
      | text | name | email_verified | false    |
    When I replace the text in the "name" field with "email_verified_by_user"
    Then I should see the following inputs within the "Basic Details" cardrow
      | Type | Name | Value                  | Disabled |
      | text | name | email_verified_by_user | false    |
    And the button labeled "Save changes" should be enabled
    And the button labeled "Cancel" should be enabled
    When I click the button labeled "Save changes"
    Then I should see the following inputs within the "Basic Details" cardrow
      | Type | Name | Value                  | Disabled |
      | text | name | email_verified_by_user | false    |

  @edit_column_default_transformer
  Scenario: Edit Column Default Transformer
    And I should see a table with ID "default-transformer-table" and the following data
      | Transformer              | Token Access Policy |
      | PassthroughUnchangedData | n/a                 |
    When I click the button labeled "Edit Column"
    Then I should see a button labeled "Cancel"
    And I should see a button labeled "Save changes"
    And I should not see a button labeled "Edit Column"
    And I should see a dropdown matching selector "[name='selected_transformer']" with the following options
      | Text                     | Value                                | Selected |
      | Passthrough              | 405d7cf0-e881-40a3-8e53-f76b502d2d76 |          |
      | Always_foo               | 00000000-e881-40a3-8e53-f76b502d2d76 |          |
      | EmailToID                | 0cedf7a4-86ab-450a-9426-478ad0a60faa |          |
      | SSNToID                  | 3f65ee22-2241-4694-bbe3-72cefbe59ff2 |          |
      | CreditCardToID           | 618a4ae7-9979-4ee8-bac5-db87335fe4d9 |          |
      | FullNameToID             | b9bf352f-b1ee-4fb2-a2eb-d0c346c6404b |          |
      | PassthroughUnchangedData | c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a |          |
      | UUID                     | e3743f5b-521e-4305-b232-ee82549e1477 |          |
      | UUIDShouldShowUpAccessor | 00000002-521e-4305-b232-ee82549e1477 |          |
    When I select "Always_foo" from the dropdown in column 1 of row 1 of the table with ID "default-transformer-table"
    Then I should see a table with ID "default-transformer-table" and the following data
      | Transformer | Token Access Policy |
      | Always_foo  | n/a                 |
    When I click the button labeled "Save changes"
    Then I should see a table with ID "default-transformer-table" and the following data
      | Transformer | Token Access Policy |
      | Always_foo  | n/a                 |

  @edit_column_access_policy
  Scenario: Edit Column Access Policy
    Given a mocked "GET" request for "access_policies"
    Then I should see a table with ID "column-access-policy" and the following data
      |       | Policy name            | Parameters |
      | Where | hellopolicy2           | {}         |
      | AND   | hello2_policy          | N/A        |
      | AND   | hellopolicy4           | {}         |
      | AND   | jj_policy              | N/A        |
      | AND   | hello3_policy_ed6a3776 | N/A        |
    When I click the button labeled "Edit Column"
    Then I should see a button labeled "Cancel"
    And I should see a button labeled "Save changes"
    And I should not see a button labeled "Edit Column"
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
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
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "column-access-policy" and the following data
      |       | Policy name            | Parameters |
      | Where | hellopolicy2           |            |
      | AND   | hello2_policy          |            |
      | AND   | hellopolicy4           |            |
      | AND   | jj_policy              |            |
      | AND   | hello3_policy_ed6a3776 |            |
      | AND   | Allow_all              |            |
    When I click the "delete" button in row 6 of the table with ID "column-access-policy"
    Then I should see a table with ID "column-access-policy" and the following data
      |       | Policy name            | Parameters |
      | Where | hellopolicy2           |            |
      | AND   | hello2_policy          |            |
      | AND   | hellopolicy4           |            |
      | AND   | jj_policy              |            |
      | AND   | hello3_policy_ed6a3776 |            |
