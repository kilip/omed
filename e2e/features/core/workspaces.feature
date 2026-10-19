@authenticated
Feature: Workspaces
  A user can view their workspaces on the dashboard and create a new workspace.

  Scenario: Create a new workspace from the dashboard
    Given I have a personal workspace
    When I open "/core"
    Then I am on "/core"
    When I click the new workspace button
    Then I am on "/core/workspaces/new"
    When I fill in the workspace name with "Acme Corp"
    And I fill in the workspace description with "Main workspace for Acme"
    And I submit the workspace form
    Then I am on "/core"
    And I see "Acme Corp" in the workspaces table
    And the active workspace is "Acme Corp"
