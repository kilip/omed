Feature: Dashboard home

  Background:
    Given I am signed in

  Scenario: Signed-in user lands on home
    When I open "/"
    Then I should be on "/home"
    And I should see the main navigation with "Home", "Finance", "Blog", "Settings"

  Scenario: Navigate via sidebar
    When I open "/home"
    And I click "Settings" in the main navigation
    Then I should be on "/settings"

