#!/usr/bin/env bats

load helpers

@test "yaman container restart a created container" {
  cid=$(run_yaman_and_get_cid container create "$DOCKER_ALPINE" -- echo "hello, world")

  run_yaman container restart "$cid"
  assert_success

  run_yaman container logs "$cid"
  assert_success
  assert_output "hello, world"

  run_yaman container delete "$cid"
  assert_success
}

@test "yaman container restart a running container" {
  cid=$(run_yaman_and_get_cid container run -d "$DOCKER_ALPINE" -- sleep 100)
  pid1=$(inspect "$cid" | jq '.Shim.State.pid')

  sleep 1
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
  cid=$(run_yaman_and_get_cid container run -d --rm "$DOCKER_ALPINE" -- sleep 100)
  pid1=$(inspect "$cid" | jq '.Shim.State.pid')

  sleep 1
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
