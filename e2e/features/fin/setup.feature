@authenticated @fin @setup
Feature: Finance Onboarding Wizard & Route Guard
  As a user in a new workspace
  I want to complete the 2-step financial onboarding wizard
  So that I can configure my base currency, period, and chart of accounts before accessing financial tools

  Scenario: Fresh workspace redirects to onboarding wizard and completes 2-step setup
    Given I have a personal workspace
    When I open "/fin"
    Then I am on "/fin/onboarding"
    And I see the setup initial period form
    When I select decimal style "dot"
    And I click the wizard next button
    Then I see account "1111" in the setup preview table
    When I click the wizard complete button
    Then I am on "/fin"
    When I open "/fin/onboarding"
    Then I am on "/fin"

  Scenario: Protected finance routes redirect un-onboarded workspace to onboarding wizard
    Given I have a personal workspace
    When I open "/fin/overview"
    Then I am on "/fin/onboarding"
    When I open "/fin/accounts"
    Then I am on "/fin/onboarding"
    When I open "/fin/periods"
    Then I am on "/fin/onboarding"
    When I click the apply template button
    Then I am on "/fin"
    When I open "/fin/accounts"
    Then I am on "/fin/accounts"
    And I see account "1111" in the accounts table

  Scenario: Completed onboarding prevents re-entering onboarding wizard
    Given I have a personal workspace
    When I open "/fin/onboarding"
    When I click the apply template button
    Then I am on "/fin"
    When I open "/fin/onboarding"
    Then I am on "/fin"
