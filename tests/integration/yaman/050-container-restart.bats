#!/usr/bin/env bats

load helpers

@test "yaman container restart a created container" {
  local cid=""
  run_yaman container create "$DOCKER_ALPINE" -- echo "hello, world"
  assert_success
  cid="$output"

  run_yaman container restart "$cid"
  assert_success

  run_yaman container logs "$cid"
  assert_success
  assert_output "hello, world"

  run_yaman container delete "$cid"
  assert_success
}

@test "yaman container restart a running container" {
  local cid=""
  local pid1=""
  local pid2=""

  run_yaman container run -d "$DOCKER_ALPINE" -- sleep 100
  assert_success
  cid="$output"
  pid1=$(inspect "$cid" | jq '.Shim.State.pid')

  run_yaman container restart "$cid"
  assert_success
  pid2=$(inspect "$cid" | jq '.Shim.State.pid')
  [ "$pid1" -lt "$pid2" ]

  run_yaman container list
  assert_success
  assert_output --regexp "$cid(.+)running"

  run_yaman container stop "$cid"
  assert_success

  run_yaman container delete "$cid"
  assert_success
}

@test "yaman container restart a running container configured with --rm" {
  local cid=""
  local pid1=""
  local pid2=""

  run_yaman container run -d --rm "$DOCKER_ALPINE" -- sleep 100
  assert_success
  cid="$output"
  pid1=$(inspect "$cid" | jq '.Shim.State.pid')

  run_yaman container restart "$cid"
  assert_success
  pid2=$(inspect "$cid" | jq '.Shim.State.pid')
  [ "$pid1" -lt "$pid2" ]

  run_yaman container list
  assert_success
  assert_output --regexp "$cid(.+)running"

  run_yaman container stop "$cid"
  assert_success
}
