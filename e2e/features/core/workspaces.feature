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

  Scenario: Switch active workspace from the dashboard table
    Given I have a personal workspace
    When I open "/core"
    When I click the new workspace button
    Then I am on "/core/workspaces/new"
    When I fill in the workspace name with "Beta Corp"
    And I submit the workspace form
    Then I am on "/core"
    And I see "Beta Corp" in the workspaces table
    And the active workspace is "Beta Corp"
    When I click the switch button for workspace "Personal Workspace"
    Then the active workspace is "Personal workspace"
    And workspace "Personal Workspace" has the active tag
    And workspace "Beta Corp" has the switch button

  Scenario: Edit workspace by clicking table row
    Given I have a personal workspace
    When I open "/core"
    When I click the new workspace button
    Then I am on "/core/workspaces/new"
    When I fill in the workspace name with "Gamma Corp"
    And I submit the workspace form
    Then I am on "/core"
    And I see "Gamma Corp" in the workspaces table
    When I click the workspace row "Gamma Corp"
    Then I am on workspace edit page
    When I fill in the workspace name with "Gamma Updated"
    And I submit the workspace form
    Then I am on "/core"
    And I see "Gamma Updated" in the workspaces table

