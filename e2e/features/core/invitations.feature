@authenticated
Feature: Workspace Invitations
  As a workspace owner
  I want to invite members by email
  So they can join my workspace via the emailed link

  Scenario: Inviting a member sends an email with an accept link
    Given I have an active workspace
    And I open "/core/invitations"
    When I create an invitation for "colleague@omed.test" with role "Admin"
    Then an invitation email should be sent to "colleague@omed.test"
    When the recipient opens the invitation link
    Then the recipient can accept the invitation
