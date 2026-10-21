@authenticated @cms
Feature: CMS Public API Content Visibility
  As an unauthenticated API client
  I want to fetch publicly visible posts and pages
  Without accessing drafts or scheduled content

  Scenario: Public API returns only published posts
    Given a workspace exists with a published post, a draft post, and a future-scheduled post
    When an unauthenticated request is made for the public posts
    Then the public response contains the published post
    And the public response does not contain the draft post
    And the public response does not contain the future-scheduled post
