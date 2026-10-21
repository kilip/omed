@authenticated @cms
Feature: CMS Pages Management
  As a workspace editor or owner
  I want to create, edit, and delete standalone pages
  So that I can publish static information

  Background:
    Given I have a personal workspace

  Scenario: Create, edit, and delete a page
    Given I open "/cms/pages"
    When I click the new page button
    And I fill in the page with title "About Us" and body "# About Our Team"
    And I save the page
    Then I see the page status "Draft"
    When I fill in the page with title "About Us Updated" and body "# About Us Content"
    And I save the page
    When I open "/cms/pages"
    Then I see "About Us Updated" in the pages table
    When I open page "About Us Updated"
    And I delete the page
    Then I am on "/cms/pages"
    And I do not see "About Us Updated" in the pages table
