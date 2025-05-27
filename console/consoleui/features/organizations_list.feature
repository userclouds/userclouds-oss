@organizations
Feature: organizations index page

  @a11y
  Scenario: organizations list accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                             | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations* | 200    | organizations_page_1.json |
    When I navigate to the page with path "/organizations?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a&limit=5"
    Then the page should have no accessibility violations

  Scenario: organizations list
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And the following mocked requests:
      | Method | Path                                                             | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations* | 200    | organizations_page_1.json |
    When I navigate to the page with path "/organizations?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a&limit=5"
    Then I should see a "p" with the text "Loading ..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a link to "/organizations/7def2e9f-0c6a-489a-8549-4a673ad001e9?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    # On the first page (no previous page)
    And I should see a table with ID "organizations" and the following data
      | Organization name            | Region        | Created               | ID           |
      | UserClouds Dev               | aws-us-east-1 | 2/4/2023, 9:42:17 PM  | Copy 1ee449… |
      | Hobart and William and Smith | aws-us-west-2 | 2/15/2023, 3:22:02 PM | Copy 508620… |
      | Test Test Test               | aws-us-east-1 | 2/15/2023, 3:22:02 PM | Copy 5ded72… |
      | Buzz Org                     | aws-us-east-1 | 2/15/2023, 3:22:02 PM | Copy 79b600… |
      | Foo                          | aws-us-west-2 | 3/20/2023, 3:00:01 PM | Copy 7def2e… |
    And the button with title "View previous page" should be disabled
    And the button with title "View next page" should be enabled
    And I should see the following text on the page
      | TagName               | TextContent |
      | button#prev[disabled] |             |
    # Navigate to page 2 (previous and next pages)
    Given the following mocked requests:
      | Method | Path                                                             | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations* | 200    | organizations_page_2.json |
    When I click the button with ID "next"
    Then the button with title "View next page" should be enabled
    Then the button with title "View previous page" should be enabled
    And I should see a link to "/organizations/7e88776c-7f90-4bc6-994e-7eb09e812c51?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "organizations" and the following data
      | Organization name    | Region        | Created               | ID           |
      | Bar Org              | aws-us-west-2 | 2/15/2023, 3:22:02 PM | Copy 7e8877… |
      | FizzBuzz Org         | aws-us-west-2 | 2/15/2023, 3:22:02 PM | Copy 87c88c… |
      | Foo Org              | aws-us-west-2 | 2/15/2023, 3:22:02 PM | Copy a353a1… |
      | The Gates Foundation | aws-us-west-2 | 2/15/2023, 3:22:02 PM | Copy a50f8b… |
      | Test Org             | aws-us-east-1 | 2/15/2023, 3:22:02 PM | Copy b34083… |
    # Navigate to page 3 (no next page)
    Given the following mocked requests:
      | Method | Path                                                             | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations* | 200    | organizations_page_3.json |
    When I click the button with ID "next"
    Then the button with title "View next page" should be disabled
    And the button with title "View previous page" should be enabled
    And I should see a link to "/organizations/c586e8a5-4a72-419a-a1f9-20186d36a9fa?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "organizations" and the following data
      | Organization name | Region        | Created               | ID           |
      | Baz Org           | aws-us-east-1 | 2/15/2023, 3:22:02 PM | Copy c586e8… |
      | Dino's Pizza      | aws-us-west-2 | 2/15/2023, 3:22:03 PM | Copy cffc2e… |
      | La Bodega         | aws-us-west-2 | 2/15/2023, 3:22:03 PM | Copy d4abe8… |
      | Gold Bar          | aws-us-west-2 | 2/15/2023, 3:22:03 PM | Copy d81646… |
      | Moe's Bar         | aws-us-east-1 | 2/15/2023, 3:22:03 PM | Copy e66465… |
    # Go back to page 2
    Given the following mocked requests:
      | Method | Path                                                             | Status | Body                      |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/organizations* | 200    | organizations_page_2.json |
    When I click the button with ID "prev"
    Then the button with title "View next page" should be enabled
    And the button with title "View previous page" should be enabled
    And I should see a link to "/organizations/7e88776c-7f90-4bc6-994e-7eb09e812c51?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a table with ID "organizations" and the following data
      | Organization name    | Region        | Created               | ID           |
      | Bar Org              | aws-us-west-2 | 2/15/2023, 3:22:02 PM | Copy 7e8877… |
      | FizzBuzz Org         | aws-us-west-2 | 2/15/2023, 3:22:02 PM | Copy 87c88c… |
      | Foo Org              | aws-us-west-2 | 2/15/2023, 3:22:02 PM | Copy a353a1… |
      | The Gates Foundation | aws-us-west-2 | 2/15/2023, 3:22:02 PM | Copy a50f8b… |
      | Test Org             | aws-us-east-1 | 2/15/2023, 3:22:02 PM | Copy b34083… |
