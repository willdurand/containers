#!/usr/bin/env bats

load helpers

@test "yaman --version" {
  run_yaman --version
  assert_success
  assert_output --partial 'yaman version '
  assert_output --partial 'commit: '
  assert_output --partial 'spec: '
  assert_output --partial 'go: '
}
