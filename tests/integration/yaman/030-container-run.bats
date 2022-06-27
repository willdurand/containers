#!/usr/bin/env bats

load helpers

@test "yaman container run" {
  value=$(random_string)
  run_yaman container run --rm "$DOCKER_ALPINE" -- echo "$value"
  assert_success
  assert_output "$value"
}

@test "yaman container run closes stdin" {
  value=$(random_string)
  run_yaman container run --rm --interactive "$DOCKER_ALPINE" -- cat <<<"$value"
  assert_success
  assert_output "$value"
}

@test "yaman container run with pipe" {
  value=$(random_string)
  run bash -c "echo $value | yaman container run --rm --interactive $DOCKER_ALPINE -- cat"
  assert_success
  assert_output "$value"
}

@test "yaman container run -d" {
  cid=$(run_yaman_and_get_cid container run --rm -d "$DOCKER_ALPINE" -- sleep 10)

  run_yaman container list
  assert_success
  assert_output --regexp "$cid(.+)running"

  run_yaman container stop "$cid"
  assert_success

  run_yaman container list
  refute_output --regexp "$cid(.+)running"
}

@test "yaman container run --hostname" {
  hostname="some-hostname"
  run_yaman container run --rm --hostname "$hostname" "$DOCKER_ALPINE" -- hostname
  assert_success
  assert_output "$hostname"
}

@test "yaman container run --name" {
  cid=$(run_yaman_and_get_cid container run -d --rm --name "some-name" "$DOCKER_ALPINE" -- sleep 10)

  run_yaman container list
  assert_success
  assert_output --regexp "$cid(.+)\ssome-name$"

  run_yaman container stop "$cid"
  assert_success
}

@test "yaman container run image without /etc/resolv.conf" {
  run_yaman container run --rm "$DOCKER_HELLO_WORLD"
  assert_success
  assert_output --partial "This message shows that your installation appears to be working correctly"
}
