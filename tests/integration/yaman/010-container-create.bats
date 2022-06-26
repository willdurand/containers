#!/usr/bin/env bats

load helpers

@test "yaman container create" {
  local cid=""
  run_yaman container create --rm "$DOCKER_ALPINE" -- echo "hello, world"
  assert_success
  cid="$output"

  run_yaman container list --all
  assert_success
  assert_output --regexp "$cid(.+)created"

  run_yaman container delete "$cid"
  assert_success
  assert_output ""
}
