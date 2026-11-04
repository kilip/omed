@authenticated @agent @launcher
Feature: Agent Launcher
  As a workspace user
  I want to access the AI agent launcher
  So that I can navigate to agent providers and settings

  Scenario: Navigate to providers and settings from the agent launcher
    Given I have an active workspace
    When I open "/agent"
    Then I see the agent launcher
    When I click the agent launcher tile "Providers"
    Then I am on "/agent/providers"
    And I see the providers page
    When I open "/agent"
    Then I see the agent launcher
    When I click the agent launcher tile "Settings"
    Then I am on "/agent/settings"
    And I see the agent settings page
