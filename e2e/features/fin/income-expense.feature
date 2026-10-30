@authenticated @fin @cashflow
Feature: Finance Income and Expense Quick Entry
  As a finance user
  I want to record simple income and expense transactions quickly
  So that I do not have to write full multi-line journal entries for everyday cash flow

  Background:
    Given I have a personal workspace
    When I open "/fin/overview"
    When I click the setup chart of accounts banner button
    Then I am on "/fin/setup"
    When I click the apply template button
    Then I am on "/fin/accounts"

  Scenario: Record an expense with receipt reference and see it in expense history
    When I open "/fin/expense"
    Then I am on "/fin/expense"
    When I click the new expense button
    Then I am on "/fin/expense/new"
    When I select payment account "1111"
    And I select expense category "5100"
    And I enter cashflow amount 75000
    And I enter cashflow description "Belanja kebutuhan kantor di Indomaret"
    And I enter cashflow reference "STRUK-001"
    Then I see cashflow accounting preview "(D) Expense:"
    And I see cashflow accounting preview "(K) Payment:"
    When I save the cashflow form
    Then I am on "/fin/expense"
    And I see cashflow item "Belanja kebutuhan kantor di Indomaret" in the table
    And I see cashflow item "STRUK-001" in the table
    And I see cashflow item "75,000" in the table

  Scenario: Record an income from Company X and see it in income history
    When I open "/fin/income"
    Then I am on "/fin/income"
    When I click the new income button
    Then I am on "/fin/income/new"
    When I select deposit account "1121"
    And I select revenue category "4110"
    And I enter cashflow amount 5000000
    And I enter cashflow description "Project Payment from Company X"
    And I enter cashflow reference "INV-2026-001"
    Then I see cashflow accounting preview "(D) Payment:"
    And I see cashflow accounting preview "(K) Revenue:"
    When I save the cashflow form
    Then I am on "/fin/income"
    And I see cashflow item "Project Payment from Company X" in the table
    And I see cashflow item "INV-2026-001" in the table
    And I see cashflow item "5,000,000" in the table
