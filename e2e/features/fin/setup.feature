@authenticated @fin @setup
Feature: Finance Chart of Accounts Setup
  As a user in a new workspace
  I want to seed standard Chart of Accounts templates
  So that I can start financial bookkeeping immediately

  Scenario: Seed freelancer template from dashboard banner
    Given I have a personal workspace
    When I open "/fin"
    Then I see the setup chart of accounts banner
    When I click the setup chart of accounts banner button
    Then I am on "/fin/setup"
    And I see account "1111" in the setup preview table
    When I click the apply template button
    Then I am on "/fin/accounts"
    And I see account "1111" in the accounts table
    And I do not see the setup template button

  Scenario: Seed template from accounts empty state
    Given I have a personal workspace
    When I open "/fin/accounts"
    Then I see the accounts empty state
    When I click the seed template button in empty state
    Then I am on "/fin/setup"
    When I click the apply template button
    Then I am on "/fin/accounts"
    And I see account "1111" in the accounts table

  Scenario: Prevent re-seeding when accounts already exist
    Given I have a personal workspace
    When I open "/fin"
    When I click the setup chart of accounts banner button
    Then I am on "/fin/setup"
    When I click the apply template button
    Then I am on "/fin/accounts"
    When I open "/fin/setup"
    Then I am on "/fin/accounts"
