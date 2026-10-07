Feature: Access control

  Scenario Outline: Anonymous visitor is sent to login
    Given I am not signed in
    When I open "<path>"
    Then I should be on "/login"

    Examples:
      | path      |
      | /home     |
      | /settings |
      | /fin      |
      | /blog     |

