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

  Scenario: Localized public post retrieval returns requested translation when published
    Given a workspace exists with a post published in "en" with slug "welcome-post" and translation in "id" with slug "selamat-datang"
    When an unauthenticated request is made for post "selamat-datang" in locale "id"
    Then the public response status is 200
    And the public post response has locale "id" and fallback false
    And the public post alternates contains locale "en"

  Scenario: Localized public post retrieval falls back to default locale when translation is missing
    Given a workspace exists with a post published only in "en" with slug "english-only-article"
    When an unauthenticated request is made for post "english-only-article" in locale "id"
    Then the public response status is 200
    And the public post response has fallback true
    And the public post alternates contains locale "en"

  Scenario: Localized public post retrieval returns 400 for unsupported locale
    Given a workspace exists with a post published only in "en" with slug "unsupported-locale-check"
    When an unauthenticated request is made for post "unsupported-locale-check" in locale "fr"
    Then the public response status is 400
