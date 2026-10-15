Feature: Authentication
  Visitors without a session can only reach the login page.

  Scenario: An anonymous visitor is redirected to the login page
    When I open "/"
    Then I am on "/login"
    And I see the login page
