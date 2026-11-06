@authenticated @fin
Feature: Talk To Agent Chat
  As a workspace member with finance and agent permissions
  I want to chat with the financial agent to analyze slips and receipts
  So that I can quickly review and save draft transactions

  Background:
    Given I have a personal workspace

  Scenario: ATM slip yields a journal draft preview and confirm saves a draft
    Given the workspace has financial agent and accounts configured
    When I navigate to the Talk to Agent page
    And I attach file "kaltimtara_atm_withdrawal_slip.png" to the chat
    Then I see attachment thumbnail "kaltimtara_atm_withdrawal_slip.png"
    And I see a draft card of kind "journal"
    When I confirm the draft proposal
    Then I see the confirmed draft state

  Scenario: Shop receipt yields a multi-item expense preview
    Given the workspace has financial agent and accounts configured
    When I navigate to the Talk to Agent page
    And I attach file "shop_stationery_receipt.png" to the chat
    Then I see attachment thumbnail "shop_stationery_receipt.png"
    And I see a draft card of kind "expense"

  Scenario: Discard saves nothing
    Given the workspace has financial agent and accounts configured
    When I navigate to the Talk to Agent page
    And I attach file "kaltimtara_atm_slip.png" to the chat
    Then I see a draft card of kind "journal"
    When I discard the draft proposal
    Then I see the discarded draft state

  Scenario: History persists after reload
    Given the workspace has financial agent and accounts configured
    When I navigate to the Talk to Agent page
    And I send chat message "Hello agent, let us analyze my receipts"
    Then I see message "Hello agent, let us analyze my receipts" in the chat
    When I reload the page
    Then I see message "Hello agent, let us analyze my receipts" in the chat

  Scenario: Permission-less user cannot open the chat
    Given a workspace exists with sample finance data where I have role "fin:viewer"
    When I navigate to the Talk to Agent page
    Then I see forbidden access denied for chat
