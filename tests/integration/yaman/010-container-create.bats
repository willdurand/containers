#!/usr/bin/env bats

load helpers

@test "yaman container create" {
  cid=$(run_yaman_and_get_cid container create --rm "$DOCKER_ALPINE" -- echo "hello, world")

  run_yaman container list --all
  assert_success
  assert_output --regexp "$cid(.+)created"

  run_yaman container delete "$cid"
  assert_success
  assert_output ""
}
