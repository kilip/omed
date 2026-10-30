@authenticated @fin @periods
Feature: Finance Accounting Periods
  As a financial user
  I want to manage accounting periods
  So that I can control posting windows and close completed periods

  Scenario: Initial period created after setup and visible in periods table
    Given I have a personal workspace
    When I open "/fin/onboarding"
    Then I see the setup initial period form
    When I click the apply template button
    Then I am on "/fin"
    When I open "/fin/periods"
    Then I see period "FY 2026" with status "Open" in the periods table

  Scenario: Create, close, reopen, and delete an accounting period
    Given I have a personal workspace
    When I open "/fin/onboarding"
    When I click the apply template button
    Then I am on "/fin"
    When I open "/fin/periods"
    When I click the new period button
    And I fill the period form with name "FY 2027" and fiscal year 2027
    And I click the save period button
    Then I see period "FY 2027" with status "Open" in the periods table
    When I close period "FY 2027"
    Then I see period "FY 2027" with status "Closed" in the periods table
    When I reopen period "FY 2027"
    Then I see period "FY 2027" with status "Open" in the periods table
    When I delete period "FY 2027"
    Then I do not see period "FY 2027" in the periods table
