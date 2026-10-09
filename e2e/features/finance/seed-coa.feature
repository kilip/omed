@seed-coa
Feature: Seed Chart of Accounts

  Background:
    Given I am signed in

  Scenario: Onboarding empty state when no accounts exist
    When I open "/fin"
    Then I should see the setup chart of accounts card
    When I click setup chart of accounts
    Then I should be on "/fin/seed-coa"
    And I should see the heading "Seed Chart of Accounts"

  Scenario: Preview accounts and search filtering
    When I open "/fin/seed-coa"
    Then I should see the account preview table
    And I should see the account row with code "1000"
    When I search preview accounts for "Assets"
    Then I should see the account row with code "1000"

  Scenario: Successfully seed chart of accounts and view accounts via menu
    When I open "/fin/seed-coa"
    Then I should see the account preview table
    When I click seed chart of accounts
    Then I should see the chart of accounts seeded successfully
    When I click go to finance
    Then I should be on "/fin"
    And I should see the view accounts menu card
    When I click view accounts
    Then I should be on "/fin/accounts"
    And I should see the heading "Chart of Accounts"
    And I should see the account row with code "1000"


