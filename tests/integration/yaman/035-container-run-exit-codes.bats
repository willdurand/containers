#!/usr/bin/env bats

load helpers

@test "yaman container run sets its exit code to 125 for internal errors" {
  run_yaman container run --rm "$DOCKER_ALPINE" --invalid-flag
  assert_failure 125
}

@test "yaman container run sets its exit code to the process exit code" {
  run_yaman container run --rm "$DOCKER_ALPINE" -- sh -c 'exit 42'
  assert_failure 42
}

@test "yaman container run sets its exit code to 127 when command is not found" {
  run -127 yaman container run --rm "$DOCKER_ALPINE" -- /bin/invalid-program
}
