@authenticated @fin @opening-balance
Feature: Opening Balance Setup and Management
  As a finance user
  I want to configure opening balances for cash, bank, and paylater accounts
  So that my balance sheet and general ledger accurately reflect my initial equity

  @opening-balance
  Scenario: Setup chart of accounts and input opening balance
    Given I have a personal workspace
    When I open "/fin/onboarding"
    When I click the apply template button
    Then I am on "/fin"
    When I open "/fin/opening-balance"
    Then I am on "/fin/opening-balance"
    When I input opening balance for account "1121" with amount "10000000"
    And I input opening balance for account "2160" with amount "1500000"
    And I click the save opening balance button
    Then I see the opening balance reentry notice
    When I open "/fin/journal"
    Then I see entry "Opening Balance / Saldo Awal" in the journal list
