@systemlog
@syslog
Feature: systemlog

  Scenario: syslog list accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "serviceinfo"
    And a mocked "GET" request for "userinfo"
    And the following mocked requests:
      | Method | Path                                                    | Status | Body             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/runs* | 200    | system_logs.json |
    When I navigate to the page with path "/systemlog?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: syslog
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "serviceinfo"
    And a mocked "GET" request for "userinfo"
    And the following mocked requests:
      | Method | Path                                                    | Status | Body |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/runs* | 200    | []   |
    When I navigate to the page with path "/systemlog?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a card with the title "System Log"
    And I should see a "h2" with the text "No entries in the system log."

  Scenario: syslog list
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "serviceinfo"
    And a mocked "GET" request for "userinfo"
    And the following mocked requests:
      | Method | Path                                                    | Status | Body             |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/runs* | 200    | system_logs.json |
    When I navigate to the page with path "/systemlog?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading entries..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a link to "/systemlog/7cafc9f2-05d5-4fe1-94e7-86c626b5c191?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    And I should see a card with the title "System Log"
    And I should see a table with ID "syslog" and the following data
      | Type | Event ID                             | Created               | Total Records | Failed | Warnings |
      | app  | 7cafc9f2-05d5-4fe1-94e7-86c626b5c19c | 6/12/2023, 4:33:24 PM | 7             | 0      | 0        |
      | user | 7cafc9f2-05d5-4fe1-94e7-86c626b5c191 | 6/12/2023, 4:33:24 PM | 10            | 1      | 1        |
