#!/usr/bin/env bats

load helpers

function teardown() {
  rm -f config.json
}

@test "yacr spec" {
  run_yacr spec
  assert_success

  assert_equal "$(jq '.linux.uidMappings[0].containerID' config.json)" "null"
}

@test "yacr spec --rootless" {
  run_yacr spec --rootless
  assert_success

  assert_equal "$(jq '.linux.uidMappings[0].containerID' config.json)" "0"
  assert_equal "$(jq '.linux.uidMappings[0].size' config.json)" "1"
}
