#!/usr/bin/env bats

load helpers

@test "yaman image pull from docker.io" {
  run_yaman image pull "$DOCKER_ALPINE:3.16"
  assert_success
  assert_output --regexp "sha256:[a-z0-9]+"
}

@test "yaman image pull from quay.io" {
  run_yaman image pull "$QUAY_ALPINE"
  assert_success
  assert_output --regexp "sha256:[a-z0-9]+"
}

@test "yaman image list" {
  run_yaman image list
  assert_success
  assert_output --regexp "aptible/alpine\s+latest"
  assert_output --regexp "library/alpine\s+3\.16"
}
