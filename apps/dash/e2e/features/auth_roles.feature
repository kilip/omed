Feature: Multi-Role Authentication with Better-Auth TestUtils
  As a developer / QA engineer
  I want to test dashboard behavior under various roles (owner, admin, member, superadmin, user)
  So that permissions and workspace access are appropriately verified

  Scenario: Workspace Owner logs in and accesses dashboard
    Given I am logged in as "owner"
    When I navigate to "/"
    Then I should see the dashboard layout
    And I should see the user name on the page

  Scenario: Workspace Admin logs in
    Given I am logged in as "admin"
    When I navigate to "/"
    Then I should see the dashboard layout
    And I should see the user name on the page

  Scenario: Workspace Member logs in
    Given I am logged in as "member"
    When I navigate to "/"
    Then I should see the dashboard layout
    And I should see the user name on the page

  Scenario: Superadmin logs in with platform privileges
    Given I am logged in as "superadmin"
    When I navigate to "/"
    Then I should see the dashboard layout
    And I should see the user name on the page

  Scenario: Standard User logs in
    Given I am logged in as "user"
    When I navigate to "/"
    Then I should see the dashboard layout
    And I should see the user name on the page
