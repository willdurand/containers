#!/usr/bin/env bats

load helpers

@test "yacs start and delete immediately" {
  local sock=""
  local state=""

  run_yacs --bundle=/tmp/alpine-bundle --container-id=alpine-bats
  assert_success
  sock="$output"

  state=$(get_state "$sock")
  [ "alpine-bats" = "$(echo "$state" | jq -r '.ID')" ]
  [ "yacr" = $(echo "$state" | jq -r '.Runtime') ]
  [ "created" = $(echo "$state" | jq -r '.State.status') ]
  
  pid=$(echo "$state" | jq -r '.State.pid')
  run ps -p "$pid" > /dev/null
  assert_success

  run curl -s -X DELETE --unix-socket "$sock" http://shim/
  assert_success
  assert_output "BYE"

  run ps -p "$pid" > /dev/null
  assert_failure
}
