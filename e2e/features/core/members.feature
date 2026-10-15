@authenticated
Feature: Workspace Member Roles
  As a workspace owner
  I want to select one role per module when inviting a member
  So that roles enforce single-selection per module while allowing multi-module roles

  Scenario: Selecting roles enforces single-selection per module
    Given I have an active workspace
    And I open "/core/members/new"
    When I select the role "core:member"
    Then the role "core:member" should be selected
    When I select the role "core:admin"
    Then the role "core:admin" should be selected
    And the role "core:member" should not be selected
    When I select the role "fin:admin"
    Then the role "core:admin" should be selected
    And the role "fin:admin" should be selected
    When I select the role "fin:accountant"
    Then the role "core:admin" should be selected
    And the role "fin:accountant" should be selected
    And the role "fin:admin" should not be selected
    When I fill the member email "newmember@omed.test"
    And I submit the member invitation form
    Then I am on "/core/members"
    And "newmember@omed.test" should be listed with role "Admin"
    And "newmember@omed.test" should be listed with role "Accountant"
