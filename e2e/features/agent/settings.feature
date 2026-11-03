@authenticated @agent
Feature: Agent Settings
  As a workspace admin
  I want to enable the financial agent and choose its provider
  So that transaction documents can be analyzed automatically

  Scenario: Admin enables the financial agent with a selected provider
    Given I have an active workspace
    And the workspace has a default agent provider "Primary Provider"
    And the workspace has an agent provider "Secondary Provider"
    When I open "/agent/settings"
    Then I see the agent settings page
    And the financial agent is disabled
    When I enable the financial agent
    And I select the financial provider "Secondary Provider"
    And I save the agent settings
    Then I see the agent settings saved message
    When I reload the page
    Then I see the agent settings page
    And the financial agent is enabled
    And the financial provider is "Secondary Provider"

  Scenario: Admin cannot enable the financial agent without a provider
    Given I have an active workspace
    When I open "/agent/settings"
    Then I see the agent settings page
    When I enable the financial agent
    And I save the agent settings
    Then I see the agent settings error "no capable provider found"
    And the financial agent is disabled

  Scenario: Viewer sees agent settings as read-only
    Given a workspace exists with a sample agent provider where I have role "agent:viewer"
    When I open "/agent/settings"
    Then I see the agent settings page
    And the financial agent switch is disabled
    And the financial provider select is disabled
    And the save agent settings button is hidden
