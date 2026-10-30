@authenticated @fin @dashboard
Feature: Finance Dashboard
  As a finance user
  I want to view the finance dashboard metrics, chart, and recent entries
  So that I can monitor workspace financial health in real time

  Background:
    Given I have a personal workspace
    When I open "/fin/onboarding"
    When I click the apply template button
    Then I am on "/fin"

  Scenario: Navigate to financial features from the launcher
    When I open "/fin"
    Then I see the finance launcher
    When I click the launcher tile "Overview"
    Then I am on "/fin/overview"

  Scenario: View finance dashboard with real posted data
    When I open "/fin/journal"
    And I click the new journal entry button
    Then I see the create entry drawer
    When I enter entry description "Dashboard Revenue Entry"
    And I add a debit line of 5000000 for account "1111"
    And I add a credit line of 5000000 for account "4140"
    And I save the entry drawer
    Then I see entry "Dashboard Revenue Entry" in the journal list
    When I click on entry "Dashboard Revenue Entry" in the list
    When I click the post entry button
    Then I see the entry status is "Posted"
    When I open "/fin/overview"
    Then I see financial statistic "Total assets" with amount "5,000,000"
    And I see financial statistic "Net income" with amount "5,000,000"
    And I see recent journal entry "Dashboard Revenue Entry" in dashboard
    And I see the revenue vs expense chart
    And I see no provisional tags on dashboard
