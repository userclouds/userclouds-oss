@user_details
Feature: User details page

  @a11y
  Scenario: User details basic info accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I navigate to the page with path "/users/029232cf-4c01-494a-b3ab-874173dcccbd?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: User details basic info
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    When I navigate to the page with path "/users/029232cf-4c01-494a-b3ab-874173dcccbd?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    Then I should see the following text on the page
      | TagName | TextContent      |
      | button  | Reveal User Data |
    And I should not see a cardrow with the title "Authentication Methods"
    And I should not see a cardrow with the title "MFA Methods"
    And I should not see a cardrow with the title "User Events"
    And I should not see a cardrow with the title "Admin"
    Given a mocked "GET" request for "user details"
    And a mocked "GET" request for "user events"
    And a mocked "GET" request for "user consented purposes"
    When I click the button labeled "Reveal User Data"
    Then I should see a "p" with the text "Fetching user..."
    And I should see a button labeled "Edit User Data"
    Then I should see a table with ID "userdata" and the following data
      | Column name    | Column value            | Consented purposes |
      | Email          | testuser@userclouds.com | operational        |
      | Email Verified | false                   | operational        |
      | Name           | Vlad Federov            | operational        |
      | Nickname       | Vlad the Lad            | operational        |
      | Picture        |                         | operational        |
    And I should see a cardrow with the title "Authentication Methods"
    And I should see a cardrow with the title "MFA Methods"
    And I should see a cardrow with the title "User Events"
    And I should not see a cardrow with the title "Admin"
