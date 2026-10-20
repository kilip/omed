@authenticated @preferences
Feature: User Preferences
  A signed-in user can change language preferences and have them persist.

  Scenario: Switching language updates the UI and persists the preference
    When I open "/"
    Then I am on "/"
    And I see the language is "EN"
    And I see the launcher title is "Apps"
    When I switch language to "Bahasa Indonesia"
    Then I see the language is "ID"
    And I see the launcher title is "Aplikasi"
    And my user preference locale is "id"
    When I reload the page
    Then I see the language is "ID"
    And I see the launcher title is "Aplikasi"
    When I switch language to "English"
    Then I see the language is "EN"
    And I see the launcher title is "Apps"
    And my user preference locale is "en"
