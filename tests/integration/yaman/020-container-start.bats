#!/usr/bin/env bats

load helpers

@test "yaman container start" {
  local cid=""
  run_yaman container create "$DOCKER_ALPINE" -- echo "hello, world"
  assert_success
  cid="$output"

  run_yaman container start "$cid"
  assert_success
  assert_output ""

  run_yaman container logs "$cid"
  assert_success
  assert_output "hello, world"

  run_yaman container delete "$cid"
  assert_success
}

@test "yaman container start --attach" {
  local cid=""
  run_yaman container create --rm "$DOCKER_ALPINE" -- echo "hello, world"
  assert_success
  cid="$output"

  run_yaman container start --attach "$cid"
  assert_success
  assert_output "hello, world"
}

@test "yaman container start --interactive" {
  local cid=""
  run_yaman container create --rm --interactive "$DOCKER_ALPINE" -- cat
  assert_success
  cid="$output"

  run bash -c "echo 'hello, world' | yaman container start --interactive $cid"
  assert_success
  assert_output "hello, world"
}

@test "yaman container start --interactive --attach" {
  local cid=""
  run_yaman container create --rm --interactive "$DOCKER_ALPINE" -- cat
  assert_success
  cid="$output"

  run bash -c "echo 'hello, world' | yaman container start --attach --interactive $cid"
  assert_success
  assert_output "hello, world"
}
