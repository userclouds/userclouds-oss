@userstore
@accessors
@accessor_details
Feature: Accessor details page

  # TODO:
  # canceling edit mode
  # error messages
  # verify more of the initial text on the page
  # verify link hrefs
  @a11y
  Scenario: Basic info accessor accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "accessor details"
    And a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    And the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/a78f1f88-3684-4e59-a01d-c121e259ec96 | 200    | global_accessor_policy.json |
    When I navigate to the page with path "/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: Basic info accessor
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "accessor details"
    And a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    And the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/a78f1f88-3684-4e59-a01d-c121e259ec96 | 200    | global_accessor_policy.json |
    When I navigate to the page with path "/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading accessor ..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic details |
      | button  | Edit Accessor |
      | button  | Test accessor |
    And I should see a cardrow with the title "Basic details"
    And I should see a cardrow with the title "Column configuration"
    And I should see a cardrow with the title "Selector"
    And I should see a cardrow with the title "Access Policy"
    And I should see a "h3" with the text "Global Policy (applied by default)"
    And I should see a "h3" with the text "Column Policies (applied to columns read by this accessor)"
    And I should see a "h3" with the text "Manual Policies (manually applied to this accessor as a whole)"
    And I should see a cardrow with the title "Execution Rate Limiting"
    And I should see a cardrow with the title "Result Rate Limiting"
    And I should see a cardrow with the title "Purposes of access"
    And I should see the following text on the page
      | TagName                                | TextContent               |
      | label                                  | Name                      |
      | label:has-text("Name") p               | My_Accessor               |
      | label                                  | ID                        |
      | label:has-text("ID") div               | 2ee449…                   |
      | label                                  | Deleted data access       |
      | label                                  | Version                   |
      | label:has-text("Version") p            | 2                         |
      | label                                  | Audit logged              |
      | label:has-text("Audit logged") svg     | On                        |
      | label                                  | Use Search Index          |
      | label:has-text("Use Search Index") svg | Off                       |
      | label                                  | Description               |
      | label:has-text("Description") p        | This is a simple accessor |
    And I should see a table with ID "purposes" and the following data
      |       | Purpose     | Description |
      | Where | operational |             |

  Scenario: Edit basic details accessor
    Given I am a logged-in user
    And the following feature flags
      | Name                   | Value |
      | global-access-policies | true  |
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "accessor details"
    And a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    And the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/a78f1f88-3684-4e59-a01d-c121e259ec96 | 200    | global_accessor_policy.json |
    When I navigate to the page with path "/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading accessor ..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic details |
      | button  | Edit Accessor |
    And I should see a cardrow with the title "Basic details"
    And I should see a cardrow with the title "Column configuration"
    And I should see a cardrow with the title "Purposes of access"
    # enter edit mode
    Given a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policies"
    When I click the button labeled "Edit Accessor"
    Then I should see the following form elements
      | TagName  | Type | Name                 | Value                     |
      | input    | text | accessor_name        | My_Accessor               |
      | textarea |      | accessor_description | This is a simple accessor |
    And I should see a dropdown matching selector "[name='accessor_purpose']" with the following options
      | Text             | Value                                | Selected |
      | Select a purpose |                                      |          |
      | marketing        | 0b112683-aa23-4269-b098-ae6fdc1a9d8d |          |
    # edit name and description
    When I replace the text in the "accessor_name" field with "Our_Accessor"
    And I replace the text in the "accessor_description" field with "Foo"
    And I select the option labeled "marketing" in the dropdown matching selector "[name='accessor_purpose']"
    Then I should see a table with ID "purposes" and the following data
      |       | Purpose     | Description                        |
      | Where | operational |                                    |
      | And   | marketing   | barraging you with emails and such |
    And I should see a button labeled "Save Accessor"
    And I should see a button labeled "Cancel"
    And I should see a "p" with the text "No purposes available. You can add purposes here."
    When I click the "delete" button in row 1 of the table with ID "purposes"
    Then I should see a table with ID "purposes" and the following data
      |       | Purpose   | Description                        |
      | Where | marketing | barraging you with emails and such |
    # save new name and description
    Given a mocked "GET" request for "updated accessor details"
    # mocks are last-in-first-out, so we have to mock reqs with same URL in reverse
    And a mocked "PUT" request for "updated accessor details"
    And a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    When I click the button labeled "Save Accessor"
    Then I should see a toast notification with the text "Successfully saved accessorClose"
    And I should see the following text on the page
      | TagName         | TextContent        |
      | label           | Name               |
      | label > div > p | Our_Accessor       |
      | label           | Version            |
      | label > div > p | 3                  |
      | label           | ID                 |
      | label > div     | 2ee449…            |
      | label           | Description        |
      | p               | Foo                |
      | h2              | Purposes of access |
    And I should see a table with ID "purposes" and the following data
      |       | Purpose   | Description                        |
      | Where | marketing | barraging you with emails and such |
    And I should not see an element matching selector "#accessorDetails button:has-text('Save changes')"
    And I should not see an element matching selector "#accessorDetails button:has-text('Cancel')"

  Scenario: Edit columns accessor
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "accessor details"
    And a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    And the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/a78f1f88-3684-4e59-a01d-c121e259ec96 | 200    | global_accessor_policy.json |
    When I navigate to the page with path "/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading accessor ..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic details |
      | button  | Edit Accessor |
    And I should see a cardrow with the title "Basic details"
    And I should see a cardrow with the title "Column configuration"
    And I should see a cardrow with the title "Purposes of access"
    Then I should see a table with ID "columnconfig" and the following data
      | Name       | Type   | Transformer |
      | baz_column | string | Passthrough |
    Given a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policies"
    # enter edit mode
    When I click the button labeled "Edit Accessor"
    Then I should see the following text on the page
      | Selector | TextContent   |
      | button   | Save Accessor |
      | button   | Cancel        |
    And I should see a dropdown matching selector "#selectUserStoreColumnToAdd" with the following options
      | Text            | Value                                | Selected |
      | Select a column |                                      | true     |
      | bar_column      | 24b52969-7124-475f-b0e9-3fef50da6b3e |          |
      | foo_column      | b0ee6ec3-3a8b-4bb7-bebb-5ca78e03e184 |          |
    # choose a column to add
    When I select the option labeled "foo_column" in the dropdown matching selector "#selectUserStoreColumnToAdd"
    Then I should see a table with ID "columnconfig" and the following data
      | Name       | Type   | Transformer              |
      | baz_column | string | Passthrough              |
      | foo_column | string | PassthroughUnchangedData |
    And I should see a dropdown in column 3 of row 1 of the table with ID "columnconfig" and the following options
      | Text                     | Value                                | Selected |
      | Select a transformer     |                                      |          |
      | Passthrough (default)    | 00000000-0000-0000-0000-000000000000 |          |
      | Passthrough              | 405d7cf0-e881-40a3-8e53-f76b502d2d76 |          |
      | Always_foo               | 00000000-e881-40a3-8e53-f76b502d2d76 |          |
      | EmailToID                | 0cedf7a4-86ab-450a-9426-478ad0a60faa |          |
      | SSNToID                  | 3f65ee22-2241-4694-bbe3-72cefbe59ff2 |          |
      | CreditCardToID           | 618a4ae7-9979-4ee8-bac5-db87335fe4d9 |          |
      | FullNameToID             | b9bf352f-b1ee-4fb2-a2eb-d0c346c6404b |          |
      | PassthroughUnchangedData | c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a |          |
      | UUID                     | e3743f5b-521e-4305-b232-ee82549e1477 |          |
      | UUIDShouldShowUpAccessor | 00000002-521e-4305-b232-ee82549e1477 |          |
    # The following explicitly shouldn't show up:
    #   | UUIDShouldntShowUp      | 00000000-521e-4305-b232-ee82549e1477 |          |
    #   | UUIDShouldShowUpMutator | 00000001-521e-4305-b232-ee82549e1477 |          |
    And I should see a dropdown matching selector "#selectUserStoreColumnToAdd" with the following options
      | Text            | Value                                | Selected |
      | Select a column |                                      | true     |
      | bar_column      | 24b52969-7124-475f-b0e9-3fef50da6b3e |          |
    When I select "Always_foo" from the dropdown in column 3 of row 2 of the table with ID "columnconfig"
    Then I should see a table with ID "columnconfig" and the following data
      | Name       | Type   | Transformer |
      | baz_column | string | Passthrough |
      | foo_column | string | Always_foo  |
    # choose another column to add
    When I select the option labeled "bar_column" in the dropdown matching selector "#selectUserStoreColumnToAdd"
    Then I should see a table with ID "columnconfig" and the following data
      | Name       | Type      | Transformer              |
      | baz_column | string    | Passthrough              |
      | foo_column | string    | Always_foo               |
      | bar_column | timestamp | PassthroughUnchangedData |
    And I should not see an element matching selector "#selectUserStoreColumnToAdd"
    # queue a persisted column for delete
    When I click the "delete" button in row 1 of the table with ID "columnconfig"
    And I click the "delete" button in row 2 of the table with ID "columnconfig"
    Then I should see a table with ID "columnconfig" and the following data
      | Name       | Type   | Transformer |
      | foo_column | string | Always_foo  |
    And I should see a dropdown matching selector "#selectUserStoreColumnToAdd" with the following options
      | Text            | Value                                | Selected |
      | Select a column |                                      | true     |
      | bar_column      | 24b52969-7124-475f-b0e9-3fef50da6b3e |          |
      | baz_column      | 175565bf-4d3e-4b01-9908-529933dfbc5e |          |
    Given a mocked "PUT" request for "accessor details"
    And a mocked "GET" request for "access_policy"
    And a mocked "GET" request for "accessor details"
    And the following mocked requests:
      | Method | Path                                                                                                         | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/0c0b7371-5175-405b-a17c-fec5969914b8       | 200    | access_policy_comp_inter.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/0c0b7371-5175-405b-a17c-fec5969914b8       | 200    | access_policy_comp_inter.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/0c0b7371-5175-405b-a17c-fec5969914b8       | 200    | access_policy_comp_inter.json |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc   | 200    | accessor_details.json         |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc?* | 200    | accessor_details.json         |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/a78f1f88-3684-4e59-a01d-c121e259ec96       | 200    | global_accessor_policy.json   |
    # save new set of columns
    When I click the button labeled "Save Accessor"
    Then I should see a toast notification with the text "Successfully saved accessorClose"
    And I should not see an element matching selector "button:has-text('Save changes')"
    And I should not see an element matching selector "button:has-text('Cancel')"

  Scenario: Edit selector config accessor
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "accessor details"
    And a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    And the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/a78f1f88-3684-4e59-a01d-c121e259ec96 | 200    | global_accessor_policy.json |
    When I navigate to the page with path "/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading accessor ..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent |
      | h2      | Selector    |
    Given a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policies"
    # enter edit mode
    When I click the button labeled "Edit Accessor"
    Then I should see the following form elements
      | TagName | Type | Name                     | Value    |
      | input   | text | accessor_selector_config | {id} = ? |
    # edit selector config
    When I replace the text in the "accessor_selector_config" field with "{id} LIKE ?"
    Then I should see the following form elements
      | TagName | Type | Name                     | Value       |
      | input   | text | accessor_selector_config | {id} LIKE ? |
    # save new selector config
    Given a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    And the following mocked requests:
      | Method | Path                                                                                                         | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/0c0b7371-5175-405b-a17c-fec5969914b8       | 200    | access_policy_comp_inter.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/0c0b7371-5175-405b-a17c-fec5969914b8       | 200    | access_policy_comp_inter.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/0c0b7371-5175-405b-a17c-fec5969914b8       | 200    | access_policy_comp_inter.json |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc   | 200    | accessor_details.json         |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc?* | 200    | accessor_details.json         |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/a78f1f88-3684-4e59-a01d-c121e259ec96       | 200    | global_accessor_policy.json   |
    When I click the button labeled "Save Accessor"
    Then I should see a toast notification with the text "Successfully saved accessorClose"
    And I should see the following text on the page
      | TagName | TextContent             |
      | label   | Selector "where" clause |
    And I should not see an element matching selector "#accessorDetails button:has-text('Save changes')"
    And I should not see an element matching selector "#accessorDetails button:has-text('Cancel')"

  Scenario: Edit access policy accessor composition type
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "accessor details"
    And a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    And the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/a78f1f88-3684-4e59-a01d-c121e259ec96 | 200    | global_accessor_policy.json |
    When I navigate to the page with path "/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading accessor ..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic details |
      | button  | Edit Accessor |
    And I should see a cardrow with the title "Basic details"
    And I should see a cardrow with the title "Column configuration"
    And I should see a cardrow with the title "Purposes of access"
    And I should see a link to "/policytemplates/b0a4c61b-d2e6-4a20-a9a6-2e89fe7f5f16/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/0ff44561-d498-49b2-8224-c8b273903e27/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/policytemplates/010f6071-8167-4767-a7f8-4e3615f05f15/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/b2c8e2b3-7232-435b-9ab2-ba9ed418d214/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/eff3f014-f8dd-47a8-b9e6-6f62e716ec2d/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters |
      | Where | hellopolicy2           | {}         |
      | AND   | hello2_policy          | N/A        |
      | AND   | hellopolicy4           | {}         |
      | AND   | jj_policy              | N/A        |
      | AND   | hello3_policy_ed6a3776 | N/A        |
    Given a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policies"
    # enter edit mode
    When I click the button labeled "Edit Accessor"
    Then I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Save Accessor"
    And I should see a button labeled "Cancel"
    And I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters | Delete     |
      | Where | hellopolicy2           |            | Delete Bin |
      | AND   | hello2_policy          |            | Delete Bin |
      | AND   | hellopolicy4           |            | Delete Bin |
      | AND   | jj_policy              |            | Delete Bin |
      | AND   | hello3_policy_ed6a3776 |            | Delete Bin |
    When I select the option labeled "OR" in the dropdown matching selector " [name='policyTypeSelector']"
    And I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters | Delete     |
      | Where | hellopolicy2           |            | Delete Bin |
      | OR    | hello2_policy          |            | Delete Bin |
      | OR    | hellopolicy4           |            | Delete Bin |
      | OR    | jj_policy              |            | Delete Bin |
      | OR    | hello3_policy_ed6a3776 |            | Delete Bin |
    Given a mocked "GET" request for "access_policy"
    And the following mocked requests:
      | Method | Path                                                                                                         | Status | Body                          |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/0c0b7371-5175-405b-a17c-fec5969914b8       | 200    | access_policy_comp_inter.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/0c0b7371-5175-405b-a17c-fec5969914b8       | 200    | access_policy_comp_inter.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/0c0b7371-5175-405b-a17c-fec5969914b8       | 200    | access_policy_comp_inter.json |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc   | 200    | accessor_details.json         |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc?* | 200    | accessor_details.json         |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/a78f1f88-3684-4e59-a01d-c121e259ec96       | 200    | global_accessor_policy.json   |
    When I click the button labeled "Save Accessor"
    Then the button labeled "Edit Accessor" should be enabled
    And I should see a toast notification with the text "Successfully saved accessorClose"
    And I should see a link to "/accesspolicies/eff3f014-f8dd-47a8-b9e6-6f62e716ec2d/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic details |
      | button  | Edit Accessor |
    And I should see a cardrow with the title "Basic details"
    And I should see a cardrow with the title "Column configuration"
    And I should see a cardrow with the title "Purposes of access"
    And I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters |
      | Where | hellopolicy2           | {}         |
      | OR    | hello2_policy          | N/A        |
      | OR    | hellopolicy4           | {}         |
      | OR    | jj_policy              | N/A        |
      | OR    | hello3_policy_ed6a3776 | N/A        |

  Scenario: Edit access policy composition type for accessor
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "token res accessor details"
    And a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    And a mocked "GET" request for "transformers"
    And the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/1f44a215-2590-488f-a254-b6e8e4b67346 | 200    | access_policy_2.json        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/a78f1f88-3684-4e59-a01d-c121e259ec96 | 200    | global_accessor_policy.json |
    When I navigate to the page with path "/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading accessor ..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic details |
      | button  | Edit Accessor |
    And I should see a cardrow with the title "Basic details"
    And I should see a cardrow with the title "Column configuration"
    And I should see a cardrow with the title "Purposes of access"
    And I should see a link to "/policytemplates/b0a4c61b-d2e6-4a20-a9a6-2e89fe7f5f16/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/0ff44561-d498-49b2-8224-c8b273903e27/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/policytemplates/010f6071-8167-4767-a7f8-4e3615f05f15/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/b2c8e2b3-7232-435b-9ab2-ba9ed418d214/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/eff3f014-f8dd-47a8-b9e6-6f62e716ec2d/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters |
      | Where | hellopolicy2           | {}         |
      | AND   | hello2_policy          | N/A        |
      | AND   | hellopolicy4           | {}         |
      | AND   | jj_policy              | N/A        |
      | AND   | hello3_policy_ed6a3776 | N/A        |
    Given a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policies"
    # enter edit mode
    When I click the button labeled "Edit Accessor"
    Then I should see 1 buttons labeled "Add Policy" and they should be enabled
    And I should see 1 buttons labeled "Add Template" and they should be enabled
    And I should see a button labeled "Save Accessor"
    And I should see a button labeled "Cancel"
    Given a mocked "GET" request for "access_policy"
    And the following mocked requests:
      | Method | Path                                                                                                         | Status | Body                            |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/0c0b7371-5175-405b-a17c-fec5969914b8*      | 200    | access_policy_comp_inter.json   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/1f44a215-2590-488f-a254-b6e8e4b67346*      | 200    | access_policy_2_comp_inter.json |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc*  | 200    | accessor_details_token_res.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc?* | 200    | accessor_details_token_res.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/0c0b7371-5175-405b-a17c-fec5969914b8*      | 200    | access_policy_comp_inter.json   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/1f44a215-2590-488f-a254-b6e8e4b67346*      | 200    | access_policy_2_comp_inter.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/a78f1f88-3684-4e59-a01d-c121e259ec96       | 200    | global_accessor_policy.json     |
    When I click the button labeled "Save Accessor"
    Then the button labeled "Edit Accessor" should be enabled
    And I should see a toast notification with the text "Successfully saved accessorClose"
    And I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic details |
      | button  | Edit Accessor |
    And I should see a cardrow with the title "Basic details"
    And I should see a cardrow with the title "Column configuration"
    And I should see a cardrow with the title "Purposes of access"

  Scenario: Edit access policy accessor add and remove policies and templates
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "accessor details"
    And a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    And the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access/a78f1f88-3684-4e59-a01d-c121e259ec96 | 200    | global_accessor_policy.json |
    When I navigate to the page with path "/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading accessor ..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent   |
      | h2      | Basic details |
      | button  | Edit Accessor |
    And I should see a cardrow with the title "Basic details"
    And I should see a cardrow with the title "Column configuration"
    And I should see a cardrow with the title "Purposes of access"
    And I should see a link to "/policytemplates/b0a4c61b-d2e6-4a20-a9a6-2e89fe7f5f16/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/0ff44561-d498-49b2-8224-c8b273903e27/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/policytemplates/010f6071-8167-4767-a7f8-4e3615f05f15/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/b2c8e2b3-7232-435b-9ab2-ba9ed418d214/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a link to "/accesspolicies/eff3f014-f8dd-47a8-b9e6-6f62e716ec2d/latest?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters |
      | Where | hellopolicy2           | {}         |
      | AND   | hello2_policy          | N/A        |
      | AND   | hellopolicy4           | {}         |
      | AND   | jj_policy              | N/A        |
      | AND   | hello3_policy_ed6a3776 | N/A        |
    # enter edit mode
    Given a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "access_policies"
    When I click the button labeled "Edit Accessor"
    Then I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    And I should see a button labeled "Save Accessor"
    And I should see a button labeled "Cancel"
    And I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters | Delete     |
      | Where | hellopolicy2           |            | Delete Bin |
      | AND   | hello2_policy          |            | Delete Bin |
      | AND   | hellopolicy4           |            | Delete Bin |
      | AND   | jj_policy              |            | Delete Bin |
      | AND   | hello3_policy_ed6a3776 |            | Delete Bin |
    Given a mocked "GET" request for "access_policies"
    When I click the button labeled "Add Policy"
    Then I should see a "p" with the text "Loading policies..."
    # this checks for the first row:
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
    And I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters | Delete     |
      | Where | hellopolicy2           |            | Delete Bin |
      | AND   | hello2_policy          |            | Delete Bin |
      | AND   | hellopolicy4           |            | Delete Bin |
      | AND   | jj_policy              |            | Delete Bin |
      | AND   | hello3_policy_ed6a3776 |            | Delete Bin |
      | AND   | Allow_all              |            | Delete Bin |
    When I click the "delete" button in row 2 of the table with ID "accessorManualPolicyComponents"
    Then I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters | Delete     |
      | Where | hellopolicy2           |            | Delete Bin |
      | AND   | hellopolicy4           |            | Delete Bin |
      | AND   | jj_policy              |            | Delete Bin |
      | AND   | hello3_policy_ed6a3776 |            | Delete Bin |
      | AND   | Allow_all              |            | Delete Bin |
    Given a mocked "GET" request for "access_policy_templates"
    When I click the button labeled "Add Template"
    Then I should see a "p" with the text "Loading templates..."
    # this checks for the first row:
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
    Then I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters | Delete     |
      | Where | hellopolicy2           |            | Delete Bin |
      | AND   | hellopolicy4           |            | Delete Bin |
      | AND   | jj_policy              |            | Delete Bin |
      | AND   | hello3_policy_ed6a3776 |            | Delete Bin |
      | AND   | Allow_all              |            | Delete Bin |
      | AND   | AllowAll               |            | Delete Bin |
    When I click the "delete" button in row 2 of the table with ID "accessorManualPolicyComponents"
    Then I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name            | Parameters | Delete     |
      | Where | hellopolicy2           |            | Delete Bin |
      | AND   | jj_policy              |            | Delete Bin |
      | AND   | hello3_policy_ed6a3776 |            | Delete Bin |
      | AND   | Allow_all              |            | Delete Bin |
      | AND   | AllowAll               |            | Delete Bin |
