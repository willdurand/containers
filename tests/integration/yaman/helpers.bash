#!/usr/bin/env bash

load '../base_helpers'

TIMEOUT=30s

DOCKER_ALPINE=docker.io/library/alpine
DOCKER_HELLO_WORLD=docker.io/willdurand/hello-world
DOCKER_REDIS=docker.io/library/redis
QUAY_ALPINE=quay.io/aptible/alpine

function run_yaman() {
  run timeout --foreground "$TIMEOUT" yaman "$@"
}

function run_yaman_and_get_cid() {
  local cid=""

  run --separate-stderr timeout --foreground "$TIMEOUT" yaman "$@"
  assert_success
  cid="$output"

  echo "$cid"
}

function random_string() {
  local length=${1:-10}

  head /dev/urandom | tr -dc A-Z0-9 | head -c"$length"
}

function inspect() {
  local cid="$1"
  
  run yaman container inspect "$@"
  assert_success

  echo "$output"
}
