#!/usr/bin/env bats

load helpers

@test "yaman container delete --all" {
  cid1=$(run_yaman_and_get_cid container create "$DOCKER_ALPINE")
  cid2=$(run_yaman_and_get_cid container create "$QUAY_ALPINE")

  run_yaman container list --all
  assert_success
  assert_output --regexp "$cid1(.+)$DOCKER_ALPINE(.+)created"
  assert_output --regexp "$cid2(.+)$QUAY_ALPINE(.+)created"

  run_yaman container delete --all
  assert_success

  run_yaman container list --all
  assert_success
  refute_output --regexp "$cid1(.+)$DOCKER_ALPINE(.+)created"
  refute_output --regexp "$cid2(.+)$QUAY_ALPINE(.+)created"
}
