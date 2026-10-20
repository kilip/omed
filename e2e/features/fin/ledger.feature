@authenticated @fin
Feature: Finance General Ledger
  As a finance user
  I want to view general ledger postings for accounts
  And navigate seamlessly between accounts, entries, and ledger
  So that I can audit posted financial transactions and account balances

  Background:
    Given I have a personal workspace
    When I open "/fin"
    When I click the setup chart of accounts banner button
    Then I am on "/fin/setup"
    When I click the apply template button
    Then I am on "/fin/accounts"

  Scenario: Post a journal entry and view it in General Ledger
    When I open "/fin/journal"
    And I click the new journal entry button
    Then I see the create entry drawer
    When I enter entry description "Client Retainer Fee"
    And I add a debit line of 5000000 for account "1111"
    And I add a credit line of 5000000 for account "4140"
    And I save the entry drawer
    Then I see entry "Client Retainer Fee" in the journal list
    When I click on entry "Client Retainer Fee" in the list
    When I click the post entry button
    Then I see the entry status is "Posted"
    When I open "/fin/ledger"
    And I select ledger account "1111"
    Then I see posting with description "Client Retainer Fee" in the ledger
    And I see the ledger running balance

  Scenario: Cross-navigation between accounts, entry detail, and ledger
    When I open "/fin/journal"
    And I click the new journal entry button
    Then I see the create entry drawer
    When I enter entry description "Software Hosting Expense"
    And I add a debit line of 750000 for account "5100"
    And I add a credit line of 750000 for account "1111"
    And I save the entry drawer
    Then I see entry "Software Hosting Expense" in the journal list
    When I click on entry "Software Hosting Expense" in the list
    When I click the post entry button
    Then I see the entry status is "Posted"
    When I click posting line account "5100" in entry detail
    Then I am on "/fin/ledger"
    And I see posting with description "Software Hosting Expense" in the ledger
    When I click entry link in ledger row
    Then I am on "/fin/journal"
    When I open "/fin/accounts"
    And I click account "1111" ledger link in accounts table
    Then I am on "/fin/ledger"
    And I see posting with description "Software Hosting Expense" in the ledger
