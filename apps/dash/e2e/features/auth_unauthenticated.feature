Feature: Unauthenticated Navigation & Protection
  As a guest user
  I want protected routes to redirect to login
  So that unauthorized access is prevented

  Scenario: Accessing protected dashboard home redirects to login
    Given I am an unauthenticated user
    When I navigate to protected page "/"
    Then I should be redirected to "/login"
    And I should see the login card with title "Login to Omed"
    And I should see social login buttons for Google and GitHub

  Scenario: Accessing protected finance route redirects to login
    Given I am an unauthenticated user
    When I navigate to protected page "/finance/accounts"
    Then I should be redirected to "/login"
    And I should see the login card with title "Login to Omed"
