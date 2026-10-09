@accounts @update-account
Feature: Update and Delete Account

  Background:
    Given I am signed in

  Scenario: Navigate to edit account page from accounts list table
    When I open "/fin/accounts/create"
    And I fill create account form with code "2001", name "Hutang Usaha", and type "Liability"
    And I submit create account form
    Then I should be on "/fin/accounts"
    And I should see the account row with code "2001"
    When I click edit button for account with code "2001"
    Then I should see the heading "Edit Account"
    And I should see account code "2001" disabled in form

  Scenario: Update account details and verify changes
    When I open "/fin/accounts/create"
    And I fill create account form with code "2001", name "Hutang Usaha", and type "Liability"
    And I submit create account form
    Then I should be on "/fin/accounts"
    And I should see the account row with code "2001"
    When I click edit button for account with code "2001"
    And I update account name to "Hutang Usaha Vendor"
    And I update account description to "Hutang usaha jangka pendek ke vendor"
    And I submit edit account form
    Then I should be on "/fin/accounts"
    And I should see the account row with code "2001"

  Scenario: Delete account and verify removal from accounts table
    When I open "/fin/accounts/create"
    And I fill create account form with code "2001", name "Hutang Usaha", and type "Liability"
    And I submit create account form
    Then I should be on "/fin/accounts"
    And I should see the account row with code "2001"
    When I click edit button for account with code "2001"
    And I click delete account button
    And I confirm account deletion
    Then I should be on "/fin/accounts"
    And I should not see the account row with code "2001"


