Feature: Dashboard Navigation and User Actions
  As an authenticated user
  I want to navigate through sidebar menu items and perform session actions
  So that I can use the dashboard features and log out safely

  Scenario: Navigate to Finance Accounts page
    Given I am logged in as "owner"
    When I navigate to "/finance/accounts"
    Then I should see the dashboard layout

  Scenario: Navigate to Settings page
    Given I am logged in as "admin"
    When I navigate to "/settings"
    Then I should see the dashboard layout

  Scenario: Navigate to Blog Articles page
    Given I am logged in as "member"
    When I navigate to "/blog/articles"
    Then I should see the dashboard layout
