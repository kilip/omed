@authenticated @cms @media
Feature: CMS Media Library
  As a workspace editor or owner
  I want to upload media files and edit metadata
  And ensure referenced media cannot be deleted

  Background:
    Given I have a personal workspace

  Scenario: Upload media and edit alt text
    Given I open "/cms/media"
    When I upload a media file named "test-banner.png"
    Then I see "test-banner.png" in the media library
    When I edit the alt text of "test-banner.png" to "Hero banner image"
    Then I see "test-banner.png" in the media library

  Scenario: Delete media is blocked when used as a post cover
    Given I open "/cms/media"
    When I upload a media file named "cover-photo.png"
    Then I see "cover-photo.png" in the media library
    When I open "/cms/posts"
    And I click the new post button
    And I fill in the post with title "Post with Cover" and body "Post body content"
    And I select the uploaded cover image
    And I save the post
    When I open "/cms/media"
    And I attempt to delete the media "cover-photo.png"
    Then the media delete is blocked with a message that it is in use
