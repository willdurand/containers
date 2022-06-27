#!/usr/bin/env bats

load helpers

@test "yaman container logs" {
  value=$(random_string)
  cid=$(run_yaman_and_get_cid container run -d "$DOCKER_ALPINE" -- echo "$value")

  run_yaman container logs "$cid"
  assert_output "$value"

  run_yaman container delete "$cid"
  assert_success
}

@test "yaman container logs --timestamps" {
  value=$(random_string)
  cid=$(run_yaman_and_get_cid container run -d "$DOCKER_ALPINE" -- echo "$value")

  run_yaman container logs --timestamps "$cid"
  assert_output --regexp "20[0-9]{2}.+ - $value"

  run_yaman container delete "$cid"
  assert_success
}
