#!/usr/bin/env bats

load helpers

@test "yaman container logs" {
  local cid=""
  value=$(random_string)
  run_yaman container run -d "$DOCKER_ALPINE" -- echo "$value"
  assert_success
  cid="$output"

  run_yaman container logs "$cid"
  assert_output "$value"

  run_yaman container delete "$cid"
  assert_success
}

@test "yaman container logs --timestamps" {
  local cid=""
  value=$(random_string)
  run_yaman container run -d "$DOCKER_ALPINE" -- echo "$value"
  assert_success
  cid="$output"

  run_yaman container logs --timestamps "$cid"
  assert_output --regexp "20[0-9]{2}.+ - $value"

  run_yaman container delete "$cid"
  assert_success
}
