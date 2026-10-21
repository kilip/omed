@authenticated @cms
Feature: CMS Posts Management
  As a workspace editor or owner
  I want to create, edit, and delete posts
  So that I can manage content articles

  Background:
    Given I have a personal workspace

  Scenario: Create, edit, and delete a post
    Given I open "/cms/posts"
    When I click the new post button
    And I fill in the post with title "Getting Started with Omed" and body "# Welcome to Omed"
    And I save the post
    Then I see the post status "Draft"
    When I fill in the post with title "Getting Started with Omed Updated" and body "# Updated Content"
    And I save the post
    When I open "/cms/posts"
    Then I see "Getting Started with Omed Updated" in the posts table
    When I open post "Getting Started with Omed Updated"
    And I delete the post
    Then I am on "/cms/posts"
    And I do not see "Getting Started with Omed Updated" in the posts table
