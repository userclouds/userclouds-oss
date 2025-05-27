@userstore @accessors @create_accessor
Feature: Create accessor page

  @a11y
  Scenario: Create accessor accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    When I navigate to the page with path "/accessors/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  @ally
  Scenario: Create accessor
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    When I navigate to the page with path "/accessors/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent     |
      | h2      | Basic Details   |
      | h2      | Columns         |
      | td      | No columns      |
      | h2      | Selector        |
      | h2      | Policy          |
      | button  | Add Policy      |
      | button  | Create Accessor |
      | button  | Cancel          |
    And I should see the following form elements
      | TagName  | Type | Name                     | Value            |
      | input    | text | accessor_name            |                  |
      | textarea |      | accessor_description     |                  |
      | select   |      | accessor_purpose         | Select a purpose |
      | select   |      | select_column            |                  |
      | input    | text | accessor_selector_config | {id} = ANY(?)    |
    And I should see a dropdown matching selector "[name='accessor_purpose']" with the following options
      | Text             | Value                                | Selected |
      | Select a purpose |                                      |          |
      | marketing        | 0b112683-aa23-4269-b098-ae6fdc1a9d8d |          |
      | operational      | 7f55f479-3822-4976-a8a9-b789d5c6f152 |          |
    And I should see a dropdown matching selector "#selectUserStoreColumnToAdd" with the following options
      | Text            | Value      | Selected |
      | Select a column |            | true     |
      | bar_column      | bar_column |          |
      | baz_column      | baz_column |          |
      | foo_column      | foo_column |          |
    # input name and description
    When I replace the text in the "accessor_name" field with "Our_Accessor"
    And I replace the text in the "accessor_description" field with "foo"
    # purposes
    And I select the option labeled "operational" in the dropdown matching selector "[name='accessor_purpose']"
    Then I should see a table with ID "purpose" and the following data
      |       | Purpose     | Description                                |
      | Where | operational | Purpose is for basic operation of the site |
    When I select the option labeled "marketing" in the dropdown matching selector "[name='accessor_purpose']"
    Then I should see a table with ID "purpose" and the following data
      |       | Purpose     | Description                                |
      | Where | operational | Purpose is for basic operation of the site |
      | And   | marketing   | barraging you with emails and such         |
    When I click the "delete" button in row 1 of the table with ID "purpose"
    Then I should see a table with ID "purpose" and the following data
      |       | Purpose   | Description                        |
      | Where | marketing | barraging you with emails and such |
    # choose a column to add
    When I select the option labeled "foo_column" in the dropdown matching selector "#selectUserStoreColumnToAdd"
    Then I should see a table with ID "accessorColumns" and the following data
      | Name       | Type   | Transformer          |
      | foo_column | string | Select a transformer |
    And I should see a dropdown matching selector "[name='selected_transformer']" with the following options
      | Text                               | Value                                | Selected |
      | Select a transformer               |                                      |          |
      | PassthroughUnchangedData (default) | 00000000-0000-0000-0000-000000000000 |          |
      | Passthrough                        | 405d7cf0-e881-40a3-8e53-f76b502d2d76 |          |
      | Always_foo                         | 00000000-e881-40a3-8e53-f76b502d2d76 |          |
      | EmailToID                          | 0cedf7a4-86ab-450a-9426-478ad0a60faa |          |
      | SSNToID                            | 3f65ee22-2241-4694-bbe3-72cefbe59ff2 |          |
      | CreditCardToID                     | 618a4ae7-9979-4ee8-bac5-db87335fe4d9 |          |
      | FullNameToID                       | b9bf352f-b1ee-4fb2-a2eb-d0c346c6404b |          |
      | PassthroughUnchangedData           | c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a |          |
      | UUID                               | e3743f5b-521e-4305-b232-ee82549e1477 |          |
      | UUIDShouldShowUpAccessor           | 00000002-521e-4305-b232-ee82549e1477 |          |
    And I should see a dropdown matching selector "[name='selected_transformer']" without the following options
      | Text                    | Value                                | Selected |
      | UUIDShouldntShowUp      | 00000000-521e-4305-b232-ee82549e1477 |          |
      | UUIDShouldShowUpMutator | 00000001-521e-4305-b232-ee82549e1477 |          |
    When I select "Passthrough" from the dropdown in column 3 of row 1 of the table with ID "accessorColumns"
    Then I should see a table with ID "accessorColumns" and the following data
      | Name       | Type   | Transformer |
      | foo_column | string | Passthrough |
    And I should see a dropdown matching selector "#selectUserStoreColumnToAdd" with the following options
      | Text            | Value      | Selected |
      | Select a column |            | true     |
      | bar_column      | bar_column |          |
      | baz_column      | baz_column |          |
    # choose another column to add
    When I select the option labeled "bar_column" in the dropdown matching selector "#selectUserStoreColumnToAdd"
    Then I should see a table with ID "accessorColumns" and the following data
      | Name       | Type      | Transformer          |
      | foo_column | string    | Passthrough          |
      | bar_column | timestamp | Select a transformer |
    And I should see a dropdown matching selector "#selectUserStoreColumnToAdd" with the following options
      | Text            | Value      | Selected |
      | Select a column |            | true     |
      | baz_column      | baz_column |          |
    # remove a just-added column
    When I click the "delete" button in row 2 of the table with ID "accessorColumns"
    Then I should see a table with ID "accessorColumns" and the following data
      | Name       | Type   | Transformer |
      | foo_column | string | Passthrough |
    And I should see a dropdown matching selector "#selectUserStoreColumnToAdd" with the following options
      | Text            | Value      | Selected |
      | Select a column |            | true     |
      | bar_column      | bar_column |          |
      | baz_column      | baz_column |          |
    # edit selector config
    When I replace the text in the "accessor_selector_config" field with "{id} LIKE ?"
    Then I should see the following form elements
      | TagName | Type | Name                     | Value       |
      | input   | text | accessor_selector_config | {id} LIKE ? |
    Given the following mocked requests:
      | Method | Path                                                               | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access* |    200 | access_policies_page_1.json |
    When I click the button labeled "Add Policy"
    Then I should see a "p" with the text "Loading policies..."
    # this is the first row in the table:
    Then I should see a "td" with the text "AllowAll"
    And I should see a table with ID "paginatedPolicyChooserPolicies" and the following data
      | Select | Name                | Type   | Version | ID                                   |
      |        | AllowAll            | Policy |       0 | 3f380e42-0b21-4570-a312-91e1b80386fa |
      |        | DenyAll             | Policy |       0 | c9c14750-b8f3-4507-bd3f-5c6562f0a6e6 |
      |        | aaaa                | Policy |       0 | c9bb52b7-3c97-4b58-be68-e466d4c3ea8f |
      |        | asdf                | Policy |       0 | e234c37f-f410-4f2c-a1d4-8444308bbf35 |
      |        | asdg                | Policy |       0 | 44ee268d-7bd5-4776-b7f1-bc5946282d53 |
      |        | asf_policy_332622a1 | Policy |       0 | aeaf2aac-f732-4370-9ac5-ead2636dc9c9 |
      |        | bcbcb               | Policy |       0 | c6e600d4-b929-4bd7-8e2b-0dd569f97d79 |
      |        | gdsag               | Policy |       0 | a76cc3a6-e47d-4dd8-991c-3a1b7a47e996 |
      |        | qwe                 | Policy |       0 | 42a16658-cc04-48af-8546-82704e6e1166 |
      |        | sdvv                | Policy |       0 | 7de53276-70ad-4816-89fd-c502f880753d |
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    Given the following mocked requests:
      | Method | Path                                                               | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access* |    200 | access_policies_page_2.json |
    #testing pagination dialog
    When I click the 1st button with ID "next"
    Then I should see a "td" with the text "gdsagaa"
    And I should see a table with ID "paginatedPolicyChooserPolicies" and the following data
      | Select | Name                | Type   | ID | Version                              |
      |        | DenyAll             | Policy |  0 | c9c14750-b8f3-4507-bd3f-5c6562f0a6e6 |
      |        | aaaa                | Policy |  0 | c9bb52b7-3c97-4b58-be68-e466d4c3ea8f |
      |        | asdf                | Policy |  0 | e234c37f-f410-4f2c-a1d4-8444308bbf35 |
      |        | asf_policy_332622a1 | Policy |  0 | aeaf2aac-f732-4370-9ac5-ead2636dc9c9 |
      |        | bcbcb               | Policy |  0 | c6e600d4-b929-4bd7-8e2b-0dd569f97d79 |
      |        | gdsagaa             | Policy |  0 | a76cc3a6-e47d-4dd8-991c-3a1b7a47e996 |
    Given the following mocked requests:
      | Method | Path                                                               | Status | Body                        |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/policies/access* |    200 | access_policies_page_1.json |
    #testing pagination dialog
    When I click the 1st button with ID "prev"
    Then I should see a "td" with the text "AllowAll"
    And I should see a table with ID "paginatedPolicyChooserPolicies" and the following data
      | Select | Name                | Type   | Version | ID                                   |
      |        | AllowAll            | Policy |       0 | 3f380e42-0b21-4570-a312-91e1b80386fa |
      |        | DenyAll             | Policy |       0 | c9c14750-b8f3-4507-bd3f-5c6562f0a6e6 |
      |        | aaaa                | Policy |       0 | c9bb52b7-3c97-4b58-be68-e466d4c3ea8f |
      |        | asdf                | Policy |       0 | e234c37f-f410-4f2c-a1d4-8444308bbf35 |
      |        | asdg                | Policy |       0 | 44ee268d-7bd5-4776-b7f1-bc5946282d53 |
      |        | asf_policy_332622a1 | Policy |       0 | aeaf2aac-f732-4370-9ac5-ead2636dc9c9 |
      |        | bcbcb               | Policy |       0 | c6e600d4-b929-4bd7-8e2b-0dd569f97d79 |
      |        | gdsag               | Policy |       0 | a76cc3a6-e47d-4dd8-991c-3a1b7a47e996 |
      |        | qwe                 | Policy |       0 | 42a16658-cc04-48af-8546-82704e6e1166 |
      |        | sdvv                | Policy |       0 | 7de53276-70ad-4816-89fd-c502f880753d |
    When I click the radio input in row 1 of the table with ID "paginatedPolicyChooserPolicies"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name | Parameters | Delete     |
      | Where | AllowAll    |            | Delete Bin |
    Given a mocked "GET" request for "access_policies"
    When I click the button labeled "Add Policy"
    # We don't need to test for loading state (I think because of browser caching KSJ)
    # this is the first row in the table:
    Then I should see a "td" with the text "Allow_all"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    And I should see a table with ID "paginatedPolicyChooserPolicies" and the following data
      | Select | Name           | Type   | Version | ID                                   |
      |        | Allow_all      | Policy |       0 | 0c0b7371-5175-405b-a17c-fec5969914b8 |
      |        | Dont_Allow_Any | Policy |       1 | 00000000-5175-405b-a17c-fec5969914b8 |
    When I click the radio input in row 2 of the table with ID "paginatedPolicyChooserPolicies"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name    | Parameters | Delete     |
      | Where | AllowAll       |            | Delete Bin |
      | AND   | Dont_Allow_Any |            | Delete Bin |
    When I click the "delete" button in row 2 of the table with ID "accessorManualPolicyComponents"
    Then I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name | Parameters | Delete     |
      | Where | AllowAll    |            | Delete Bin |
    Given a mocked "GET" request for "access_policy_templates"
    When I click the button labeled "Add Template"
    Then I should see a "p" with the text "Loading templates..."
    # this is the first row in the table:
    Then I should see a "td" with the text "CheckAttribute"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    And I should see a table with ID "paginatedPolicyChooserTemplates" and the following data
      | Select | Name              | Type     | Version | ID                                   |
      |        | hellopolicy4      | Template |       0 | 010f6071-8167-4767-a7f8-4e3615f05f15 |
      |        | AllowAll          | Template |       0 | 1e742248-fdde-4c88-9ea7-2c2106ec7aa8 |
      |        | jsljkdfkjnsdajnfl | Template |       0 | 23a4909c-8c1a-450e-8f9b-abc90e9771d8 |
      |        | hefa              | Template |       0 | 23e84792-3971-4d08-80da-729b608f0b01 |
      |        | testwkyle         | Template |       0 | 3053a10b-49a9-4da0-a392-fb9228e98c00 |
      |        | ksdfjasdfasdf     | Template |       0 | 61851da2-810c-4bf9-9656-583d94ba7d5f |
      |        | a                 | Template |       0 | a3cde40f-1cbc-491a-b423-826f5041b858 |
      |        | hellopolicy2      | Template |       0 | b0a4c61b-d2e6-4a20-a9a6-2e89fe7f5f16 |
      |        | somethinga        | Template |       0 | cafd385e-8b2b-49df-bbc7-69e55a163fe1 |
      |        | hellopolicy1      | Template |       0 | cc3051ef-1dd2-4aef-bd40-fd046f8a3cd5 |
      |        | aslkjdfhaslkjdhf  | Template |       0 | d4a48099-3850-4b44-b282-8dc3d843a762 |
      |        | tyler1            | Template |       0 | db97e49f-806e-473d-9ad3-b0d5303e2263 |
      |        | hellopolicy3      | Template |       0 | e123f616-e546-4686-9009-d13f6908c721 |
      |        | henlo             | Template |       0 | e7e7a8bc-1e68-4ee0-84cf-e373a1f19bd8 |
      |        | atemp             | Template |       0 | 95235dfc-ba40-4828-a923-6ac7e616d281 |
      |        | checkIfEven       | Template |       0 | aa412fd1-7c82-4b54-9ffd-b50f589642c6 |
      |        | CheckAttribute    | Template |       0 | aad2bf25-311f-467e-9169-a6a89b6d34a6 |
    When I click the radio input in row 3 of the table with ID "paginatedPolicyChooserTemplates"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name       | Parameters | Delete     |
      | Where | AllowAll          |            | Delete Bin |
      | AND   | jsljkdfkjnsdajnfl |            | Delete Bin |
    # save the new policy
    Given a mocked "POST" request for "accessors"
    # user is redirected to details page
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "created accessor details"
    When I click the button labeled "Create Accessor"
    Then the page should have no accessibility violations
    Then I should see the following text on the page
      | TagName            | TextContent        |
      | label              | Name               |
      | label > div > p    | Our_Accessor       |
      | label              | Version            |
      | label > div > p    |                  0 |
      | label              | ID                 |
      | label > div > span |            2ee449… |
      | label              | Description        |
      | p                  | foo                |
      | h2                 | Purposes of access |
      | button             | Edit Accessor      |
    And I should see a table with ID "purposes" and the following data
      |       | Purpose     | Description |
      | Where | operational |             |
    And I should be on the page with the path "/accessors/2ee4497e-c326-4068-94ed-3dcdaaaa53bc/latest"
    And I should see a toast notification with the text "Successfully created accessorClose"

  Scenario: Error creating accessor
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    When I navigate to the page with path "/accessors/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent |
    # input name and description
    # input name and description
    When I replace the text in the "accessor_name" field with "Our_Accessor"
    And I replace the text in the "accessor_description" field with "foo"
    And I select the option labeled "operational" in the dropdown matching selector "[name='accessor_purpose']"
    # choose a column to add
    When I select the option labeled "foo_column" in the dropdown matching selector "#selectUserStoreColumnToAdd"
    Then I should see a table with ID "accessorColumns" and the following data
      | Name       | Type   | Transformer          |
      | foo_column | string | Select a transformer |
    When I select "Passthrough" from the dropdown in column 3 of row 1 of the table with ID "accessorColumns"
    Then I should see a table with ID "accessorColumns" and the following data
      | Name       | Type   | Transformer |
      | foo_column | string | Passthrough |
    And I should see a dropdown matching selector "#selectUserStoreColumnToAdd" with the following options
      | Text            | Value      | Selected |
      | Select a column |            | true     |
      | bar_column      | bar_column |          |
      | baz_column      | baz_column |          |
    # choose another column to add
    When I select the option labeled "bar_column" in the dropdown matching selector "#selectUserStoreColumnToAdd"
    Then I should see a table with ID "accessorColumns" and the following data
      | Name       | Type      | Transformer          |
      | foo_column | string    | Passthrough          |
      | bar_column | timestamp | Select a transformer |
    And I should see a dropdown matching selector "#selectUserStoreColumnToAdd" with the following options
      | Text            | Value      | Selected |
      | Select a column |            | true     |
      | baz_column      | baz_column |          |
    # remove a just-added column
    When I click the "delete" button in row 2 of the table with ID "accessorColumns"
    Then I should see a table with ID "accessorColumns" and the following data
      | Name       | Type   | Transformer |
      | foo_column | string | Passthrough |
    And I should see a dropdown matching selector "#selectUserStoreColumnToAdd" with the following options
      | Text            | Value      | Selected |
      | Select a column |            | true     |
      | bar_column      | bar_column |          |
      | baz_column      | baz_column |          |
    # edit selector config
    When I replace the text in the "accessor_selector_config" field with "{id} LIKE ?"
    Then I should see the following form elements
      | TagName | Type | Name                     | Value       |
      | input   | text | accessor_selector_config | {id} LIKE ? |
    Given a mocked "GET" request for "access_policies"
    When I click the button labeled "Add Policy"
    Then I should see a "p" with the text "Loading policies..."
    # this is the first row in the table:
    Then I should see a "td" with the text "Allow_all"
    And I should see a button labeled "Save selection"
    And the button labeled "Save selection" should be disabled
    And I should see a table with ID "paginatedPolicyChooserPolicies" and the following data
      | Select | Name           | Type   | Version | ID                                   |
      |        | Allow_all      | Policy |       0 | 0c0b7371-5175-405b-a17c-fec5969914b8 |
      |        | Dont_Allow_Any | Policy |       1 | 00000000-5175-405b-a17c-fec5969914b8 |
    When I click the radio input in row 1 of the table with ID "paginatedPolicyChooserPolicies"
    Then the button labeled "Save selection" should be enabled
    When I click the button labeled "Save selection"
    Then I should see a table with ID "accessorManualPolicyComponents" and the following data
      |       | Policy name | Parameters | Delete     |
      | Where | Allow_all   |            | Delete Bin |
    Given the following mocked requests:
      | Method | Path                                                                  | Status | Body                                                                  |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/accessors |    400 | {"error":"uh-oh","request_id":"9a0f0b22-dabf-40b2-8f82-4a2caab9e605"} |
    When I click the button labeled "Create Accessor"
    Then I should see the following text on the page
      | TagName | TextContent |
      | p       | uh-oh       |
    And I should be on the page with the path "/accessors/create"

  Scenario: Cancel accessor creation
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "access_policies"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "purposes"
    And a mocked "GET" request for "access_policy"
    When I navigate to the page with path "/accessors/create?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see the following text on the page
      | TagName | TextContent     |
      | h2      | Basic Details   |
      | h2      | Columns         |
      | td      | No columns      |
      | h2      | Selector        |
      | h2      | Policy          |
      | button  | Add Policy      |
      | button  | Create Accessor |
      | button  | Cancel          |
    And I should see the following form elements
      | TagName  | Type | Name                     | Value         |
      | input    | text | accessor_name            |               |
      | textarea |      | accessor_description     |               |
      | select   |      | select_column            |               |
      | input    | text | accessor_selector_config | {id} = ANY(?) |
    And I should see a button labeled "Add Policy"
    And I should see a button labeled "Add Template"
    # input name and description
    When I replace the text in the "accessor_name" field with "Our_Accessor"
    And I replace the text in the "accessor_description" field with "foo"
    Given I intend to accept the confirm dialog
    Given a mocked "GET" request for "tenants"
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "accessors"
    And a mocked "GET" request for "transformers"
    And a mocked "GET" request for "mutators"
    And a mocked "GET" request for "user store schema edit"
    And a mocked "GET" request for "purposes"
    When I click the button labeled "Cancel"
    Then I should see the following text on the page
      | TagName | TextContent |
      | td      | baz_column  |
    And I should be on the page with the path "/accessors"
