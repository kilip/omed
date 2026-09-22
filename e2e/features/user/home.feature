Feature: User Homepage

Scenario: Access homepage as guest
  Given I go to "/"
  Then I should be on "/login"

Scenario: Access homepage as test user
  Given I have logged in as test user
  And I go to "/home"
  Then page title should contain "Home"
