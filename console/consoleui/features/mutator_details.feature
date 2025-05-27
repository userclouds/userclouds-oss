@userstore
@mutators
@mutator_details
Feature: Mutator details page

  # TODO:
  # canceling edit mode
  # error messages
  # verify more of the initial text on the page
  # verify link hrefs
  @a11y
  Scenario: Basic info mutator accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a request for user store schema edit system columns
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policy"
    And a mocked "GET" request for "mutator details"
    When I navigate to the page with path "/mutators/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: Basic info mutator
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a request for user store schema edit system columns
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policy"
    And a mocked "GET" request for "mutator details"
    When I navigate to the page with path "/mutators/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading mutator..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a cardrow with the title "Basic Details"
    And I should see a cardrow with the title "Columns"
    And I should see a cardrow with the title "Selector"
    And I should see a cardrow with the title "Access Policy"
    And I should see the following text on the page
      | TagName            | TextContent              |
      | label              | Name                     |
      | label > div > p    | My_Mutator               |
      | label              | Version                  |
      | label > div > p    | 2                        |
      | label              | ID                       |
      | label > div > span | 2ee449…                  |
      | label              | Description              |
      | label + p          | This is a simple mutator |

  Scenario: Edit basic details mutator
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a request for user store schema edit system columns
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policy"
    And a mocked "GET" request for "mutator details"
    When I navigate to the page with path "/mutators/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading mutator..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent  |
      | button  | Edit Mutator |
    And I should see a cardrow with the title "Basic Details"
    # enter edit mode
    When I click the button labeled "Edit Mutator"
    Then I should see the following form elements
      | TagName  | Type | Name                | Value                    |
      | input    | text | mutator_name        | My_Mutator               |
      | textarea |      | mutator_description | This is a simple mutator |
    Then I should see the following text on the page
      | Selector         | TextContent  |
      | button[disabled] | Save Mutator |
      | button           | Cancel       |
    # edit name and description
    When I replace the text in the "mutator_name" field with "Our_Mutator"
    And I replace the text in the "mutator_description" field with "Foo"
    Then I should see the following text on the page
      | Selector               | TextContent  |
      | button:not([disabled]) | Save Mutator |
      | button:not([disabled]) | Cancel       |
    # save new name and description
    Given a mocked "GET" request for "updated mutator details"
    # mocks are last-in-first-out, so we have to mock reqs with same URL in reverse
    And a mocked "PUT" request for "updated mutator details"
    And a mocked "GET" request for "access_policies"
    And a request for user store schema edit system columns
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policy"
    When I click the button labeled "Save Mutator"
    Then I should see a toast notification with the text "Successfully updated mutatorClose"
    And I should see the following text on the page
      | TagName            | TextContent |
      | label              | Name        |
      | label > div > p    | Our_Mutator |
      | label              | Version     |
      | label > div > p    | 3           |
      | label              | ID          |
      | label > div > span | 2ee449…     |
      | label              | Description |
      | label + p          | Foo         |
    And I should not see an element matching selector "button:has-text('Save Mutator')"
    And I should not see an element matching selector "button:has-text('Cancel')"

  Scenario: Edit columns mutator
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a request for user store schema edit system columns
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policy"
    And a mocked "GET" request for "mutator details"
    When I navigate to the page with path "/mutators/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading mutator..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent  |
      | button  | Edit Mutator |
    And I should see a cardrow with the title "Columns"
    Then I should see a table with ID "columns" and the following data
      | Name       | Type   | Array | Normalizer |
      | baz_column | string | Off   | Always_foo |
    Given a request for user store schema edit system columns
    And a mocked "GET" request for "transformers"
    # enter edit mode
    When I click the button labeled "Edit Mutator"
    Then I should see the following text on the page
      | Selector | TextContent  |
      | button   | Save Mutator |
      | button   | Cancel       |
    And I should see a dropdown matching selector "#selectUserStoreColumnToAdd" with the following options
      | Text            | Value                                | Selected |
      | Select a column |                                      | true     |
      | bar_column      | 24b52969-7124-475f-b0e9-3fef50da6b3e |          |
      | foo_column      | b0ee6ec3-3a8b-4bb7-bebb-5ca78e03e184 |          |
    And I should see a dropdown matching selector "[name='selected_normalizer']" with the following options
      | Text                     | Value                                | Selected |
      | Select a normalizer      |                                      |          |
      | Passthrough              | 405d7cf0-e881-40a3-8e53-f76b502d2d76 |          |
      | Always_foo               | 00000000-e881-40a3-8e53-f76b502d2d76 |          |
      | EmailToID                | 0cedf7a4-86ab-450a-9426-478ad0a60faa |          |
      | SSNToID                  | 3f65ee22-2241-4694-bbe3-72cefbe59ff2 |          |
      | CreditCardToID           | 618a4ae7-9979-4ee8-bac5-db87335fe4d9 |          |
      | FullNameToID             | b9bf352f-b1ee-4fb2-a2eb-d0c346c6404b |          |
      | PassthroughUnchangedData | c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a |          |
      | UUIDShouldShowUpMutator  | 00000001-521e-4305-b232-ee82549e1477 |          |
    And I should see a dropdown matching selector "[name='selected_normalizer']" without the following options
      | Text                     | Value                                | Selected |
      | UUIDShouldntShowUp       | 00000000-521e-4305-b232-ee82549e1477 |          |
      | UUIDShouldShowUpAccessor | 00000002-521e-4305-b232-ee82549e1477 |          |
    # add column
    When I select the option labeled "foo_column" in the dropdown matching selector "#selectUserStoreColumnToAdd"
    Then I should see a table with ID "columns" and the following data
      | Name       | Type   | Array | Normalizer          |
      | baz_column | string | Off   | Always_foo          |
      | foo_column | string | Off   | Select a normalizer |
    And I should see a dropdown matching selector "#selectUserStoreColumnToAdd" with the following options
      | Text            | Value                                | Selected |
      | Select a column |                                      | true     |
      | bar_column      | 24b52969-7124-475f-b0e9-3fef50da6b3e |          |
    When I select "Always_foo" from the dropdown in column 4 of row 2 of the table with ID "columns"
    Then I should see a table with ID "columns" and the following data
      | Name       | Type   | Array | Normalizer |
      | baz_column | string | Off   | Always_foo |
      | foo_column | string | Off   | Always_foo |
    # add column
    When I select the option labeled "bar_column" in the dropdown matching selector "#selectUserStoreColumnToAdd"
    Then I should see a table with ID "columns" and the following data
      | Name       | Type      | Array | Normalizer          |
      | baz_column | string    | Off   | Always_foo          |
      | foo_column | string    | Off   | Always_foo          |
      | bar_column | timestamp | Off   | Select a normalizer |
    And I should not see an element matching selector "#selectUserStoreColumnToAdd"
    # queue a persisted column for delete
    When I click the "delete" button in row 1 of the table with ID "columns"
    Then row 1 of the table with ID "columns" should be marked for delete
    # remove unsaved columns
    When I click the "delete" button in row 3 of the table with ID "columns"
    Then I click the "delete" button in row 2 of the table with ID "columns"
    # unqueue persisted column
    When I click the "delete" button in row 1 of the table with ID "columns"
    Then row 1 of the table with ID "columns" should not be marked for delete
    And I should see a table with ID "columns" and the following data
      | Name       | Type   | Array | Normalizer |
      | baz_column | string | Off   | Always_foo |
    And I should see a dropdown matching selector "#selectUserStoreColumnToAdd" with the following options
      | Text            | Value                                | Selected |
      | Select a column |                                      | true     |
      | bar_column      | 24b52969-7124-475f-b0e9-3fef50da6b3e |          |
      | foo_column      | b0ee6ec3-3a8b-4bb7-bebb-5ca78e03e184 |          |
    When I select "PassthroughUnchangedData" from the dropdown in column 4 of row 1 of the table with ID "columns"
    Then I should see a table with ID "columns" and the following data
      | Name       | Type   | Array | Normalizer               |
      | baz_column | string | Off   | PassthroughUnchangedData |
    Given a mocked "GET" request for "updated mutator details"
    # mocks are last-in-first-out, so we have to mock reqs with same URL in reverse
    And a mocked "PUT" request for "updated mutator details"
    And a mocked request to save a mutator with 1 column
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policy"
    And a mocked "GET" request for "mutator details"
    # save new set of columns
    When I click the button labeled "Save Mutator"
    Then I should see a toast notification with the text "Successfully updated mutatorClose"
    And I should see the following text on the page
      | Selector | TextContent  |
      | button   | Edit Mutator |
    And I should see a table with ID "columns" and the following data
      | Name       | Type   | Array | Transformer |
      | baz_column | string | Off   | Always_foo  |
    And I should not see an element matching selector "button:has-text('Save Mutator')"
    And I should not see an element matching selector "button:has-text('Cancel')"

  Scenario: Edit selector config mutator
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a request for user store schema edit system columns
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policy"
    And a mocked "GET" request for "mutator details"
    When I navigate to the page with path "/mutators/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading mutator..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent  |
      | button  | Edit Mutator |
    And I should see a cardrow with the title "Selector"
    # enter edit mode
    When I click the button labeled "Edit Mutator"
    Then I should see the following form elements
      | TagName | Type | Name                    | Value    |
      | input   | text | mutator_selector_config | {id} = ? |
    Then I should see the following text on the page
      | Selector         | TextContent  |
      | button[disabled] | Save Mutator |
      | button           | Cancel       |
    # edit selector config
    When I replace the text in the "mutator_selector_config" field with "{id} LIKE ?"
    Then I should see the following text on the page
      | Selector               | TextContent  |
      | button:not([disabled]) | Save Mutator |
      | button:not([disabled]) | Cancel       |
    # save new selector config
    Given a mocked "GET" request for "updated mutator details"
    # mocks are last-in-first-out, so we have to mock reqs with same URL in reverse
    And a mocked "PUT" request for "updated mutator details"
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policy"
    And a mocked "GET" request for "mutator details"
    When I click the button labeled "Save Mutator"
    Then I should see a toast notification with the text "Successfully updated mutatorClose"
    And I should see the following text on the page
      | TagName         | TextContent             |
      | label           | Selector "where" clause |
      | label > div > p | {id} LIKE ?             |
      | button          | Edit Mutator            |
    And I should not see an element matching selector "button:has-text('Save Mutator')"
    And I should not see an element matching selector "button:has-text('Cancel')"

  Scenario: Edit access policy for a mutator
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a request for user store schema edit system columns
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policy"
    And a mocked "GET" request for "mutator details"
    When I navigate to the page with path "/mutators/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading mutator..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent  |
      | button  | Edit Mutator |
    And I should see a link to "/policytemplates/b0a4c61b-d2e6-4a20-a9a6-2e89fe7f5f16/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/0ff44561-d498-49b2-8224-c8b273903e27/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/policytemplates/010f6071-8167-4767-a7f8-4e3615f05f15/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/b2c8e2b3-7232-435b-9ab2-ba9ed418d214/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/eff3f014-f8dd-47a8-b9e6-6f62e716ec2d/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "mutatorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters |
      | Where | hellopolicy2           | {}         |
      | AND   | hello2_policy          | N/A        |
      | AND   | hellopolicy4           | {}         |
      | AND   | jj_policy              | N/A        |
      | AND   | hello3_policy_ed6a3776 | N/A        |
    And I should see a cardrow with the title "Access Policy"
    Given a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "access_policy"
    When I click the button labeled "Edit Mutator"
    Then I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Save Mutator"
    And I should see a button labeled "Cancel"
    And I should see a table with ID "mutatorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters | Delete     |
      | Where | hellopolicy2           |            | Delete Bin |
      | AND   | hello2_policy          |            | Delete Bin |
      | AND   | hellopolicy4           |            | Delete Bin |
      | AND   | jj_policy              |            | Delete Bin |
      | AND   | hello3_policy_ed6a3776 |            | Delete Bin |
    When I select the option labeled "OR" in the dropdown matching selector " [name='policyTypeSelector']"
    Then I should see a table with ID "mutatorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters | Delete     |
      | Where | hellopolicy2           |            | Delete Bin |
      | OR    | hello2_policy          |            | Delete Bin |
      | OR    | hellopolicy4           |            | Delete Bin |
      | OR    | jj_policy              |            | Delete Bin |
      | OR    | hello3_policy_ed6a3776 |            | Delete Bin |
    Given a mocked "GET" request for "updated mutator details"
    And the following mocked requests:
      | Method | Path                                                                                                       | Status | Body                          |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc | 200    | updated_mutator_details.json  |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/0c0b7371-5175-405b-a17c-fec5969914b8     | 200    | access_policy_comp_inter.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/0c0b7371-5175-405b-a17c-fec5969914b8     | 200    | access_policy_comp_inter.json |
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "access_policies"
    When I click the button labeled "Save Mutator"
    Then I should see a table with ID "mutatorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters |
      | Where | hellopolicy2           |            |
      | OR    | hello2_policy          |            |
      | OR    | hellopolicy4           |            |
      | OR    | jj_policy              |            |
      | OR    | hello3_policy_ed6a3776 |            |

  Scenario: Edit access policy mutator add and remove policies and templates
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a request for user store schema edit system columns
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "access_policy_templates"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policy"
    And a mocked "GET" request for "mutator details"
    When I navigate to the page with path "/mutators/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading mutator..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent  |
      | button  | Edit Mutator |
    And I should see a link to "/policytemplates/b0a4c61b-d2e6-4a20-a9a6-2e89fe7f5f16/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/0ff44561-d498-49b2-8224-c8b273903e27/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/policytemplates/010f6071-8167-4767-a7f8-4e3615f05f15/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/b2c8e2b3-7232-435b-9ab2-ba9ed418d214/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/eff3f014-f8dd-47a8-b9e6-6f62e716ec2d/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "mutatorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters |
      | Where | hellopolicy2           | {}         |
      | AND   | hello2_policy          | N/A        |
      | AND   | hellopolicy4           | {}         |
      | AND   | jj_policy              | N/A        |
      | AND   | hello3_policy_ed6a3776 | N/A        |
    And I should see a cardrow with the title "Access Policy"
    Given a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "access_policy_templates"
    # enter edit mode
    When I click the button labeled "Edit Mutator"
    Then I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Save Mutator"
    And I should see a button labeled "Cancel"
    And I should see a table with ID "mutatorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters | Delete     |
      | Where | hellopolicy2           |            | Delete Bin |
      | AND   | hello2_policy          |            | Delete Bin |
      | AND   | hellopolicy4           |            | Delete Bin |
      | AND   | jj_policy              |            | Delete Bin |
      | AND   | hello3_policy_ed6a3776 |            | Delete Bin |
    When I click the button labeled "Add Policy"
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
    Then I should see a table with ID "mutatorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters | Delete     |
      | Where | hellopolicy2           |            | Delete Bin |
      | AND   | hello2_policy          |            | Delete Bin |
      | AND   | hellopolicy4           |            | Delete Bin |
      | AND   | jj_policy              |            | Delete Bin |
      | AND   | hello3_policy_ed6a3776 |            | Delete Bin |
      | AND   | Allow_all              |            | Delete Bin |
    When I click the "delete" button in row 2 of the table with ID "mutatorManualPolicyComponents"
    Then I should see a table with ID "mutatorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters | Delete     |
      | Where | hellopolicy2           |            | Delete Bin |
      | AND   | hellopolicy4           |            | Delete Bin |
      | AND   | jj_policy              |            | Delete Bin |
      | AND   | hello3_policy_ed6a3776 |            | Delete Bin |
      | AND   | Allow_all              |            | Delete Bin |
    When I click the button labeled "Add Template"
    Then I should see a "dialog td" with the text "CheckAttribute"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    And I should see a table with ID "paginatedPolicyChooserTemplates" and the following data
      | Select | Name              | Type     | Version | ID                                   |
      |        | hellopolicy4      | Template | 0       | 010f6071-8167-4767-a7f8-4e3615f05f15 |
      |        | AllowAll          | Template | 0       | 1e742248-fdde-4c88-9ea7-2c2106ec7aa8 |
      |        | jsljkdfkjnsdajnfl | Template | 0       | 23a4909c-8c1a-450e-8f9b-abc90e9771d8 |
      |        | hefa              | Template | 0       | 23e84792-3971-4d08-80da-729b608f0b01 |
      |        | testwkyle         | Template | 0       | 3053a10b-49a9-4da0-a392-fb9228e98c00 |
      |        | ksdfjasdfasdf     | Template | 0       | 61851da2-810c-4bf9-9656-583d94ba7d5f |
      |        | a                 | Template | 0       | a3cde40f-1cbc-491a-b423-826f5041b858 |
      |        | hellopolicy2      | Template | 0       | b0a4c61b-d2e6-4a20-a9a6-2e89fe7f5f16 |
      |        | somethinga        | Template | 0       | cafd385e-8b2b-49df-bbc7-69e55a163fe1 |
      |        | hellopolicy1      | Template | 0       | cc3051ef-1dd2-4aef-bd40-fd046f8a3cd5 |
      |        | aslkjdfhaslkjdhf  | Template | 0       | d4a48099-3850-4b44-b282-8dc3d843a762 |
      |        | tyler1            | Template | 0       | db97e49f-806e-473d-9ad3-b0d5303e2263 |
      |        | hellopolicy3      | Template | 0       | e123f616-e546-4686-9009-d13f6908c721 |
      |        | henlo             | Template | 0       | e7e7a8bc-1e68-4ee0-84cf-e373a1f19bd8 |
      |        | atemp             | Template | 0       | 95235dfc-ba40-4828-a923-6ac7e616d281 |
      |        | checkIfEven       | Template | 0       | aa412fd1-7c82-4b54-9ffd-b50f589642c6 |
      |        | CheckAttribute    | Template | 0       | aad2bf25-311f-467e-9169-a6a89b6d34a6 |
    When I click the radio input in row 2 of the table with ID "paginatedPolicyChooserTemplates"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "mutatorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters | Delete     |
      | Where | hellopolicy2           |            | Delete Bin |
      | AND   | hellopolicy4           |            | Delete Bin |
      | AND   | jj_policy              |            | Delete Bin |
      | AND   | hello3_policy_ed6a3776 |            | Delete Bin |
      | AND   | Allow_all              |            | Delete Bin |
      | AND   | AllowAll               |            | Delete Bin |
    When I click the "delete" button in row 2 of the table with ID "mutatorManualPolicyComponents"
    Then I should see a table with ID "mutatorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters | Delete     |
      | Where | hellopolicy2           |            | Delete Bin |
      | AND   | jj_policy              |            | Delete Bin |
      | AND   | hello3_policy_ed6a3776 |            | Delete Bin |
      | AND   | Allow_all              |            | Delete Bin |
      | AND   | AllowAll               |            | Delete Bin |
