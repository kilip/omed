@accounts @create-account
Feature: Create Account

  Background:
    Given I am signed in

  Scenario: Navigate to create account page from accounts list
    When I open "/fin/accounts"
    And I click create account button
    Then I should be on "/fin/accounts/create"
    And I should see the heading "Create Account"

  Scenario: Successfully create a new account and verify in accounts table
    When I open "/fin/accounts/create"
    And I fill create account form with code "1001", name "Kas Utama", and type "Asset"
    And I submit create account form
    Then I should be on "/fin/accounts"
    And I should see the account row with code "1001"

