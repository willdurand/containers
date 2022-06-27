#!/usr/bin/env bats

load helpers

@test "yaman container run --pull=always" {
  run_yaman container run --rm --pull=always "$DOCKER_HELLO_WORLD"
  assert_success
  assert_output --partial "Pulling $DOCKER_HELLO_WORLD"
}

@test "yaman container run --pull=missing" {
  run_yaman container run --rm --pull=missing "$DOCKER_HELLO_WORLD"
  assert_success
  refute_output --partial "Pulling $DOCKER_HELLO_WORLD"
}

@test "yaman container run --pull=never" {
  run_yaman container run --rm --pull=never "$DOCKER_HELLO_WORLD"
  assert_success
  refute_output --partial "Pulling $DOCKER_HELLO_WORLD"
}
