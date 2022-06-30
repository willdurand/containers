#!/usr/bin/env bats

load helpers

@test "yaman container stop" {
  cid1=$(run_yaman_and_get_cid container run --rm -d "$DOCKER_ALPINE" -- sleep 10)
  cid2=$(run_yaman_and_get_cid container run --rm -d "$DOCKER_ALPINE" -- sleep 10)

  run_yaman container list
  assert_success
  assert_output --regexp "$cid1(.+)$DOCKER_ALPINE(.+)running"
  assert_output --regexp "$cid2(.+)$DOCKER_ALPINE(.+)running"

  run_yaman container stop "$cid1" "$cid2"
  assert_success

  run_yaman container list --all
  assert_success
  refute_output --partial "$cid1"
  refute_output --partial "$cid2"
}
