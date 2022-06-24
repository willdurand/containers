#!/usr/bin/env bats

load helpers

@test "yacs with alpine bundle" {
  local sock=""
  local state=""

  run_yacs --bundle=/tmp/alpine-bundle --container-id=alpine-bats
  assert_success
  sock="$output"

  state=$(get_state "$sock")
  [ "alpine-bats" = "$(echo "$state" | jq -r '.ID')" ]
  [ "yacr" = $(echo "$state" | jq -r '.Runtime') ]
  [ "created" = $(echo "$state" | jq -r '.State.status') ]

  run curl -s -X POST -d 'cmd=start' --unix-socket "$sock" http://shim/
  assert_success
  state="$output"

  [ "running" = $(echo "$state" | jq -r '.State.status') ]

  run kill -9 $(echo "$state" | jq -r '.State.pid')
  assert_success

  state=$(get_state "$sock")
  [ "stopped" = $(echo "$state" | jq -r '.State.status') ]

  run curl -s -X DELETE --unix-socket "$sock" http://shim/
  assert_success
  assert_output "BYE"
}
