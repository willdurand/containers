#!/usr/bin/env bats

load helpers

@test "yaman container run --runtime=runc" {
  run_yaman container run --rm --runtime=runc "$DOCKER_ALPINE" -- id
  assert_success
  assert_output --partial "uid=0(root) gid=0(root)"
}

@test "yaman container run --runtime=runc --tty" {
  run_yaman container run --rm --runtime=runc --tty "$DOCKER_ALPINE" -- tty
  assert_success
  assert_output --partial "/dev/pts/0"
}
