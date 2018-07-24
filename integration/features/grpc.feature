Feature: test gRPC
  In order to ensure gRPC integration works
  As a developer
  I need to test a gRPC integration

  Scenario: Simple Echo
    Given the gRPC echo server and proxy is running
    And the router is running
    When I send a request to the router
    Then the gRPC servers response should be echoed
