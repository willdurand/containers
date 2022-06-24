#!/usr/bin/env bash

load '../base_helpers'

function run_yacs() {
  run yacs "$@"
}

function get_state() {
  local sock="$1"
  
  run curl -s --unix-socket "$sock" http://shim/
  assert_success

  echo "$output"
}