#!/usr/bin/env bats

load helpers

@test "yaman container list" {
  cid=$(run_yaman_and_get_cid container run -d "$DOCKER_ALPINE" -- sleep 10)

  run_yaman container list
  assert_success
  assert_output --regexp "$cid(.+)$DOCKER_ALPINE(.+)running"

  run_yaman container stop "$cid"
  assert_success

  sleep 1
  run_yaman container list
  assert_success
  refute_output --partial "$cid"

  run_yaman container list --all
  assert_success
  assert_output --regexp "$cid(.+)$DOCKER_ALPINE(.+)Exited"

  run_yaman container delete "$cid"
  assert_success

  sleep 1
  run_yaman container list --all
  assert_success
  refute_output --partial "$cid"
}
