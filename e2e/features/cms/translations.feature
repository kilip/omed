@authenticated @cms
Feature: CMS Multi-Language Content
  As an author
  I want to create and edit post translations per locale
  So that multi-language audiences can read localized content

  Background:
    Given I have a personal workspace

  Scenario: Create a post, switch locale tabs, and add a translation
    Given I open "/cms/posts"
    When I click the new post button
    And I fill in the post with title "English Article" and body "Content in English"
    And I save the post
    When I open "/cms/posts"
    Then I see "English Article" in the posts table
    When I open post "English Article"
    And I switch to translation tab "ID"
    And I fill in the post with title "Artikel Bahasa Indonesia" and body "Konten bahasa Indonesia"
    And I save the post
    Then I see the post status "Draft"

  Scenario: Post translation creation, tab switching, and verifying title retention on both tabs
    Given I open "/cms/posts"
    When I click the new post button
    And I fill in the post with title "Original English Post" and body "English body content"
    And I save the post
    When I switch to translation tab "ID"
    And I fill in the post with title "Posting Bahasa Indonesia" and body "Konten posting Indonesia"
    And I save the post
    When I switch to translation tab "EN"
    Then the post title is "Original English Post"
    When I switch to translation tab "ID"
    Then the post title is "Posting Bahasa Indonesia"

  Scenario: Independent publishing per locale
    Given I open "/cms/posts"
    When I click the new post button
    And I fill in the post with title "Bilingual Headline" and body "Content EN"
    And I save the post
    When I switch to translation tab "ID"
    And I fill in the post with title "Tajuk Dwibahasa" and body "Konten ID"
    And I save the post
    When I switch to translation tab "EN"
    And I publish the post
    Then I see the post status "Published"
    When I switch to translation tab "ID"
    Then I see the post status "Draft"
    When I publish the post
    Then I see the post status "Published"

  Scenario: Page translation creation and retention
    Given I open "/cms/pages"
    When I click the new page button
    And I fill in the page with title "About Company" and body "# About Us in English"
    And I save the page
    When I switch to translation tab "ID"
    And I fill in the page with title "Tentang Perusahaan" and body "# Tentang Kami dalam Bahasa Indonesia"
    And I save the page
    When I switch to translation tab "EN"
    Then the page title is "About Company"
    When I switch to translation tab "ID"
    Then the page title is "Tentang Perusahaan"

  Scenario: Guard preventing deletion of default translation while other translation exists
    Given I open "/cms/posts"
    When I click the new post button
    And I fill in the post with title "Protected Default Post" and body "Default body"
    And I save the post
    When I switch to translation tab "ID"
    And I fill in the post with title "Terjemahan Tambahan" and body "Badan ID"
    And I save the post
    When I switch to translation tab "EN"
    Then the delete translation button is hidden
    When I switch to translation tab "ID"
    Then the delete translation button is visible
    When I delete the translation
    Then the post title is "Protected Default Post"
    When I switch to translation tab "ID"
    Then the delete translation button is hidden

  Scenario: Updating CMS workspace supported locales and saving in settings
    Given I open "/cms/settings"
    Then the CMS settings page is displayed
    When I add supported locale "ja"
    And I save the CMS settings
    Then the CMS settings are saved successfully
