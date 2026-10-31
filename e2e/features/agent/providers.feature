@authenticated @agent
Feature: Agent Providers Management
  As a workspace admin
  I want to manage AI model providers
  So that the workspace has configured models for agents

  Scenario: Admin adds a provider, tests connection, sets default, and deletes it
    Given I have an active workspace
    When I open "/agent/providers"
    Then I see the providers page
    When I click the add provider button
    And I fill the provider form with:
      | name    | OpenAI Main              |
      | kind    | OpenAI Compatible        |
      | model   | gpt-4o                   |
      | baseUrl | http://127.0.0.1:4010/v1 |
      | apiKey  | sk-test-key-1234         |
    And I submit the provider form
    Then I see "OpenAI Main" in the providers table
    When I test the connection for provider "OpenAI Main"
    Then the connection test for "OpenAI Main" succeeds
    When I click the add provider button
    And I fill the provider form with:
      | name    | OpenAI Second            |
      | kind    | OpenAI Compatible        |
      | model   | gpt-4o-mini              |
      | baseUrl | http://127.0.0.1:4010/v1 |
      | apiKey  | sk-test-key-5678         |
    And I submit the provider form
    Then I see "OpenAI Second" in the providers table
    When I set "OpenAI Second" as default provider
    Then "OpenAI Second" is marked as default
    When I delete the provider "OpenAI Second"
    Then I do not see "OpenAI Second" in the providers table
