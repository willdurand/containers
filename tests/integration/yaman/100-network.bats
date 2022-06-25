#!/usr/bin/env bats

load helpers

@test "container has internet access" {
  if [ "$CI" == "true" ]; then
    skip "not working in GitHub Actions"
  fi

  run_yaman container run --rm "$DOCKER_ALPINE" -- ping -c 1 1.1.1.1
  assert_success
  assert_output --partial "1 packets transmitted, 1 packets received"
}

@test "container DNS is configured" {
  if [ "$CI" == "true" ]; then
    skip "not working in GitHub Actions"
  fi

  run_yaman container run --rm "$DOCKER_ALPINE" -- ping -c 1 github.com
  assert_success
  assert_output --partial "1 packets transmitted, 1 packets received"
}

@test "/etc/resolv.conf should be created" {
  run_yaman container run --rm "$DOCKER_ALPINE" -- cat /etc/resolv.conf
  assert_success
  assert_output "nameserver 10.0.2.3"
}
