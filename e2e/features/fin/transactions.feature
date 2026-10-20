@authenticated @fin @transactions
Feature: Finance Transactions
  As a finance user
  I want to create, search, edit, and manage transactions and their entries
  So that I can group related journal entries into business transactions

  Background:
    Given I have a personal workspace
    When I open "/fin"
    When I click the setup chart of accounts banner button
    Then I am on "/fin/setup"
    When I click the apply template button
    Then I am on "/fin/accounts"

  Scenario: Create transaction and filter list
    When I open "/fin/transactions"
    And I click the new transaction button
    Then I am on "/fin/transactions/new"
    When I enter transaction description "Cloud Hosting Subscription"
    And I enter transaction reference "INV-2026-001"
    And I attach file "hosting-invoice.pdf" to the new transaction form
    And I save the transaction form
    Then I see attachment "hosting-invoice.pdf" on transaction detail
    When I upload attachment "payment-receipt.png" on transaction detail
    Then I see attachment "payment-receipt.png" on transaction detail
    When I delete attachment "payment-receipt.png" on transaction detail
    Then I do not see attachment "payment-receipt.png" on transaction detail
    When I open "/fin/transactions"
    Then I see transaction "Cloud Hosting Subscription" in the transactions list
    When I search transactions for "Hosting"
    Then I see transaction "Cloud Hosting Subscription" in the transactions list

  Scenario: Edit draft transaction and add journal entry
    When I open "/fin/transactions"
    And I click the new transaction button
    Then I am on "/fin/transactions/new"
    When I enter transaction description "Laptop Purchase"
    And I enter transaction reference "PO-100"
    And I save the transaction form
    When I click the edit transaction button
    And I change transaction reference to "PO-100-REV"
    And I change transaction description to "Laptop Purchase Revised"
    And I save the transaction changes
    Then I see text "PO-100-REV" on transaction detail
    When I click add entry to transaction button
    Then I see the create entry drawer
    When I enter entry description "Laptop down payment"
    And I add a debit line of 10000000 for account "5100"
    And I add a credit line of 10000000 for account "1111"
    And I save the entry drawer
    When I open "/fin/transactions"
    And I click on transaction "Laptop Purchase Revised" in the list
    Then I see entry "Laptop down payment" in transaction entries table

  Scenario: Transaction with posted entry cannot be edited or deleted
    When I open "/fin/transactions"
    And I click the new transaction button
    Then I am on "/fin/transactions/new"
    When I enter transaction description "Fixed Asset Acquisition"
    And I enter transaction reference "FA-01"
    And I save the transaction form
    When I click add entry to transaction button
    Then I see the create entry drawer
    When I enter entry description "Asset Initial Payment"
    And I add a debit line of 5000000 for account "5100"
    And I add a credit line of 5000000 for account "1111"
    And I save the entry drawer
    When I open "/fin/transactions"
    And I click on transaction "Fixed Asset Acquisition" in the list
    When I click entry "Asset Initial Payment" in transaction entries table
    When I click the post entry button
    Then I see the entry status is "Posted"
    When I open "/fin/transactions"
    And I click on transaction "Fixed Asset Acquisition" in the list
    Then I cannot edit or delete the transaction
