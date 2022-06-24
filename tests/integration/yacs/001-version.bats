#!/usr/bin/env bats

load helpers

@test "yacs --version" {
  run_yacs --version
  assert_success
  assert_output --partial 'yacs version '
  assert_output --partial 'commit: '
  assert_output --partial 'spec: '
  assert_output --partial 'go: '
}
