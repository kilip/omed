@authenticated @fin
Feature: Finance Journal Entries
  As a finance user
  I want to create, search, post, edit, reverse, and void journal entries
  So that I can maintain balanced double-entry accounting records

  Background:
    Given I have a personal workspace
    When I open "/fin/onboarding"
    When I click the apply template button
    Then I am on "/fin"

  Scenario: Create a balanced draft journal entry with new transaction
    When I open "/fin/journal"
    And I click the new journal entry button
    Then I see the create entry drawer
    When I enter entry description "Office Supplies Purchase"
    And I add a debit line of quantity 3 and unit price 50000 for account "5100"
    And I add a credit line of 150000 for account "1111"
    And I save the entry drawer
    Then I see entry "Office Supplies Purchase" in the journal list
    When I click on entry "Office Supplies Purchase" in the list
    Then I see the entry status is "Draft"

  Scenario: Reject unbalanced journal entry
    When I open "/fin/journal"
    And I click the new journal entry button
    Then I see the create entry drawer
    When I enter entry description "Unbalanced Equipment"
    And I add a debit line of 100000 for account "5100"
    And I add a credit line of 50000 for account "1111"
    Then I see the entry unbalanced warning

  Scenario: Edit, Post, and Void a draft journal entry
    When I open "/fin/journal"
    And I click the new journal entry button
    Then I see the create entry drawer
    When I enter entry description "Draft Subscription"
    And I add a debit line of 200000 for account "5100"
    And I add a credit line of 200000 for account "1111"
    And I save the entry drawer
    Then I see entry "Draft Subscription" in the journal list
    When I click on entry "Draft Subscription" in the list
    Then I see the entry status is "Draft"
    When I click the edit entry button
    And I change entry description to "Updated Subscription"
    And I save the entry changes
    Then I see the entry status is "Draft"
    When I click the post entry button
    Then I see the entry status is "Posted"
    When I click the void entry button
    Then I see the entry status is "Void"

  Scenario: Reverse a posted journal entry
    When I open "/fin/journal"
    And I click the new journal entry button
    Then I see the create entry drawer
    When I enter entry description "Equipment Invoice"
    And I add a debit line of 500000 for account "1111"
    And I add a credit line of 500000 for account "3100"
    And I save the entry drawer
    Then I see entry "Equipment Invoice" in the journal list
    When I click on entry "Equipment Invoice" in the list
    When I click the post entry button
    Then I see the entry status is "Posted"
    When I click the reverse entry button
    And I confirm the reverse modal
    Then I see the entry status is "Posted"
