@authenticated @cms
Feature: CMS Role Permissions
  As a workspace member
  I should only perform CMS actions allowed by my assigned role
  So that viewers are read-only while editors can manage content

  Scenario: Viewer has read-only access to CMS
    Given a workspace exists where I have role "cms:viewer"
    When I open "/cms/posts"
    Then the new post button is hidden
    When I open "/cms/pages"
    Then the new page button is hidden
    When I open "/cms/media"
    Then the upload media button is hidden

  Scenario: Editor has full edit access to CMS
    Given a workspace exists where I have role "cms:editor"
    When I open "/cms/posts"
    Then the new post button is visible
    When I open "/cms/pages"
    Then the new page button is visible
    When I open "/cms/media"
    Then the upload media button is visible
