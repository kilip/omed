@authenticated @fin @accounts
Feature: Finance Accounts Management
  As a finance user
  I want to create, edit, deactivate, and delete chart of accounts entries
  So that I can customize my workspace accounts structure

  Background:
    Given I have a personal workspace
    When I open "/fin/onboarding"
    When I click the apply template button
    Then I am on "/fin"

  Scenario: Create, edit, deactivate, and delete an account
    When I open "/fin/accounts"
    And I click the new account button
    Then I see the account drawer
    When I fill the account form with code "1099", name "Petty Cash Stash", type "Asset"
    And I click the save account button
    Then I see account "1099" in the accounts table
    And I see account "1099" with status "Active" in the accounts table
    When I click the edit button for account "1099"
    Then I see the account drawer
    When I update the account name to "Petty Cash Reserve"
    And I toggle the account active switch
    And I click the save account button
    Then I see account "1099" with name "Petty Cash Reserve" in the accounts table
    And I see account "1099" with status "Inactive" in the accounts table
    When I delete account "1099"
    Then I do not see account "1099" in the accounts table
