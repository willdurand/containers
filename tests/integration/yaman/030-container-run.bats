#!/usr/bin/env bats

load helpers

@test "yaman container run" {
  value=$(random_string)
  run_yaman container run --rm "$DOCKER_ALPINE" -- echo "$value"
  assert_success
  assert_output "$value"
}

@test "yaman container run closes stdin" {
  value=$(random_string)
  run_yaman container run --rm --interactive "$DOCKER_ALPINE" -- cat <<<"$value"
  assert_success
  assert_output "$value"
}

@test "yaman container run with pipe" {
  value=$(random_string)
  run bash -c "echo $value | yaman container run --rm --interactive $DOCKER_ALPINE -- cat"
  assert_success
  assert_output "$value"
}

@test "yaman container run -d" {
  local cid=""
  run_yaman container run --rm -d "$DOCKER_ALPINE" -- sleep 10
  assert_success
  cid="$output"

  run_yaman container list
  assert_success
  assert_output --regexp "$cid(.+)running"

  run_yaman container stop "$cid"
  assert_success

  run_yaman container list
  refute_output --regexp "$cid(.+)running"
}

@test "yaman container run --hostname" {
  hostname="some-hostname"
  run_yaman container run --rm --hostname "$hostname" "$DOCKER_ALPINE" -- hostname
  assert_success
  assert_output "$hostname"
}

@test "yaman container run --name" {
  local cid=""
  run_yaman container run -d --rm --name "some-name" "$DOCKER_ALPINE" -- sleep 10
  assert_success
  cid="$output"

  run_yaman container list
  assert_success
  assert_output --regexp "$cid(.+)\ssome-name$"

  run_yaman container stop "$cid"
  assert_success
}
