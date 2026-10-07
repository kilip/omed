Feature: Language switcher

  Scenario: Switch language on the login page
    Given I am not signed in
    When I open "/login"
    And I switch the language to "Bahasa Indonesia"
    Then I should see the heading "Masuk ke Omed"

  Scenario: Language preference is saved to my account
    Given I am signed in
    When I open "/home"
    And I switch the language to "Bahasa Indonesia"
    And I open "/settings" in a fresh browser session
    Then I should see the heading "Pengaturan"

