@authenticated @cms
Feature: CMS Content Publishing
  As a workspace editor or owner
  I want to publish now, schedule, and unpublish content
  So that I can control the content lifecycle

  Background:
    Given I have a personal workspace

  Scenario: Publish and unpublish a post
    Given I open "/cms/posts"
    When I click the new post button
    And I fill in the post with title "Launch Announcement" and body "We are launching today!"
    And I save the post
    Then I see the post status "Draft"
    When I publish the post
    Then I see the post status "Published"
    When I unpublish the post
    Then I see the post status "Draft"

  Scenario: Schedule a post for future publication
    Given I open "/cms/posts"
    When I click the new post button
    And I fill in the post with title "Future Announcement" and body "Coming soon!"
    And I save the post
    Then I see the post status "Draft"
    When I schedule the post for publication
    Then I see the post status "Scheduled"
