@authenticated @fin @cashflow
Feature: Finance Income and Expense Quick Entry
  As a finance user
  I want to record simple income and expense transactions quickly
  So that I do not have to write full multi-line journal entries for everyday cash flow

  Background:
    Given I have a personal workspace
    When I open "/fin/onboarding"
    When I click the apply template button
    Then I am on "/fin"

  Scenario: Finance Income and Expense - Record a multi-item income with posting and see it in income history
    When I open "/fin/income"
    Then I am on "/fin/income"
    When I click the new income button
    Then I am on "/fin/income/new"
    When I select deposit account "1121"
    And I enter cashflow reference "inv-123456"
    And I add cashflow item 0 with account "4130", description "Konsultasi Improvement", quantity 1 and unit price 1000000
    And I add cashflow item 1 with account "4110", description "Scaffold base project structure and ci/cd", quantity 6 and unit price 500000
    When I click the cashflow posting button
    Then I am on "/fin/income"
    And I see cashflow item "inv-123456" in the table
    And I see cashflow item "4,000,000" in the table
    And I see cashflow status "Posted"

  Scenario: Finance Income and Expense - Record a multi-item expense as draft, edit and post it
    When I open "/fin/expense"
    Then I am on "/fin/expense"
    When I click the new expense button
    Then I am on "/fin/expense/new"
    When I select payment account "1111"
    And I add cashflow item 0 with account "6710", description "telur", quantity 2 and unit price 55000
    And I add cashflow item 1 with account "6710", description "indomie", quantity 5 and unit price 3000
    When I click the cashflow draft button
    Then I am on "/fin/expense"
    And I see cashflow status "Draft"
    And I see cashflow item "125,000" in the table
    When I click cashflow row "125,000"
    And I click the edit cashflow button
    When I click the cashflow posting button
    Then I am on "/fin/expense"
    And I see cashflow status "Posted"
