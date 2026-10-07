Feature: Login page

  Scenario: Shows social providers
    Given I am not signed in
    When I open "/login"
    Then I should see the heading "Sign in to Omed"
    And I should see the button "Continue with Google"
    And I should see the button "Continue with GitHub"

