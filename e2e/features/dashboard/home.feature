@authenticated
Feature: Homepage
  A signed-in user lands on the module launcher.

  Scenario: An authenticated user sees the homepage
    When I open "/"
    Then I am on "/"
    And I see the homepage
