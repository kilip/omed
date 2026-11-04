@authenticated @fin @settings
Feature: Finance Settings
  As a finance user
  I want to configure currency formatting, fiscal periods, and FX accounts
  So that financial records follow my workspace conventions

  Background:
    Given I have a personal workspace
    When I open "/fin/onboarding"
    And I click the apply template button
    Then I am on "/fin"

  Scenario: Update currency, decimal format, and FX account mappings and persist after reload
    When I open "/fin/settings"
    Then I see the finance settings page
    And the decimal style in finance settings is "dot"
    And I see period "FY 2026" with status "Open" in finance settings
    When I select base currency "IDR" in finance settings
    And I select decimal style "comma" in finance settings
    And I save currency and number format settings
    Then I see the finance settings saved message
    When I select FX account "4140" for "Realized FX Gain Account"
    And I select FX account "5100" for "Rounding Difference Account"
    And I save FX account mappings
    Then I see the finance settings saved message
    When I reload the page
    Then I see the finance settings page
    And the base currency in finance settings is "IDR"
    And the decimal style in finance settings is "comma"
    And the FX account for "Realized FX Gain Account" is "4140"
    And the FX account for "Rounding Difference Account" is "5100"

  Scenario: Create a new fiscal period from finance settings
    When I open "/fin/settings"
    Then I see the finance settings page
    When I create a fiscal period "FY 2027" for fiscal year 2027 from finance settings
    Then I see the fiscal period created message
    And I see period "FY 2027" with status "Open" in finance settings
