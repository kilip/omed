@authenticated @cms @launcher
Feature: CMS Launcher
  As a workspace user
  I want to access the content launcher
  So that I can navigate to posts, pages, and media

  Scenario: Navigate to posts, pages, and media from the cms launcher
    Given I have an active workspace
    When I open "/cms"
    Then I see the cms launcher
    When I click the cms launcher tile "Posts"
    Then I am on "/cms/posts"
    And I see the posts page
    When I open "/cms"
    Then I see the cms launcher
    When I click the cms launcher tile "Pages"
    Then I am on "/cms/pages"
    And I see the pages page
    When I open "/cms"
    Then I see the cms launcher
    When I click the cms launcher tile "Media"
    Then I am on "/cms/media"
    And I see the media library page
