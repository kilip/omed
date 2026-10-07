Feature: Settings

  Background:
    Given I am signed in
    And I open "/settings"

  Scenario: Default currency is IDR for a new user
    Then the default currency should be "IDR"

  Scenario: Change default currency
    When I set the default currency to "USD"
    Then I should see the message "Default currency saved"
    When I open "/settings" in a fresh browser session
    Then the default currency should be "USD"

  Scenario: Change language from settings
    When I choose the language "Bahasa Indonesia"
    Then I should see the heading "Pengaturan"

  Scenario: Toggle dark theme
    When I choose the theme "Dark"
    Then the page should use the dark theme

