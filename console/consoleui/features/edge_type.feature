@authz
@edge_type
Feature: edge type page

  @a11y
  Scenario: edge type accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
    When I navigate to the page with path "/edgetypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: create edge type
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
    When I navigate to the page with path "/edgetypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a card with the title "Edge types"
    When I click the button labeled "Create Edge Type"
    Then I should see a card with the title "Edge Type"
    And I should see a dropdown matching selector "[name='source_object_type']" with the following options
      | Text                    | Value                   | Selected |
      | _user                   | _user                   | true     |
      | _access_policy_template | _access_policy_template |          |
      | _transformer            | _transformer            |          |
      | _login_app              | _login_app              |          |
      | _access_policy          | _access_policy          |          |
      | _policies               | _policies               |          |
      | _group                  | _group                  |          |
    And I should see a dropdown matching selector "[name='target_object_type']" with the following options
      | Text                    | Value                   | Selected |
      | _user                   | _user                   | true     |
      | _access_policy_template | _access_policy_template |          |
      | _transformer            | _transformer            |          |
      | _login_app              | _login_app              |          |
      | _access_policy          | _access_policy          |          |
      | _policies               | _policies               |          |
      | _group                  | _group                  |          |
    And the button labeled "Add Attribute" should be enabled
    And the button labeled "Create Edge Type" should be disabled
    When I type "new edge type" into the input with ID "name"
    Then the button labeled "Create Edge Type" should be enabled
    Given the following mocked requests:
      | Method | Path                                                               | Status | Body                           |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes* | 200    | authz_created_edge_type_1.json |
    When I click the button labeled "Create Edge Type"
    Then I should see a card with the title "Edge types"
    And I should see a toast notification with the text "Successfully created edge typeClose"
    And I should be on the page with the path "/edgetypes/900474aa-13ee-481a-8ea0-0f0f4374364b/"

  Scenario: create edge type with attributes
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
    When I navigate to the page with path "/edgetypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    When I click the button labeled "Create Edge Type"
    Then I should see a card with the title "Edge Type"
    And the button labeled "Add Attribute" should be enabled
    And the button labeled "Create Edge Type" should be disabled
    And I should see a dropdown matching selector "[name='source_object_type']" with the following options
      | Text                    | Value                   | Selected |
      | _user                   | _user                   | true     |
      | _access_policy_template | _access_policy_template |          |
      | _transformer            | _transformer            |          |
      | _login_app              | _login_app              |          |
      | _access_policy          | _access_policy          |          |
      | _policies               | _policies               |          |
      | _group                  | _group                  |          |
    And I should see a dropdown matching selector "[name='target_object_type']" with the following options
      | Text                    | Value                   | Selected |
      | _user                   | _user                   | true     |
      | _access_policy_template | _access_policy_template |          |
      | _transformer            | _transformer            |          |
      | _login_app              | _login_app              |          |
      | _access_policy          | _access_policy          |          |
      | _policies               | _policies               |          |
      | _group                  | _group                  |          |
    When I type "new edge type" into the input with ID "name"
    And I click the button labeled "Add Attribute"
    And I click the button labeled "Add Attribute"
    Then I should see a dropdown matching selector "[name='attribute_flavor_0']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    | true     |
      | Inherit   | Inherit   |          |
      | Propagate | Propagate |          |
    And I should see a dropdown matching selector "[name='attribute_flavor_1']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    | true     |
      | Inherit   | Inherit   |          |
      | Propagate | Propagate |          |
    When I type "attribute 0" into the input with ID "attributeName0"
    And I select the option labeled "Inherit" in the dropdown matching selector "[name='attribute_flavor_0']"
    And I type "attribute 1" into the input with ID "attributeName1"
    And I select the option labeled "_group" in the dropdown matching selector "[name='source_object_type']"
    Then I should see a dropdown matching selector "[name='attribute_flavor_0']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    |          |
      | Inherit   | Inherit   | true     |
      | Propagate | Propagate |          |
    And I should see a dropdown matching selector "[name='attribute_flavor_1']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    | true     |
      | Inherit   | Inherit   |          |
      | Propagate | Propagate |          |
    And I should see a dropdown matching selector "[name='source_object_type']" with the following options
      | Text                    | Value                   | Selected |
      | _user                   | _user                   |          |
      | _access_policy_template | _access_policy_template |          |
      | _transformer            | _transformer            |          |
      | _login_app              | _login_app              |          |
      | _access_policy          | _access_policy          |          |
      | _policies               | _policies               |          |
      | _group                  | _group                  | true     |
    And the button labeled "Create Edge Type" should be enabled
    Given the following mocked requests:
      | Method | Path                                                               | Status | Body                           |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes* | 200    | authz_created_edge_type_2.json |
    When I click the button labeled "Create Edge Type"
    Then I should see a card with the title "Edge types"
    And I should see a toast notification with the text "Successfully created edge typeClose"
    And I should be on the page with the path "/edgetypes/daa5334c-a2c6-4a71-b5b1-a40c844797f2/"

  Scenario: create edge type with attributes error
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
    When I navigate to the page with path "/edgetypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    When I click the button labeled "Create Edge Type"
    Then I should see a card with the title "Edge Type"
    And the button labeled "Add Attribute" should be enabled
    And the button labeled "Create Edge Type" should be disabled
    And I should see a dropdown matching selector "[name='source_object_type']" with the following options
      | Text                    | Value                   | Selected |
      | _user                   | _user                   | true     |
      | _access_policy_template | _access_policy_template |          |
      | _transformer            | _transformer            |          |
      | _login_app              | _login_app              |          |
      | _access_policy          | _access_policy          |          |
      | _policies               | _policies               |          |
      | _group                  | _group                  |          |
    And I should see a dropdown matching selector "[name='target_object_type']" with the following options
      | Text                    | Value                   | Selected |
      | _user                   | _user                   | true     |
      | _access_policy_template | _access_policy_template |          |
      | _transformer            | _transformer            |          |
      | _login_app              | _login_app              |          |
      | _access_policy          | _access_policy          |          |
      | _policies               | _policies               |          |
      | _group                  | _group                  |          |
    When I type "new edge type" into the input with ID "name"
    And I click the button labeled "Add Attribute"
    And I click the button labeled "Add Attribute"
    Then I should see a dropdown matching selector "[name='attribute_flavor_0']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    | true     |
      | Inherit   | Inherit   |          |
      | Propagate | Propagate |          |
    And I should see a dropdown matching selector "[name='attribute_flavor_1']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    | true     |
      | Inherit   | Inherit   |          |
      | Propagate | Propagate |          |
    When I type "attribute 0" into the input with ID "attributeName0"
    And I select the option labeled "Inherit" in the dropdown matching selector "[name='attribute_flavor_0']"
    And I type "attribute 1" into the input with ID "attributeName1"
    And I select the option labeled "_group" in the dropdown matching selector "[name='source_object_type']"
    Then I should see a dropdown matching selector "[name='attribute_flavor_0']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    |          |
      | Inherit   | Inherit   | true     |
      | Propagate | Propagate |          |
    And I should see a dropdown matching selector "[name='attribute_flavor_1']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    | true     |
      | Inherit   | Inherit   |          |
      | Propagate | Propagate |          |
    And I should see a dropdown matching selector "[name='source_object_type']" with the following options
      | Text                    | Value                   | Selected |
      | _user                   | _user                   |          |
      | _access_policy_template | _access_policy_template |          |
      | _transformer            | _transformer            |          |
      | _login_app              | _login_app              |          |
      | _access_policy          | _access_policy          |          |
      | _policies               | _policies               |          |
      | _group                  | _group                  | true     |
    And the button labeled "Create Edge Type" should be enabled
    Given the following mocked requests:
      | Method | Path                                                               | Status | Body                                       |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes* | 400    | {"error": "Attribute.Name can't be empty"} |
    When I click the button labeled "Create Edge Type"
    Then I should see a "p" with the text "error creating edge type: Attribute.Name can't be empty"

  Scenario: create edge type handle deleted attributes correctly
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_1.json |
    When I navigate to the page with path "/edgetypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    When I click the button labeled "Create Edge Type"
    Then I should see a card with the title "Edge Type"
    And the button labeled "Add Attribute" should be enabled
    And the button labeled "Create Edge Type" should be disabled
    And I should see a dropdown matching selector "[name='source_object_type']" with the following options
      | Text                    | Value                   | Selected |
      | _user                   | _user                   | true     |
      | _access_policy_template | _access_policy_template |          |
      | _transformer            | _transformer            |          |
      | _login_app              | _login_app              |          |
      | _access_policy          | _access_policy          |          |
      | _policies               | _policies               |          |
      | _group                  | _group                  |          |
    And I should see a dropdown matching selector "[name='target_object_type']" with the following options
      | Text                    | Value                   | Selected |
      | _user                   | _user                   | true     |
      | _access_policy_template | _access_policy_template |          |
      | _transformer            | _transformer            |          |
      | _login_app              | _login_app              |          |
      | _access_policy          | _access_policy          |          |
      | _policies               | _policies               |          |
      | _group                  | _group                  |          |
    When I type "new edge type" into the input with ID "name"
    And I click the button labeled "Add Attribute"
    And I click the button labeled "Add Attribute"
    Then I should see a dropdown matching selector "[name='attribute_flavor_0']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    | true     |
      | Inherit   | Inherit   |          |
      | Propagate | Propagate |          |
    And I should see a dropdown matching selector "[name='attribute_flavor_1']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    | true     |
      | Inherit   | Inherit   |          |
      | Propagate | Propagate |          |
    When I type "attribute 0" into the input with ID "attributeName0"
    And I select the option labeled "Inherit" in the dropdown matching selector "[name='attribute_flavor_0']"
    And I type "attribute 1" into the input with ID "attributeName1"
    And I select the option labeled "_group" in the dropdown matching selector "[name='source_object_type']"
    Then I should see a dropdown matching selector "[name='attribute_flavor_0']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    |          |
      | Inherit   | Inherit   | true     |
      | Propagate | Propagate |          |
    And I should see a dropdown matching selector "[name='attribute_flavor_1']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    | true     |
      | Inherit   | Inherit   |          |
      | Propagate | Propagate |          |
    And I should see a dropdown matching selector "[name='source_object_type']" with the following options
      | Text                    | Value                   | Selected |
      | _user                   | _user                   |          |
      | _access_policy_template | _access_policy_template |          |
      | _transformer            | _transformer            |          |
      | _login_app              | _login_app              |          |
      | _access_policy          | _access_policy          |          |
      | _policies               | _policies               |          |
      | _group                  | _group                  | true     |
    And the button labeled "Create Edge Type" should be enabled
    When I click the button labeled "Add Attribute"
    Then I should see a dropdown matching selector "[name='attribute_flavor_2']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    | true     |
      | Inherit   | Inherit   |          |
      | Propagate | Propagate |          |
    And I should see an element matching selector "#attributeName2"
    When I click the "delete" button in row 2 of the table with ID "attributes"
    Then I should not see an element matching selector "#attributeName2"
    Given the following mocked requests:
      | Method | Path                                                               | Status | Body                           |
      | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes* | 200    | authz_created_edge_type_2.json |
    When I click the button labeled "Create Edge Type"
    Then I should see a card with the title "Edge types"
    And I should see a toast notification with the text "Successfully created edge typeClose"
    And I should be on the page with the path "/edgetypes/daa5334c-a2c6-4a71-b5b1-a40c844797f2/"

  Scenario: edge type detail displays correctly
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_2.json |
    When I navigate to the page with path "/edgetypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a link to "/edgetypes/daa5334c-a2c6-4a71-b5b1-a40c844797f2?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                           |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes/daa5334c-a2c6-4a71-b5b1-a40c844797f2 | 200    | authz_created_edge_type_2.json |
    When I click the link with the href "/edgetypes/daa5334c-a2c6-4a71-b5b1-a40c844797f2?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching Edge Type..."
    And I should see a card with the title "Edge Type"
    And I should see a cardrow with the title "Basic Details"
    And I should see a cardrow with the title "Attributes"
    And I should see the following text on the page
      | TagName            | TextContent        |
      | label              | Name               |
      | label > div > p    | new edge type      |
      | label              | ID                 |
      | label > div > span | daa533…            |
      | label              | Source Object Type |
      | label > div > p    | _group             |
      | label              | Target Object Type |
      | label > div > p    | _user              |
    And I should see a table with ID "attributes" and the following data
      | Attribute Name | Flavor  |
      | attribute0     | Inherit |
      | attribute1     | Direct  |
    And the button labeled "Edit Edge Type" should be enabled
    And I should not see a button labeled "Add Attribute"
    And I should not see a button labeled "Save Edge Type"
    And I should not see a button labeled "Cancel"

  @a11y
  Scenario: edge type detail allows for basic edits
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_2.json |
    When I navigate to the page with path "/edgetypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a link to "/edgetypes/daa5334c-a2c6-4a71-b5b1-a40c844797f2?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                           |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes/daa5334c-a2c6-4a71-b5b1-a40c844797f2 | 200    | authz_created_edge_type_2.json |
    When I click the link with the href "/edgetypes/daa5334c-a2c6-4a71-b5b1-a40c844797f2?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching Edge Type..."
    And I should see a card with the title "Edge Type"
    And I should see the following text on the page
      | TagName            | TextContent        |
      | label              | Name               |
      | label > div > p    | new edge type      |
      | label              | ID                 |
      | label > div > span | daa533…            |
      | label              | Source Object Type |
      | label > div > p    | _group             |
      | label              | Target Object Type |
      | label > div > p    | _user              |
    And I should see a table with ID "attributes" and the following data
      | Attribute Name | Flavor  |
      | attribute0     | Inherit |
      | attribute1     | Direct  |
    And the button labeled "Edit Edge Type" should be enabled
    And I should not see a button labeled "Add Attribute"
    And I should not see a button labeled "Save Edge Type"
    And I should not see a button labeled "Cancel"
    When I click the button labeled "Edit Edge Type"
    Then the page should have no accessibility violations
    And I should see a dropdown matching selector "[name='attribute_flavor_0']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    |          |
      | Inherit   | Inherit   | true     |
      | Propagate | Propagate |          |
    And I should see a dropdown matching selector "[name='attribute_flavor_1']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    | true     |
      | Inherit   | Inherit   |          |
      | Propagate | Propagate |          |
    When I click the button labeled "Add Attribute"
    Then I should see a dropdown matching selector "[name='attribute_flavor_2']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    | true     |
      | Inherit   | Inherit   |          |
      | Propagate | Propagate |          |
    And I should see an element matching selector "#attributeName2"
    When I click the "delete" button in row 3 of the table with ID "attributes"
    Then I should not see an element matching selector "#attributeName2"
    And the button labeled "Add Attribute" should be enabled
    And the button labeled "Save Edge Type" should be enabled
    And the button labeled "Cancel" should be enabled
    And I should not see a button labeled "Edit Edge Type"
    Given the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                                |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes/daa5334c-a2c6-4a71-b5b1-a40c844797f2 | 200    | authz_created_edge_type_2_edit.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*                                     | 200    | authz_edge_types_2.json             |
    And I click the button labeled "Save Edge Type"
    And I should see a card with the title "Edge Type"
    And I should see the following text on the page
      | TagName            | TextContent        |
      | label              | Name               |
      | label > div > p    | new edge type      |
      | label              | ID                 |
      | label > div > span | daa533…            |
      | label              | Source Object Type |
      | label > div > p    | _group             |
      | label              | Target Object Type |
      | label > div > p    | _user              |
    And I should see a table with ID "attributes" and the following data
      | Attribute Name | Flavor  |
      | attribute0     | Inherit |
      | attribute1     | Direct  |
    And the button labeled "Edit Edge Type" should be enabled
    And I should not see a button labeled "Add Attribute"
    And I should not see a button labeled "Save Edge Type"
    And I should not see a button labeled "Cancel"

  Scenario: edge type detail error
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                                 | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objects*     | 200    | authz_objects.json      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/objecttypes* | 200    | authz_object_types.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes*   | 200    | authz_edge_types_2.json |
    When I navigate to the page with path "/edgetypes?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a link to "/edgetypes/daa5334c-a2c6-4a71-b5b1-a40c844797f2?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Given the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                           |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes/daa5334c-a2c6-4a71-b5b1-a40c844797f2 | 200    | authz_created_edge_type_2.json |
    When I click the link with the href "/edgetypes/daa5334c-a2c6-4a71-b5b1-a40c844797f2?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Fetching Edge Type..."
    And I should see a card with the title "Edge Type"
    And I should see the following text on the page
      | TagName            | TextContent        |
      | label              | Name               |
      | label > div > p    | new edge type      |
      | label              | ID                 |
      | label > div > span | daa533…            |
      | label              | Source Object Type |
      | label > div > p    | _group             |
      | label              | Target Object Type |
      | label > div > p    | _user              |
    And I should see a table with ID "attributes" and the following data
      | Attribute Name | Flavor  |
      | attribute0     | Inherit |
      | attribute1     | Direct  |
    And the button labeled "Edit Edge Type" should be enabled
    And I should not see a button labeled "Add Attribute"
    And I should not see a button labeled "Save Edge Type"
    And I should not see a button labeled "Cancel"
    When I click the button labeled "Edit Edge Type"
    Then I should see a dropdown matching selector "[name='attribute_flavor_0']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    |          |
      | Inherit   | Inherit   | true     |
      | Propagate | Propagate |          |
    And I should see a dropdown matching selector "[name='attribute_flavor_1']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    | true     |
      | Inherit   | Inherit   |          |
      | Propagate | Propagate |          |
    When I click the button labeled "Add Attribute"
    Then I should see a dropdown matching selector "[name='attribute_flavor_2']" with the following options
      | Text      | Value     | Selected |
      | Direct    | Direct    | true     |
      | Inherit   | Inherit   |          |
      | Propagate | Propagate |          |
    And I should see an element matching selector "#attributeName2"
    When I click the "delete" button in row 3 of the table with ID "attributes"
    Then I should not see an element matching selector "#attributeName2"
    And the button labeled "Add Attribute" should be enabled
    And the button labeled "Save Edge Type" should be enabled
    And the button labeled "Cancel" should be enabled
    And I should not see a button labeled "Edit Edge Type"
    Given the following mocked requests:
      | Method | Path                                                                                                   | Status | Body                                       |
      | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/authz/edgetypes/daa5334c-a2c6-4a71-b5b1-a40c844797f2 | 400    | {"error": "Attribute.Name can't be empty"} |
    When I click the button labeled "Save Edge Type"
    Then I should see a "p" with the text "error updating edge type: Attribute.Name can't be empty"
