#!/usr/bin/env bash

load '../base_helpers'

TIMEOUT=30s

DOCKER_ALPINE=docker.io/library/alpine
QUAY_ALPINE=quay.io/aptible/alpine

function run_yaman() {
  run timeout --foreground "$TIMEOUT" yaman "$@" 3> /dev/null
}

function random_string() {
  local length=${1:-10}

  head /dev/urandom | tr -dc A-Z0-9 | head -c"$length"
}
