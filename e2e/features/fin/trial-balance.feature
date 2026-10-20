@authenticated @fin @trial-balance
Feature: Finance Trial Balance
  As a finance user
  I want to view the trial balance report as of a specified date
  So that I can verify accounting balance equality and ledger integrity

  Background:
    Given I have a personal workspace
    When I open "/fin"
    When I click the setup chart of accounts banner button
    Then I am on "/fin/setup"
    When I click the apply template button
    Then I am on "/fin/accounts"

  Scenario: View balanced trial balance with real posted entry data
    When I open "/fin/journal"
    And I click the new journal entry button
    Then I see the create entry drawer
    When I enter entry description "Trial Balance Sample Entry"
    And I add a debit line of 5000000 for account "1111"
    And I add a credit line of 5000000 for account "4140"
    And I save the entry drawer
    Then I see entry "Trial Balance Sample Entry" in the journal list
    When I click on entry "Trial Balance Sample Entry" in the list
    When I click the post entry button
    Then I see the entry status is "Posted"
    When I open "/fin/trial-balance"
    Then I see the trial balance table
    And I see no provisional notice in trial balance
    And I see the trial balance status is balanced
    And I see account "1111" has balance "5,000,000" in trial balance
    And I see account "4140" has balance "5,000,000" in trial balance
    And I see total trial balance amount "5,000,000"
    When I set trial balance as of date to "2020-01-01"
    Then I see total trial balance amount "0"
