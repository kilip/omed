@authenticated
Feature: Access Tokens
  As an authenticated user
  I want to create and revoke personal access tokens
  So that external clients can access the API on my behalf

  Scenario: A user creates an access token and sees it only once
    Given I have a personal workspace
    When I open "/core/access-tokens"
    Then I am on "/core/access-tokens"
    When I create an access token named "Claude Desktop" with expiry "90 days"
    Then the generated token starting with "omed_pat_" is displayed
    When I close the token modal
    Then the token reveal is no longer visible
    And the access token "Claude Desktop" is listed with status "Active"

  Scenario: A user revokes an access token
    Given I have a personal workspace
    When I open "/core/access-tokens"
    Then I am on "/core/access-tokens"
    When I create an access token named "Claude Desktop" with expiry "90 days"
    And I close the token modal
    When I revoke the access token "Claude Desktop"
    Then the access token "Claude Desktop" is listed with status "Revoked"
