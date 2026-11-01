@otp
Feature: Passwordless OTP login
  Users can sign in or sign up using a one-time code sent to their email.

  Scenario: A new user signs in with email OTP and lands on "/"
    Given the mailbox is empty
    When I open "/login"
    And I request a sign-in code for a new email
    And I enter the code from the email
    Then I am on "/"

  Scenario: An existing user signs in with email OTP and lands on "/"
    Given the mailbox is empty
    When I open "/login"
    And I request a sign-in code for the existing user
    And I enter the code from the email
    Then I am on "/"

  Scenario: An invalid code displays the error message and stays on "/login"
    Given the mailbox is empty
    When I open "/login"
    And I request a sign-in code for a new email
    And I enter the code "000000"
    Then I see the invalid code message
    And I am on "/login"

  Scenario: Resend button is disabled with countdown immediately after requesting a code
    Given the mailbox is empty
    When I open "/login"
    And I request a sign-in code for a new email
    Then the resend button is disabled
