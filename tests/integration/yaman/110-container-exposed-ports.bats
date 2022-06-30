#!/usr/bin/env bats

load helpers

@test "container run --publish-all" {
  # We need `--entrypoint` to by-pass an issue with rootless containers and
  # non-root users in containers.
  cid=$(run_yaman_and_get_cid container run -d --rm --entrypoint='["sh", "-c"]' --publish-all "$DOCKER_REDIS")

  run_yaman container list
  assert_success
  assert_output --regexp "$cid(.+)running"

  port=$(inspect "$cid" | jq '.ExposedPorts[0].HostPort')
  run bash -c "echo 'QUIT' | nc 127.0.0.1 $port"
  assert_success
  assert_output --partial "+OK"

  run_yaman container stop "$cid"
  assert_success
}
