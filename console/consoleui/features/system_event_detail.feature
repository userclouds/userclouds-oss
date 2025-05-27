@systemevent
@syseventdetail
Feature: systemevent

  @a11y
  Scenario: system event accessibility
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "serviceinfo"
    And a mocked "GET" request for "userinfo"
    And the following mocked requests:
      | Method | Path                                                     | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/runs*  | 200    | system_logs_single.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/runs/* | 200    | system_events.json      |
    When I navigate to the page with path "/systemlog/7cafc9f2-05d5-4fe1-94e7-86c626b5c191?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page should have no accessibility violations

  Scenario: systemevent for run that doesn't exist
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "serviceinfo"
    And a mocked "GET" request for "userinfo"
    And the following mocked requests:
      | Method | Path                                                     | Status | Body |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/runs*  | 200    | []   |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/runs/* | 200    | []   |
    When I navigate to the page with path "/systemlog/7cafc9f2-05d5-4fe1-94e7-86c626b5c191?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then the page title should be "[dev] UserClouds Console"
    And I should see a card with the title "Summary"
    And I should see a "p" with the text "Entry not found"
    And I should see a card with the title "Sync Records"
    And I should see a "h2" with the text "No entries in the system log."

  Scenario: systemevent for run that exists with no details
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "serviceinfo"
    And a mocked "GET" request for "userinfo"
    And the following mocked requests:
      | Method | Path                                                     | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/runs*  | 200    | system_logs_single.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/runs/* | 200    | []                      |
    When I navigate to the page with path "/systemlog/7cafc9f2-05d5-4fe1-94e7-86c626b5c191?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading ..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a card with the title "Summary"
    And I should see the following text on the page
      | TagName         | TextContent                                         |
      | label           | Event Type                                          |
      | label > div > p | app                                                 |
      | label           | Event ID                                            |
      | label > div > p | 7cafc9f2-05d5-4fe1-94e7-86c626b5c191                |
      | label           | Created Date                                        |
      | label > div > p | 2023-06-12T23:33:24.400562Z                         |
      | label           | Event Summary                                       |
      | label > div > p | There were 7 total records. 0 failures. 0 warnings. |
    And I should see a card with the title "Sync Records"
    And I should see a "h2" with the text "No entries in the system log."

  Scenario: systemevent for run that exists with details
    Given I am a logged-in user
    And a mocked "GET" request for "tenants"
    And a mocked "GET" request for "selectedTenant"
    And a mocked "GET" request for "serviceinfo"
    And a mocked "GET" request for "userinfo"
    And the following mocked requests:
      | Method | Path                                                     | Status | Body                    |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/runs*  | 200    | system_logs_single.json |
      | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/runs/* | 200    | system_events.json      |
    When I navigate to the page with path "/systemlog/7cafc9f2-05d5-4fe1-94e7-86c626b5c191?company_id=1ee4497e-c326-4068-94ed-3dcdaaaa53bc&tenant_id=41ab79a8-0dff-418e-9d42-e1694469120a"
    Then I should see a "p" with the text "Loading ..."
    And the page title should be "[dev] UserClouds Console"
    Then I should see a card with the title "Summary"
    And I should see the following text on the page
      | TagName         | TextContent                                         |
      | label           | Event Type                                          |
      | label > div > p | app                                                 |
      | label           | Event ID                                            |
      | label > div > p | 7cafc9f2-05d5-4fe1-94e7-86c626b5c191                |
      | label           | Created Date                                        |
      | label > div > p | 2023-06-12T23:33:24.400562Z                         |
      | label           | Event Summary                                       |
      | label > div > p | There were 7 total records. 0 failures. 0 warnings. |
    And I should see a card with the title "Sync Records"
    And I should see a table within the "Sync Records" card with the following data
      | Object ID                        | Event ID                             | User ID | Created               | Warning | Error |
      | LNQaXUqpcsWt4jHFn5lf7q0D6SKPRk1m | 515330a6-91ec-4a4b-9132-55ee28012543 |         | 6/12/2023, 4:33:24 PM |         |       |
      | 9dzd3Y2ydcBjOgSOO3m2mfft0vng0y6i | 66e51069-255e-4a71-bebd-0507a3e590f9 |         | 6/12/2023, 4:33:24 PM |         |       |
      | 2u0NpFen6RcxQj7MN5E1bLObU9zbh9EZ | 77a1f7ec-653d-4098-ad2b-da56d78cfd30 |         | 6/12/2023, 4:33:24 PM |         |       |
      | blQIL2lhLaw4CGl5ZpJNUw7WjXtUNz0C | 82e41602-b975-436c-972b-13afbc29ebf1 |         | 6/12/2023, 4:33:25 PM |         |       |
      | Py9Li5X8TJEsGzhD78hJAurkaGvabxgy | 9f67378a-cc8a-4734-9db5-921fd2b83d07 |         | 6/12/2023, 4:33:25 PM |         |       |
      | uqARTnY25kGrChzYGkYZXwM7zHwAcV8d | dd16b9cb-5f44-439e-8c20-fd28a55c40ac |         | 6/12/2023, 4:33:24 PM |         |       |
      | jVLPrESkrVm0HxyBKvxQVuQAelrRXn2i | e06bec75-d860-4fa8-9a2f-ca295a941889 |         | 6/12/2023, 4:33:25 PM |         |       |
