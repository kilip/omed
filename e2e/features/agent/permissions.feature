@authenticated @agent
Feature: Agent Role Permissions
  As a workspace member
  I should only see agent features allowed by my assigned role
  So that read-only viewers cannot create, edit, test, or delete providers

  Scenario: Viewer sees providers list without management controls
    Given a workspace exists with a sample agent provider where I have role "agent:viewer"
    When I open "/agent/providers"
    Then I see the providers page
    And I see "Sample Provider" in the providers table
    And the add provider button is hidden
    And the test connection button is hidden for "Sample Provider"
    And the set default button is hidden for "Sample Provider"
    And the edit provider button is hidden for "Sample Provider"
    And the delete provider button is hidden for "Sample Provider"
