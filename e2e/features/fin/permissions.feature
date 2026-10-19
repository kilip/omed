@authenticated @fin
Feature: Finance Role Permissions
  As a workspace member
  I should only see and perform financial actions allowed by my assigned fin role
  So that read-only viewers cannot modify or create records while accountants can

  Scenario: Viewer has read-only access across transactions and journal entries
    Given a workspace exists with sample finance data where I have role "fin:viewer"
    When I open "/fin/transactions"
    Then the new transaction button is hidden
    When I open the sample transaction detail page
    Then the edit transaction button is hidden
    And the delete transaction button is hidden
    And the add entry button is hidden
    When I open "/fin/journal"
    Then the new entry button is hidden
    When I open the sample journal entry detail page
    Then the post entry button is hidden
    And the edit entry button is hidden
    And the delete entry button is hidden
    And the void entry button is hidden
    When I open the sample posted journal entry detail page
    Then the reverse entry button is hidden
    And the void entry button is hidden

  Scenario: Accountant has full management access across transactions and journal entries
    Given a workspace exists with sample finance data where I have role "fin:accountant"
    When I open "/fin/transactions"
    Then the new transaction button is visible
    When I open the sample transaction detail page
    Then the edit transaction button is visible
    And the delete transaction button is visible
    And the add entry button is visible
    When I open "/fin/journal"
    Then the new entry button is visible
    When I open the sample journal entry detail page
    Then the post entry button is visible
    And the edit entry button is visible
    And the delete entry button is visible
    And the void entry button is visible
    When I open the sample posted journal entry detail page
    Then the reverse entry button is visible
    And the void entry button is visible
